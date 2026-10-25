package collab

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/db"
	"gitee.com/yunque/yunque/backend/internal/docs"
	"gitee.com/yunque/yunque/backend/internal/userlookup"
)

func setupCollab(t *testing.T) (*Hub, *Store, string, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	authStore := auth.NewStore(sqlDB)
	authSvc := auth.NewService(authStore, time.Hour)
	user, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "collabuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	docsSvc := docs.NewService(docs.NewStore(sqlDB), userlookup.FromAuthStore(authStore))
	doc, err := docsSvc.CreateDocument(context.Background(), docs.CreateDocumentInput{
		ActorID: user.ID,
		Title:   "Collab Doc",
	})
	if err != nil {
		t.Fatalf("create doc: %v", err)
	}

	store := NewStore(sqlDB)
	hub := NewHub(store, nil)
	return hub, store, doc.ID, user.ID
}

func TestAppendDedupAndListAfter(t *testing.T) {
	_, store, docID, userID := setupCollab(t)
	ctx := context.Background()

	u1, err := store.AppendUpdate(ctx, docID, "op-1", userID, []byte{0x01, 0x02})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := store.AppendUpdate(ctx, docID, "op-1", userID, []byte{0x01, 0x02}); err == nil {
		t.Fatal("expected duplicate error")
	}

	u2, err := store.AppendUpdate(ctx, docID, "op-2", userID, []byte{0x03})
	if err != nil {
		t.Fatalf("append2: %v", err)
	}

	after, err := store.ListUpdatesAfter(ctx, docID, u1.Seq)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(after) != 1 || after[0].OperationID != "op-2" {
		t.Fatalf("after=%+v", after)
	}
	_ = u2
}

func TestHubBroadcastAndReconnectSync(t *testing.T) {
	hub, store, docID, userID := setupCollab(t)
	ctx := context.Background()

	a := hub.AttachClient(docID, userID, true)
	b := hub.AttachClient(docID, userID, true)
	defer hub.DetachClient(docID, a)
	defer hub.DetachClient(docID, b)

	payload := base64.StdEncoding.EncodeToString([]byte{0x0a, 0x0b})
	if err := hub.HandleUpdate(ctx, docID, userID, Message{
		Type:        TypeUpdate,
		OperationID: "upd-1",
		Update:      payload,
	}, a); err != nil {
		t.Fatalf("handle: %v", err)
	}

	select {
	case msg := <-b.Outgoing():
		if msg.Type != TypeUpdate || msg.OperationID != "upd-1" {
			t.Fatalf("got %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting broadcast")
	}

	// reconnect: new client should see persisted update via BuildSyncMessage
	hub.DetachClient(docID, a)
	hub.DetachClient(docID, b)

	sync, empty, err := hub.BuildSyncMessage(ctx, docID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if empty {
		t.Fatal("expected non-empty")
	}
	if len(sync.Updates) != 1 {
		t.Fatalf("updates=%d", len(sync.Updates))
	}

	n, err := store.CountUpdates(ctx, docID)
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}

func TestSnapshotCompactionLoad(t *testing.T) {
	hub, store, docID, userID := setupCollab(t)
	ctx := context.Background()

	_, _ = store.AppendUpdate(ctx, docID, "a", userID, []byte{1})
	u2, _ := store.AppendUpdate(ctx, docID, "b", userID, []byte{2})

	c := hub.AttachClient(docID, userID, true)
	defer hub.DetachClient(docID, c)

	snapB64 := base64.StdEncoding.EncodeToString([]byte{0xff, 0xfe})
	if err := hub.HandleSnapshot(ctx, docID, Message{
		Type:      TypeSnapshot,
		Snapshot:  snapB64,
		UpdateSeq: u2.Seq,
	}, c); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	sync, empty, err := hub.BuildSyncMessage(ctx, docID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if empty || sync.Snapshot == "" {
		t.Fatalf("expected snapshot, empty=%v", empty)
	}
	// updates after snapshot seq should be empty
	if len(sync.Updates) != 0 {
		t.Fatalf("expected no updates after snapshot, got %d", len(sync.Updates))
	}
}
