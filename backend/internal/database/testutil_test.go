package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

type fixture struct {
	accA, accB                   int64
	teamA, teamB                 int64
	memA, memB                   int64
	projA, projB                 int64
	todoA, doneA, cancelA, todoB int64
	taskA, taskB                 int64
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := New(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("seed %q: %v", query, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

func seed(t *testing.T, db *sql.DB) fixture {
	t.Helper()
	var f fixture

	f.accA = insertID(t, db, `INSERT INTO accounts (email, password_hash, display_name) VALUES ('a@x.com', 'h', 'A')`)
	f.accB = insertID(t, db, `INSERT INTO accounts (email, password_hash, display_name) VALUES ('b@x.com', 'h', 'B')`)

	f.teamA = insertID(t, db, `INSERT INTO teams (name) VALUES ('team-a')`)
	f.teamB = insertID(t, db, `INSERT INTO teams (name) VALUES ('team-b')`)

	member := `INSERT INTO memberships (team_id, account_id, role) VALUES (?, ?, 'owner')`
	f.memA = insertID(t, db, member, f.teamA, f.accA)
	f.memB = insertID(t, db, member, f.teamB, f.accB)

	project := `INSERT INTO projects (team_id, name) VALUES (?, 'Alpha')`
	f.projA = insertID(t, db, project, f.teamA)
	f.projB = insertID(t, db, project, f.teamB)

	status := func(projectID int64, title, category string, pos int) int64 {
		return insertID(t, db,
			`INSERT INTO statuses (project_id, title, color, category, position) VALUES (?, ?, '#aabbcc', ?, ?)`,
			projectID, title, category, pos)
	}
	f.todoA = status(f.projA, "Todo", "todo", 0)
	f.doneA = status(f.projA, "Done", "done", 1)
	f.cancelA = status(f.projA, "Cancelled", "cancelled", 2)
	f.todoB = status(f.projB, "Todo", "todo", 0)

	task := `INSERT INTO tasks (team_id, project_id, status_id, created_by, title) VALUES (?, ?, ?, ?, 'seed')`
	f.taskA = insertID(t, db, task, f.teamA, f.projA, f.todoA, f.memA)
	f.taskB = insertID(t, db, task, f.teamB, f.projB, f.todoB, f.memB)

	return f
}

func newTask(t *testing.T, db *sql.DB, f fixture, statusID int64) int64 {
	t.Helper()
	return insertID(t, db,
		`INSERT INTO tasks (team_id, project_id, status_id, created_by, title) VALUES (?, ?, ?, ?, 'x')`,
		f.teamA, f.projA, statusID, f.memA)
}

func closedAt(t *testing.T, db *sql.DB, taskID int64) sql.NullInt64 {
	t.Helper()
	var v sql.NullInt64
	if err := db.QueryRow(`SELECT closed_at FROM tasks WHERE id = ?`, taskID).Scan(&v); err != nil {
		t.Fatalf("closed_at: %v", err)
	}
	return v
}

func count(t *testing.T, db *sql.DB, table, where string, args ...any) int {
	t.Helper()
	var n int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, table, where)
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}