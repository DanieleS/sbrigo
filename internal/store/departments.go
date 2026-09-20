package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danieles/sbrigo/internal/model"
)

const departmentColumns = `id, name, description, default_sort_order`

func scanDepartment(row pgx.Row) (model.Department, error) {
	var d model.Department
	err := row.Scan(&d.ID, &d.Name, &d.Description, &d.DefaultSortOrder)
	return d, err
}

// ListDepartments returns all departments in default order.
func (s *Store) ListDepartments(ctx context.Context) ([]model.Department, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+departmentColumns+` FROM departments ORDER BY default_sort_order, name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Department, error) { return scanDepartment(row) })
}

// GetDepartment returns one department.
func (s *Store) GetDepartment(ctx context.Context, id uuid.UUID) (model.Department, error) {
	d, err := scanDepartment(s.pool.QueryRow(ctx, `SELECT `+departmentColumns+` FROM departments WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}

// CreateDepartment inserts a department. A zero id is replaced by a fresh UUID.
func (s *Store) CreateDepartment(ctx context.Context, d model.Department) (model.Department, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO departments (id, name, description, default_sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING `+departmentColumns, d.ID, d.Name, d.Description, d.DefaultSortOrder)
	return scanDepartment(row)
}

// UpdateDepartment replaces name, description and default order.
func (s *Store) UpdateDepartment(ctx context.Context, d model.Department) (model.Department, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE departments SET name = $2, description = $3, default_sort_order = $4
		WHERE id = $1
		RETURNING `+departmentColumns, d.ID, d.Name, d.Description, d.DefaultSortOrder)
	out, err := scanDepartment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

// DeleteDepartment removes a department; tasks referencing it fall back to NULL.
func (s *Store) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM departments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
