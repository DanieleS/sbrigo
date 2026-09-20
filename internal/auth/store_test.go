package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func newRedisStore(t *testing.T) (*RedisSessions, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return NewRedisSessionsFromClient(rdb), mr
}

func TestRedisSessionsLifecycle(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)
	user := uuid.New()

	tok1, err := store.Create(ctx, user, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := store.Create(ctx, user, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if tok1 == tok2 {
		t.Fatal("tokens must be unique")
	}
	for _, k := range mr.Keys() {
		if mr.Exists(k) && (len(k) > 0) && containsToken(k, tok1) {
			t.Fatal("raw token must not be used as a key")
		}
	}

	id, ok, err := store.Resolve(ctx, tok1)
	if err != nil || !ok || id != user {
		t.Fatalf("resolve: id=%s ok=%v err=%v", id, ok, err)
	}
	if _, ok, _ := store.Resolve(ctx, "garbage"); ok {
		t.Fatal("unknown token resolved")
	}

	if err := store.Revoke(ctx, tok1); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.Resolve(ctx, tok1); ok {
		t.Fatal("revoked token still valid")
	}
	if _, ok, _ := store.Resolve(ctx, tok2); !ok {
		t.Fatal("other session must survive a single revoke")
	}

	tok3, _ := store.Create(ctx, user, time.Hour)
	if err := store.RevokeAll(ctx, user); err != nil {
		t.Fatal(err)
	}
	for _, tok := range []string{tok2, tok3} {
		if _, ok, _ := store.Resolve(ctx, tok); ok {
			t.Fatal("RevokeAll left a session alive")
		}
	}
}

func TestRedisSessionsExpire(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)
	tok, err := store.Create(ctx, uuid.New(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	mr.FastForward(2 * time.Minute)
	if _, ok, _ := store.Resolve(ctx, tok); ok {
		t.Fatal("expired session resolved")
	}
}

func TestSignedSessionsCannotRevokeAll(t *testing.T) {
	s := NewSignedSessions(NewSigner("0123456789abcdef0123456789abcdef"))
	if err := s.RevokeAll(context.Background(), uuid.New()); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

func containsToken(key, token string) bool {
	return len(token) > 0 && len(key) >= len(token) && (key == token || indexOf(key, token) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
