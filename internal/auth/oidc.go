package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/danieles/sbrigo/internal/model"
)

// UserSyncer creates or refreshes the local profile of a user on login.
type UserSyncer interface {
	UpsertUser(ctx context.Context, identityID, email, displayName string) (model.User, error)
}

// OIDCOptions configures the login flow.
type OIDCOptions struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	PublicURL    string // base URL of the app, used to build the redirect URI
	SessionTTL   time.Duration
	Secure       bool // set Secure on cookies (HTTPS deployments)
}

// OIDC handles /auth/login, /auth/callback and /auth/logout using Authorization Code + PKCE.
type OIDC struct {
	opts     OIDCOptions
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
	signer   *Signer
	sessions SessionStore
	users    UserSyncer
	log      *slog.Logger
}

// NewOIDC discovers the provider and prepares the flow.
func NewOIDC(ctx context.Context, opts OIDCOptions, signer *Signer, sessions SessionStore, users UserSyncer, log *slog.Logger) (*OIDC, error) {
	provider, err := oidc.NewProvider(ctx, opts.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery for %s: %w", opts.Issuer, err)
	}
	return &OIDC{
		opts:     opts,
		verifier: provider.Verifier(&oidc.Config{ClientID: opts.ClientID}),
		oauth: oauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  strings.TrimRight(opts.PublicURL, "/") + "/auth/callback",
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		signer:   signer,
		sessions: sessions,
		users:    users,
		log:      log,
	}, nil
}

// Register mounts the auth routes.
func (o *OIDC) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", o.login)
	mux.HandleFunc("GET /auth/callback", o.callback)
	mux.HandleFunc("POST /auth/logout", o.logout)
	mux.HandleFunc("POST /auth/logout-all", o.logoutAll)
}

// login starts the flow: state + PKCE verifier are kept in a short-lived signed cookie.
func (o *OIDC) login(w http.ResponseWriter, r *http.Request) {
	state := randomString(24)
	verifier := oauth2.GenerateVerifier()
	returnTo := r.URL.Query().Get("return_to")
	if !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		returnTo = "/"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     LoginCookie,
		Value:    o.signer.Sign(state+"|"+verifier+"|"+returnTo, time.Now().Add(10*time.Minute)),
		Path:     "/auth",
		HttpOnly: true,
		Secure:   o.opts.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, o.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

// callback exchanges the code, verifies the ID token, syncs the user and issues the session.
func (o *OIDC) callback(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, LoginCookie, "/auth", o.opts.Secure)

	c, err := r.Cookie(LoginCookie)
	if err != nil {
		http.Error(w, "login session missing or expired, please retry", http.StatusBadRequest)
		return
	}
	payload, err := o.signer.Verify(c.Value)
	if err != nil {
		http.Error(w, "login session invalid, please retry", http.StatusBadRequest)
		return
	}
	parts := strings.SplitN(payload, "|", 3)
	if len(parts) != 3 {
		http.Error(w, "login session invalid, please retry", http.StatusBadRequest)
		return
	}
	state, verifier, returnTo := parts[0], parts[1], parts[2]

	q := r.URL.Query()
	if errCode := q.Get("error"); errCode != "" {
		http.Error(w, "identity provider error: "+errCode+" "+q.Get("error_description"), http.StatusBadGateway)
		return
	}
	if q.Get("state") != state {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	token, err := o.oauth.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		o.log.Error("oidc code exchange failed", "err", err)
		http.Error(w, "code exchange failed", http.StatusBadGateway)
		return
	}
	rawID, _ := token.Extra("id_token").(string)
	if rawID == "" {
		http.Error(w, "id_token missing in token response", http.StatusBadGateway)
		return
	}
	idToken, err := o.verifier.Verify(ctx, rawID)
	if err != nil {
		o.log.Error("id token verification failed", "err", err)
		http.Error(w, "invalid id_token", http.StatusBadGateway)
		return
	}

	var claims struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "unreadable claims", http.StatusBadGateway)
		return
	}
	displayName := firstNonEmpty(claims.Name, claims.Username, emailLocalPart(claims.Email), idToken.Subject)

	user, err := o.users.UpsertUser(ctx, idToken.Subject, claims.Email, displayName)
	if err != nil {
		o.log.Error("user sync failed", "err", err)
		http.Error(w, "could not synchronise user profile", http.StatusInternalServerError)
		return
	}

	sessionToken, err := o.sessions.Create(ctx, user.ID, o.opts.SessionTTL)
	if err != nil {
		o.log.Error("session creation failed", "err", err)
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   o.opts.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(o.opts.SessionTTL.Seconds()),
	})
	o.log.Info("user logged in", "user_id", user.ID, "email", user.Email)
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (o *OIDC) logout(w http.ResponseWriter, r *http.Request) {
	LogoutHandler(o.sessions, o.opts.Secure, o.log)(w, r)
}

// logoutAll revokes every session of the calling user ("esci da tutti i dispositivi").
func (o *OIDC) logoutAll(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(SessionCookie)
	if err != nil || c.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID, ok, err := o.sessions.Resolve(r.Context(), c.Value)
	if err != nil {
		o.log.Error("session lookup failed", "err", err)
		http.Error(w, "session store unavailable", http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := o.sessions.RevokeAll(r.Context(), userID); err != nil {
		if errors.Is(err, ErrNotSupported) {
			http.Error(w, "logout from all devices requires the Redis session store", http.StatusNotImplemented)
			return
		}
		o.log.Error("revoke all sessions failed", "err", err)
		http.Error(w, "could not revoke sessions", http.StatusInternalServerError)
		return
	}
	clearCookie(w, SessionCookie, "/", o.opts.Secure)
	w.WriteHeader(http.StatusNoContent)
}

// LogoutHandler revokes the current session and clears the cookie.
func LogoutHandler(sessions SessionStore, secure bool, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
			if err := sessions.Revoke(r.Context(), c.Value); err != nil {
				log.Error("session revoke failed", "err", err)
			}
		}
		clearCookie(w, SessionCookie, "/", secure)
		w.WriteHeader(http.StatusNoContent)
	}
}

func clearCookie(w http.ResponseWriter, name, path string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: path, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(errors.Join(errors.New("crypto/rand unavailable"), err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func emailLocalPart(email string) string {
	local, _, _ := strings.Cut(email, "@")
	return local
}
