
package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	path := filepath.Join(t.TempDir(), "nested", "app.db")

	db, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("PingContext() error = %v", err)
	}
}

func TestNewPragmas(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")

	db, err := New(ctx, path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("Conn() error = %v", err)
	}
	defer conn.Close()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "foreign_keys",
			query:    "PRAGMA foreign_keys",
			expected: "1",
		},
		{
			name:     "journal_mode",
			query:    "PRAGMA journal_mode",
			expected: "wal",
		},
		{
			name:     "busy_timeout",
			query:    "PRAGMA busy_timeout",
			expected: "5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string

			if err := conn.QueryRowContext(ctx, tt.query).Scan(&got); err != nil {
				t.Fatalf("QueryRowContext() error = %v", err)
			}

			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewInvalidDirectory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(file, "app.db")

	db, err := New(ctx, path)
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("New() expected an error, got nil")
	}
}
