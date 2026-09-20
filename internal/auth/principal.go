// Package auth implements login via OIDC (Authorization Code + PKCE), signed session cookies and
// the static API key used by the automation agent.
package auth

import (
	"context"

	"github.com/google/uuid"
)

// Kind tells how the caller authenticated.
type Kind string

// Principal kinds.
const (
	KindUser  Kind = "user"  // a family member with a session cookie
	KindAgent Kind = "agent" // the automation agent using the static API key
)

// Principal is the authenticated caller attached to the request context.
type Principal struct {
	Kind   Kind
	UserID uuid.UUID // zero for agents
}

type ctxKey struct{}

// WithPrincipal stores the principal in the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// PrincipalFrom extracts the principal set by the middleware.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
