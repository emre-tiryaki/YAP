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
	db          *sql.DB
	session     model.Session
	projectRepo *repository.ProjectRepository
	projects    *ProjectService
	tasks       *TaskService
}

func newTaskEnv(t *testing.T) taskEnv {
	t.Helper()
	db, session := newTestDB(t)
	projectRepo := repository.NewProjectRepository()
	return taskEnv{
		db:          db,
		session:     session,
		projectRepo: projectRepo,
		projects:    NewProjectService(db, projectRepo),
		tasks:       NewTaskService(db, projectRepo, repository.NewTaskRepository()),
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

func (e taskEnv) planOf(t *testing.T, projectID string) string {
	t.Helper()
	var id string
	if err := e.db.QueryRow(`SELECT id FROM plans WHERE project_id = ? ORDER BY created_at, id LIMIT 1`, projectID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e taskEnv) plan(t *testing.T, projectID, name string, withTodo bool) model.Plan {
	t.Helper()
	ctx := context.Background()
	plan := model.Plan{ID: model.NewID(), WorkspaceID: e.session.WorkspaceID, ProjectID: projectID, Name: name}
	if err := e.projectRepo.InsertPlan(ctx, e.db, plan, nowUTC()); err != nil {
		t.Fatalf("InsertPlan() error = %v", err)
	}
	if withTodo {
		status := model.Status{
			ID: model.NewID(), WorkspaceID: e.session.WorkspaceID, PlanID: plan.ID,
			Title: "To Do", Color: "#000", Category: model.StatusToDo, Position: "a0",
		}
		if err := e.projectRepo.InsertStatus(ctx, e.db, status, nowUTC()); err != nil {
			t.Fatalf("InsertStatus() error = %v", err)
		}
	}
	return plan
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
	if task.ProjectName != "Backend" {
		t.Errorf("ProjectName = %q, want Backend", task.ProjectName)
	}
	if want := env.planOf(t, project.ID); task.PlanID != want {
		t.Errorf("PlanID = %q, want default plan %q", task.PlanID, want)
	}

	var statusCategory, title, dueDate, planID string
	err = env.db.QueryRowContext(ctx,
		`SELECT s.category, t.title, t.due_date, t.plan_id
		 FROM tasks t JOIN statuses s ON s.id = t.status_id WHERE t.id = ?`, task.ID,
	).Scan(&statusCategory, &title, &dueDate, &planID)
	if err != nil {
		t.Fatal(err)
	}
	if statusCategory != "todo" || title != "Write tests" || dueDate != "2026-12-01T10:00:00Z" || planID != task.PlanID {
		t.Errorf("stored = %q %q %q %q", statusCategory, title, dueDate, planID)
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

func TestTaskCreateInExplicitPlan(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")
	plan := env.plan(t, project.ID, "Second", true)

	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "x", PlanID: plan.ID})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.PlanID != plan.ID || task.Position != "a0" {
		t.Errorf("task = %+v, want plan %q and first position in that plan", task, plan.ID)
	}

	var taskPlan, statusPlan string
	err = env.db.QueryRowContext(ctx,
		`SELECT t.plan_id, s.plan_id FROM tasks t JOIN statuses s ON s.id = t.status_id WHERE t.id = ?`, task.ID,
	).Scan(&taskPlan, &statusPlan)
	if err != nil {
		t.Fatal(err)
	}
	if taskPlan != plan.ID || statusPlan != plan.ID {
		t.Errorf("task plan = %q, status plan = %q, want both %q", taskPlan, statusPlan, plan.ID)
	}
}

func TestTaskCreateDefaultsToWholeProjectPlan(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")
	defaultPlan := env.planOf(t, project.ID)

	teamID := model.NewID()
	now := nowUTC()
	if _, err := env.db.Exec(
		`INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Backend', ?, ?)`,
		teamID, env.session.WorkspaceID, now, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := env.db.Exec(
		`INSERT INTO project_teams (workspace_id, project_id, team_id, added_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		env.session.WorkspaceID, project.ID, teamID, now, now,
	); err != nil {
		t.Fatal(err)
	}
	teamPlan := model.Plan{
		ID: model.NewID(), WorkspaceID: env.session.WorkspaceID, ProjectID: project.ID,
		TeamID: teamID, Name: "Team plan",
	}
	if err := env.projectRepo.InsertPlan(ctx, env.db, teamPlan, "2020-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "x"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.PlanID != defaultPlan {
		t.Errorf("PlanID = %q, want whole-project plan %q", task.PlanID, defaultPlan)
	}
}

func TestTaskCreateNotFoundCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(t *testing.T, env taskEnv) (projectID, planID string)
	}{
		{"soft deleted project", func(t *testing.T, env taskEnv) (string, string) {
			p := env.project(t, "gone")
			if _, err := env.db.Exec(`UPDATE projects SET deleted_at = ? WHERE id = ?`, nowUTC(), p.ID); err != nil {
				t.Fatal(err)
			}
			return p.ID, ""
		}},
		{"project without todo status", func(t *testing.T, env taskEnv) (string, string) {
			p := env.project(t, "no todo")
			if _, err := env.db.Exec(
				`UPDATE statuses SET deleted_at = ?
				 WHERE category = 'todo' AND plan_id IN (SELECT id FROM plans WHERE project_id = ?)`,
				nowUTC(), p.ID,
			); err != nil {
				t.Fatal(err)
			}
			return p.ID, ""
		}},
		{"project without default plan", func(t *testing.T, env taskEnv) (string, string) {
			p := env.project(t, "no plan")
			if _, err := env.db.Exec(`UPDATE plans SET deleted_at = ? WHERE project_id = ?`, nowUTC(), p.ID); err != nil {
				t.Fatal(err)
			}
			return p.ID, ""
		}},
		{"project in another workspace", func(t *testing.T, env taskEnv) (string, string) {
			otherWorkspace := model.NewID()
			if err := repository.NewSessionRepository().InsertWorkspace(ctx, env.db, otherWorkspace, model.KindPersonal, "Other", nowUTC()); err != nil {
				t.Fatal(err)
			}
			p, err := env.projects.Create(ctx, otherWorkspace, CreateProjectInput{Name: "foreign"})
			if err != nil {
				t.Fatal(err)
			}
			return p.ID, ""
		}},
		{"plan of another project", func(t *testing.T, env taskEnv) (string, string) {
			a := env.project(t, "a")
			b := env.project(t, "b")
			return a.ID, env.planOf(t, b.ID)
		}},
		{"archived explicit plan", func(t *testing.T, env taskEnv) (string, string) {
			p := env.project(t, "archived")
			plan := env.plan(t, p.ID, "Old", true)
			if _, err := env.db.Exec(`UPDATE plans SET archived_at = ? WHERE id = ?`, nowUTC(), plan.ID); err != nil {
				t.Fatal(err)
			}
			return p.ID, plan.ID
		}},
		{"explicit plan without todo status", func(t *testing.T, env taskEnv) (string, string) {
			p := env.project(t, "empty plan")
			plan := env.plan(t, p.ID, "Empty", false)
			return p.ID, plan.ID
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTaskEnv(t)
			projectID, planID := tt.setup(t, env)

			_, err := env.tasks.Create(ctx, env.session, projectID, CreateTaskInput{Title: "x", PlanID: planID})
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

func TestTaskUpcomingSpansPlans(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	second := env.plan(t, project.ID, "Second", true)

	soon := time.Now().Add(time.Hour)
	later := time.Now().Add(2 * time.Hour)
	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "later", DueDate: &later}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "soon", DueDate: &soon, PlanID: second.ID}); err != nil {
		t.Fatal(err)
	}

	got, err := env.tasks.Upcoming(ctx, env.session)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Title != "soon" || got[0].PlanID != second.ID || got[1].Title != "later" {
		t.Errorf("Upcoming() = %+v, want [soon (second plan), later]", got)
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
	if _, err := env.tasks.SetDone(ctx, env.session, finished.ID, true); err != nil {
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