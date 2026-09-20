package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionStore issues and resolves opaque session tokens.
type SessionStore interface {
	// Create starts a session for userID and returns the token to place in the cookie.
	Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error)
	// Resolve returns the user owning the token; ok is false for unknown or expired tokens.
	Resolve(ctx context.Context, token string) (userID uuid.UUID, ok bool, err error)
	// Revoke ends one session.
	Revoke(ctx context.Context, token string) error
	// RevokeAll ends every session of a user ("esci da tutti i dispositivi").
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

// ---- Redis-backed sessions -------------------------------------------------------------------

// RedisSessions keeps sessions server-side so they can be revoked. Only a hash of the token is
// stored: a leaked Redis dump does not yield usable cookies.
type RedisSessions struct {
	rdb    redis.UniversalClient
	prefix string
}

type sessionRecord struct {
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewRedisSessions connects using a redis:// or rediss:// URL.
func NewRedisSessions(ctx context.Context, url string) (*RedisSessions, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisSessions{rdb: rdb, prefix: "sbrigo:"}, nil
}

// NewRedisSessionsFromClient wraps an existing client (tests).
func NewRedisSessionsFromClient(rdb redis.UniversalClient) *RedisSessions {
	return &RedisSessions{rdb: rdb, prefix: "sbrigo:"}
}

// Close releases the connection.
func (s *RedisSessions) Close() error { return s.rdb.Close() }

func (s *RedisSessions) sessionKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return s.prefix + "session:" + hex.EncodeToString(sum[:])
}

func (s *RedisSessions) userKey(userID uuid.UUID) string {
	return s.prefix + "user_sessions:" + userID.String()
}

// Create implements SessionStore.
func (s *RedisSessions) Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now().UTC()
	rec, err := json.Marshal(sessionRecord{UserID: userID, CreatedAt: now, ExpiresAt: now.Add(ttl)})
	if err != nil {
		return "", err
	}
	key := s.sessionKey(token)
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, key, rec, ttl)
	pipe.SAdd(ctx, s.userKey(userID), key)
	// The index lives a little longer than the sessions it references so it never outlives them silently.
	pipe.Expire(ctx, s.userKey(userID), ttl+24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}
	return token, nil
}

// Resolve implements SessionStore.
func (s *RedisSessions) Resolve(ctx context.Context, token string) (uuid.UUID, bool, error) {
	raw, err := s.rdb.Get(ctx, s.sessionKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("read session: %w", err)
	}
	var rec sessionRecord
	if err := json.Unmarshal(raw, &rec); err != nil || rec.UserID == uuid.Nil {
		return uuid.Nil, false, nil
	}
	return rec.UserID, true, nil
}

// Revoke implements SessionStore.
func (s *RedisSessions) Revoke(ctx context.Context, token string) error {
	key := s.sessionKey(token)
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err == nil {
		var rec sessionRecord
		if json.Unmarshal(raw, &rec) == nil {
			s.rdb.SRem(ctx, s.userKey(rec.UserID), key)
		}
	}
	return s.rdb.Del(ctx, key).Err()
}

// RevokeAll implements SessionStore.
func (s *RedisSessions) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	keys, err := s.rdb.SMembers(ctx, s.userKey(userID)).Result()
	if err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	if len(keys) > 0 {
		pipe.Del(ctx, keys...)
	}
	pipe.Del(ctx, s.userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

// ---- Stateless fallback ----------------------------------------------------------------------

// SignedSessions encodes the user id in an HMAC-signed cookie. It needs no infrastructure but
// cannot revoke a session before it expires. Used when SBRIGO_REDIS_URL is not set.
type SignedSessions struct {
	signer *Signer
}

// NewSignedSessions builds the fallback store.
func NewSignedSessions(signer *Signer) *SignedSessions { return &SignedSessions{signer: signer} }

// Create implements SessionStore.
func (s *SignedSessions) Create(_ context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	return s.signer.SessionToken(userID, ttl), nil
}

// Resolve implements SessionStore.
func (s *SignedSessions) Resolve(_ context.Context, token string) (uuid.UUID, bool, error) {
	id, err := s.signer.ParseSession(token)
	if err != nil {
		return uuid.Nil, false, nil
	}
	return id, true, nil
}

// Revoke implements SessionStore: the cookie is cleared by the caller; nothing to do server-side.
func (s *SignedSessions) Revoke(context.Context, string) error { return nil }

// RevokeAll implements SessionStore: not supported without server-side state.
func (s *SignedSessions) RevokeAll(context.Context, uuid.UUID) error { return ErrNotSupported }

// ErrNotSupported is returned by operations the stateless store cannot perform.
var ErrNotSupported = errors.New("not supported by the stateless session store")
