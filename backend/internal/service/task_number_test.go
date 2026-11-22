package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestProjectGetsUniqueIdentifiers(t *testing.T) {
	env := newTaskEnv(t)
	a := env.project(t, "Kitap Kulübü")
	b := env.project(t, "Kitchen")
	c := env.project(t, "x")
	if a.Identifier != "KIT" || b.Identifier != "KIT2" || c.Identifier != "PRJ" {
		t.Errorf("identifiers = %q, %q, %q; want KIT, KIT2, PRJ", a.Identifier, b.Identifier, c.Identifier)
	}

	// A deleted project frees its identifier.
	if err := env.projects.Delete(context.Background(), env.session.WorkspaceID, a.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if d := env.project(t, "Kitap Evi"); d.Identifier != "KIT" {
		t.Errorf("identifier after delete = %q, want KIT", d.Identifier)
	}
}

func TestTaskNumbersCountPerProject(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	first := env.project(t, "First")
	second := env.project(t, "Second")

	var numbers []int
	for _, projectID := range []string{first.ID, first.ID, second.ID, first.ID} {
		task, err := env.tasks.Create(ctx, env.session, projectID, CreateTaskInput{Title: "T"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		numbers = append(numbers, task.Number)
	}
	if want := []int{1, 2, 1, 3}; !slices.Equal(numbers, want) {
		t.Errorf("numbers = %v, want %v", numbers, want)
	}

	// Subtasks count too, and a deleted task's number is not reused.
	parent, err := env.tasks.Create(ctx, env.session, first.ID, CreateTaskInput{Title: "Parent"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := env.tasks.CreateSubtask(ctx, env.session, parent.ID, "Sub")
	if err != nil {
		t.Fatal(err)
	}
	if parent.Number != 4 || sub.Number != 5 {
		t.Errorf("parent and subtask numbers = %d, %d; want 4, 5", parent.Number, sub.Number)
	}
	if err := env.tasks.Delete(ctx, env.session, sub.ID); err != nil {
		t.Fatal(err)
	}
	next, err := env.tasks.Create(ctx, env.session, first.ID, CreateTaskInput{Title: "Next"})
	if err != nil || next.Number != 6 {
		t.Errorf("number after delete = %d (err %v), want 6", next.Number, err)
	}

	got, err := env.tasks.ListByPlan(ctx, env.session, first.ID, next.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range got {
		if task.Number == 0 {
			t.Errorf("task %q listed without a number", task.Title)
		}
	}
}

func TestUpdateProjectIdentifier(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	first := env.project(t, "Kitap")
	second := env.project(t, "Web")

	str := func(s string) *string { return &s }
	tests := []struct {
		name    string
		project string
		input   *string
		want    string
		wantErr error
	}{
		{"changes", first.ID, str("book"), "BOOK", nil},
		{"same value again", first.ID, str("BOOK"), "BOOK", nil},
		{"taken by another project", second.ID, str("BOOK"), "", ErrIdentifierTaken},
		{"too short", second.ID, str("W"), "", ErrInvalidInput},
		{"starts with a digit", second.ID, str("2WEB"), "", ErrInvalidInput},
		{"too long", second.ID, str("WEBSITE"), "", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.projects.Update(ctx, env.session.WorkspaceID, tt.project, UpdateProjectInput{Identifier: tt.input})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got.Identifier != tt.want {
				t.Errorf("Identifier = %q, want %q", got.Identifier, tt.want)
			}
		})
	}
	if !errors.Is(ErrIdentifierTaken, ErrConflict) {
		t.Error("ErrIdentifierTaken must be a conflict (HTTP 409)")
	}
	if got, _ := env.projects.Get(ctx, env.session.WorkspaceID, second.ID); got.Identifier != "WEB" {
		t.Errorf("rejected updates changed the identifier to %q", got.Identifier)
	}
}

func TestSearchFindsTaskByKey(t *testing.T) {
	ctx := context.Background()
	env := newResourceEnv(t)
	projectRepo := repository.NewProjectRepository()
	projects := NewProjectService(env.db, projectRepo, repository.NewSettingsRepository())
	tasks := NewTaskService(env.db, projectRepo, repository.NewTaskRepository(), repository.NewTaskActivityRepository())
	search := NewSearchService(env.db, repository.NewSearchRepository())

	project, err := projects.Create(ctx, env.session.WorkspaceID, CreateProjectInput{Name: "Website"})
	if err != nil {
		t.Fatal(err)
	}
	var second string
	for _, title := range []string{"Alpha", "Beta", "Gamma"} {
		task, err := tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: title})
		if err != nil {
			t.Fatal(err)
		}
		if task.Number == 2 {
			second = task.ID
		}
	}

	for _, query := range []string{"WEB-2", "web-2", " Web-2 "} {
		res, err := search.Search(ctx, env.session, query)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", query, err)
		}
		if len(res.Tasks) != 1 || res.Tasks[0].ID != second {
			t.Errorf("Search(%q) tasks = %+v, want only task Beta", query, res.Tasks)
		}
	}
	res, err := search.Search(ctx, env.session, "WEB-9")
	if err != nil || len(res.Tasks) != 0 {
		t.Errorf("Search(WEB-9) = %+v (err %v), want no tasks", res.Tasks, err)
	}
}
