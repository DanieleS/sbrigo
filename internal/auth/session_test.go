package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSignerRoundTrip(t *testing.T) {
	s := NewSigner("0123456789abcdef0123456789abcdef")
	id := uuid.New()
	tok := s.SessionToken(id, time.Hour)

	got, err := s.ParseSession(tok)
	if err != nil {
		t.Fatalf("ParseSession: %v", err)
	}
	if got != id {
		t.Fatalf("got %s want %s", got, id)
	}
}

func TestSignerRejectsTamperingAndExpiry(t *testing.T) {
	s := NewSigner("0123456789abcdef0123456789abcdef")
	other := NewSigner("another-secret-another-secret-12")
	id := uuid.New()

	if _, err := s.ParseSession(other.SessionToken(id, time.Hour)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token signed with a different key accepted: %v", err)
	}
	if _, err := s.ParseSession(s.SessionToken(id, time.Hour) + "x"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("tampered token accepted: %v", err)
	}
	s.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	tok := s.SessionToken(id, time.Hour) // expired one hour ago relative to real now
	s.now = time.Now
	if _, err := s.ParseSession(tok); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired token accepted: %v", err)
	}
}

func TestAuthenticatorResolve(t *testing.T) {
	signer := NewSigner("0123456789abcdef0123456789abcdef")
	a := NewAuthenticator(signer, "agent-key")
	userID := uuid.New()

	cases := []struct {
		name     string
		prepare  func(r *http.Request)
		wantOK   bool
		wantKind Kind
	}{
		{"anonymous", func(*http.Request) {}, false, ""},
		{"valid api key", func(r *http.Request) { r.Header.Set(APIKeyHeader, "agent-key") }, true, KindAgent},
		{"wrong api key", func(r *http.Request) { r.Header.Set(APIKeyHeader, "nope") }, false, ""},
		{"session cookie", func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: SessionCookie, Value: signer.SessionToken(userID, time.Hour)})
		}, true, KindUser},
		{"garbage cookie", func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "garbage"})
		}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
			tc.prepare(r)
			p, ok := a.Resolve(r)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if ok && p.Kind != tc.wantKind {
				t.Fatalf("kind=%s want %s", p.Kind, tc.wantKind)
			}
			if ok && p.Kind == KindUser && p.UserID != userID {
				t.Fatalf("user id mismatch")
			}
		})
	}
}

func TestAuthenticatorDisabledAPIKey(t *testing.T) {
	a := NewAuthenticator(NewSigner("0123456789abcdef0123456789abcdef"), "")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(APIKeyHeader, "")
	if _, ok := a.Resolve(r); ok {
		t.Fatal("empty api key must never authenticate")
	}
}

func TestRequireReturns401(t *testing.T) {
	a := NewAuthenticator(NewSigner("0123456789abcdef0123456789abcdef"), "")
	called := false
	h := a.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := PrincipalFrom(r.Context()); !ok {
			t.Error("principal missing from context")
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized || called {
		t.Fatalf("anonymous request: code=%d called=%v", rec.Code, called)
	}

	a.EnableDevAutoLogin(uuid.New())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("dev auto-login: code=%d called=%v", rec.Code, called)
	}
}
