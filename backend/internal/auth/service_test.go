package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitee.com/yunque/yunque/backend/internal/db"
)

func setupService(t *testing.T) *Service {
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

	return NewService(NewStore(sqlDB), time.Hour)
}

func TestRegisterLoginAuthenticateLogout(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, RegisterInput{
		Username:    "alice",
		Password:    "password123",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("username = %q", user.Username)
	}
	if !user.IsAdmin {
		t.Fatal("expected first user to be admin")
	}
	if user.AvatarColor == "" {
		t.Fatal("expected avatar_color on register")
	}
	found := false
	for _, c := range AvatarPalette {
		if c == user.AvatarColor {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("avatar_color %q not in palette", user.AvatarColor)
	}

	_, err = svc.Register(ctx, RegisterInput{
		Username: "alice",
		Password: "password123",
	})
	if err != ErrUsernameTaken {
		t.Fatalf("expected username taken, got %v", err)
	}

	loggedIn, session, err := svc.Login(ctx, LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedIn.ID != user.ID {
		t.Fatalf("user id mismatch")
	}
	if session.ID == "" {
		t.Fatal("empty session id")
	}

	authed, _, err := svc.Authenticate(ctx, session.ID)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if authed.ID != user.ID {
		t.Fatalf("authed user mismatch")
	}

	if err := svc.Logout(ctx, session.ID); err != nil {
		t.Fatalf("logout: %v", err)
	}
	_, _, err = svc.Authenticate(ctx, session.ID)
	if err != ErrUnauthorized {
		t.Fatalf("expected unauthorized after logout, got %v", err)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Username: "bob",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, _, err = svc.Login(ctx, LoginInput{Username: "bob", Password: "wrong-pass"})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{Username: "ab", Password: "password123"})
	if err == nil {
		t.Fatal("expected username validation error")
	}

	_, err = svc.Register(ctx, RegisterInput{Username: "valid_user", Password: "short"})
	if err == nil {
		t.Fatal("expected password validation error")
	}
}

func TestRegistrationGateAndAdmin(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	enabled, err := svc.RegistrationEnabled(ctx)
	if err != nil || !enabled {
		t.Fatalf("empty db should allow registration: enabled=%v err=%v", enabled, err)
	}

	admin, err := svc.Register(ctx, RegisterInput{Username: "admin1", Password: "password123"})
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}
	if !admin.IsAdmin {
		t.Fatal("first user must be admin")
	}

	peer, err := svc.Register(ctx, RegisterInput{Username: "peer1", Password: "password123"})
	if err != nil {
		t.Fatalf("register peer while open: %v", err)
	}
	if peer.IsAdmin {
		t.Fatal("second user must not be admin")
	}

	_, err = svc.UpdateSiteSettings(ctx, admin, SiteSettings{RegistrationEnabled: false})
	if err != nil {
		t.Fatalf("close registration: %v", err)
	}
	enabled, err = svc.RegistrationEnabled(ctx)
	if err != nil || enabled {
		t.Fatalf("expected registration closed, enabled=%v err=%v", enabled, err)
	}

	_, err = svc.Register(ctx, RegisterInput{Username: "blocked", Password: "password123"})
	if err != ErrRegistrationClosed {
		t.Fatalf("expected registration closed, got %v", err)
	}

	_, err = svc.UpdateSiteSettings(ctx, peer, SiteSettings{RegistrationEnabled: true})
	if err != ErrForbidden {
		t.Fatalf("non-admin must not update settings, got %v", err)
	}

	users, err := svc.ListUsers(ctx, admin)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestDisableAndDeleteUser(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	admin, err := svc.Register(ctx, RegisterInput{Username: "admin1", Password: "password123"})
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}
	peer, err := svc.Register(ctx, RegisterInput{Username: "peer1", Password: "password123"})
	if err != nil {
		t.Fatalf("register peer: %v", err)
	}
	_, session, err := svc.Login(ctx, LoginInput{Username: "peer1", Password: "password123"})
	if err != nil {
		t.Fatalf("login peer: %v", err)
	}

	_, err = svc.SetUserDisabled(ctx, admin, admin.ID, true)
	if err != ErrCannotModifySelf {
		t.Fatalf("expected cannot modify self, got %v", err)
	}
	err = svc.DeleteUser(ctx, admin, admin.ID)
	if err != ErrCannotModifySelf {
		t.Fatalf("expected cannot delete self, got %v", err)
	}

	item, err := svc.SetUserDisabled(ctx, admin, peer.ID, true)
	if err != nil {
		t.Fatalf("disable peer: %v", err)
	}
	if item.DisabledAt == nil {
		t.Fatal("expected disabled_at set")
	}
	_, _, err = svc.Login(ctx, LoginInput{Username: "peer1", Password: "password123"})
	if err != ErrAccountDisabled {
		t.Fatalf("expected account disabled, got %v", err)
	}
	_, _, err = svc.Authenticate(ctx, session.ID)
	if err != ErrUnauthorized {
		t.Fatalf("expected session unauthorized after disable, got %v", err)
	}

	item, err = svc.SetUserDisabled(ctx, admin, peer.ID, false)
	if err != nil {
		t.Fatalf("enable peer: %v", err)
	}
	if item.DisabledAt != nil {
		t.Fatal("expected disabled_at cleared")
	}

	err = svc.DeleteUser(ctx, admin, peer.ID)
	if err != nil {
		t.Fatalf("delete peer: %v", err)
	}
	users, err := svc.ListUsers(ctx, admin)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user after delete, got %d", len(users))
	}

	err = svc.DeleteUser(ctx, admin, admin.ID)
	if err != ErrCannotModifySelf {
		t.Fatalf("expected cannot delete self, got %v", err)
	}
}
