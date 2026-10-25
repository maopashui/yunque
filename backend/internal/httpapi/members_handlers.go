package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitee.com/yunque/yunque/backend/internal/docs"
)

func (h *Handlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	members, err := h.Docs.ListMembers(r.Context(), user.ID, docID)
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

type addMemberRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *Handlers) AddMember(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	member, err := h.Docs.AddMember(r.Context(), docs.AddMemberInput{
		ActorID:    user.ID,
		DocumentID: docID,
		Username:   req.Username,
		Role:       docs.Role(req.Role),
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"member": member})
}

type patchMemberRequest struct {
	Role string `json:"role"`
}

func (h *Handlers) UpdateMember(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	memberUserID := chi.URLParam(r, "userId")
	if memberUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "user id required")
		return
	}
	var req patchMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	member, err := h.Docs.UpdateMemberRole(r.Context(), docs.UpdateMemberInput{
		ActorID:    user.ID,
		DocumentID: docID,
		UserID:     memberUserID,
		Role:       docs.Role(req.Role),
	})
	if err != nil {
		h.writeDocsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"member": member})
}

func (h *Handlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	user, docID, ok := h.actorDoc(w, r)
	if !ok {
		return
	}
	memberUserID := chi.URLParam(r, "userId")
	if memberUserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "user id required")
		return
	}
	if err := h.Docs.RemoveMember(r.Context(), user.ID, docID, memberUserID); err != nil {
		h.writeDocsError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
