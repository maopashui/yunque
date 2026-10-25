package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/collab"
	"gitee.com/yunque/yunque/backend/internal/config"
	"gitee.com/yunque/yunque/backend/internal/db"
	"gitee.com/yunque/yunque/backend/internal/docs"
	"gitee.com/yunque/yunque/backend/internal/httpapi"
	"gitee.com/yunque/yunque/backend/internal/userlookup"
)

func setupRouter(t *testing.T) (http.Handler, config.Config) {
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

	cfg := config.Config{
		CookieName:      "yunque_session",
		SessionHTTPOnly: true,
		CookieSecure:    false,
		AllowedOrigin:   "http://localhost:5173",
		Version:         "test",
		SessionTTL:      time.Hour,
	}
	authStore := auth.NewStore(sqlDB)
	svc := auth.NewService(authStore, cfg.SessionTTL)
	docsSvc := docs.NewService(docs.NewStore(sqlDB), userlookup.FromAuthStore(authStore))
	collabHub := collab.NewHub(collab.NewStore(sqlDB), nil)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpapi.NewRouter(cfg, svc, docsSvc, collabHub, logger), cfg
}

func TestHealth(t *testing.T) {
	router, _ := setupRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %#v", body)
	}
}

func TestAuthFlow(t *testing.T) {
	router, cfg := setupRouter(t)

	registerBody := `{"username":"carol","password":"password123","display_name":"Carol"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d body=%s", rec.Code, rec.Body.String())
	}

	loginBody := `{"username":"carol","password":"password123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == cfg.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected session cookie")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d", rec.Code)
	}
}

func TestMeUnauthorized(t *testing.T) {
	router, _ := setupRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}
