package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
)

// openAtVersion returns a database migrated only up to version, so a later
// migration can be run against data written under the older schema.
func openAtVersion(t *testing.T, version int) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?%s", path, pragmas))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := fs.ReadDir(migrationFS, "migrations/sqlite")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files[:version] {
		body, err := migrationFS.ReadFile("migrations/sqlite/" + f.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("%s: %v", f.Name(), err)
		}
	}
	mustExec(t, db, fmt.Sprintf("PRAGMA user_version = %d", version))
	return db
}

func TestMigration005BackfillsCompletedAt(t *testing.T) {
	const (
		idDone     = "00000000-0000-7000-8000-000000000301"
		idOpen     = "00000000-0000-7000-8000-000000000302"
		idDeleted  = "00000000-0000-7000-8000-000000000303"
		idDoneStat = "00000000-0000-7000-8000-000000000304"
		doneAt     = "2026-03-04T05:06:07Z"
	)
	ctx := context.Background()
	db := openAtVersion(t, 4)
	seedBase(t, db)
	mustExec(t, db, `INSERT INTO statuses (id, workspace_id, plan_id, title, color, category, position, updated_at)
		VALUES (?, ?, ?, 'Done', '#000', 'done', 'a1', ?)`, idDoneStat, idW1, idL1, ts)
	mustExec(t, db, insertTask, idDone, idW1, idP1, idL1, idDoneStat, nil, idM1, ts, doneAt)
	mustExec(t, db, insertTask, idOpen, idW1, idP1, idL1, idS1, nil, idM1, ts, ts)
	mustExec(t, db, insertTask, idDeleted, idW1, idP1, idL1, idDoneStat, nil, idM1, ts, ts)
	mustExec(t, db, `UPDATE tasks SET deleted_at = ? WHERE id = ?`, ts, idDeleted)

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate() error = %v", err)
	}

	tests := []struct {
		name, id string
		want     sql.NullString
	}{
		{"task in done status is finished at its last update", idDone, sql.NullString{String: doneAt, Valid: true}},
		{"task in todo status stays open", idOpen, sql.NullString{}},
		{"deleted task is left alone", idDeleted, sql.NullString{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got sql.NullString
			if err := db.QueryRowContext(ctx, `SELECT completed_at FROM tasks WHERE id = ?`, tt.id).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("completed_at = %+v, want %+v", got, tt.want)
			}
		})
	}
}
