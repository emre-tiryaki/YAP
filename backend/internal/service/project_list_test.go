package service

import (
	"context"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestProjectListEmpty(t *testing.T) {
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	got, err := svc.List(context.Background(), session.WorkspaceID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("List() = %#v, want empty non-nil slice", got)
	}
}

func TestProjectCreateTrimsAndPersistsFields(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	tests := []struct {
		name     string
		input    CreateProjectInput
		wantName string
		wantDesc string
	}{
		{"trimmed", CreateProjectInput{Name: "  Alpha  ", Description: "  text  "}, "Alpha", "text"},
		{"no description", CreateProjectInput{Name: "Beta"}, "Beta", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := svc.Create(ctx, session.WorkspaceID, tt.input)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if p.Name != tt.wantName || p.Description != tt.wantDesc || p.WorkspaceID != session.WorkspaceID {
				t.Errorf("Create() = %+v", p)
			}
			if len(p.ID) != 36 {
				t.Errorf("ID = %q, want a 36 character id", p.ID)
			}
		})
	}
}

func TestProjectCreateWhitespaceNameIsInvalid(t *testing.T) {
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	_, err := svc.Create(context.Background(), session.WorkspaceID, CreateProjectInput{Name: "   "})
	if err == nil {
		t.Fatal("Create() expected an error for a whitespace-only name")
	}
}

func TestProjectCreateDefaultStatusOrderAndCategories(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	p, err := svc.Create(ctx, session.WorkspaceID, CreateProjectInput{Name: "Board"})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.QueryContext(ctx,
		`SELECT title, category, position FROM statuses WHERE project_id = ? ORDER BY position`, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var gotTitles []string
	var gotCategories []model.StatusCategory
	prev := ""
	for rows.Next() {
		var title, category, pos string
		if err := rows.Scan(&title, &category, &pos); err != nil {
			t.Fatal(err)
		}
		if pos <= prev {
			t.Errorf("position %q does not sort after %q", pos, prev)
		}
		prev = pos
		gotTitles = append(gotTitles, title)
		gotCategories = append(gotCategories, model.StatusCategory(category))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(gotTitles) != len(defaultStatuses) {
		t.Fatalf("statuses = %v, want %d", gotTitles, len(defaultStatuses))
	}
	for i, d := range defaultStatuses {
		if gotTitles[i] != d.title || gotCategories[i] != d.category {
			t.Errorf("status[%d] = %q/%q, want %q/%q", i, gotTitles[i], gotCategories[i], d.title, d.category)
		}
	}
}

func TestProjectCreateRollsBackOnFailure(t *testing.T) {
	ctx := context.Background()
	db, _ := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	_, err := svc.Create(ctx, model.NewID(), CreateProjectInput{Name: "orphan"})
	if err == nil {
		t.Fatal("Create() expected an error for an unknown workspace")
	}

	var projects, statuses int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&projects); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM statuses").Scan(&statuses); err != nil {
		t.Fatal(err)
	}
	if projects != 0 || statuses != 0 {
		t.Errorf("projects = %d, statuses = %d, want 0 after rollback", projects, statuses)
	}
}

func TestProjectListIsWorkspaceScopedAndNewestFirst(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)
	svc := NewProjectService(db, repository.NewProjectRepository())

	otherWorkspace := model.NewID()
	if err := repository.NewSessionRepository().InsertWorkspace(ctx, db, otherWorkspace, model.KindPersonal, "Other", nowUTC()); err != nil {
		t.Fatal(err)
	}

	older, err := svc.Create(ctx, session.WorkspaceID, CreateProjectInput{Name: "older"})
	if err != nil {
		t.Fatal(err)
	}
	newer, err := svc.Create(ctx, session.WorkspaceID, CreateProjectInput{Name: "newer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, otherWorkspace, CreateProjectInput{Name: "foreign"}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.List(ctx, session.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != newer.ID || got[1].ID != older.ID {
		t.Errorf("List() = %+v, want [newer, older]", got)
	}
}