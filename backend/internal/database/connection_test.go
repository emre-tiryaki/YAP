package database

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"regexp"
	"testing"
)

func TestNew_PragmasApplyToEveryPooledConnection(t *testing.T) {
	db := newTestDB(t)
	const n = 4
	db.SetMaxOpenConns(n)
	ctx := context.Background()

	conns := make([]*sql.Conn, 0, n)
	for i := 0; i < n; i++ {
		c, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("Conn %d: %v", i, err)
		}
		t.Cleanup(func() { c.Close() })
		conns = append(conns, c)
	}

	for i, c := range conns {
		var fk, busy int
		var journal string
		if err := c.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
			t.Fatal(err)
		}
		if err := c.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
			t.Fatal(err)
		}
		if err := c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journal); err != nil {
			t.Fatal(err)
		}
		if fk != 1 || busy != 5000 || journal != "wal" {
			t.Errorf("conn %d: foreign_keys=%d busy_timeout=%d journal_mode=%s", i, fk, busy, journal)
		}
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	files, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "idem.db")

	for run := 1; run <= 3; run++ {
		db, err := New(context.Background(), path)
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		var version int
		if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
			t.Fatal(err)
		}
		db.Close()
		if version != len(files) {
			t.Fatalf("run %d: user_version=%d, want %d", run, version, len(files))
		}
	}
}

func TestMigrate_FileNamesAreOrdered(t *testing.T) {
	files, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no migration files embedded")
	}
	pattern := regexp.MustCompile(`^(\d{3})_[a-z0-9_]+\.sql$`)
	for i, f := range files {
		m := pattern.FindStringSubmatch(f.Name())
		if m == nil {
			t.Errorf("bad migration name: %s", f.Name())
			continue
		}
		want := i + 1
		var got int
		for _, ch := range m[1] {
			got = got*10 + int(ch-'0')
		}
		if got != want {
			t.Errorf("%s: sequence number %d, want %d (gap or duplicate)", f.Name(), got, want)
		}
	}
}

func TestMigrate_CreatesExpectedObjects(t *testing.T) {
	db := newTestDB(t)
	cases := []struct {
		kind string
		name string
	}{
		{"table", "accounts"}, {"table", "teams"}, {"table", "memberships"},
		{"table", "projects"}, {"table", "statuses"}, {"table", "tasks"},
		{"table", "task_assignees"},
		{"trigger", "trg_tasks_closed_on_insert"}, {"trigger", "trg_tasks_closed_on_status"},
		{"index", "idx_tasks_project_status"}, {"index", "idx_tasks_parent"},
		{"index", "idx_memberships_account"}, {"index", "idx_task_assignees_member"},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.name, func(t *testing.T) {
			if n := count(t, db, "sqlite_master", "type = ? AND name = ?", tc.kind, tc.name); n != 1 {
				t.Errorf("found %d, want 1", n)
			}
		})
	}
}