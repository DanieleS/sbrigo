package api

import (
	"net/http"

	"github.com/danieles/sbrigo/internal/auth"
)

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	if p.Kind == auth.KindAgent {
		writeJSON(w, http.StatusOK, map[string]any{"kind": p.Kind, "user": nil})
		return
	}
	u, err := s.store.GetUser(r.Context(), p.UserID)
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": p.Kind, "user": u})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, users)
}
