package repository

import (
	"context"
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
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)

	seedTask(t, db, s, p.ID, statuses[model.StatusToDo], "one", "a0", nil)
	seedTask(t, db, s, p.ID, statuses[model.StatusToDo], "two", "a1", nil)
	gone := seedTask(t, db, s, p.ID, statuses[model.StatusToDo], "gone", "a2", nil)
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
		t.Errorf("TaskCount = %d, want 2", counts[p.ID])
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

func TestProjectInsertStatus(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewProjectRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)

	valid := model.Status{
		ID: model.NewID(), WorkspaceID: s.WorkspaceID, ProjectID: p.ID,
		Title: "To Do", Color: "#fff", Category: model.StatusToDo, Position: "a0",
	}
	if err := repo.InsertStatus(ctx, db, valid, baseNow); err != nil {
		t.Fatalf("InsertStatus() error = %v", err)
	}

	var title, category, position string
	err := db.QueryRowContext(ctx, `SELECT title, category, position FROM statuses WHERE id = ?`, valid.ID).
		Scan(&title, &category, &position)
	if err != nil {
		t.Fatal(err)
	}
	if title != "To Do" || category != "todo" || position != "a0" {
		t.Errorf("stored status = %q %q %q", title, category, position)
	}

	tests := []struct {
		name   string
		mutate func(s *model.Status)
	}{
		{"duplicate id", func(s *model.Status) { s.ID = valid.ID }},
		{"invalid category", func(s *model.Status) { s.ID = model.NewID(); s.Category = "blocked" }},
		{"unknown project", func(s *model.Status) { s.ID = model.NewID(); s.ProjectID = model.NewID() }},
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

func TestProjectExists(t *testing.T) {
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
		want        bool
	}{
		{"existing", a.WorkspaceID, live.ID, true},
		{"soft deleted", a.WorkspaceID, deleted.ID, false},
		{"other workspace", b.WorkspaceID, live.ID, false},
		{"unknown id", a.WorkspaceID, "missing", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Exists(ctx, db, tt.workspaceID, tt.projectID)
			if err != nil {
				t.Fatalf("Exists() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}