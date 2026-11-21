package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

func TestMigration011BackfillsIdentifiersAndNumbers(t *testing.T) {
	ctx := context.Background()
	db := openAtVersion(t, 10)
	seedBase(t, db) // project idP1 is named "p": one letter, so it gets PRJ.

	proj := func(id, ws, name, createdAt string) {
		mustExec(t, db, `INSERT INTO projects (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			id, ws, name, createdAt, createdAt)
	}
	pid := func(n int) string { return fmt.Sprintf("00000000-0000-7000-8000-0000000004%02d", n) }
	proj(pid(1), idW1, "Kitap Kulübü", "2026-01-01T00:00:01Z")
	proj(pid(2), idW1, "Kitchen", "2026-01-01T00:00:02Z") // same base KIT: second
	proj(pid(3), idW1, "kit-3d", "2026-01-01T00:00:03Z")  // KIT again: third
	proj(pid(4), idW2, "Kitap", "2026-01-01T00:00:04Z")   // other workspace: KIT again, first there
	proj(pid(5), idW1, "  go ", "2026-01-01T00:00:05Z")   // GO
	proj(pid(6), idW1, "Öğrenme", "2026-01-01T00:00:06Z") // only RENME letters after the non-ASCII ones: REN
	proj(pid(7), idW1, "12345", "2026-01-01T00:00:07Z")   // no letters: PRJ again
	mustExec(t, db, `UPDATE projects SET deleted_at = ? WHERE id = ?`, ts, pid(7))

	// Tasks of the first project, created out of id order; one is deleted.
	task := func(id, project, createdAt string) {
		mustExec(t, db, `INSERT INTO tasks (id, workspace_id, project_id, plan_id, status_id, created_by, title, position, created_at, updated_at)
			SELECT ?, workspace_id, ?, id, (SELECT id FROM statuses WHERE plan_id = plans.id LIMIT 1), ?, 't', 'a0', ?, ?
			  FROM plans WHERE project_id = ? LIMIT 1`, id, project, idM1, createdAt, createdAt, project)
	}
	tid := func(n int) string { return fmt.Sprintf("00000000-0000-7000-8000-0000000005%02d", n) }
	mustExec(t, db, insertPlan, "00000000-0000-7000-8000-000000000601", idW1, pid(1), nil, "Main", ts, ts)
	mustExec(t, db, insertStatus, "00000000-0000-7000-8000-000000000602", idW1, "00000000-0000-7000-8000-000000000601", ts)
	task(tid(3), pid(1), "2026-02-01T00:00:03Z")
	task(tid(1), pid(1), "2026-02-01T00:00:01Z")
	task(tid(2), pid(1), "2026-02-01T00:00:02Z")
	mustExec(t, db, `UPDATE tasks SET deleted_at = ? WHERE id = ?`, ts, tid(2))

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate() error = %v", err)
	}

	wantIdent := map[string]string{
		idP1: "PRJ", pid(1): "KIT", pid(2): "KIT2", pid(3): "KIT3", pid(4): "KIT",
		pid(5): "GO", pid(6): "REN", pid(7): "PRJ2",
	}
	for id, want := range wantIdent {
		var got sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT identifier FROM projects WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got.String != want {
			t.Errorf("identifier of %s = %q, want %q", id, got.String, want)
		}
	}

	wantNumber := map[string]int{tid(1): 1, tid(2): 2, tid(3): 3}
	for id, want := range wantNumber {
		var got int
		if err := db.QueryRowContext(ctx, `SELECT number FROM tasks WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("number of %s = %d, want %d", id, got, want)
		}
	}
	var next, empty int
	if err := db.QueryRowContext(ctx, `SELECT next_task_number FROM projects WHERE id = ?`, pid(1)).Scan(&next); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT next_task_number FROM projects WHERE id = ?`, pid(2)).Scan(&empty); err != nil {
		t.Fatal(err)
	}
	if next != 4 || empty != 1 {
		t.Errorf("next_task_number = %d and %d, want 4 and 1", next, empty)
	}

	// The unique indexes hold.
	wantExecError(t, db, `INSERT INTO projects (id, workspace_id, name, identifier, created_at, updated_at) VALUES (?, ?, 'x', 'KIT', ?, ?)`,
		"00000000-0000-7000-8000-000000000999", idW1, ts, ts)
	wantExecError(t, db, `UPDATE tasks SET number = 1 WHERE id = ?`, tid(3))
}
