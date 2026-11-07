package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func TestProjectInsertAndList(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")

	withDesc := model.Project{ID: model.NewID(), WorkspaceID: s.WorkspaceID, Name: "With", Description: "desc"}
	noDesc := model.Project{ID: model.NewID(), WorkspaceID: s.WorkspaceID, Name: "Without"}
	if err := repo.Insert(ctx, db, withDesc, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Insert(ctx, db, noDesc, "2026-01-02T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.List(ctx, db, s.WorkspaceID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() returned %d projects, want 2", len(got))
	}
	if got[0].ID != noDesc.ID || got[0].Description != "" || got[0].Name != "Without" {
		t.Errorf("got[0] = %+v, want newest project with empty description", got[0])
	}
	if got[1].ID != withDesc.ID || got[1].Description != "desc" {
		t.Errorf("got[1] = %+v", got[1])
	}
	for _, p := range got {
		if p.WorkspaceID != s.WorkspaceID {
			t.Errorf("WorkspaceID = %q, want %q", p.WorkspaceID, s.WorkspaceID)
		}
	}
}

func TestProjectListEmptyIsNonNil(t *testing.T) {
	db := newTestDB(t)
	s := seedWorkspace(t, db, "owner")

	got, err := NewProjectRepository().List(context.Background(), db, s.WorkspaceID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("List() = %#v, want empty non-nil slice", got)
	}
}

func TestProjectListFilters(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	a := seedWorkspace(t, db, "a")
	b := seedWorkspace(t, db, "b")

	active := seedProject(t, db, a.WorkspaceID, "active", baseNow)
	deleted := seedProject(t, db, a.WorkspaceID, "deleted", baseNow)
	archived := seedProject(t, db, a.WorkspaceID, "archived", baseNow)
	seedProject(t, db, b.WorkspaceID, "foreign", baseNow)

	mustExec(t, db, `UPDATE projects SET deleted_at = ? WHERE id = ?`, baseNow, deleted.ID)
	mustExec(t, db, `UPDATE projects SET archived_at = ? WHERE id = ?`, baseNow, archived.ID)

	got, err := repo.List(ctx, db, a.WorkspaceID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != active.ID {
		t.Errorf("List() = %+v, want only %s", got, active.ID)
	}
}

func TestProjectListTaskCount(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")

	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	empty := seedProject(t, db, s.WorkspaceID, "empty", "2025-12-31T00:00:00Z")
	main := seedPlan(t, db, s.WorkspaceID, p.ID, "Main")
	second := seedPlan(t, db, s.WorkspaceID, p.ID, "Second")
	mainStatuses := seedStatuses(t, db, s.WorkspaceID, main.ID)
	secondStatuses := seedStatuses(t, db, s.WorkspaceID, second.ID)

	seedTask(t, db, s, main, mainStatuses[model.StatusToDo], "one", "a0", nil)
	seedTask(t, db, s, second, secondStatuses[model.StatusToDo], "two", "a1", nil)
	gone := seedTask(t, db, s, main, mainStatuses[model.StatusToDo], "gone", "a2", nil)
	mustExec(t, db, `UPDATE tasks SET deleted_at = ? WHERE id = ?`, baseNow, gone.ID)

	got, err := repo.List(ctx, db, s.WorkspaceID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	counts := map[string]int{}
	for _, pr := range got {
		counts[pr.ID] = pr.TaskCount
	}
	if counts[p.ID] != 2 {
		t.Errorf("TaskCount = %d, want 2 across plans", counts[p.ID])
	}
	if counts[empty.ID] != 0 {
		t.Errorf("empty TaskCount = %d, want 0", counts[empty.ID])
	}
}

func TestProjectInsertErrors(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")

	p := model.Project{ID: model.NewID(), WorkspaceID: s.WorkspaceID, Name: "p"}
	if err := repo.Insert(ctx, db, p, baseNow); err != nil {
		t.Fatal(err)
	}
	if err := repo.Insert(ctx, db, p, baseNow); err == nil {
		t.Error("Insert() expected an error for a duplicate id")
	}
	orphan := model.Project{ID: model.NewID(), WorkspaceID: model.NewID(), Name: "orphan"}
	if err := repo.Insert(ctx, db, orphan, baseNow); err == nil {
		t.Error("Insert() expected an error for an unknown workspace")
	}
}

func TestProjectInsertPlan(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	teamID := seedTeam(t, db, s.WorkspaceID, "Backend")
	seedProjectTeam(t, db, s.WorkspaceID, p.ID, teamID)

	tests := []struct {
		name     string
		plan     model.Plan
		wantTeam sql.NullString
		wantDesc sql.NullString
	}{
		{"whole project", model.Plan{Name: "Main"}, sql.NullString{}, sql.NullString{}},
		{
			"team plan",
			model.Plan{Name: "API", TeamID: teamID, Description: "endpoints"},
			sql.NullString{String: teamID, Valid: true},
			sql.NullString{String: "endpoints", Valid: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := tt.plan
			plan.ID = model.NewID()
			plan.WorkspaceID = s.WorkspaceID
			plan.ProjectID = p.ID
			if err := repo.InsertPlan(ctx, db, plan, baseNow); err != nil {
				t.Fatalf("InsertPlan() error = %v", err)
			}

			var name, createdAt, updatedAt string
			var team, desc sql.NullString
			err := db.QueryRowContext(ctx,
				`SELECT name, team_id, description, created_at, updated_at FROM plans WHERE id = ?`, plan.ID,
			).Scan(&name, &team, &desc, &createdAt, &updatedAt)
			if err != nil {
				t.Fatal(err)
			}
			if name != tt.plan.Name || team != tt.wantTeam || desc != tt.wantDesc {
				t.Errorf("stored = %q %v %v", name, team, desc)
			}
			if createdAt != baseNow || updatedAt != baseNow {
				t.Errorf("timestamps = %q %q, want %q", createdAt, updatedAt, baseNow)
			}
		})
	}
}

func TestProjectInsertPlanErrors(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	unlinkedTeam := seedTeam(t, db, s.WorkspaceID, "Unlinked")

	valid := model.Plan{ID: model.NewID(), WorkspaceID: s.WorkspaceID, ProjectID: p.ID, Name: "Main"}
	if err := repo.InsertPlan(ctx, db, valid, baseNow); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(p *model.Plan)
	}{
		{"duplicate id", func(p *model.Plan) { p.ID = valid.ID }},
		{"duplicate name in scope", func(p *model.Plan) { p.ID = model.NewID() }},
		{"unknown project", func(p *model.Plan) { p.ID = model.NewID(); p.ProjectID = model.NewID() }},
		{"unknown workspace", func(p *model.Plan) { p.ID = model.NewID(); p.WorkspaceID = model.NewID() }},
		{"team not linked to project", func(p *model.Plan) {
			p.ID = model.NewID()
			p.Name = "Other"
			p.TeamID = unlinkedTeam
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad := valid
			tt.mutate(&bad)
			if err := repo.InsertPlan(ctx, db, bad, baseNow); err == nil {
				t.Error("InsertPlan() expected an error")
			}
		})
	}
}

func TestProjectInsertStatus(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	plan := seedPlan(t, db, s.WorkspaceID, p.ID, "Main")

	valid := model.Status{
		ID: model.NewID(), WorkspaceID: s.WorkspaceID, PlanID: plan.ID,
		Title: "To Do", Color: "#fff", Category: model.StatusToDo, Position: "a0",
	}
	if err := repo.InsertStatus(ctx, db, valid, baseNow); err != nil {
		t.Fatalf("InsertStatus() error = %v", err)
	}

	var title, category, position, planID string
	err := db.QueryRowContext(ctx, `SELECT title, category, position, plan_id FROM statuses WHERE id = ?`, valid.ID).
		Scan(&title, &category, &position, &planID)
	if err != nil {
		t.Fatal(err)
	}
	if title != "To Do" || category != "todo" || position != "a0" || planID != plan.ID {
		t.Errorf("stored status = %q %q %q %q", title, category, position, planID)
	}

	tests := []struct {
		name   string
		mutate func(s *model.Status)
	}{
		{"duplicate id", func(s *model.Status) { s.ID = valid.ID }},
		{"invalid category", func(s *model.Status) { s.ID = model.NewID(); s.Category = "blocked" }},
		{"unknown plan", func(s *model.Status) { s.ID = model.NewID(); s.PlanID = model.NewID() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad := valid
			tt.mutate(&bad)
			if err := repo.InsertStatus(ctx, db, bad, baseNow); err == nil {
				t.Error("InsertStatus() expected an error")
			}
		})
	}
}

func TestProjectName(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	a := seedWorkspace(t, db, "a")
	b := seedWorkspace(t, db, "b")

	live := seedProject(t, db, a.WorkspaceID, "live", baseNow)
	deleted := seedProject(t, db, a.WorkspaceID, "deleted", baseNow)
	mustExec(t, db, `UPDATE projects SET deleted_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), deleted.ID)

	tests := []struct {
		name        string
		workspaceID string
		projectID   string
		want        string
		wantNoRows  bool
	}{
		{"existing", a.WorkspaceID, live.ID, "live", false},
		{"soft deleted", a.WorkspaceID, deleted.ID, "", true},
		{"other workspace", b.WorkspaceID, live.ID, "", true},
		{"unknown id", a.WorkspaceID, "missing", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Name(ctx, db, tt.workspaceID, tt.projectID)
			if tt.wantNoRows {
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("Name() error = %v, want sql.ErrNoRows", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Name() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectDefaultPlanID(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	a := seedWorkspace(t, db, "a")
	b := seedWorkspace(t, db, "b")
	p := seedProject(t, db, a.WorkspaceID, "p", baseNow)

	teamID := seedTeam(t, db, a.WorkspaceID, "Backend")
	seedProjectTeam(t, db, a.WorkspaceID, p.ID, teamID)
	teamPlan := model.Plan{ID: model.NewID(), WorkspaceID: a.WorkspaceID, ProjectID: p.ID, TeamID: teamID, Name: "Team"}
	if err := repo.InsertPlan(ctx, db, teamPlan, "2025-12-31T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.DefaultPlanID(ctx, db, a.WorkspaceID, p.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("DefaultPlanID() with only a team plan error = %v, want sql.ErrNoRows", err)
	}

	earliest := model.Plan{ID: model.NewID(), WorkspaceID: a.WorkspaceID, ProjectID: p.ID, Name: "Earliest"}
	later := model.Plan{ID: model.NewID(), WorkspaceID: a.WorkspaceID, ProjectID: p.ID, Name: "Later"}
	if err := repo.InsertPlan(ctx, db, later, "2026-01-02T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertPlan(ctx, db, earliest, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.DefaultPlanID(ctx, db, a.WorkspaceID, p.ID)
	if err != nil {
		t.Fatalf("DefaultPlanID() error = %v", err)
	}
	if got != earliest.ID {
		t.Errorf("DefaultPlanID() = %q, want earliest whole-project plan %q", got, earliest.ID)
	}

	mustExec(t, db, `UPDATE plans SET archived_at = ? WHERE id = ?`, baseNow, earliest.ID)
	got, err = repo.DefaultPlanID(ctx, db, a.WorkspaceID, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != later.ID {
		t.Errorf("DefaultPlanID() = %q, want archived plan to be skipped", got)
	}

	mustExec(t, db, `UPDATE plans SET deleted_at = ? WHERE id = ?`, baseNow, later.ID)
	if _, err := repo.DefaultPlanID(ctx, db, a.WorkspaceID, p.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("DefaultPlanID() after deleting all error = %v, want sql.ErrNoRows", err)
	}

	if _, err := repo.DefaultPlanID(ctx, db, b.WorkspaceID, p.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("DefaultPlanID() other workspace error = %v, want sql.ErrNoRows", err)
	}
}

func TestProjectPlanID(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	a := seedWorkspace(t, db, "a")
	b := seedWorkspace(t, db, "b")

	p := seedProject(t, db, a.WorkspaceID, "p", baseNow)
	other := seedProject(t, db, a.WorkspaceID, "other", baseNow)
	live := seedPlan(t, db, a.WorkspaceID, p.ID, "live")
	deleted := seedPlan(t, db, a.WorkspaceID, p.ID, "deleted")
	archived := seedPlan(t, db, a.WorkspaceID, p.ID, "archived")
	mustExec(t, db, `UPDATE plans SET deleted_at = ? WHERE id = ?`, baseNow, deleted.ID)
	mustExec(t, db, `UPDATE plans SET archived_at = ? WHERE id = ?`, baseNow, archived.ID)

	tests := []struct {
		name        string
		workspaceID string
		projectID   string
		planID      string
		wantNoRows  bool
	}{
		{"existing", a.WorkspaceID, p.ID, live.ID, false},
		{"soft deleted", a.WorkspaceID, p.ID, deleted.ID, true},
		{"archived", a.WorkspaceID, p.ID, archived.ID, true},
		{"other project", a.WorkspaceID, other.ID, live.ID, true},
		{"other workspace", b.WorkspaceID, p.ID, live.ID, true},
		{"unknown id", a.WorkspaceID, p.ID, "missing", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.PlanID(ctx, db, tt.workspaceID, tt.projectID, tt.planID)
			if tt.wantNoRows {
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("PlanID() error = %v, want sql.ErrNoRows", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("PlanID() error = %v", err)
			}
			if got != tt.planID {
				t.Errorf("PlanID() = %q, want %q", got, tt.planID)
			}
		})
	}
}