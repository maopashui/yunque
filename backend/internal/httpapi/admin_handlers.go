package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) RegistrationStatus(w http.ResponseWriter, r *http.Request) {
	enabled, err := h.Auth.RegistrationEnabled(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled})
}

func (h *Handlers) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	settings, err := h.Auth.GetSiteSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

type patchAdminSettingsRequest struct {
	RegistrationEnabled *bool `json:"registration_enabled"`
}

func (h *Handlers) PatchAdminSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	var req patchAdminSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	current, err := h.Auth.GetSiteSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	next := current
	if req.RegistrationEnabled != nil {
		next.RegistrationEnabled = *req.RegistrationEnabled
	}
	settings, err := h.Auth.UpdateSiteSettings(r.Context(), user, next)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *Handlers) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	users, err := h.Auth.ListUsers(r.Context(), user)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

type patchAdminUserRequest struct {
	Disabled *bool `json:"disabled"`
}

func (h *Handlers) PatchAdminUser(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	targetID := chi.URLParam(r, "userId")
	var req patchAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Disabled == nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "disabled is required")
		return
	}
	item, err := h.Auth.SetUserDisabled(r.Context(), user, targetID, *req.Disabled)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": item})
}

func (h *Handlers) DeleteAdminUser(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	targetID := chi.URLParam(r, "userId")
	if err := h.Auth.DeleteUser(r.Context(), user, targetID); err != nil {
		h.writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
