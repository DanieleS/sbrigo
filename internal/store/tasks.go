package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danieles/sbrigo/internal/model"
)

const taskColumns = `id, list_id, title, notes, is_completed, item_type, department_id, assignee_id, due_date, position, created_at, updated_at`

func scanTask(row pgx.Row) (model.Task, error) {
	var t model.Task
	err := row.Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.IsCompleted, &t.ItemType, &t.DepartmentID, &t.AssigneeID, &t.DueDate, &t.Position, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// ListTasks returns tasks matching the filter, open items first, newest last.
func (s *Store) ListTasks(ctx context.Context, f model.TaskFilter) ([]model.Task, error) {
	var (
		where []string
		args  []any
	)
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.ListID != nil {
		add("list_id = $%d", *f.ListID)
	}
	if f.ItemType != "" {
		add("item_type = $%d", f.ItemType)
	}
	if f.IsCompleted != nil {
		add("is_completed = $%d", *f.IsCompleted)
	}
	if f.DepartmentID != nil {
		add("department_id = $%d", *f.DepartmentID)
	}
	if f.AssigneeID != nil {
		add("assignee_id = $%d", *f.AssigneeID)
	}
	if f.UpdatedSince != nil {
		add("updated_at > $%d", *f.UpdatedSince)
	}
	q := `SELECT ` + taskColumns + ` FROM tasks`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY is_completed, position, created_at, id`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Task, error) { return scanTask(row) })
}

// GetTask returns one task.
func (s *Store) GetTask(ctx context.Context, id uuid.UUID) (model.Task, error) {
	t, err := scanTask(s.pool.QueryRow(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// ErrUnknownList is returned when a task references a list that does not exist.
var ErrUnknownList = errors.New("unknown list")

// resolveList makes list and item type agree: an explicit list sets the type from its kind, a
// missing list falls back to the default list of the requested type.
func (s *Store) resolveList(ctx context.Context, listID uuid.UUID, itemType string) (uuid.UUID, string, error) {
	if listID != uuid.Nil {
		l, err := s.GetList(ctx, listID)
		if errors.Is(err, ErrNotFound) {
			return uuid.Nil, "", ErrUnknownList
		}
		if err != nil {
			return uuid.Nil, "", err
		}
		return l.ID, l.Kind, nil
	}
	if itemType == "" {
		itemType = model.ItemTypeGrocery
	}
	id, err := s.DefaultListID(ctx, itemType)
	if errors.Is(err, ErrNotFound) {
		return uuid.Nil, "", ErrUnknownList
	}
	return id, itemType, err
}

// CreateTask inserts a task. Clients may supply their own id so that an offline queue can be
// replayed idempotently: re-inserting an existing id returns the stored row unchanged and
// created=false.
func (s *Store) CreateTask(ctx context.Context, t model.Task) (model.Task, bool, error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	listID, itemType, err := s.resolveList(ctx, t.ListID, t.ItemType)
	if err != nil {
		return model.Task{}, false, err
	}
	t.ListID, t.ItemType = listID, itemType
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = t.CreatedAt
	}
	if t.Position == 0 {
		t.Position = float64(t.CreatedAt.UnixMilli())
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO tasks (id, list_id, title, notes, is_completed, item_type, department_id, assignee_id, due_date, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
		RETURNING `+taskColumns,
		t.ID, t.ListID, t.Title, t.Notes, t.IsCompleted, t.ItemType, t.DepartmentID, t.AssigneeID, t.DueDate, t.Position, t.CreatedAt, t.UpdatedAt)
	out, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, err := s.GetTask(ctx, t.ID)
		return existing, false, err
	}
	return out, true, err
}

// UpdateTask applies a partial update with last-write-wins semantics: when the patch carries a
// client timestamp older than the stored updated_at the update is rejected with ErrConflict and
// the current row is returned so the caller can reconcile.
func (s *Store) UpdateTask(ctx context.Context, id uuid.UUID, p model.TaskPatch) (model.Task, error) {
	ts := time.Now().UTC()
	if p.UpdatedAt != nil {
		ts = p.UpdatedAt.UTC()
		// Guard against wildly skewed client clocks: a future timestamp would block every later edit.
		if ts.After(time.Now().Add(5 * time.Minute)) {
			ts = time.Now().UTC()
		}
	}

	sets := []string{"updated_at = $2"}
	args := []any{id, ts}
	set := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	// Moving between lists (or changing the type) keeps list and type consistent.
	if p.ListID.Set || p.ItemType.Set {
		listID, itemType, err := s.resolveList(ctx, p.ListID.Value, p.ItemType.Value)
		if err != nil {
			return model.Task{}, err
		}
		set("list_id", listID)
		set("item_type", itemType)
	}
	if p.Title.Set {
		set("title", p.Title.Value)
	}
	if p.Notes.Set {
		set("notes", p.Notes.Value)
	}
	if p.IsCompleted.Set {
		set("is_completed", p.IsCompleted.Value)
	}
	if p.DepartmentID.Set {
		set("department_id", p.DepartmentID.Value)
	}
	if p.AssigneeID.Set {
		set("assignee_id", p.AssigneeID.Value)
	}
	if p.DueDate.Set {
		set("due_date", p.DueDate.Value)
	}
	if p.Position.Set {
		set("position", p.Position.Value)
	}

	row := s.pool.QueryRow(ctx, `UPDATE tasks SET `+strings.Join(sets, ", ")+`
		WHERE id = $1 AND updated_at <= $2
		RETURNING `+taskColumns, args...)
	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		current, gerr := s.GetTask(ctx, id)
		if gerr != nil {
			return current, gerr // ErrNotFound or a real failure
		}
		return current, ErrConflict
	}
	return t, err
}

// DeleteTask removes a task.
func (s *Store) DeleteTask(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCompletedTasks removes every completed task of the given type and/or list (empty = all)
// and returns the ids removed so they can be broadcast.
func (s *Store) DeleteCompletedTasks(ctx context.Context, itemType string, listID *uuid.UUID) ([]uuid.UUID, error) {
	q := `DELETE FROM tasks WHERE is_completed`
	var args []any
	if itemType != "" {
		args = append(args, itemType)
		q += fmt.Sprintf(` AND item_type = $%d`, len(args))
	}
	if listID != nil {
		args = append(args, *listID)
		q += fmt.Sprintf(` AND list_id = $%d`, len(args))
	}
	rows, err := s.pool.Query(ctx, q+` RETURNING id`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		err := row.Scan(&id)
		return id, err
	})
}
