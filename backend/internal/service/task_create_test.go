package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

type taskEnv struct {
	db       *sql.DB
	session  model.Session
	projects *ProjectService
	tasks    *TaskService
}

func newTaskEnv(t *testing.T) taskEnv {
	t.Helper()
	db, session := newTestDB(t)
	projectRepo := repository.NewProjectRepository()
	return taskEnv{
		db:       db,
		session:  session,
		projects: NewProjectService(db, projectRepo),
		tasks:    NewTaskService(db, projectRepo, repository.NewTaskRepository()),
	}
}

func (e taskEnv) project(t *testing.T, name string) model.Project {
	t.Helper()
	p, err := e.projects.Create(context.Background(), e.session.WorkspaceID, CreateProjectInput{Name: name})
	if err != nil {
		t.Fatalf("project Create() error = %v", err)
	}
	return p
}

func TestTaskCreatePersistsAndAssignsCreator(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")
	due := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)

	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{
		Title: "  Write tests  ", Description: "  all of them  ", DueDate: &due,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.Title != "Write tests" || task.Description != "all of them" {
		t.Errorf("task = %+v, want trimmed title and description", task)
	}
	if task.StatusCategory != model.StatusToDo || task.CreatedBy != env.session.MembershipID {
		t.Errorf("task = %+v", task)
	}

	var statusCategory, title, dueDate string
	err = env.db.QueryRowContext(ctx,
		`SELECT s.category, t.title, t.due_date FROM tasks t JOIN statuses s ON s.id = t.status_id WHERE t.id = ?`, task.ID,
	).Scan(&statusCategory, &title, &dueDate)
	if err != nil {
		t.Fatal(err)
	}
	if statusCategory != "todo" || title != "Write tests" || dueDate != "2026-12-01T10:00:00Z" {
		t.Errorf("stored = %q %q %q", statusCategory, title, dueDate)
	}

	var assignees int
	err = env.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_assignees WHERE task_id = ? AND membership_id = ? AND removed_at IS NULL`,
		task.ID, env.session.MembershipID,
	).Scan(&assignees)
	if err != nil {
		t.Fatal(err)
	}
	if assignees != 1 {
		t.Errorf("assignees = %d, want 1", assignees)
	}
}

func TestTaskCreatePositionsIncrease(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")

	prev := ""
	for i := 0; i < 5; i++ {
		task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: fmt.Sprintf("t%d", i)})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if task.Position <= prev {
			t.Errorf("position %q does not sort after %q", task.Position, prev)
		}
		prev = task.Position
	}
}

func TestTaskCreateNotFoundCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(t *testing.T, env taskEnv) string
	}{
		{"soft deleted project", func(t *testing.T, env taskEnv) string {
			p := env.project(t, "gone")
			if _, err := env.db.Exec(`UPDATE projects SET deleted_at = ? WHERE id = ?`, nowUTC(), p.ID); err != nil {
				t.Fatal(err)
			}
			return p.ID
		}},
		{"project without todo status", func(t *testing.T, env taskEnv) string {
			p := env.project(t, "no todo")
			if _, err := env.db.Exec(`UPDATE statuses SET deleted_at = ? WHERE project_id = ? AND category = 'todo'`, nowUTC(), p.ID); err != nil {
				t.Fatal(err)
			}
			return p.ID
		}},
		{"project in another workspace", func(t *testing.T, env taskEnv) string {
			otherWorkspace := model.NewID()
			if err := repository.NewSessionRepository().InsertWorkspace(ctx, env.db, otherWorkspace, model.KindPersonal, "Other", nowUTC()); err != nil {
				t.Fatal(err)
			}
			p, err := env.projects.Create(ctx, otherWorkspace, CreateProjectInput{Name: "foreign"})
			if err != nil {
				t.Fatal(err)
			}
			return p.ID
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTaskEnv(t)
			projectID := tt.setup(t, env)

			_, err := env.tasks.Create(ctx, env.session, projectID, CreateTaskInput{Title: "x"})
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("Create() error = %v, want ErrNotFound", err)
			}

			var n int
			if err := env.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Errorf("tasks = %d, want 0", n)
			}
		})
	}
}

func TestTaskUpcomingLimitAndProjectName(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")

	base := time.Now().Add(24 * time.Hour)
	total := upcomingTaskLimit + 2
	for i := 0; i < total; i++ {
		due := base.Add(time.Duration(i) * time.Hour)
		if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{
			Title: fmt.Sprintf("task-%02d", i), DueDate: &due,
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := env.tasks.Upcoming(ctx, env.session)
	if err != nil {
		t.Fatalf("Upcoming() error = %v", err)
	}
	if len(got) != upcomingTaskLimit {
		t.Fatalf("Upcoming() returned %d tasks, want %d", len(got), upcomingTaskLimit)
	}
	if got[0].Title != "task-00" || got[0].ProjectName != "Alpha" {
		t.Errorf("got[0] = %+v", got[0])
	}
}

func TestTaskUpcomingExcludesDoneTasks(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	due := time.Now().Add(time.Hour)

	open, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "open", DueDate: &due})
	if err != nil {
		t.Fatal(err)
	}
	finished, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "finished", DueDate: &due})
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.db.Exec(
		`UPDATE tasks SET status_id = (SELECT id FROM statuses WHERE project_id = ? AND category = 'done') WHERE id = ?`,
		project.ID, finished.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := env.tasks.Upcoming(ctx, env.session)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != open.ID {
		t.Errorf("Upcoming() = %+v, want only the open task", got)
	}
}