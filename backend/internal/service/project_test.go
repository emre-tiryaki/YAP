package service

import (
	"context"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestProjectCreate(t *testing.T) {
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	tests := []struct{
		name string
		input CreateProjectInput
		wantErr error
	}{
		{"valid", CreateProjectInput{Name: "Backend"}, nil},
		{"blank name", CreateProjectInput{Name: ""}, ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), session.WorkspaceID, tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProjectCreateAddsDefaultStatuses(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	project, err := svc.Create(ctx, session.WorkspaceID, CreateProjectInput{Name: "Backend"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM statuses WHERE project_id = ?", project.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != len(defaultStatuses) {
		t.Errorf("statuses = %d, want %d", count, len(defaultStatuses))
	}

	projects, err := svc.List(ctx, session.WorkspaceID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("List() = %+v, want onşy %s", projects, project.ID)
	}
}