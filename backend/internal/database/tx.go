package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
)

// sqliteConstraint is SQLITE_CONSTRAINT, the primary result code of every
// constraint failure.
const sqliteConstraint = 19

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Commit tx: %w", err)
	}
	return nil
}

// IsConstraint reports whether err is SQLite refusing a write because a
// constraint (CHECK, NOT NULL, UNIQUE, FOREIGN KEY) failed.
func IsConstraint(err error) bool {
	var se *sqlite.Error
	// The primary result code is the low byte of the extended code.
	return errors.As(err, &se) && se.Code()&0xff == sqliteConstraint
}
