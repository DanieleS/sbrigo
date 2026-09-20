// Package store implements persistence on top of PostgreSQL.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors returned by the store.
var (
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when a last-write-wins update loses against a newer server-side version.
	ErrConflict = errors.New("stale update")
)

// Store groups all repositories over a single pool.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps a pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
