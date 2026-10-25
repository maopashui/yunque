package docs

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/db"
)

type testUserLookup struct {
	store *auth.Store
}

func (l testUserLookup) LookupByUsername(ctx context.Context, username string) (UserRef, error) {
	u, err := l.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return UserRef{}, ErrNotFound
		}
		return UserRef{}, err
	}
	return UserRef{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarColor: auth.ResolveAvatarColor(u.ID, u.AvatarColor),
	}, nil
}

func setupDocs(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	migrationsDir := filepath.Join("..", "..", "migrations")
	if err := db.Migrate(sqlDB, migrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	authStore := auth.NewStore(sqlDB)
	authSvc := auth.NewService(authStore, time.Hour)
	user, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "docuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return NewService(NewStore(sqlDB), testUserLookup{store: authStore}), user.ID
}

func TestDocumentAndBlockCRUD(t *testing.T) {
	svc, userID := setupDocs(t)
	ctx := context.Background()

	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{ActorID: userID, Title: "Hello"})
	if err != nil {
		t.Fatalf("create doc: %v", err)
	}

	content, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "hi"}},
	})
	block, doc, err := svc.CreateBlock(ctx, CreateBlockInput{
		ActorID:    userID,
		DocumentID: doc.ID,
		BlockType:  BlockParagraph,
		Content:    content,
	})
	if err != nil {
		t.Fatalf("create block: %v", err)
	}
	if doc.BlockCount != 1 {
		t.Fatalf("block_count=%d", doc.BlockCount)
	}

	updatedContent, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "updated"}},
	})
	base := block.Revision
	block, doc, err = svc.UpdateBlock(ctx, UpdateBlockInput{
		ActorID:      userID,
		DocumentID:   doc.ID,
		BlockID:      block.ID,
		Content:      updatedContent,
		BaseRevision: &base,
	})
	if err != nil {
		t.Fatalf("update block: %v", err)
	}
	if block.Revision != 2 {
		t.Fatalf("revision=%d", block.Revision)
	}

	blocks, err := svc.ListBlocks(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("list blocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("len=%d", len(blocks))
	}

	if err := svc.DeleteDocument(ctx, userID, doc.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	restored, err := svc.RestoreDocument(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Fatal("expected restored")
	}
}

func TestBlockIndexAndTOC(t *testing.T) {
	svc, userID := setupDocs(t)
	ctx := context.Background()

	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{ActorID: userID, Title: "TOC Doc"})
	if err != nil {
		t.Fatal(err)
	}
	headingContent, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "第一章"}},
	})
	headingAttrs, _ := json.Marshal(map[string]any{"level": 1})
	_, _, err = svc.CreateBlock(ctx, CreateBlockInput{
		ActorID:    userID,
		DocumentID: doc.ID,
		BlockType:  BlockHeading,
		Content:    headingContent,
		Attrs:      headingAttrs,
	})
	if err != nil {
		t.Fatal(err)
	}
	para, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "body"}},
	})
	_, _, err = svc.CreateBlock(ctx, CreateBlockInput{
		ActorID:    userID,
		DocumentID: doc.ID,
		BlockType:  BlockParagraph,
		Content:    para,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, index, err := svc.ListBlockIndex(ctx, userID, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(index) != 2 {
		t.Fatalf("index len=%d", len(index))
	}
	toc, err := svc.ListTOC(ctx, userID, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(toc) != 1 || toc[0].Text != "第一章" {
		t.Fatalf("toc=%+v", toc)
	}
}

func TestSyncContent(t *testing.T) {
	svc, userID := setupDocs(t)
	ctx := context.Background()

	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{ActorID: userID, Title: "Edit Me"})
	if err != nil {
		t.Fatal(err)
	}
	base := doc.Revision
	content, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "hello"}},
	})
	doc, blocks, err := svc.SyncContent(ctx, SyncContentInput{
		ActorID:      userID,
		DocumentID:   doc.ID,
		BaseRevision: &base,
		Blocks: []ContentBlockInput{{
			BlockType: BlockParagraph,
			Content:   content,
			Attrs:     json.RawMessage(`{}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.BlockCount != 1 || len(blocks) != 1 {
		t.Fatalf("count=%d blocks=%d", doc.BlockCount, len(blocks))
	}
	stale := base
	_, _, err = svc.SyncContent(ctx, SyncContentInput{
		ActorID:      userID,
		DocumentID:   doc.ID,
		BaseRevision: &stale,
		Blocks:       []ContentBlockInput{},
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestShareMemberCanListDocument(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	authStore := auth.NewStore(sqlDB)
	authSvc := auth.NewService(authStore, time.Hour)
	owner, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "owner_a", Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	peer, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "peer_b", Password: "password123", DisplayName: "Peer B",
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(NewStore(sqlDB), testUserLookup{store: authStore})
	ctx := context.Background()
	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{ActorID: owner.ID, Title: "Shared"})
	if err != nil {
		t.Fatal(err)
	}

	listed, err := svc.ListDocuments(ctx, ListDocumentsQuery{UserID: peer.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Documents) != 0 || listed.Total != 0 {
		t.Fatalf("peer should see 0 docs before share, got %d total=%d", len(listed.Documents), listed.Total)
	}

	member, err := svc.AddMember(ctx, AddMemberInput{
		ActorID: owner.ID, DocumentID: doc.ID, Username: "peer_b", Role: RoleEditor,
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if member.UserID != peer.ID || member.Role != RoleEditor {
		t.Fatalf("member=%+v", member)
	}

	listed, err = svc.ListDocuments(ctx, ListDocumentsQuery{UserID: peer.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Documents) != 1 || listed.Total != 1 || listed.Documents[0].ID != doc.ID || listed.Documents[0].Role != RoleEditor {
		t.Fatalf("after share listed=%+v", listed)
	}

	got, err := svc.GetDocument(ctx, peer.ID, doc.ID)
	if err != nil {
		t.Fatalf("peer get: %v", err)
	}
	if got.Role != RoleEditor {
		t.Fatalf("role=%s", got.Role)
	}

	members, err := svc.ListMembers(ctx, owner.ID, doc.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("members=%d err=%v", len(members), err)
	}

	if err := svc.RemoveMember(ctx, owner.ID, doc.ID, peer.ID); err != nil {
		t.Fatal(err)
	}
	listed, err = svc.ListDocuments(ctx, ListDocumentsQuery{UserID: peer.ID})
	if err != nil || len(listed.Documents) != 0 || listed.Total != 0 {
		t.Fatalf("after remove listed=%d total=%d err=%v", len(listed.Documents), listed.Total, err)
	}
}

func TestFavoriteIsPerUser(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(filepath.Join(dir, "fav.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	authStore := auth.NewStore(sqlDB)
	authSvc := auth.NewService(authStore, time.Hour)
	owner, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "fav_owner", Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	peer, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "fav_peer", Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewStore(sqlDB), testUserLookup{store: authStore})
	ctx := context.Background()
	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{ActorID: owner.ID, Title: "FavDoc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMember(ctx, AddMemberInput{
		ActorID: owner.ID, DocumentID: doc.ID, Username: "fav_peer", Role: RoleViewer,
	}); err != nil {
		t.Fatal(err)
	}

	patched, err := svc.PatchDocument(ctx, PatchDocumentInput{
		ActorID: owner.ID, DocumentID: doc.ID, IsFavorite: boolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !patched.IsFavorite {
		t.Fatal("owner should see favorite")
	}

	peerDoc, err := svc.GetDocument(ctx, peer.ID, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if peerDoc.IsFavorite {
		t.Fatal("peer must not inherit owner's favorite")
	}

	ownerList, err := svc.ListDocuments(ctx, ListDocumentsQuery{UserID: owner.ID, Favorite: true})
	if err != nil || len(ownerList.Documents) != 1 {
		t.Fatalf("owner favorites=%d err=%v", len(ownerList.Documents), err)
	}
	peerList, err := svc.ListDocuments(ctx, ListDocumentsQuery{UserID: peer.ID, Favorite: true})
	if err != nil || len(peerList.Documents) != 0 {
		t.Fatalf("peer favorites=%d err=%v", len(peerList.Documents), err)
	}

	if _, err := svc.PatchDocument(ctx, PatchDocumentInput{
		ActorID: peer.ID, DocumentID: doc.ID, IsFavorite: boolPtr(true),
	}); err != nil {
		t.Fatalf("viewer favorite: %v", err)
	}
	peerList, err = svc.ListDocuments(ctx, ListDocumentsQuery{UserID: peer.ID, Favorite: true})
	if err != nil || len(peerList.Documents) != 1 {
		t.Fatalf("peer favorites after=%d err=%v", len(peerList.Documents), err)
	}
	ownerAgain, err := svc.GetDocument(ctx, owner.ID, doc.ID)
	if err != nil || !ownerAgain.IsFavorite {
		t.Fatalf("owner favorite should stay true: %+v err=%v", ownerAgain, err)
	}
}

func boolPtr(v bool) *bool { return &v }

func TestSnapshotRestorePreservesHistory(t *testing.T) {
	svc, userID := setupDocs(t)
	ctx := context.Background()

	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{ActorID: userID, Title: "Hist"})
	if err != nil {
		t.Fatal(err)
	}
	base := doc.Revision
	content1, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "v1"}},
	})
	doc, blocks, err := svc.SyncContent(ctx, SyncContentInput{
		ActorID: userID, DocumentID: doc.ID, BaseRevision: &base,
		Blocks: []ContentBlockInput{{BlockType: BlockParagraph, Content: content1, Attrs: json.RawMessage(`{}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d", len(blocks))
	}

	snaps, err := svc.ListSnapshots(ctx, userID, doc.ID)
	if err != nil || len(snaps) < 1 {
		t.Fatalf("auto snaps=%d err=%v", len(snaps), err)
	}
	firstID := snaps[0].ID

	content2, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "v2"}},
	})
	rev := doc.Revision
	doc, _, err = svc.SyncContent(ctx, SyncContentInput{
		ActorID: userID, DocumentID: doc.ID, BaseRevision: &rev,
		Blocks: []ContentBlockInput{{BlockType: BlockParagraph, Content: content2, Attrs: json.RawMessage(`{}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}

	diff, err := svc.CompareSnapshot(ctx, userID, doc.ID, firstID)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changed)+len(diff.Added)+len(diff.Removed) == 0 {
		t.Fatalf("expected content diff, got %+v", diff)
	}
	for _, e := range diff.Added {
		if e.ID == "" {
			t.Fatalf("added entry missing id: %+v", e)
		}
	}
	for _, e := range diff.Removed {
		if e.ID == "" {
			t.Fatalf("removed entry missing id: %+v", e)
		}
	}
	for _, e := range diff.Changed {
		if e.ID == "" || e.FromPreview == "" || e.ToPreview == "" {
			t.Fatalf("changed entry incomplete: %+v", e)
		}
	}

	beforeCount, err := svc.ListSnapshots(ctx, userID, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	doc, restored, _, err := svc.RestoreSnapshot(ctx, userID, doc.ID, firstID)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 {
		t.Fatalf("restored blocks=%d", len(restored))
	}
	text := string(restored[0].Content)
	if !strings.Contains(text, "v1") {
		t.Fatalf("expected v1 content, got %s", text)
	}
	afterCount, err := svc.ListSnapshots(ctx, userID, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterCount) <= len(beforeCount) {
		t.Fatalf("restore should add history snapshots: before=%d after=%d", len(beforeCount), len(afterCount))
	}
	_ = doc
}

func TestDiffSnapshotsIgnoresRegeneratedIDs(t *testing.T) {
	same, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "keep me"}},
	})
	oldLine, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "old line"}},
	})
	newLine, _ := json.Marshal(map[string]any{
		"inlines": []map[string]any{{"type": "text", "text": "new line"}},
	})
	// Simulate materialize regenerating UUIDs while only one paragraph changed.
	from := SnapshotPayload{
		Title: "Doc",
		Blocks: []Block{
			{ID: "old-a", BlockType: BlockParagraph, Content: same, Attrs: json.RawMessage(`{}`)},
			{ID: "old-b", BlockType: BlockParagraph, Content: oldLine, Attrs: json.RawMessage(`{}`)},
			{ID: "old-c", BlockType: BlockParagraph, Content: same, Attrs: json.RawMessage(`{}`)},
		},
	}
	to := SnapshotPayload{
		Title: "Doc",
		Blocks: []Block{
			{ID: "new-a", BlockType: BlockParagraph, Content: same, Attrs: json.RawMessage(`{}`)},
			{ID: "new-b", BlockType: BlockParagraph, Content: newLine, Attrs: json.RawMessage(`{}`)},
			{ID: "new-c", BlockType: BlockParagraph, Content: same, Attrs: json.RawMessage(`{}`)},
		},
	}
	diff := diffSnapshots(from, to)
	if diff.Unchanged != 2 {
		t.Fatalf("unchanged=%d want 2, diff=%+v", diff.Unchanged, diff)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Preview != "old line" {
		t.Fatalf("removed=%+v", diff.Removed)
	}
	if len(diff.Added) != 1 || diff.Added[0].Preview != "new line" {
		t.Fatalf("added=%+v", diff.Added)
	}
}
