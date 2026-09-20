package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danieles/sbrigo/internal/model"
)

func scanSupermarket(row pgx.Row) (model.Supermarket, error) {
	var m model.Supermarket
	err := row.Scan(&m.ID, &m.Name)
	return m, err
}

// ListSupermarkets returns all supermarkets alphabetically.
func (s *Store) ListSupermarkets(ctx context.Context) ([]model.Supermarket, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM supermarkets ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Supermarket, error) { return scanSupermarket(row) })
}

// GetSupermarket returns one supermarket.
func (s *Store) GetSupermarket(ctx context.Context, id uuid.UUID) (model.Supermarket, error) {
	m, err := scanSupermarket(s.pool.QueryRow(ctx, `SELECT id, name FROM supermarkets WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

// CreateSupermarket inserts a supermarket. A zero id is replaced by a fresh UUID.
func (s *Store) CreateSupermarket(ctx context.Context, m model.Supermarket) (model.Supermarket, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	row := s.pool.QueryRow(ctx, `INSERT INTO supermarkets (id, name) VALUES ($1, $2) RETURNING id, name`, m.ID, m.Name)
	return scanSupermarket(row)
}

// UpdateSupermarket renames a supermarket.
func (s *Store) UpdateSupermarket(ctx context.Context, m model.Supermarket) (model.Supermarket, error) {
	row := s.pool.QueryRow(ctx, `UPDATE supermarkets SET name = $2 WHERE id = $1 RETURNING id, name`, m.ID, m.Name)
	out, err := scanSupermarket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

// DeleteSupermarket removes a supermarket and, by cascade, its aisle order.
func (s *Store) DeleteSupermarket(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM supermarkets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListDepartmentOrder returns the aisle sequence configured for a supermarket.
func (s *Store) ListDepartmentOrder(ctx context.Context, supermarketID uuid.UUID) ([]model.DepartmentOrder, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT department_id, sort_order FROM supermarket_department_orders
		WHERE supermarket_id = $1 ORDER BY sort_order`, supermarketID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.DepartmentOrder, error) {
		var o model.DepartmentOrder
		err := row.Scan(&o.DepartmentID, &o.SortOrder)
		return o, err
	})
}

// ReplaceDepartmentOrder atomically replaces the aisle sequence of a supermarket with the given
// department ids, in order. Departments not listed keep no explicit position and fall back to
// their default order in the UI.
func (s *Store) ReplaceDepartmentOrder(ctx context.Context, supermarketID uuid.UUID, departmentIDs []uuid.UUID) ([]model.DepartmentOrder, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx)) //nolint:errcheck

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM supermarkets WHERE id = $1)`, supermarketID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM supermarket_department_orders WHERE supermarket_id = $1`, supermarketID); err != nil {
		return nil, err
	}
	out := make([]model.DepartmentOrder, 0, len(departmentIDs))
	for i, depID := range departmentIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO supermarket_department_orders (supermarket_id, department_id, sort_order)
			VALUES ($1, $2, $3)`, supermarketID, depID, i+1); err != nil {
			return nil, err
		}
		out = append(out, model.DepartmentOrder{DepartmentID: depID, SortOrder: i + 1})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
