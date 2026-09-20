package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danieles/sbrigo/internal/model"
)

const userColumns = `id, identity_id, email, display_name, created_at`

func scanUser(row pgx.Row) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.IdentityID, &u.Email, &u.DisplayName, &u.CreatedAt)
	return u, err
}

// UpsertUser creates the local profile on first login and refreshes email/name on later logins.
func (s *Store) UpsertUser(ctx context.Context, identityID, email, displayName string) (model.User, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (id, identity_id, email, display_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (identity_id) DO UPDATE SET
			email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE users.email END,
			display_name = CASE WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name ELSE users.display_name END
		RETURNING `+userColumns,
		uuid.New(), identityID, email, displayName)
	u, err := scanUser(row)
	if err != nil {
		return u, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

// GetUser returns one user by id.
func (s *Store) GetUser(ctx context.Context, id uuid.UUID) (model.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// ListUsers returns every family member, useful for assignment pickers.
func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY display_name, email`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.User, error) { return scanUser(row) })
}
