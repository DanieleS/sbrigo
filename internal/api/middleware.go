package api

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// SecurityHeaders sets conservative browser hardening headers. The CSP allows only same-origin
// resources plus inline styles (Vue scoped styles and the PWA manifest need them).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'",
			"script-src 'self'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data:",
			"font-src 'self'",
			"connect-src 'self'",
			"manifest-src 'self'",
			"worker-src 'self'",
			"frame-ancestors 'none'",
			"base-uri 'self'",
			"form-action 'self'",
		}, "; "))
		next.ServeHTTP(w, r)
	})
}

// RequestTimeout bounds the lifetime of ordinary API requests. The SSE stream is excluded
// because it is intentionally long-lived.
func RequestTimeout(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireJSONForWrites rejects state-changing requests that do not declare a JSON body. Together
// with SameSite cookies this closes the classic HTML-form CSRF vector.
func RequireJSONForWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			if r.ContentLength != 0 && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
