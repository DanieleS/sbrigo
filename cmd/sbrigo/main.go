// Command sbrigo runs the Sbrigo backend: REST API, SSE stream, OIDC login and the embedded PWA.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/danieles/sbrigo/internal/api"
	"github.com/danieles/sbrigo/internal/auth"
	"github.com/danieles/sbrigo/internal/config"
	"github.com/danieles/sbrigo/internal/db"
	"github.com/danieles/sbrigo/internal/realtime"
	"github.com/danieles/sbrigo/internal/store"
	"github.com/danieles/sbrigo/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, log); err != nil {
		return err
	}

	st := store.New(pool)
	broker := realtime.NewBroker(log)
	signer := auth.NewSigner(cfg.SessionSecret)
	var sessions auth.SessionStore
	if cfg.RedisURL != "" {
		rs, err := auth.NewRedisSessions(ctx, cfg.RedisURL)
		if err != nil {
			return err
		}
		defer func() { _ = rs.Close() }()
		sessions = rs
		log.Info("session store: redis")
	} else {
		sessions = auth.NewSignedSessions(signer)
		log.Warn("session store: signed cookies (no revocation); set SBRIGO_REDIS_URL to enable server-side sessions")
	}
	authn := auth.NewAuthenticator(sessions, cfg.APIKey, log)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	if cfg.OIDCEnabled() {
		o, err := auth.NewOIDC(ctx, auth.OIDCOptions{
			Issuer:       cfg.OIDCIssuer,
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			PublicURL:    cfg.PublicURL,
			SessionTTL:   cfg.SessionTTL,
			Secure:       cfg.SecureCookies(),
		}, signer, sessions, st, log)
		if err != nil {
			return err
		}
		o.Register(mux)
		log.Info("oidc login enabled", "issuer", cfg.OIDCIssuer)
	} else {
		mux.Handle("POST /auth/logout", auth.LogoutHandler(sessions, cfg.SecureCookies(), log))
		mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "OIDC login is not configured", http.StatusNotImplemented)
		})
	}

	if cfg.DevAutoLoginEmail != "" {
		u, err := st.UpsertUser(ctx, "dev:"+cfg.DevAutoLoginEmail, cfg.DevAutoLoginEmail, strings.Split(cfg.DevAutoLoginEmail, "@")[0])
		if err != nil {
			return err
		}
		authn.EnableDevAutoLogin(u.ID)
		log.Warn("DEV AUTO-LOGIN ENABLED: every anonymous request acts as this user. Never use in production.", "email", u.Email)
	}
	if cfg.APIKey != "" {
		log.Info("agent api key enabled", "header", auth.APIKeyHeader)
	}

	api.New(st, broker, authn, log).Register(mux)
	mux.Handle("/", api.SPAHandler(web.Dist()))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Logging(log, api.SecurityHeaders(api.RequestTimeout(30*time.Second, mux))),
		ReadHeaderTimeout: 10 * time.Second,
		// No WriteTimeout: the SSE stream is long-lived.
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("sbrigo listening", "addr", cfg.Addr, "public_url", cfg.PublicURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
