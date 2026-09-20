package api

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// isForeignKeyViolation reports whether err is PostgreSQL error 23503.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
