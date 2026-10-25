package auth

import "time"

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name"`
	AvatarColor  string     `json:"avatar_color"`
	IsAdmin      bool       `json:"is_admin"`
	PasswordHash string     `json:"-"`
	DisabledAt   *time.Time `json:"disabled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (u User) Disabled() bool {
	return u.DisabledAt != nil
}

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type PublicUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarColor string `json:"avatar_color"`
	IsAdmin     bool   `json:"is_admin"`
}

type UserListItem struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	AvatarColor string     `json:"avatar_color"`
	IsAdmin     bool       `json:"is_admin"`
	DisabledAt  *time.Time `json:"disabled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (u UserListItem) Disabled() bool {
	return u.DisabledAt != nil
}

type SiteSettings struct {
	RegistrationEnabled bool `json:"registration_enabled"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarColor: ResolveAvatarColor(u.ID, u.AvatarColor),
		IsAdmin:     u.IsAdmin,
	}
}
