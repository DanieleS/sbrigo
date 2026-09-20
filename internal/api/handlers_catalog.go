package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/danieles/sbrigo/internal/model"
)

type departmentInput struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	DefaultSortOrder int    `json:"default_sort_order"`
}

func (in *departmentInput) validate(w http.ResponseWriter) bool {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return false
	}
	return true
}

func (s *Server) listDepartments(w http.ResponseWriter, r *http.Request) {
	out, err := s.store.ListDepartments(r.Context())
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createDepartment(w http.ResponseWriter, r *http.Request) {
	var in departmentInput
	if !readJSON(w, r, &in) || !in.validate(w) {
		return
	}
	d, err := s.store.CreateDepartment(r.Context(), model.Department{Name: in.Name, Description: in.Description, DefaultSortOrder: in.DefaultSortOrder})
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("department.created", d)
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) updateDepartment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var in departmentInput
	if !readJSON(w, r, &in) || !in.validate(w) {
		return
	}
	d, err := s.store.UpdateDepartment(r.Context(), model.Department{ID: id, Name: in.Name, Description: in.Description, DefaultSortOrder: in.DefaultSortOrder})
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("department.updated", d)
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if s.storeError(w, s.store.DeleteDepartment(r.Context(), id)) {
		return
	}
	s.broker.Publish("department.deleted", map[string]uuid.UUID{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

type supermarketInput struct {
	Name string `json:"name"`
}

func (s *Server) listSupermarkets(w http.ResponseWriter, r *http.Request) {
	out, err := s.store.ListSupermarkets(r.Context())
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createSupermarket(w http.ResponseWriter, r *http.Request) {
	var in supermarketInput
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	m, err := s.store.CreateSupermarket(r.Context(), model.Supermarket{Name: in.Name})
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("supermarket.created", m)
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) updateSupermarket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var in supermarketInput
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	m, err := s.store.UpdateSupermarket(r.Context(), model.Supermarket{ID: id, Name: in.Name})
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("supermarket.updated", m)
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) deleteSupermarket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if s.storeError(w, s.store.DeleteSupermarket(r.Context(), id)) {
		return
	}
	s.broker.Publish("supermarket.deleted", map[string]uuid.UUID{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getDepartmentOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.store.GetSupermarket(r.Context(), id); s.storeError(w, err) {
		return
	}
	out, err := s.store.ListDepartmentOrder(r.Context(), id)
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type departmentOrderInput struct {
	DepartmentIDs []uuid.UUID `json:"department_ids"`
}

func (s *Server) putDepartmentOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var in departmentOrderInput
	if !readJSON(w, r, &in) {
		return
	}
	seen := map[uuid.UUID]bool{}
	for _, depID := range in.DepartmentIDs {
		if seen[depID] {
			writeError(w, http.StatusBadRequest, "duplicate department id "+depID.String())
			return
		}
		seen[depID] = true
	}
	out, err := s.store.ReplaceDepartmentOrder(r.Context(), id, in.DepartmentIDs)
	if s.storeError(w, err) {
		return
	}
	s.broker.Publish("supermarket.order_updated", map[string]any{"supermarket_id": id, "order": out})
	writeJSON(w, http.StatusOK, out)
}
