// Package api exposes the REST endpoints and the SSE stream under /api/v1.
package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/danieles/sbrigo/internal/auth"
	"github.com/danieles/sbrigo/internal/realtime"
	"github.com/danieles/sbrigo/internal/store"
)

// Server holds the handler dependencies.
type Server struct {
	store  *store.Store
	broker *realtime.Broker
	auth   *auth.Authenticator
	log    *slog.Logger
}

// New creates the API server.
func New(st *store.Store, broker *realtime.Broker, authn *auth.Authenticator, log *slog.Logger) *Server {
	return &Server{store: st, broker: broker, auth: authn, log: log}
}

// Register mounts the protected API on mux.
func (s *Server) Register(mux *http.ServeMux) {
	api := http.NewServeMux()

	api.HandleFunc("GET /api/v1/me", s.getMe)
	api.HandleFunc("GET /api/v1/users", s.listUsers)

	api.HandleFunc("GET /api/v1/departments", s.listDepartments)
	api.HandleFunc("POST /api/v1/departments", s.createDepartment)
	api.HandleFunc("PUT /api/v1/departments/{id}", s.updateDepartment)
	api.HandleFunc("DELETE /api/v1/departments/{id}", s.deleteDepartment)

	api.HandleFunc("GET /api/v1/supermarkets", s.listSupermarkets)
	api.HandleFunc("POST /api/v1/supermarkets", s.createSupermarket)
	api.HandleFunc("PUT /api/v1/supermarkets/{id}", s.updateSupermarket)
	api.HandleFunc("DELETE /api/v1/supermarkets/{id}", s.deleteSupermarket)
	api.HandleFunc("GET /api/v1/supermarkets/{id}/department-order", s.getDepartmentOrder)
	api.HandleFunc("PUT /api/v1/supermarkets/{id}/department-order", s.putDepartmentOrder)

	api.HandleFunc("GET /api/v1/lists", s.listLists)
	api.HandleFunc("POST /api/v1/lists", s.createList)
	api.HandleFunc("PUT /api/v1/lists/{id}", s.updateList)
	api.HandleFunc("DELETE /api/v1/lists/{id}", s.deleteList)

	api.HandleFunc("GET /api/v1/tasks", s.listTasks)
	api.HandleFunc("POST /api/v1/tasks", s.createTask)
	api.HandleFunc("DELETE /api/v1/tasks", s.deleteCompletedTasks)
	api.HandleFunc("GET /api/v1/tasks/{id}", s.getTask)
	api.HandleFunc("PATCH /api/v1/tasks/{id}", s.updateTask)
	api.HandleFunc("DELETE /api/v1/tasks/{id}", s.deleteTask)

	api.Handle("GET /api/v1/events", s.broker)

	mux.Handle("/api/", s.auth.Require(RequireJSONForWrites(api)))
}

// SPAHandler serves the embedded frontend, falling back to index.html for client-side routes.
func SPAHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if f, err := dist.Open(p); err == nil {
			f.Close()
			switch {
			case strings.HasPrefix(p, "assets/"):
				// Vite emits content-hashed file names: safe to cache forever.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			default:
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		// Client-side route: serve the app shell.
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "frontend not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index) //nolint:errcheck
	})
}

// Logging wraps a handler with structured request logging.
func Logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events" { // long-lived stream, not worth a line per connection
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Debug("http", "method", r.Method, "path", r.URL.Path, "status", rw.status, "dur", time.Since(start).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush keeps SSE working through the recorder.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
