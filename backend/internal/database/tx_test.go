package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func newTxTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := New(context.Background(), filepath.Join(t.TempDir(), "tx.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE tx_probe (v TEXT NOT NULL) STRICT`); err != nil {
		t.Fatal(err)
	}
	return db
}

func countProbe(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tx_probe`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWithTxCommit(t *testing.T) {
	db := newTxTestDB(t)

	err := WithTx(context.Background(), db, func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO tx_probe (v) VALUES ('a')`)
		return err
	})
	if err != nil {
		t.Fatalf("WithTx() error = %v", err)
	}
	if n := countProbe(t, db); n != 1 {
		t.Errorf("rows = %d, want 1", n)
	}
}

func TestWithTxRollbackOnError(t *testing.T) {
	db := newTxTestDB(t)
	sentinel := errors.New("boom")

	err := WithTx(context.Background(), db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO tx_probe (v) VALUES ('a')`); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("WithTx() error = %v, want %v", err, sentinel)
	}
	if n := countProbe(t, db); n != 0 {
		t.Errorf("rows = %d, want 0 after rollback", n)
	}
}

func TestWithTxCommitFailureIsReported(t *testing.T) {
	db := newTxTestDB(t)

	err := WithTx(context.Background(), db, func(tx *sql.Tx) error {
		return tx.Commit()
	})
	if err == nil {
		t.Fatal("WithTx() expected an error when the transaction is already committed")
	}
}

func TestWithTxCanceledContext(t *testing.T) {
	db := newTxTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	err := WithTx(ctx, db, func(tx *sql.Tx) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("WithTx() expected an error for a canceled context")
	}
	if called {
		t.Error("fn must not run when the transaction cannot begin")
	}
}
func TestIsConstraint(t *testing.T) {
	ctx := context.Background()
	db, err := New(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, fkErr := db.ExecContext(ctx,
		`INSERT INTO projects (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'p', 'x', 'x')`,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002")
	_, syntaxErr := db.ExecContext(ctx, `INSERT INTO nowhere VALUES (1)`)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"foreign key", fkErr, true},
		{"other sqlite error", syntaxErr, false},
		{"plain error", errors.New("x"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConstraint(tt.err); got != tt.want {
				t.Errorf("IsConstraint(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
