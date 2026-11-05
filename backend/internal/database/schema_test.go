package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

const ts = "2026-01-01T00:00:00Z"

const (
	idW1 = "00000000-0000-7000-8000-0000000000a1"
	idW2 = "00000000-0000-7000-8000-0000000000a2"
	idU1 = "00000000-0000-7000-8000-0000000000b1"
	idM1 = "00000000-0000-7000-8000-0000000000c1"
	idP1 = "00000000-0000-7000-8000-0000000000d1"
	idS1 = "00000000-0000-7000-8000-0000000000e1"
	idT1 = "00000000-0000-7000-8000-0000000000f1"
	idK1 = "00000000-0000-7000-8000-000000000101"
	idK2 = "00000000-0000-7000-8000-000000000102"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := New(context.Background(), filepath.Join(t.TempDir(), "schema.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func wantExecError(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err == nil {
		t.Fatalf("exec %q: expected an error, got nil", query)
	}
}

func seedBase(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `INSERT INTO users (id, display_name, created_at, updated_at) VALUES (?, 'u', ?, ?)`, idU1, ts, ts)
	mustExec(t, db, `INSERT INTO workspaces (id, kind, name, created_at, updated_at) VALUES (?, 'team', 'w1', ?, ?)`, idW1, ts, ts)
	mustExec(t, db, `INSERT INTO workspaces (id, kind, name, created_at, updated_at) VALUES (?, 'team', 'w2', ?, ?)`, idW2, ts, ts)
	mustExec(t, db, `INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at) VALUES (?, ?, ?, 'owner', ?, ?)`, idM1, idW1, idU1, ts, ts)
	mustExec(t, db, `INSERT INTO projects (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'p', ?, ?)`, idP1, idW1, ts, ts)
	mustExec(t, db, `INSERT INTO statuses (id, workspace_id, project_id, title, color, category, position, updated_at) VALUES (?, ?, ?, 's', '#000', 'todo', 'a0', ?)`, idS1, idW1, idP1, ts)
	mustExec(t, db, `INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Backend', ?, ?)`, idT1, idW1, ts, ts)
}

func TestSchemaTablesExistAndAreStrict(t *testing.T) {
	db := newSchemaDB(t)

	want := []string{
		"users", "workspaces", "memberships", "teams", "projects",
		"statuses", "project_teams", "team_members", "tasks", "task_assignees",
	}
	for _, name := range want {
		t.Run(name, func(t *testing.T) {
			var strict int
			err := db.QueryRow(
				`SELECT strict FROM pragma_table_list WHERE schema = 'main' AND name = ?`, name,
			).Scan(&strict)
			if err != nil {
				t.Fatalf("table %s missing: %v", name, err)
			}
			if strict != 1 {
				t.Errorf("table %s is not STRICT", name)
			}
		})
	}
}

func TestSchemaRejectsInvalidIDLength(t *testing.T) {
	db := newSchemaDB(t)
	wantExecError(t, db, `INSERT INTO users (id, display_name, created_at, updated_at) VALUES ('short', 'u', ?, ?)`, ts, ts)
}

func TestSchemaEnumChecks(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	tests := []struct {
		name  string
		query string
		args  []any
	}{
		{"workspace kind", `INSERT INTO workspaces (id, kind, name, created_at, updated_at) VALUES (?, 'bogus', 'x', ?, ?)`, []any{idK1, ts, ts}},
		{"membership role", `INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at) VALUES (?, ?, ?, 'god', ?, ?)`, []any{idK1, idW2, idU1, ts, ts}},
		{"status category", `INSERT INTO statuses (id, workspace_id, project_id, title, color, category, position, updated_at) VALUES (?, ?, ?, 's', '#000', 'blocked', 'a1', ?)`, []any{idK1, idW1, idP1, ts}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantExecError(t, db, tt.query, tt.args...)
		})
	}
}

func TestSchemaWorkspaceIsolation(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	t.Run("status cannot point at another workspace's project", func(t *testing.T) {
		wantExecError(t, db,
			`INSERT INTO statuses (id, workspace_id, project_id, title, color, category, position, updated_at)
			 VALUES (?, ?, ?, 's', '#000', 'todo', 'a1', ?)`, idK1, idW2, idP1, ts)
	})

	t.Run("project_team cannot join across workspaces", func(t *testing.T) {
		mustExec(t, db, `INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Other', ?, ?)`, idK1, idW2, ts, ts)
		wantExecError(t, db,
			`INSERT INTO project_teams (workspace_id, project_id, team_id, added_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			idW1, idP1, idK1, ts, ts)
	})

	t.Run("task creator must belong to the task workspace", func(t *testing.T) {
		wantExecError(t, db,
			`INSERT INTO tasks (id, workspace_id, project_id, status_id, created_by, title, position, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, 't', 'a0', ?, ?)`, idK1, idW2, idP1, idS1, idM1, ts, ts)
	})
}

func TestSchemaForeignKeysAreEnforced(t *testing.T) {
	db := newSchemaDB(t)
	wantExecError(t, db,
		`INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at) VALUES (?, ?, ?, 'owner', ?, ?)`,
		idM1, idW1, idU1, ts, ts)
}

func TestSchemaMembershipUniquePerWorkspace(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	wantExecError(t, db,
		`INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at) VALUES (?, ?, ?, 'member', ?, ?)`,
		idK1, idW1, idU1, ts, ts)
}

func TestSchemaTeamNameUniqueAmongActiveTeams(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	wantExecError(t, db,
		`INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Backend', ?, ?)`,
		idK1, idW1, ts, ts)

	mustExec(t, db, `UPDATE teams SET deleted_at = ? WHERE id = ?`, ts, idT1)
	mustExec(t, db,
		`INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Backend', ?, ?)`,
		idK1, idW1, ts, ts)
}

func TestSchemaTaskTeamMustBelongToProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	insert := `INSERT INTO tasks (id, workspace_id, project_id, status_id, team_id, created_by, title, position, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, ?, 't', 'a0', ?, ?)`

	wantExecError(t, db, insert, idK1, idW1, idP1, idS1, idT1, idM1, ts, ts)

	mustExec(t, db,
		`INSERT INTO project_teams (workspace_id, project_id, team_id, added_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		idW1, idP1, idT1, ts, ts)
	mustExec(t, db, insert, idK1, idW1, idP1, idS1, idT1, idM1, ts, ts)
}

func TestSchemaTaskWithoutTeamSkipsTeamConstraint(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	mustExec(t, db,
		`INSERT INTO tasks (id, workspace_id, project_id, status_id, created_by, title, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 't', 'a0', ?, ?)`, idK1, idW1, idP1, idS1, idM1, ts, ts)
}

func TestSchemaSubtaskMustShareProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	mustExec(t, db,
		`INSERT INTO tasks (id, workspace_id, project_id, status_id, created_by, title, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'parent', 'a0', ?, ?)`, idK1, idW1, idP1, idS1, idM1, ts, ts)
	mustExec(t, db,
		`INSERT INTO tasks (id, workspace_id, project_id, status_id, parent_id, created_by, title, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'child', 'a1', ?, ?)`, idK2, idW1, idP1, idS1, idK1, idM1, ts, ts)

	p2 := "00000000-0000-7000-8000-000000000201"
	s2 := "00000000-0000-7000-8000-000000000202"
	k3 := "00000000-0000-7000-8000-000000000203"
	mustExec(t, db, `INSERT INTO projects (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'p2', ?, ?)`, p2, idW1, ts, ts)
	mustExec(t, db, `INSERT INTO statuses (id, workspace_id, project_id, title, color, category, position, updated_at) VALUES (?, ?, ?, 's', '#000', 'todo', 'a0', ?)`, s2, idW1, p2, ts)
	wantExecError(t, db,
		`INSERT INTO tasks (id, workspace_id, project_id, status_id, parent_id, created_by, title, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'cross', 'a0', ?, ?)`, k3, idW1, p2, s2, idK1, idM1, ts, ts)
}

func TestSchemaTaskStatusMustShareProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	p2 := "00000000-0000-7000-8000-000000000201"
	mustExec(t, db, `INSERT INTO projects (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'p2', ?, ?)`, p2, idW1, ts, ts)
	wantExecError(t, db,
		`INSERT INTO tasks (id, workspace_id, project_id, status_id, created_by, title, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 't', 'a0', ?, ?)`, idK1, idW1, p2, idS1, idM1, ts, ts)
}

func TestSchemaForeignKeyRestrictsDelete(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	wantExecError(t, db, `DELETE FROM projects WHERE id = ?`, idP1)
}

func TestSchemaIndexesExist(t *testing.T) {
	db := newSchemaDB(t)

	want := []string{
		"idx_memberships_user_id",
		"idx_teams_workspace_id_name",
		"idx_projects_workspace_id",
		"idx_statuses_project_id_position",
		"idx_project_teams_team_id",
		"idx_team_members_team_id",
		"idx_tasks_parent_id",
		"idx_tasks_project_id_team_id",
		"idx_tasks_project_id_status_id_position",
		"idx_tasks_workspace_id_updated_at",
		"idx_task_assignees_membership_id",
	}
	for _, name := range want {
		t.Run(name, func(t *testing.T) {
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 1 {
				t.Errorf("index %s missing", name)
			}
		})
	}
}