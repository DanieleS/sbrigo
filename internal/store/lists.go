package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danieles/sbrigo/internal/model"
)

// ErrLastList is returned when deleting the only list of its kind.
var ErrLastList = errors.New("cannot delete the last list of its kind")

const listColumns = `id, name, kind, sort_order, created_at, updated_at`

func scanList(row pgx.Row) (model.List, error) {
	var l model.List
	err := row.Scan(&l.ID, &l.Name, &l.Kind, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

// ListLists returns every list with its preview figures.
func (s *Store) ListLists(ctx context.Context) ([]model.ListSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT l.id, l.name, l.kind, l.sort_order, l.created_at, l.updated_at,
		       COUNT(t.id) FILTER (WHERE NOT t.is_completed),
		       COUNT(t.id) FILTER (WHERE t.is_completed),
		       COUNT(t.id) FILTER (WHERE NOT t.is_completed AND t.due_date < date_trunc('day', now())),
		       MIN(t.due_date) FILTER (WHERE NOT t.is_completed)
		FROM lists l LEFT JOIN tasks t ON t.list_id = l.id
		GROUP BY l.id
		ORDER BY l.kind, l.sort_order, l.name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ListSummary, error) {
		var l model.ListSummary
		err := row.Scan(&l.ID, &l.Name, &l.Kind, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt,
			&l.OpenCount, &l.DoneCount, &l.OverdueCount, &l.NextDue)
		return l, err
	})
}

// GetList returns one list.
func (s *Store) GetList(ctx context.Context, id uuid.UUID) (model.List, error) {
	l, err := scanList(s.pool.QueryRow(ctx, `SELECT `+listColumns+` FROM lists WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// DefaultListID returns the first list of the given kind.
func (s *Store) DefaultListID(ctx context.Context, kind string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM lists WHERE kind = $1 ORDER BY sort_order, name LIMIT 1`, kind).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// CreateList inserts a list.
func (s *Store) CreateList(ctx context.Context, l model.List) (model.List, error) {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO lists (id, name, kind, sort_order) VALUES ($1, $2, $3, $4)
		RETURNING `+listColumns, l.ID, l.Name, l.Kind, l.SortOrder)
	return scanList(row)
}

// UpdateList renames or reorders a list; its kind is immutable.
func (s *Store) UpdateList(ctx context.Context, l model.List) (model.List, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE lists SET name = $2, sort_order = $3, updated_at = now() WHERE id = $1
		RETURNING `+listColumns, l.ID, l.Name, l.SortOrder)
	out, err := scanList(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

// DeleteList removes a list and, by cascade, its tasks. The last list of a kind is protected so
// that quick-add always has a destination.
func (s *Store) DeleteList(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx)) //nolint:errcheck

	var kind string
	if err := tx.QueryRow(ctx, `SELECT kind FROM lists WHERE id = $1 FOR UPDATE`, id).Scan(&kind); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var siblings int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM lists WHERE kind = $1`, kind).Scan(&siblings); err != nil {
		return err
	}
	if siblings <= 1 {
		return ErrLastList
	}
	if _, err := tx.Exec(ctx, `DELETE FROM lists WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
