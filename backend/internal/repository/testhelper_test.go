package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const baseNow = "2026-01-01T00:00:00Z"

type workspaceSeed struct {
	UserID       string
	WorkspaceID  string
	MembershipID string
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.New(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("database.New() error = %v", err)
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

func seedWorkspace(t *testing.T, db *sql.DB, name string) workspaceSeed {
	t.Helper()
	ctx := context.Background()
	repo := NewSessionRepository()

	user := model.User{ID: model.NewID(), DisplayName: name}
	if err := repo.InserUser(ctx, db, user, baseNow); err != nil {
		t.Fatal(err)
	}
	return seedWorkspaceFor(t, db, user.ID, name)
}

func seedWorkspaceFor(t *testing.T, db *sql.DB, userID, name string) workspaceSeed {
	t.Helper()
	ctx := context.Background()
	repo := NewSessionRepository()

	workspaceID := model.NewID()
	if err := repo.InsertWorkspace(ctx, db, workspaceID, model.KindPersonal, name, baseNow); err != nil {
		t.Fatal(err)
	}
	membershipID := model.NewID()
	if err := repo.InsertMembership(ctx, db, membershipID, workspaceID, userID, model.RoleOwner, baseNow); err != nil {
		t.Fatal(err)
	}
	return workspaceSeed{UserID: userID, WorkspaceID: workspaceID, MembershipID: membershipID}
}

func addMember(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	ctx := context.Background()
	repo := NewSessionRepository()

	user := model.User{ID: model.NewID(), DisplayName: "member"}
	if err := repo.InserUser(ctx, db, user, baseNow); err != nil {
		t.Fatal(err)
	}
	membershipID := model.NewID()
	if err := repo.InsertMembership(ctx, db, membershipID, workspaceID, user.ID, "member", baseNow); err != nil {
		t.Fatal(err)
	}
	return membershipID
}

func seedProject(t *testing.T, db *sql.DB, workspaceID, name, now string) model.Project {
	t.Helper()
	p := model.Project{ID: model.NewID(), WorkspaceID: workspaceID, Name: name}
	if err := NewProjectRepository().Insert(context.Background(), db, p, now); err != nil {
		t.Fatal(err)
	}
	return p
}

func seedPlan(t *testing.T, db *sql.DB, workspaceID, projectID, name string) model.Plan {
	t.Helper()
	p := model.Plan{ID: model.NewID(), WorkspaceID: workspaceID, ProjectID: projectID, Name: name}
	if err := NewProjectRepository().InsertPlan(context.Background(), db, p, baseNow); err != nil {
		t.Fatal(err)
	}
	return p
}

func seedTeam(t *testing.T, db *sql.DB, workspaceID, name string) string {
	t.Helper()
	id := model.NewID()
	mustExec(t, db,
		`INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, workspaceID, name, baseNow, baseNow)
	return id
}

func seedProjectTeam(t *testing.T, db *sql.DB, workspaceID, projectID, teamID string) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO project_teams (workspace_id, project_id, team_id, added_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		workspaceID, projectID, teamID, baseNow, baseNow)
}

func seedStatuses(t *testing.T, db *sql.DB, workspaceID, planID string) map[model.StatusCategory]string {
	t.Helper()
	repo := NewProjectRepository()
	ids := map[model.StatusCategory]string{}
	defs := []struct {
		category model.StatusCategory
		position string
	}{
		{model.StatusToDo, "a1"},
		{model.StatusInProgress, "a2"},
		{model.StatusDone, "a3"},
	}
	for _, d := range defs {
		s := model.Status{
			ID:          model.NewID(),
			WorkspaceID: workspaceID,
			PlanID:      planID,
			Title:       string(d.category),
			Color:       "#000000",
			Category:    d.category,
			Position:    d.position,
		}
		if err := repo.InsertStatus(context.Background(), db, s, baseNow); err != nil {
			t.Fatal(err)
		}
		ids[d.category] = s.ID
	}
	return ids
}

func seedTask(t *testing.T, db *sql.DB, s workspaceSeed, plan model.Plan, statusID, title, pos string, due *time.Time) model.Task {
	t.Helper()
	task := model.Task{
		ID:          model.NewID(),
		WorkspaceID: s.WorkspaceID,
		ProjectID:   plan.ProjectID,
		PlanID:      plan.ID,
		StatusID:    statusID,
		CreatedBy:   s.MembershipID,
		Title:       title,
		Position:    pos,
		DueDate:     due,
	}
	if err := NewTaskRepository().Insert(context.Background(), db, task, baseNow); err != nil {
		t.Fatal(err)
	}
	return task
}

func assign(t *testing.T, db *sql.DB, workspaceID, taskID, membershipID string) {
	t.Helper()
	if err := NewTaskRepository().InsertAssignee(context.Background(), db, workspaceID, taskID, membershipID, baseNow); err != nil {
		t.Fatal(err)
	}
}