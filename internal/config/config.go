// Package config reads the runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config is the full runtime configuration. Every variable is prefixed with SBRIGO_.
type Config struct {
	Addr        string // SBRIGO_ADDR, listen address (default ":8080")
	PublicURL   string // SBRIGO_PUBLIC_URL, external base URL used for OIDC redirects and cookies
	DatabaseURL string // SBRIGO_DATABASE_URL, PostgreSQL connection string

	SessionSecret string        // SBRIGO_SESSION_SECRET, HMAC key for session cookies (>= 32 chars)
	SessionTTL    time.Duration // SBRIGO_SESSION_TTL (default 720h)

	OIDCIssuer       string // SBRIGO_OIDC_ISSUER, e.g. https://logto.example.com/oidc
	OIDCClientID     string // SBRIGO_OIDC_CLIENT_ID
	OIDCClientSecret string // SBRIGO_OIDC_CLIENT_SECRET (optional for public clients)

	APIKey string // SBRIGO_API_KEY, static key for the agent; empty disables API-key access

	// DevAutoLoginEmail, when set, authenticates every anonymous request as this user without an
	// identity provider. Local development only; never set it in production.
	DevAutoLoginEmail string // SBRIGO_DEV_AUTO_LOGIN_EMAIL

	LogLevel string // SBRIGO_LOG_LEVEL: debug, info (default), warn, error
}

// Load reads and validates the environment.
func Load() (Config, error) {
	c := Config{
		Addr:              getenv("SBRIGO_ADDR", ":8080"),
		PublicURL:         strings.TrimRight(getenv("SBRIGO_PUBLIC_URL", "http://localhost:8080"), "/"),
		DatabaseURL:       os.Getenv("SBRIGO_DATABASE_URL"),
		SessionSecret:     os.Getenv("SBRIGO_SESSION_SECRET"),
		OIDCIssuer:        os.Getenv("SBRIGO_OIDC_ISSUER"),
		OIDCClientID:      os.Getenv("SBRIGO_OIDC_CLIENT_ID"),
		OIDCClientSecret:  os.Getenv("SBRIGO_OIDC_CLIENT_SECRET"),
		APIKey:            os.Getenv("SBRIGO_API_KEY"),
		DevAutoLoginEmail: os.Getenv("SBRIGO_DEV_AUTO_LOGIN_EMAIL"),
		LogLevel:          getenv("SBRIGO_LOG_LEVEL", "info"),
	}

	ttl := getenv("SBRIGO_SESSION_TTL", "720h")
	d, err := time.ParseDuration(ttl)
	if err != nil || d <= 0 {
		return c, fmt.Errorf("SBRIGO_SESSION_TTL: invalid duration %q", ttl)
	}
	c.SessionTTL = d

	var problems []error
	if c.DatabaseURL == "" {
		problems = append(problems, errors.New("SBRIGO_DATABASE_URL is required"))
	}
	if len(c.SessionSecret) < 32 {
		problems = append(problems, errors.New("SBRIGO_SESSION_SECRET must be at least 32 characters"))
	}
	if _, err := url.ParseRequestURI(c.PublicURL); err != nil {
		problems = append(problems, fmt.Errorf("SBRIGO_PUBLIC_URL: %w", err))
	}
	if !c.OIDCEnabled() && c.DevAutoLoginEmail == "" && c.APIKey == "" {
		problems = append(problems, errors.New("no authentication configured: set SBRIGO_OIDC_ISSUER and SBRIGO_OIDC_CLIENT_ID (or SBRIGO_DEV_AUTO_LOGIN_EMAIL for local development)"))
	}
	if (c.OIDCIssuer == "") != (c.OIDCClientID == "") {
		problems = append(problems, errors.New("SBRIGO_OIDC_ISSUER and SBRIGO_OIDC_CLIENT_ID must be set together"))
	}
	return c, errors.Join(problems...)
}

// OIDCEnabled reports whether the identity provider is configured.
func (c Config) OIDCEnabled() bool {
	return c.OIDCIssuer != "" && c.OIDCClientID != ""
}

// SecureCookies is true when the app is served over HTTPS.
func (c Config) SecureCookies() bool {
	return strings.HasPrefix(c.PublicURL, "https://")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
