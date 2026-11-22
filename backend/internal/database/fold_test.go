package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFold(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Ödev", "ödev"},
		{"ŞEKER", "şeker"},
		{"plain", "plain"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := Fold(tt.in); got != tt.want {
			t.Errorf("Fold(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFoldFunctionInSQL(t *testing.T) {
	ctx := context.Background()
	db, err := New(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var text string
	if err := db.QueryRowContext(ctx, `SELECT `+FoldFunction+`('ÖDEV Planı')`).Scan(&text); err != nil || text != "ödev planı" {
		t.Fatalf("text fold = %q, %v", text, err)
	}
	// SQLite's own lower() leaves non-ASCII letters alone, which is why the
	// function exists.
	var builtin string
	if err := db.QueryRowContext(ctx, `SELECT lower('ÖDEV')`).Scan(&builtin); err != nil || builtin == "ödev" {
		t.Fatalf("lower() = %q, %v; the custom function would be pointless", builtin, err)
	}

	var blob string
	if err := db.QueryRowContext(ctx, `SELECT `+FoldFunction+`(CAST('ŞEKER' AS BLOB))`).Scan(&blob); err != nil || blob != "şeker" {
		t.Fatalf("blob fold = %q, %v", blob, err)
	}

	// Other types pass through untouched.
	var n int
	if err := db.QueryRowContext(ctx, `SELECT `+FoldFunction+`(42)`).Scan(&n); err != nil || n != 42 {
		t.Fatalf("int fold = %d, %v", n, err)
	}
	var null *string
	if err := db.QueryRowContext(ctx, `SELECT `+FoldFunction+`(NULL)`).Scan(&null); err != nil || null != nil {
		t.Fatalf("null fold = %v, %v", null, err)
	}

	var match int
	if err := db.QueryRowContext(ctx, `SELECT `+FoldFunction+`('Ödev') LIKE '%'||`+FoldFunction+`('ÖD')||'%'`).Scan(&match); err != nil || match != 1 {
		t.Fatalf("folded LIKE = %d, %v", match, err)
	}
}

func TestNewCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db, err := New(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err == nil {
		db.Close()
		t.Fatal("New() with a canceled context returned no error")
	}
}

func TestMigrateLeavesNewerDatabaseAlone(t *testing.T) {
	ctx := context.Background()
	db, err := New(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		t.Fatal(err)
	}
	// A database written by a newer build must not be downgraded or fail.
	if _, err := db.ExecContext(ctx, "PRAGMA user_version = 999"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate() on a newer database error = %v", err)
	}
	var after int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&after); err != nil || after != 999 {
		t.Fatalf("user_version = %d, %v; want 999 (was %d)", after, err, current)
	}
}

func TestMigrateStopsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db, err := New(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db); err == nil {
		t.Fatal("migrate() with a canceled context returned no error")
	}
}
