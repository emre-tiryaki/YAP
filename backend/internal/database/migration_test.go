
package database

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestMigrate(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open(
		"sqlite",
		"file:"+filepath.Join(t.TempDir(), "test.db")+"?"+pragmas,
	)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate() error = %v", err)
	}

	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("reading user_version: %v", err)
	}

	files, err := fs.ReadDir(migrationFS, "migrations/sqlite")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if version != len(files) {
		t.Errorf("user_version = %d, want %d", version, len(files))
	}
}

func TestMigrateIdempotency(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open(
		"sqlite",
		"file:"+filepath.Join(t.TempDir(), "test.db")+"?"+pragmas,
	)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("first migrate() error = %v", err)
	}

	var firstVersion int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&firstVersion); err != nil {
		t.Fatal(err)
	}

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("second migrate() error = %v", err)
	}

	var secondVersion int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&secondVersion); err != nil {
		t.Fatal(err)
	}

	if firstVersion != secondVersion {
		t.Errorf(
			"migration version changed: first = %d, second = %d",
			firstVersion,
			secondVersion,
		)
	}
}
