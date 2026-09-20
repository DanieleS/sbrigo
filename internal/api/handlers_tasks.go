package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/danieles/sbrigo/internal/model"
	"github.com/danieles/sbrigo/internal/store"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f model.TaskFilter
	if t := q.Get("type"); t != "" {
		if !model.ValidItemType(t) {
			writeError(w, http.StatusBadRequest, "invalid type")
			return
		}
		f.ItemType = t
	}
	if c := q.Get("completed"); c != "" {
		b, err := strconv.ParseBool(c)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid completed flag")
			return
		}
		f.IsCompleted = &b
	}
	for name, dst := range map[string]**uuid.UUID{"list_id": &f.ListID, "department_id": &f.DepartmentID, "assignee_id": &f.AssigneeID} {
		if v := q.Get(name); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+name)
				return
			}
			*dst = &id
		}
	}
	if v := q.Get("since"); v != "" {
		ts, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since (RFC 3339 expected)")
			return
		}
		f.UpdatedSince = &ts
	}
	out, err := s.store.ListTasks(r.Context(), f)
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	t, err := s.store.GetTask(r.Context(), id)
	if s.storeError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// taskInput is the create payload. Clients may pass id/created_at/updated_at so that offline
// creations replay idempotently with their original timestamps.
type taskInput struct {
	ID           *uuid.UUID `json:"id"`
	ListID       *uuid.UUID `json:"list_id"`
	Title        string     `json:"title"`
	Notes        *string    `json:"notes"`
	IsCompleted  bool       `json:"is_completed"`
	ItemType     string     `json:"item_type"`
	DepartmentID *uuid.UUID `json:"department_id"`
	AssigneeID   *uuid.UUID `json:"assignee_id"`
	DueDate      *time.Time `json:"due_date"`
	Position     *float64   `json:"position"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var in taskInput
	if !readJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if in.ItemType != "" && !model.ValidItemType(in.ItemType) {
		writeError(w, http.StatusBadRequest, "invalid item_type")
		return
	}
	t := model.Task{
		Title:        in.Title,
		Notes:        in.Notes,
		IsCompleted:  in.IsCompleted,
		ItemType:     in.ItemType,
		DepartmentID: in.DepartmentID,
		AssigneeID:   in.AssigneeID,
		DueDate:      in.DueDate,
	}
	if in.ID != nil {
		t.ID = *in.ID
	}
	if in.ListID != nil {
		t.ListID = *in.ListID
	}
	if in.Position != nil {
		t.Position = *in.Position
	}
	now := time.Now().UTC()
	if in.CreatedAt != nil && !in.CreatedAt.After(now.Add(5*time.Minute)) {
		t.CreatedAt = in.CreatedAt.UTC()
	}
	if in.UpdatedAt != nil && !in.UpdatedAt.After(now.Add(5*time.Minute)) {
		t.UpdatedAt = in.UpdatedAt.UTC()
	}

	out, created, err := s.store.CreateTask(r.Context(), t)
	if s.storeErrorFK(w, err) {
		return
	}
	if !created {
		// Idempotent replay of an offline creation: nothing changed, report the stored row.
		writeJSON(w, http.StatusOK, out)
		return
	}
	s.broker.Publish("task.created", out)
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var p model.TaskPatch
	if !readJSON(w, r, &p) {
		return
	}
	if p.Empty() {
		writeError(w, http.StatusBadRequest, "empty patch")
		return
	}
	if p.Title.Set {
		p.Title.Value = strings.TrimSpace(p.Title.Value)
		if p.Title.Value == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
	}
	if p.ItemType.Set && !model.ValidItemType(p.ItemType.Value) {
		writeError(w, http.StatusBadRequest, "invalid item_type")
		return
	}
	t, err := s.store.UpdateTask(r.Context(), id, p)
	if errors.Is(err, store.ErrConflict) {
		// Last-write-wins: a newer version is already stored. Hand it back so the client converges.
		writeJSON(w, http.StatusConflict, t)
		return
	}
	if s.storeErrorFK(w, err) {
		return
	}
	s.broker.Publish("task.updated", t)
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if s.storeError(w, s.store.DeleteTask(r.Context(), id)) {
		return
	}
	s.broker.Publish("task.deleted", map[string]uuid.UUID{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

// deleteCompletedTasks handles DELETE /api/v1/tasks?completed=true[&type=grocery][&list_id=...].
func (s *Server) deleteCompletedTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if ok, _ := strconv.ParseBool(q.Get("completed")); !ok {
		writeError(w, http.StatusBadRequest, "bulk delete requires completed=true")
		return
	}
	itemType := q.Get("type")
	if itemType != "" && !model.ValidItemType(itemType) {
		writeError(w, http.StatusBadRequest, "invalid type")
		return
	}
	var listID *uuid.UUID
	if v := q.Get("list_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid list_id")
			return
		}
		listID = &id
	}
	ids, err := s.store.DeleteCompletedTasks(r.Context(), itemType, listID)
	if s.storeError(w, err) {
		return
	}
	for _, id := range ids {
		s.broker.Publish("task.deleted", map[string]uuid.UUID{"id": id})
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": len(ids), "ids": ids})
}

// storeErrorFK additionally maps reference errors (unknown list/department/assignee) to 400.
func (s *Server) storeErrorFK(w http.ResponseWriter, err error) bool {
	if errors.Is(err, store.ErrUnknownList) {
		writeError(w, http.StatusBadRequest, "unknown list_id")
		return true
	}
	if err != nil && isForeignKeyViolation(err) {
		writeError(w, http.StatusBadRequest, "unknown department_id or assignee_id")
		return true
	}
	return s.storeError(w, err)
}
