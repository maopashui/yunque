package httpapi

import (
	"context"

	"gitee.com/yunque/yunque/backend/internal/auth"
)

type ctxKey int

const userKey ctxKey = 1

func withUser(ctx context.Context, user auth.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) (auth.User, bool) {
	u, ok := ctx.Value(userKey).(auth.User)
	return u, ok
}
