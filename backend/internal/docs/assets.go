package docs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultMaxAssetBytes = 5 << 20 // 5 MiB

type Asset struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	UploaderID string    `json:"uploader_id"`
	Filename   string    `json:"filename"`
	MIME       string    `json:"mime"`
	SizeBytes  int64     `json:"size"`
	Ext        string    `json:"-"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Service) ConfigureAssets(dir string, maxBytes int64) {
	if strings.TrimSpace(dir) != "" {
		s.assetsDir = dir
	}
	if maxBytes > 0 {
		s.maxAssetBytes = maxBytes
	}
}

func (s *Service) assetsRoot() string {
	if s.assetsDir != "" {
		return s.assetsDir
	}
	return "data/assets"
}

func (s *Service) assetMax() int64 {
	if s.maxAssetBytes > 0 {
		return s.maxAssetBytes
	}
	return DefaultMaxAssetBytes
}

func (s *Service) assetDiskPath(documentID, assetID, ext string) string {
	return filepath.Join(s.assetsRoot(), documentID, assetID+ext)
}

func AssetPublicURL(documentID, assetID string) string {
	return fmt.Sprintf("/api/documents/%s/assets/%s", documentID, assetID)
}

var allowedImageMIME = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func extForMIME(mime string) (string, bool) {
	ext, ok := allowedImageMIME[mime]
	return ext, ok
}

type UploadAssetInput struct {
	ActorID    string
	DocumentID string
	Filename   string
	Body       io.Reader
}

func (s *Service) UploadAsset(ctx context.Context, in UploadAssetInput) (Asset, error) {
	if _, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanEdit); err != nil {
		return Asset{}, err
	}

	max := s.assetMax()
	limited := io.LimitReader(in.Body, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Asset{}, fmt.Errorf("%w: read upload failed", ErrInvalidInput)
	}
	if int64(len(data)) > max {
		return Asset{}, fmt.Errorf("%w: file too large (max %d bytes)", ErrInvalidInput, max)
	}
	if len(data) == 0 {
		return Asset{}, fmt.Errorf("%w: empty file", ErrInvalidInput)
	}

	mime := http.DetectContentType(data)
	// DetectContentType may return "image/jpeg" etc.; normalize sniff edge cases.
	if mime == "application/octet-stream" || mime == "text/plain; charset=utf-8" {
		// Prefer extension hint only when sniff fails for webp etc.
		lower := strings.ToLower(in.Filename)
		switch {
		case strings.HasSuffix(lower, ".webp"):
			mime = "image/webp"
		case strings.HasSuffix(lower, ".png"):
			mime = "image/png"
		case strings.HasSuffix(lower, ".gif"):
			mime = "image/gif"
		case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
			mime = "image/jpeg"
		}
	}
	ext, ok := extForMIME(mime)
	if !ok {
		return Asset{}, fmt.Errorf("%w: unsupported image type", ErrInvalidInput)
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	asset := Asset{
		ID:         id,
		DocumentID: in.DocumentID,
		UploaderID: in.ActorID,
		Filename:   filepath.Base(strings.TrimSpace(in.Filename)),
		MIME:       mime,
		SizeBytes:  int64(len(data)),
		Ext:        ext,
		URL:        AssetPublicURL(in.DocumentID, id),
		CreatedAt:  now,
	}

	dir := filepath.Join(s.assetsRoot(), in.DocumentID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return Asset{}, err
	}
	path := s.assetDiskPath(in.DocumentID, id, ext)
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return Asset{}, err
	}
	if err := s.store.InsertAsset(ctx, asset); err != nil {
		_ = os.Remove(path)
		return Asset{}, err
	}
	return asset, nil
}

// OpenAsset returns metadata and an open file for a document asset.
// Caller must close the returned ReadCloser.
func (s *Service) OpenAsset(ctx context.Context, actorID, documentID, assetID string) (Asset, io.ReadCloser, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return Asset{}, nil, err
	}
	asset, err := s.store.GetAsset(ctx, documentID, assetID)
	if err != nil {
		return Asset{}, nil, err
	}
	asset.URL = AssetPublicURL(documentID, assetID)
	path := s.assetDiskPath(documentID, assetID, asset.Ext)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Asset{}, nil, ErrNotFound
		}
		return Asset{}, nil, err
	}
	return asset, f, nil
}
