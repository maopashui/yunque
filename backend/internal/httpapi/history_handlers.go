package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gitee.com/yunque/yunque/backend/internal/docs"
)

func (h *Handlers) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	list, err := h.Docs.ListSnapshots(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	if list == nil {
		list = []docs.SnapshotMeta{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshots": list})
}

func (h *Handlers) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	snapID := chi.URLParam(r, "snapshotId")
	detail, err := h.Docs.GetSnapshot(r.Context(), user.ID, docID, snapID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": detail})
}

type createSnapshotRequest struct {
	Label string `json:"label"`
}

func (h *Handlers) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var req createSnapshotRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	meta, err := h.Docs.CreateNamedSnapshot(r.Context(), user.ID, docID, req.Label)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"snapshot": meta})
}

func (h *Handlers) CompareSnapshot(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	snapID := chi.URLParam(r, "snapshotId")
	diff, err := h.Docs.CompareSnapshot(r.Context(), user.ID, docID, snapID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"diff": diff})
}

func (h *Handlers) RestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	snapID := chi.URLParam(r, "snapshotId")
	doc, blocks, snap, err := h.Docs.RestoreSnapshot(r.Context(), user.ID, docID, snapID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"document": doc,
		"blocks":   blocks,
		"snapshot": snap,
	})
}

func (h *Handlers) CleanupSnapshots(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	keep := 50
	if v := r.URL.Query().Get("keep"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			keep = n
		}
	}
	deleted, err := h.Docs.CleanupSnapshots(r.Context(), user.ID, docID, keep)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted, "keep": keep})
}
