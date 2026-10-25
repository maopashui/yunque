package docs

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/db"
)

func TestUploadAndOpenAssetRequiresRole(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(filepath.Join(dir, "t.db"))
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
		Username: "owner_a", Password: "password123", DisplayName: "Owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Username: "stranger_b", Password: "password123", DisplayName: "Stranger",
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(NewStore(sqlDB), testUserLookup{store: authStore})
	assetsDir := filepath.Join(dir, "assets")
	svc.ConfigureAssets(assetsDir, 1<<20)

	doc, err := svc.CreateDocument(context.Background(), CreateDocumentInput{
		ActorID: owner.ID,
		Title:   "img doc",
	})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	asset, err := svc.UploadAsset(context.Background(), UploadAssetInput{
		ActorID:    owner.ID,
		DocumentID: doc.ID,
		Filename:   "dot.png",
		Body:       bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if asset.URL == "" || asset.MIME != "image/png" {
		t.Fatalf("asset=%+v", asset)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, doc.ID, asset.ID+".png")); err != nil {
		t.Fatalf("disk file: %v", err)
	}

	_, err = svc.UploadAsset(context.Background(), UploadAssetInput{
		ActorID:    stranger.ID,
		DocumentID: doc.ID,
		Filename:   "nope.png",
		Body:       bytes.NewReader(buf.Bytes()),
	})
	if err != ErrForbidden {
		t.Fatalf("stranger upload want forbidden, got %v", err)
	}

	_, rc, err := svc.OpenAsset(context.Background(), owner.ID, doc.ID, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(got, buf.Bytes()) {
		t.Fatalf("bytes mismatch len=%d", len(got))
	}

	_, _, err = svc.OpenAsset(context.Background(), stranger.ID, doc.ID, asset.ID)
	if err != ErrForbidden {
		t.Fatalf("stranger open want forbidden, got %v", err)
	}
}
