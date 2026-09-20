package auth

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// APIKeyHeader carries the static agent key.
const APIKeyHeader = "X-API-Key"

// Authenticator resolves the principal of incoming requests.
type Authenticator struct {
	sessions SessionStore
	apiKey   string
	log      *slog.Logger
	// devUser, when non-nil, is applied to every anonymous request (local development only).
	devUser *uuid.UUID
}

// NewAuthenticator creates the middleware. apiKey may be empty to disable agent access.
func NewAuthenticator(sessions SessionStore, apiKey string, log *slog.Logger) *Authenticator {
	return &Authenticator{sessions: sessions, apiKey: apiKey, log: log}
}

// EnableDevAutoLogin makes every anonymous request act as userID.
func (a *Authenticator) EnableDevAutoLogin(userID uuid.UUID) {
	a.devUser = &userID
}

// Resolve returns the principal for the request, if any.
func (a *Authenticator) Resolve(r *http.Request) (Principal, bool) {
	if key := r.Header.Get(APIKeyHeader); key != "" && a.apiKey != "" {
		if subtle.ConstantTimeCompare([]byte(key), []byte(a.apiKey)) == 1 {
			return Principal{Kind: KindAgent}, true
		}
		return Principal{}, false
	}
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		id, ok, err := a.sessions.Resolve(r.Context(), c.Value)
		if err != nil {
			a.log.Error("session lookup failed", "err", err)
		} else if ok {
			return Principal{Kind: KindUser, UserID: id}, true
		}
	}
	if a.devUser != nil {
		return Principal{Kind: KindUser, UserID: *a.devUser}, true
	}
	return Principal{}, false
}

// Require rejects unauthenticated requests with 401 and stores the principal in the context.
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := a.Resolve(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	})
}
