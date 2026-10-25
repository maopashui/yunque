package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrSessionNotFound = errors.New("session not found")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(ctx context.Context, user User) error {
	isAdmin := 0
	if user.IsAdmin {
		isAdmin = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, display_name, password_hash, avatar_color, is_admin, disabled_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?)`,
		user.ID, user.Username, user.DisplayName, user.PasswordHash, user.AvatarColor, isAdmin,
		user.CreatedAt.UTC().Format(time.RFC3339Nano),
		user.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUsernameTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (s *Store) CountActiveAdmins(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM users
		WHERE is_admin = 1 AND disabled_at IS NULL`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count active admins: %w", err)
	}
	return n, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]UserListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, avatar_color, is_admin, disabled_at, created_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []UserListItem
	for rows.Next() {
		var item UserListItem
		var isAdmin int
		var disabledAt sql.NullString
		var createdAt string
		if err := rows.Scan(
			&item.ID, &item.Username, &item.DisplayName, &item.AvatarColor, &isAdmin, &disabledAt, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan user list: %w", err)
		}
		item.IsAdmin = isAdmin != 0
		item.AvatarColor = ResolveAvatarColor(item.ID, item.AvatarColor)
		if disabledAt.Valid && disabledAt.String != "" {
			t, err := time.Parse(time.RFC3339Nano, disabledAt.String)
			if err != nil {
				return nil, fmt.Errorf("parse disabled_at: %w", err)
			}
			item.DisabledAt = &t
		}
		t, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		item.CreatedAt = t
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []UserListItem{}
	}
	return out, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, password_hash, avatar_color, is_admin, disabled_at, created_at, updated_at
		FROM users WHERE username = ?`, username)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, password_hash, avatar_color, is_admin, disabled_at, created_at, updated_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Store) SetUserDisabledAt(ctx context.Context, userID string, disabledAt *time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var disabled any
	if disabledAt != nil {
		disabled = disabledAt.UTC().Format(time.RFC3339Nano)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE users SET disabled_at = ?, updated_at = ? WHERE id = ?`,
		disabled, now, userID,
	)
	if err != nil {
		return fmt.Errorf("set disabled_at: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, userID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Store) UpdateAvatarColor(ctx context.Context, userID, color string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET avatar_color = ?, updated_at = ?
		WHERE id = ? AND (avatar_color = '' OR avatar_color IS NULL)`,
		color, time.Now().UTC().Format(time.RFC3339Nano), userID,
	)
	if err != nil {
		return fmt.Errorf("update avatar_color: %w", err)
	}
	return nil
}

func (s *Store) GetSetting(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM site_settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get setting: %w", err)
	}
	return value, true, nil
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO site_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, session Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at, revoked_at)
		VALUES (?, ?, ?, ?, NULL)`,
		session.ID, session.UserID,
		session.ExpiresAt.UTC().Format(time.RFC3339Nano),
		session.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, id string) (Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, created_at, revoked_at
		FROM sessions WHERE id = ?`, id)

	var sess Session
	var expiresAt, createdAt string
	var revokedAt sql.NullString
	if err := row.Scan(&sess.ID, &sess.UserID, &expiresAt, &createdAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("scan session: %w", err)
	}

	var err error
	sess.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse expires_at: %w", err)
	}
	sess.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse created_at: %w", err)
	}
	if revokedAt.Valid {
		t, err := time.Parse(time.RFC3339Nano, revokedAt.String)
		if err != nil {
			return Session{}, fmt.Errorf("parse revoked_at: %w", err)
		}
		sess.RevokedAt = &t
	}
	return sess, nil
}

func (s *Store) RevokeSession(ctx context.Context, id string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET revoked_at = ?
		WHERE id = ? AND revoked_at IS NULL`,
		at.UTC().Format(time.RFC3339Nano), id,
	)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *Store) RevokeAllSessionsForUser(ctx context.Context, userID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET revoked_at = ?
		WHERE user_id = ? AND revoked_at IS NULL`,
		at.UTC().Format(time.RFC3339Nano), userID,
	)
	if err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}

func scanUser(row *sql.Row) (User, error) {
	var u User
	var isAdmin int
	var disabledAt sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarColor, &isAdmin, &disabledAt, &createdAt, &updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	u.IsAdmin = isAdmin != 0
	if disabledAt.Valid && disabledAt.String != "" {
		t, err := time.Parse(time.RFC3339Nano, disabledAt.String)
		if err != nil {
			return User{}, fmt.Errorf("parse disabled_at: %w", err)
		}
		u.DisabledAt = &t
	}
	var err error
	u.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return User{}, fmt.Errorf("parse created_at: %w", err)
	}
	u.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return User{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return u, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
