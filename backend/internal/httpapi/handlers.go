package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/collab"
	"gitee.com/yunque/yunque/backend/internal/config"
	"gitee.com/yunque/yunque/backend/internal/docs"
)

type Handlers struct {
	Auth    *auth.Service
	Docs    *docs.Service
	Collab  *collab.Hub
	Config  config.Config
	Version string
}

type registerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": h.Version,
	})
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	user, err := h.Auth.Register(r.Context(), auth.RegisterInput{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": user.Public(),
	})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	user, session, err := h.Auth.Login(r.Context(), auth.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	h.setSessionCookie(w, session.ID, session.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"user": user.Public(),
	})
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.Config.CookieName)
	if err == nil && cookie.Value != "" {
		_ = h.Auth.Logout(r.Context(), cookie.Value)
	}
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": user.Public(),
	})
}

func (h *Handlers) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, auth.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username_taken", "username already taken")
	case errors.Is(err, auth.ErrRegistrationClosed):
		writeError(w, http.StatusForbidden, "registration_closed", "registration is closed")
	case errors.Is(err, auth.ErrAccountDisabled):
		writeError(w, http.StatusForbidden, "account_disabled", "account is disabled")
	case errors.Is(err, auth.ErrCannotModifySelf):
		writeError(w, http.StatusBadRequest, "cannot_modify_self", "cannot modify your own account")
	case errors.Is(err, auth.ErrLastAdmin):
		writeError(w, http.StatusConflict, "last_admin", "cannot disable or delete the last admin")
	case errors.Is(err, auth.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "not_found", "user not found")
	case errors.Is(err, auth.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
	case errors.Is(err, auth.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (h *Handlers) setSessionCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.CookieName,
		Value:    value,
		Path:     h.Config.CookiePath(),
		Expires:  expires,
		HttpOnly: h.Config.SessionHTTPOnly,
		Secure:   h.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.Config.CookieName,
		Value:    "",
		Path:     h.Config.CookiePath(),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: h.Config.SessionHTTPOnly,
		Secure:   h.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
