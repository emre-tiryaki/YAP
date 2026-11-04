package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestTaskCreateErrors(t *testing.T) {
	db, session := newTestDB(t)
	projectRepo := repository.NewProjectRepository()
	projects := NewProjectService(db, projectRepo)
	tasks := NewTaskService(db, projectRepo, repository.NewTaskRepository())

	project, err := projects.Create(context.Background(), session.WorkspaceID, CreateProjectInput{Name: "Backend"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name      string
		projectID string
		title     string
		wantErr   error
	}{
		{"valid", project.ID, "Write tests", nil},
		{"valid", project.ID, " ", ErrInvalidInput},
		{"unknown project", "missing", "Write tests", ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tasks.Create(context.Background(), session, tt.projectID, CreateTaskInput{Title: tt.title})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskUpcomingOrdersByDueDateAndSkipsUndated(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)
	projectRepo := repository.NewProjectRepository()
	projects := NewProjectService(db, projectRepo)
	tasks := NewTaskService(db, projectRepo, repository.NewTaskRepository())

	project, err := projects.Create(ctx, session.WorkspaceID, CreateProjectInput{Name: "Backend"})
	if err != nil {
		t.Fatalf("Creat() error = %v", err)
	}

	soon := time.Now().Add(24 * time.Hour)
	later := time.Now().Add(49 * time.Hour)
	inputs := []CreateTaskInput{
		{Title: "later", DueDate: &later},
		{Title: "undated"},
		{Title: "soon", DueDate: &soon},
	}
	for _, in := range inputs {
		if _, err := tasks.Create(ctx, session, project.ID, in); err != nil {
			t.Fatalf("Create(%q) error = %v", in.Title, err)
		}
	}

	got, err := tasks.Upcoming(ctx, session)
	if err != nil {
		t.Fatalf("Upcoming() error = %v", err)
	}

	want := []string{"soon", "later"}
	if len(got) != len(want) {
		t.Fatalf("Upcoming() returned %d tasks, want %d", len(got), len(want))
	}
	for i, title := range want {
		if got[i].Title != title {
			t.Errorf("Upcoming()[%d] = %q, want %q", i, got[i].Title, title)
		}
	}
}
