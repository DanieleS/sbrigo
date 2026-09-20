package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/danieles/sbrigo/internal/model"
	"github.com/danieles/sbrigo/internal/store"
)

type listInput struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	SortOrder int    `json:"sort_order"`
}

func (s *Server) listLists(w http.ResponseWriter, r *http.Request) {
	out, err := s.store.ListLists(r.Context())
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createList(w http.ResponseWriter, r *http.Request) {
	var in listInput
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !model.ValidItemType(in.Kind) {
		writeError(w, http.StatusBadRequest, "kind must be grocery or general_task")
		return
	}
	l, err := s.store.CreateList(r.Context(), model.List{Name: in.Name, Kind: in.Kind, SortOrder: in.SortOrder})
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("list.created", l)
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) updateList(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var in listInput
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	l, err := s.store.UpdateList(r.Context(), model.List{ID: id, Name: in.Name, SortOrder: in.SortOrder})
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("list.updated", l)
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) deleteList(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	err := s.store.DeleteList(r.Context(), id)
	if errors.Is(err, store.ErrLastList) {
		writeError(w, http.StatusConflict, "cannot delete the last list of its kind")
		return
	}
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("list.deleted", map[string]uuid.UUID{"id": id})
	w.WriteHeader(http.StatusNoContent)
}
