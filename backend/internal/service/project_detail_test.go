package service

import (
	"context"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestProjectGet(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project, err := env.projects.Create(ctx, env.session.WorkspaceID, CreateProjectInput{Name: "Alpha", Description: "text"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "one"}); err != nil {
		t.Fatal(err)
	}

	deleted := env.project(t, "gone")
	if _, err := env.db.Exec(`UPDATE projects SET deleted_at = ? WHERE id = ?`, nowUTC(), deleted.ID); err != nil {
		t.Fatal(err)
	}

	otherWorkspace := model.NewID()
	if err := repository.NewSessionRepository().InsertWorkspace(ctx, env.db, otherWorkspace, model.KindPersonal, "Other", nowUTC()); err != nil {
		t.Fatal(err)
	}
	foreign, err := env.projects.Create(ctx, otherWorkspace, CreateProjectInput{Name: "foreign"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		projectID string
		wantErr   error
	}{
		{"existing", project.ID, nil},
		{"unknown", "missing", ErrNotFound},
		{"soft deleted", deleted.ID, ErrNotFound},
		{"other workspace", foreign.ID, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.projects.Get(ctx, env.session.WorkspaceID, tt.projectID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.ID != project.ID || got.Name != "Alpha" || got.Description != "text" || got.TaskCount != 1 {
				t.Errorf("Get() = %+v", got)
			}
		})
	}
}

func TestProjectPlans(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	second := env.plan(t, project.ID, "Second", true)

	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "two", PlanID: second.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "three", PlanID: second.ID}); err != nil {
		t.Fatal(err)
	}

	got, err := env.projects.Plans(ctx, env.session.WorkspaceID, project.ID)
	if err != nil {
		t.Fatalf("Plans() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Plans() returned %d plans, want 2", len(got))
	}
	if got[0].Name != defaultPlanName || got[0].TeamID != "" || got[0].TaskCount != 1 {
		t.Errorf("got[0] = %+v, want default plan with 1 task", got[0])
	}
	if got[1].ID != second.ID || got[1].TaskCount != 2 {
		t.Errorf("got[1] = %+v, want second plan with 2 tasks", got[1])
	}
}

func TestProjectPlansNotFound(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	deleted := env.project(t, "gone")
	if _, err := env.db.Exec(`UPDATE projects SET deleted_at = ? WHERE id = ?`, nowUTC(), deleted.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		projectID string
	}{
		{"unknown", "missing"},
		{"soft deleted", deleted.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.projects.Plans(ctx, env.session.WorkspaceID, tt.projectID)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("Plans() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestProjectPlansSkipsArchivedAndDeleted(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	archived := env.plan(t, project.ID, "Archived", true)
	removed := env.plan(t, project.ID, "Removed", true)
	if _, err := env.db.Exec(`UPDATE plans SET archived_at = ? WHERE id = ?`, nowUTC(), archived.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.db.Exec(`UPDATE plans SET deleted_at = ? WHERE id = ?`, nowUTC(), removed.ID); err != nil {
		t.Fatal(err)
	}

	got, err := env.projects.Plans(ctx, env.session.WorkspaceID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != defaultPlanName {
		t.Errorf("Plans() = %+v, want only the default plan", got)
	}
}