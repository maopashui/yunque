package httpapi

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitee.com/yunque/yunque/backend/internal/docs"
)

func (h *Handlers) UploadDocumentAsset(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}

	max := h.Config.MaxAssetBytes
	if max <= 0 {
		max = docs.DefaultMaxAssetBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, max+1024)
	if err := r.ParseMultipartForm(max + 1024); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid multipart body or file too large")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "file field required")
		return
	}
	defer file.Close()

	filename := ""
	if header != nil {
		filename = header.Filename
	}
	asset, err := h.Docs.UploadAsset(r.Context(), docs.UploadAssetInput{
		ActorID:    user.ID,
		DocumentID: docID,
		Filename:   filename,
		Body:       file,
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"asset": asset})
}

func (h *Handlers) GetDocumentAsset(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	assetID := chi.URLParam(r, "assetId")
	if assetID == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "asset id required")
		return
	}

	asset, rc, err := h.Docs.OpenAsset(r.Context(), user.ID, docID, assetID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", asset.MIME)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}
