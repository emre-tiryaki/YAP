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