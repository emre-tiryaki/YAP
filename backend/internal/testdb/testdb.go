// Package testdb hands tests a migrated SQLite database quickly. The schema
// is built once per test binary with database.New, so migrations and pragmas
// are the production ones, and every test gets its own copy of that file.
// Only tests import this package.
package testdb

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
)

var (
	once     sync.Once
	template string
	dir      string
	buildErr error
)

func build() {
	dir, buildErr = os.MkdirTemp("", "yap-testdb-")
	if buildErr != nil {
		return
	}
	template = filepath.Join(dir, "template.db")
	db, err := database.New(context.Background(), template)
	if err != nil {
		buildErr = err
		return
	}
	// Closing the last connection checkpoints the WAL into the main file.
	buildErr = db.Close()
}

// Path returns the path of a fresh, fully migrated database file inside
// t.TempDir(). Open it with database.New.
func Path(t testing.TB) string {
	t.Helper()
	once.Do(build)
	if buildErr != nil {
		t.Fatalf("build template database: %v", buildErr)
	}
	dst := filepath.Join(t.TempDir(), "app.db")
	if err := copyFile(template, dst); err != nil {
		t.Fatalf("copy template database: %v", err)
	}
	return dst
}

// Run runs the tests of a package and removes the template afterwards. Call
// it from TestMain: os.Exit(testdb.Run(m)).
func Run(m *testing.M) int {
	code := m.Run()
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	return code
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy: %w", err)
	}
	return out.Close()
}
