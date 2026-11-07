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
	idK3 = "00000000-0000-7000-8000-000000000103"
	idL1 = "00000000-0000-7000-8000-000000000111"
	idL2 = "00000000-0000-7000-8000-000000000112"
	idS2 = "00000000-0000-7000-8000-000000000113"
)

const (
	insertPlan = `INSERT INTO plans (id, workspace_id, project_id, team_id, name, created_at, updated_at)
	              VALUES (?, ?, ?, ?, ?, ?, ?)`
	insertStatus = `INSERT INTO statuses (id, workspace_id, plan_id, title, color, category, position, updated_at)
	                VALUES (?, ?, ?, 's', '#000', 'todo', 'a0', ?)`
	insertTask = `INSERT INTO tasks (id, workspace_id, project_id, plan_id, status_id, parent_id, created_by, title, position, created_at, updated_at)
	              VALUES (?, ?, ?, ?, ?, ?, ?, 't', 'a0', ?, ?)`
	insertProjectTeam = `INSERT INTO project_teams (workspace_id, project_id, team_id, added_at, updated_at) VALUES (?, ?, ?, ?, ?)`
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
	mustExec(t, db, insertPlan, idL1, idW1, idP1, nil, "Main", ts, ts)
	mustExec(t, db, insertStatus, idS1, idW1, idL1, ts)
	mustExec(t, db, `INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Backend', ?, ?)`, idT1, idW1, ts, ts)
}

func seedSecondPlan(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, insertPlan, idL2, idW1, idP1, nil, "Second", ts, ts)
	mustExec(t, db, insertStatus, idS2, idW1, idL2, ts)
}

type secondProject struct {
	projectID string
	planID    string
	statusID  string
}

func seedSecondProject(t *testing.T, db *sql.DB) secondProject {
	t.Helper()
	sp := secondProject{
		projectID: "00000000-0000-7000-8000-000000000201",
		statusID:  "00000000-0000-7000-8000-000000000202",
		planID:    "00000000-0000-7000-8000-000000000204",
	}
	mustExec(t, db, `INSERT INTO projects (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'p2', ?, ?)`, sp.projectID, idW1, ts, ts)
	mustExec(t, db, insertPlan, sp.planID, idW1, sp.projectID, nil, "Main", ts, ts)
	mustExec(t, db, insertStatus, sp.statusID, idW1, sp.planID, ts)
	return sp
}

func TestSchemaTablesExistAndAreStrict(t *testing.T) {
	db := newSchemaDB(t)

	want := []string{
		"users", "workspaces", "memberships", "teams", "projects", "plans",
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
		{"status category", `INSERT INTO statuses (id, workspace_id, plan_id, title, color, category, position, updated_at) VALUES (?, ?, ?, 's', '#000', 'blocked', 'a1', ?)`, []any{idK1, idW1, idL1, ts}},
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

	t.Run("plan cannot point at another workspace's project", func(t *testing.T) {
		wantExecError(t, db, insertPlan, idK1, idW2, idP1, nil, "x", ts, ts)
	})

	t.Run("status cannot point at another workspace's plan", func(t *testing.T) {
		wantExecError(t, db, insertStatus, idK1, idW2, idL1, ts)
	})

	t.Run("project_team cannot join across workspaces", func(t *testing.T) {
		mustExec(t, db, `INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Other', ?, ?)`, idK1, idW2, ts, ts)
		wantExecError(t, db, insertProjectTeam, idW1, idP1, idK1, ts, ts)
	})

	t.Run("task creator must belong to the task workspace", func(t *testing.T) {
		wantExecError(t, db, insertTask, idK1, idW2, idP1, idL1, idS1, nil, idM1, ts, ts)
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

func TestSchemaPlanTeamMustBelongToProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	wantExecError(t, db, insertPlan, idK1, idW1, idP1, idT1, "Backend plan", ts, ts)

	mustExec(t, db, insertProjectTeam, idW1, idP1, idT1, ts, ts)
	mustExec(t, db, insertPlan, idK1, idW1, idP1, idT1, "Backend plan", ts, ts)
}

func TestSchemaPlanWithoutTeamSkipsTeamConstraint(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	mustExec(t, db, insertPlan, idK1, idW1, idP1, nil, "Whole project", ts, ts)
}

func TestSchemaPlanNameUniquePerScope(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	mustExec(t, db, insertProjectTeam, idW1, idP1, idT1, ts, ts)

	wantExecError(t, db, insertPlan, idK1, idW1, idP1, nil, "Main", ts, ts)

	mustExec(t, db, insertPlan, idK1, idW1, idP1, idT1, "Main", ts, ts)
	wantExecError(t, db, insertPlan, idK2, idW1, idP1, idT1, "Main", ts, ts)

	mustExec(t, db, `UPDATE plans SET deleted_at = ? WHERE id = ?`, ts, idL1)
	mustExec(t, db, insertPlan, idK3, idW1, idP1, nil, "Main", ts, ts)
}

func TestSchemaSubtaskMustShareProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	mustExec(t, db, insertTask, idK1, idW1, idP1, idL1, idS1, nil, idM1, ts, ts)
	mustExec(t, db, insertTask, idK2, idW1, idP1, idL1, idS1, idK1, idM1, ts, ts)

	sp := seedSecondProject(t, db)
	wantExecError(t, db, insertTask, idK3, idW1, sp.projectID, sp.planID, sp.statusID, idK1, idM1, ts, ts)
}

func TestSchemaSubtaskMayLiveInAnotherPlan(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	seedSecondPlan(t, db)

	mustExec(t, db, insertTask, idK1, idW1, idP1, idL1, idS1, nil, idM1, ts, ts)
	mustExec(t, db, insertTask, idK2, idW1, idP1, idL2, idS2, idK1, idM1, ts, ts)
}

func TestSchemaTaskStatusMustShareProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	sp := seedSecondProject(t, db)

	wantExecError(t, db, insertTask, idK1, idW1, sp.projectID, sp.planID, idS1, nil, idM1, ts, ts)
}

func TestSchemaTaskStatusMustShareTaskPlan(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	seedSecondPlan(t, db)

	wantExecError(t, db, insertTask, idK1, idW1, idP1, idL1, idS2, nil, idM1, ts, ts)
	mustExec(t, db, insertTask, idK1, idW1, idP1, idL2, idS2, nil, idM1, ts, ts)
}

func TestSchemaTaskPlanMustShareTaskProject(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	sp := seedSecondProject(t, db)

	wantExecError(t, db, insertTask, idK1, idW1, sp.projectID, idL1, idS1, nil, idM1, ts, ts)
}

func TestSchemaTaskRequiresPlan(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)

	wantExecError(t, db,
		`INSERT INTO tasks (id, workspace_id, project_id, status_id, created_by, title, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 't', 'a0', ?, ?)`, idK1, idW1, idP1, idS1, idM1, ts, ts)
}

func TestSchemaForeignKeyRestrictsDelete(t *testing.T) {
	db := newSchemaDB(t)
	seedBase(t, db)
	wantExecError(t, db, `DELETE FROM projects WHERE id = ?`, idP1)
	wantExecError(t, db, `DELETE FROM plans WHERE id = ?`, idL1)
}

func TestSchemaIndexesExist(t *testing.T) {
	db := newSchemaDB(t)

	want := []string{
		"idx_memberships_user_id",
		"idx_teams_workspace_id_name",
		"idx_projects_workspace_id",
		"idx_plans_project_id_team_id",
		"idx_plans_project_id_team_id_name",
		"idx_statuses_plan_id_position",
		"idx_project_teams_team_id",
		"idx_team_members_team_id",
		"idx_tasks_parent_id",
		"idx_tasks_project_id",
		"idx_tasks_plan_id_status_id_position",
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