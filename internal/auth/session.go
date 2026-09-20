package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Cookie names.
const (
	SessionCookie = "sbrigo_session"
	LoginCookie   = "sbrigo_login"
)

// ErrInvalidToken is returned for malformed, tampered or expired tokens.
var ErrInvalidToken = errors.New("invalid token")

// Signer issues and verifies compact HMAC-signed tokens of the form base64(payload).base64(mac).
type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner builds a signer from the shared secret.
func NewSigner(secret string) *Signer {
	return &Signer{key: []byte(secret), now: time.Now}
}

// Sign produces a token that carries the payload and expires at exp.
func (s *Signer) Sign(payload string, exp time.Time) string {
	body := strconv.FormatInt(exp.Unix(), 10) + "|" + payload
	enc := base64.RawURLEncoding.EncodeToString([]byte(body))
	return enc + "." + s.mac(enc)
}

// Verify checks signature and expiry and returns the payload.
func (s *Signer) Verify(token string) (string, error) {
	enc, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(enc))) {
		return "", ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", ErrInvalidToken
	}
	expStr, payload, ok := strings.Cut(string(raw), "|")
	if !ok {
		return "", ErrInvalidToken
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return "", ErrInvalidToken
	}
	return payload, nil
}

func (s *Signer) mac(data string) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// SessionToken issues a session for the given user.
func (s *Signer) SessionToken(userID uuid.UUID, ttl time.Duration) string {
	return s.Sign(userID.String(), s.now().Add(ttl))
}

// ParseSession validates a session token and returns the user id.
func (s *Signer) ParseSession(token string) (uuid.UUID, error) {
	payload, err := s.Verify(token)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: bad user id", ErrInvalidToken)
	}
	return id, nil
}
