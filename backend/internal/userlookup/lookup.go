package userlookup

import (
	"context"
	"errors"

	"gitee.com/yunque/yunque/backend/internal/auth"
	"gitee.com/yunque/yunque/backend/internal/docs"
)

type authAdapter struct {
	store *auth.Store
}

func FromAuthStore(store *auth.Store) docs.UserLookup {
	return authAdapter{store: store}
}

func (a authAdapter) LookupByUsername(ctx context.Context, username string) (docs.UserRef, error) {
	u, err := a.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return docs.UserRef{}, docs.ErrNotFound
		}
		return docs.UserRef{}, err
	}
	return docs.UserRef{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarColor: auth.ResolveAvatarColor(u.ID, u.AvatarColor),
	}, nil
}
