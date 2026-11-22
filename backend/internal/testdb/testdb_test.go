package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func TestMain(m *testing.M) { os.Exit(Run(m)) }

func TestPathGivesIndependentMigratedCopies(t *testing.T) {
	ctx := context.Background()
	a, err := database.New(ctx, Path(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := database.New(ctx, Path(t))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	var version int
	if err := a.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version == 0 {
		t.Fatalf("user_version = %d, %v; want a migrated database", version, err)
	}
	var fk int
	if err := a.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys = %d, %v", fk, err)
	}
	if _, err := a.Exec(`INSERT INTO users (id, display_name, created_at, updated_at) VALUES (?, 'x', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`, model.NewID()); err != nil {
		t.Fatalf("insert into the first copy: %v", err)
	}
	var n int
	if err := b.QueryRow("SELECT COUNT(*) FROM users").Scan(&n); err != nil || n != 0 {
		t.Fatalf("second copy has %d users (%v); copies must be independent", n, err)
	}
}
