package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrRegistrationClosed = errors.New("registration closed")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrCannotModifySelf   = errors.New("cannot modify self")
	ErrLastAdmin          = errors.New("cannot modify last admin")
)

const settingRegistrationEnabled = "registration_enabled"

type Service struct {
	store      *Store
	sessionTTL time.Duration
}

func NewService(store *Store, sessionTTL time.Duration) *Service {
	return &Service{store: store, sessionTTL: sessionTTL}
}

type RegisterInput struct {
	Username    string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Username string
	Password string
}

func (s *Service) RegistrationEnabled(ctx context.Context) (bool, error) {
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return true, nil
	}
	return s.registrationSetting(ctx)
}

func (s *Service) GetSiteSettings(ctx context.Context) (SiteSettings, error) {
	enabled, err := s.registrationSetting(ctx)
	if err != nil {
		return SiteSettings{}, err
	}
	return SiteSettings{RegistrationEnabled: enabled}, nil
}

func (s *Service) UpdateSiteSettings(ctx context.Context, actor User, in SiteSettings) (SiteSettings, error) {
	if !actor.IsAdmin {
		return SiteSettings{}, ErrForbidden
	}
	value := "0"
	if in.RegistrationEnabled {
		value = "1"
	}
	if err := s.store.SetSetting(ctx, settingRegistrationEnabled, value); err != nil {
		return SiteSettings{}, err
	}
	return SiteSettings{RegistrationEnabled: in.RegistrationEnabled}, nil
}

func (s *Service) ListUsers(ctx context.Context, actor User) ([]UserListItem, error) {
	if !actor.IsAdmin {
		return nil, ErrForbidden
	}
	return s.store.ListUsers(ctx)
}

func (s *Service) SetUserDisabled(ctx context.Context, actor User, targetID string, disabled bool) (UserListItem, error) {
	if !actor.IsAdmin {
		return UserListItem{}, ErrForbidden
	}
	if targetID == actor.ID {
		return UserListItem{}, ErrCannotModifySelf
	}
	target, err := s.store.GetUserByID(ctx, targetID)
	if err != nil {
		return UserListItem{}, err
	}
	if disabled && target.IsAdmin && !target.Disabled() {
		if err := s.ensureNotLastActiveAdmin(ctx, target.ID); err != nil {
			return UserListItem{}, err
		}
	}

	now := time.Now().UTC()
	var at *time.Time
	if disabled {
		at = &now
	}
	if err := s.store.SetUserDisabledAt(ctx, targetID, at); err != nil {
		return UserListItem{}, err
	}
	if disabled {
		_ = s.store.RevokeAllSessionsForUser(ctx, targetID, now)
	}
	target, err = s.store.GetUserByID(ctx, targetID)
	if err != nil {
		return UserListItem{}, err
	}
	return userToListItem(target), nil
}

func (s *Service) DeleteUser(ctx context.Context, actor User, targetID string) error {
	if !actor.IsAdmin {
		return ErrForbidden
	}
	if targetID == actor.ID {
		return ErrCannotModifySelf
	}
	target, err := s.store.GetUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.IsAdmin && !target.Disabled() {
		if err := s.ensureNotLastActiveAdmin(ctx, target.ID); err != nil {
			return err
		}
	}
	_ = s.store.RevokeAllSessionsForUser(ctx, targetID, time.Now().UTC())
	return s.store.DeleteUser(ctx, targetID)
}

func (s *Service) ensureNotLastActiveAdmin(ctx context.Context, excludingUserID string) error {
	n, err := s.store.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	// Target is an active admin; if they are the only one, block.
	if n <= 1 {
		return ErrLastAdmin
	}
	_ = excludingUserID
	return nil
}

func userToListItem(u User) UserListItem {
	return UserListItem{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarColor: ResolveAvatarColor(u.ID, u.AvatarColor),
		IsAdmin:     u.IsAdmin,
		DisabledAt:  u.DisabledAt,
		CreatedAt:   u.CreatedAt,
	}
}

func (s *Service) registrationSetting(ctx context.Context) (bool, error) {
	value, ok, err := s.store.GetSetting(ctx, settingRegistrationEnabled)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	return value == "1" || strings.EqualFold(value, "true"), nil
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	username := strings.TrimSpace(in.Username)
	displayName := strings.TrimSpace(in.DisplayName)
	if err := validateUsername(username); err != nil {
		return User{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return User{}, err
	}
	if displayName == "" {
		displayName = username
	}
	if len(displayName) > 64 {
		return User{}, fmt.Errorf("%w: display_name too long", ErrInvalidInput)
	}

	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return User{}, err
	}
	isAdmin := count == 0
	if count > 0 {
		enabled, err := s.registrationSetting(ctx)
		if err != nil {
			return User{}, err
		}
		if !enabled {
			return User{}, ErrRegistrationClosed
		}
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}

	now := time.Now().UTC()
	user := User{
		ID:           uuid.NewString(),
		Username:     username,
		DisplayName:  displayName,
		AvatarColor:  RandomAvatarColor(),
		IsAdmin:      isAdmin,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput) (User, Session, error) {
	username := strings.TrimSpace(in.Username)
	if username == "" || in.Password == "" {
		return User{}, Session{}, ErrInvalidCredentials
	}

	user, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, Session{}, ErrInvalidCredentials
		}
		return User{}, Session{}, err
	}
	if user.Disabled() {
		return User{}, Session{}, ErrAccountDisabled
	}

	ok, err := VerifyPassword(in.Password, user.PasswordHash)
	if err != nil {
		return User{}, Session{}, err
	}
	if !ok {
		return User{}, Session{}, ErrInvalidCredentials
	}

	user, err = s.ensureAvatarColor(ctx, user)
	if err != nil {
		return User{}, Session{}, err
	}

	token, err := newSessionToken()
	if err != nil {
		return User{}, Session{}, err
	}

	now := time.Now().UTC()
	session := Session{
		ID:        token,
		UserID:    user.ID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
	}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return User{}, Session{}, err
	}
	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	err := s.store.RevokeSession(ctx, sessionID, time.Now().UTC())
	if errors.Is(err, ErrSessionNotFound) {
		return nil
	}
	return err
}

func (s *Service) Authenticate(ctx context.Context, sessionID string) (User, Session, error) {
	if sessionID == "" {
		return User{}, Session{}, ErrUnauthorized
	}

	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return User{}, Session{}, ErrUnauthorized
		}
		return User{}, Session{}, err
	}
	if session.RevokedAt != nil || time.Now().UTC().After(session.ExpiresAt) {
		return User{}, Session{}, ErrUnauthorized
	}

	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, Session{}, ErrUnauthorized
		}
		return User{}, Session{}, err
	}
	if user.Disabled() {
		_ = s.store.RevokeSession(ctx, sessionID, time.Now().UTC())
		return User{}, Session{}, ErrUnauthorized
	}
	user, err = s.ensureAvatarColor(ctx, user)
	if err != nil {
		return User{}, Session{}, err
	}
	return user, session, nil
}

func (s *Service) ensureAvatarColor(ctx context.Context, user User) (User, error) {
	if user.AvatarColor != "" {
		return user, nil
	}
	color := AvatarColorForID(user.ID)
	if err := s.store.UpdateAvatarColor(ctx, user.ID, color); err != nil {
		return user, err
	}
	user.AvatarColor = color
	return user, nil
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("%w: username must be 3-32 characters", ErrInvalidInput)
	}
	for _, r := range username {
		ok := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-'
		if !ok {
			return fmt.Errorf("%w: username may only contain letters, digits, _ and -", ErrInvalidInput)
		}
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return fmt.Errorf("%w: password must be 8-128 characters", ErrInvalidInput)
	}
	return nil
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
