package middleware

import (
	"context"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

type contextKey string

const userKey contextKey = "user"

// WithUser attaches a user to the context.
func WithUser(ctx context.Context, u *model.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext extracts the user from the context, or nil.
func UserFromContext(ctx context.Context) *model.User {
	u, _ := ctx.Value(userKey).(*model.User)
	return u
}
