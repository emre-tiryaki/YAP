package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func strPtr(s string) *string { return &s }

// nullableColumn reads a single nullable TEXT column, such as deleted_at.
func (e taskEnv) nullableColumn(t *testing.T, query string, args ...any) *string {
	t.Helper()
	var v *string
	if err := e.db.QueryRow(query, args...).Scan(&v); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return v
}

func (e taskEnv) createPlan(t *testing.T, projectID, name string) model.Plan {
	t.Helper()
	plan, err := e.projects.CreatePlan(context.Background(), e.session.WorkspaceID, projectID, CreatePlanInput{Name: name})
	if err != nil {
		t.Fatalf("CreatePlan(%q) error = %v", name, err)
	}
	return plan
}

func TestProjectUpdate(t *testing.T) {
	tests := []struct {
		name            string
		in              UpdateProjectInput
		wantName        string
		wantDescription string
	}{
		{"rename only keeps description", UpdateProjectInput{Name: strPtr("  Beta  ")}, "Beta", "first"},
		{"description only keeps name", UpdateProjectInput{Description: strPtr(" second ")}, "Alpha", "second"},
		{"both fields", UpdateProjectInput{Name: strPtr("Beta"), Description: strPtr("second")}, "Beta", "second"},
		{"clear description", UpdateProjectInput{Description: strPtr("")}, "Alpha", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			created, err := env.projects.Create(ctx, env.session.WorkspaceID, CreateProjectInput{Name: "Alpha", Description: "first"})
			if err != nil {
				t.Fatal(err)
			}
			env.createTasks(t, created.ID, "one", "two")
			before := env.nullableColumn(t, `SELECT updated_at FROM projects WHERE id = ?`, created.ID)

			got, err := env.projects.Update(ctx, env.session.WorkspaceID, created.ID, tt.in)
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if got.ID != created.ID || got.Name != tt.wantName || got.Description != tt.wantDescription {
				t.Errorf("Update() = %+v, want name %q description %q", got, tt.wantName, tt.wantDescription)
			}
			if got.TaskCount != 2 {
				t.Errorf("TaskCount = %d, want 2", got.TaskCount)
			}

			stored, err := env.projects.Get(ctx, env.session.WorkspaceID, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Name != tt.wantName || stored.Description != tt.wantDescription {
				t.Errorf("stored = %+v, want name %q description %q", stored, tt.wantName, tt.wantDescription)
			}
			after := env.nullableColumn(t, `SELECT updated_at FROM projects WHERE id = ?`, created.ID)
			if after == nil || before == nil || *after < *before {
				t.Errorf("updated_at = %v, want >= %v", after, before)
			}
		})
	}
}

func TestProjectUpdateErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  func(t *testing.T, env taskEnv) string
		in      UpdateProjectInput
		wantErr error
	}{
		{"blank name", func(t *testing.T, env taskEnv) string { return env.project(t, "Alpha").ID },
			UpdateProjectInput{Name: strPtr("   ")}, ErrInvalidInput},
		{"no fields", func(t *testing.T, env taskEnv) string { return env.project(t, "Alpha").ID },
			UpdateProjectInput{}, ErrInvalidInput},
		{"unknown project", func(t *testing.T, env taskEnv) string { return model.NewID() },
			UpdateProjectInput{Name: strPtr("x")}, ErrNotFound},
		{"deleted project", func(t *testing.T, env taskEnv) string {
			p := env.project(t, "gone")
			if err := env.projects.Delete(context.Background(), env.session.WorkspaceID, p.ID); err != nil {
				t.Fatal(err)
			}
			return p.ID
		}, UpdateProjectInput{Name: strPtr("x")}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTaskEnv(t)
			id := tt.target(t, env)
			if _, err := env.projects.Update(context.Background(), env.session.WorkspaceID, id, tt.in); !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProjectUpdateOtherWorkspace(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	p := env.project(t, "Alpha")

	if _, err := env.projects.Update(ctx, model.NewID(), p.ID, UpdateProjectInput{Name: strPtr("x")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
	if err := env.projects.Delete(ctx, model.NewID(), p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestProjectDeleteSoftDeletesChildren(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	second := env.createPlan(t, project.ID, "Second")
	tasks := env.createTasks(t, project.ID, "one", "two")
	if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "three", PlanID: second.ID}); err != nil {
		t.Fatal(err)
	}

	teamID := model.NewID()
	now := nowUTC()
	if _, err := env.db.Exec(`INSERT INTO teams (id, workspace_id, name, created_at, updated_at) VALUES (?, ?, 'Team', ?, ?)`,
		teamID, env.session.WorkspaceID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := env.db.Exec(`INSERT INTO project_teams (workspace_id, project_id, team_id, added_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		env.session.WorkspaceID, project.ID, teamID, now, now); err != nil {
		t.Fatal(err)
	}

	other := env.project(t, "Other")
	otherTasks := env.createTasks(t, other.ID, "keep")

	if err := env.projects.Delete(ctx, env.session.WorkspaceID, project.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if env.nullableColumn(t, `SELECT deleted_at FROM projects WHERE id = ?`, project.ID) == nil {
		t.Error("project deleted_at not set")
	}
	var open int
	for _, check := range []struct{ name, query string }{
		{"plans", `SELECT COUNT(*) FROM plans WHERE project_id = ? AND deleted_at IS NULL`},
		{"statuses", `SELECT COUNT(*) FROM statuses s JOIN plans p ON p.id = s.plan_id WHERE p.project_id = ? AND s.deleted_at IS NULL`},
		{"tasks", `SELECT COUNT(*) FROM tasks WHERE project_id = ? AND deleted_at IS NULL`},
		{"project_teams", `SELECT COUNT(*) FROM project_teams WHERE project_id = ? AND removed_at IS NULL`},
	} {
		if err := env.db.QueryRow(check.query, project.ID).Scan(&open); err != nil {
			t.Fatal(err)
		}
		if open != 0 {
			t.Errorf("%s: %d rows not deleted", check.name, open)
		}
	}
	if env.nullableColumn(t, `SELECT deleted_at FROM tasks WHERE id = ?`, tasks[0].ID) == nil {
		t.Error("task deleted_at not set")
	}

	// The other project is untouched.
	if env.nullableColumn(t, `SELECT deleted_at FROM tasks WHERE id = ?`, otherTasks[0].ID) != nil {
		t.Error("task of another project was deleted")
	}
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM plans WHERE project_id = ? AND deleted_at IS NULL`, other.ID).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Errorf("other project plans = %d, want 1", open)
	}

	projects, err := env.projects.List(ctx, env.session.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != other.ID {
		t.Errorf("List() = %+v, want only Other", projects)
	}
	if _, err := env.projects.Get(ctx, env.session.WorkspaceID, project.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
	if err := env.projects.Delete(ctx, env.session.WorkspaceID, project.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete() error = %v, want ErrNotFound", err)
	}
	if err := env.projects.Delete(ctx, env.session.WorkspaceID, model.NewID()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete(unknown) error = %v, want ErrNotFound", err)
	}
}

func TestPlanUpdate(t *testing.T) {
	tests := []struct {
		name            string
		in              UpdatePlanInput
		wantName        string
		wantDescription string
	}{
		{"rename only keeps description", UpdatePlanInput{Name: strPtr(" Sprint 2 ")}, "Sprint 2", "goals"},
		{"description only keeps name", UpdatePlanInput{Description: strPtr(" new goals ")}, "Sprint", "new goals"},
		{"both fields", UpdatePlanInput{Name: strPtr("Sprint 2"), Description: strPtr("x")}, "Sprint 2", "x"},
		{"same name is allowed", UpdatePlanInput{Name: strPtr("Sprint")}, "Sprint", "goals"},
		{"clear description", UpdatePlanInput{Description: strPtr("")}, "Sprint", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			project := env.project(t, "Alpha")
			plan, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Sprint", Description: "goals"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "t", PlanID: plan.ID}); err != nil {
				t.Fatal(err)
			}

			got, err := env.projects.UpdatePlan(ctx, env.session.WorkspaceID, project.ID, plan.ID, tt.in)
			if err != nil {
				t.Fatalf("UpdatePlan() error = %v", err)
			}
			if got.ID != plan.ID || got.ProjectID != project.ID || got.Name != tt.wantName || got.Description != tt.wantDescription {
				t.Errorf("UpdatePlan() = %+v, want name %q description %q", got, tt.wantName, tt.wantDescription)
			}
			if got.TaskCount != 1 {
				t.Errorf("TaskCount = %d, want 1", got.TaskCount)
			}

			stored, err := env.projects.GetPlan(ctx, env.session.WorkspaceID, project.ID, plan.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Name != tt.wantName || stored.Description != tt.wantDescription {
				t.Errorf("stored = %+v", stored)
			}
		})
	}
}

func TestPlanUpdateErrors(t *testing.T) {
	type ids struct{ projectID, planID string }
	tests := []struct {
		name    string
		setup   func(t *testing.T, env taskEnv) ids
		in      UpdatePlanInput
		wantErr error
	}{
		{"blank name", func(t *testing.T, env taskEnv) ids {
			p := env.project(t, "Alpha")
			return ids{p.ID, env.planOf(t, p.ID)}
		}, UpdatePlanInput{Name: strPtr(" ")}, ErrInvalidInput},
		{"no fields", func(t *testing.T, env taskEnv) ids {
			p := env.project(t, "Alpha")
			return ids{p.ID, env.planOf(t, p.ID)}
		}, UpdatePlanInput{}, ErrInvalidInput},
		{"name used by another plan", func(t *testing.T, env taskEnv) ids {
			p := env.project(t, "Alpha")
			env.createPlan(t, p.ID, "Taken")
			return ids{p.ID, env.planOf(t, p.ID)}
		}, UpdatePlanInput{Name: strPtr(" Taken ")}, ErrConflict},
		{"unknown plan", func(t *testing.T, env taskEnv) ids {
			p := env.project(t, "Alpha")
			return ids{p.ID, model.NewID()}
		}, UpdatePlanInput{Name: strPtr("x")}, ErrNotFound},
		{"plan of another project", func(t *testing.T, env taskEnv) ids {
			p := env.project(t, "Alpha")
			other := env.project(t, "Other")
			return ids{p.ID, env.planOf(t, other.ID)}
		}, UpdatePlanInput{Name: strPtr("x")}, ErrNotFound},
		{"deleted plan", func(t *testing.T, env taskEnv) ids {
			p := env.project(t, "Alpha")
			plan := env.createPlan(t, p.ID, "Gone")
			if err := env.projects.DeletePlan(context.Background(), env.session.WorkspaceID, p.ID, plan.ID); err != nil {
				t.Fatal(err)
			}
			return ids{p.ID, plan.ID}
		}, UpdatePlanInput{Name: strPtr("x")}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTaskEnv(t)
			target := tt.setup(t, env)
			_, err := env.projects.UpdatePlan(context.Background(), env.session.WorkspaceID, target.projectID, target.planID, tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdatePlan() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlanUpdateNameOfDeletedPlanIsFree(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	gone := env.createPlan(t, project.ID, "Old")
	if err := env.projects.DeletePlan(ctx, env.session.WorkspaceID, project.ID, gone.ID); err != nil {
		t.Fatal(err)
	}

	got, err := env.projects.UpdatePlan(ctx, env.session.WorkspaceID, project.ID, env.planOf(t, project.ID), UpdatePlanInput{Name: strPtr("Old")})
	if err != nil {
		t.Fatalf("UpdatePlan() error = %v", err)
	}
	if got.Name != "Old" {
		t.Errorf("Name = %q, want Old", got.Name)
	}
}

func TestPlanDeleteSoftDeletesStatusesAndTasks(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	mainPlan := env.planOf(t, project.ID)
	plan := env.createPlan(t, project.ID, "Sprint")
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "in sprint", PlanID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	kept := env.createTasks(t, project.ID, "in main")

	if err := env.projects.DeletePlan(ctx, env.session.WorkspaceID, project.ID, plan.ID); err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}

	if env.nullableColumn(t, `SELECT deleted_at FROM plans WHERE id = ?`, plan.ID) == nil {
		t.Error("plan deleted_at not set")
	}
	if env.nullableColumn(t, `SELECT deleted_at FROM tasks WHERE id = ?`, task.ID) == nil {
		t.Error("task deleted_at not set")
	}
	var open int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM statuses WHERE plan_id = ? AND deleted_at IS NULL`, plan.ID).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 0 {
		t.Errorf("%d statuses not deleted", open)
	}

	// The project's other plan is untouched.
	if env.nullableColumn(t, `SELECT deleted_at FROM tasks WHERE id = ?`, kept[0].ID) != nil {
		t.Error("task of another plan was deleted")
	}
	if got := env.statuses(t, project.ID, mainPlan); len(got) != 3 {
		t.Errorf("main plan statuses = %d, want 3", len(got))
	}

	plans, err := env.projects.Plans(ctx, env.session.WorkspaceID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].ID != mainPlan {
		t.Errorf("Plans() = %+v, want only main plan", plans)
	}
	stored, err := env.projects.Get(ctx, env.session.WorkspaceID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.TaskCount != 1 {
		t.Errorf("project TaskCount = %d, want 1", stored.TaskCount)
	}
	if _, err := env.projects.GetPlan(ctx, env.session.WorkspaceID, project.ID, plan.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetPlan() error = %v, want ErrNotFound", err)
	}
	if err := env.projects.DeletePlan(ctx, env.session.WorkspaceID, project.ID, plan.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second DeletePlan() error = %v, want ErrNotFound", err)
	}

	// The name can be reused once the plan is deleted.
	env.createPlan(t, project.ID, "Sprint")
}

func TestPlanDeleteKeepsOneWholeProjectPlan(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	mainPlan := env.planOf(t, project.ID)

	if err := env.projects.DeletePlan(ctx, env.session.WorkspaceID, project.ID, mainPlan); !errors.Is(err, ErrConflict) {
		t.Fatalf("DeletePlan(last plan) error = %v, want ErrConflict", err)
	}
	if env.nullableColumn(t, `SELECT deleted_at FROM plans WHERE id = ?`, mainPlan) != nil {
		t.Error("last plan was deleted")
	}

	// With a second plan the original default plan can go, and new tasks
	// without an explicit plan land in the remaining one.
	second := env.createPlan(t, project.ID, "Second")
	if err := env.projects.DeletePlan(ctx, env.session.WorkspaceID, project.ID, mainPlan); err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "t"})
	if err != nil {
		t.Fatalf("task Create() error = %v", err)
	}
	if task.PlanID != second.ID {
		t.Errorf("task PlanID = %q, want %q", task.PlanID, second.ID)
	}
}

func TestPlanDeleteErrors(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	other := env.project(t, "Other")
	env.createPlan(t, other.ID, "Extra")

	tests := []struct {
		name              string
		projectID, planID string
	}{
		{"unknown plan", project.ID, model.NewID()},
		{"unknown project", model.NewID(), env.planOf(t, project.ID)},
		{"plan of another project", project.ID, env.planOf(t, other.ID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := env.projects.DeletePlan(ctx, env.session.WorkspaceID, tt.projectID, tt.planID); !errors.Is(err, ErrNotFound) {
				t.Errorf("DeletePlan() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestTaskUpdate(t *testing.T) {
	due := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	later := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name            string
		in              UpdateTaskInput
		wantTitle       string
		wantDescription string
		wantDue         *time.Time
	}{
		{"rename only keeps the rest", UpdateTaskInput{Title: strPtr("  Renamed ")}, "Renamed", "desc", &due},
		{"description only", UpdateTaskInput{Description: strPtr(" other ")}, "Write", "other", &due},
		{"set due date", UpdateTaskInput{DueDate: &later, DueDateSet: true}, "Write", "desc", &later},
		{"clear due date", UpdateTaskInput{DueDateSet: true}, "Write", "desc", nil},
		{"all fields", UpdateTaskInput{Title: strPtr("A"), Description: strPtr(""), DueDate: &later, DueDateSet: true}, "A", "", &later},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			project := env.project(t, "Alpha")
			planID := env.planOf(t, project.ID)
			created, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "Write", Description: "desc", DueDate: &due})
			if err != nil {
				t.Fatal(err)
			}

			got, err := env.tasks.Update(ctx, env.session, created.ID, tt.in)
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if got.ID != created.ID || got.Title != tt.wantTitle || got.Description != tt.wantDescription {
				t.Errorf("Update() = %+v, want title %q description %q", got, tt.wantTitle, tt.wantDescription)
			}
			if got.ProjectID != project.ID || got.ProjectName != "Alpha" || got.PlanID != planID ||
				got.StatusID != created.StatusID || got.StatusCategory != model.StatusToDo || got.Position != created.Position {
				t.Errorf("Update() placement = %+v, want unchanged from %+v", got, created)
			}
			checkDue := func(label string, d *time.Time) {
				t.Helper()
				switch {
				case tt.wantDue == nil && d != nil:
					t.Errorf("%s DueDate = %v, want nil", label, d)
				case tt.wantDue != nil && (d == nil || !d.Equal(*tt.wantDue)):
					t.Errorf("%s DueDate = %v, want %v", label, d, tt.wantDue)
				}
			}
			checkDue("returned", got.DueDate)

			listed, err := env.tasks.ListByPlan(ctx, env.session, project.ID, planID)
			if err != nil {
				t.Fatal(err)
			}
			if len(listed) != 1 || listed[0].Title != tt.wantTitle || listed[0].Description != tt.wantDescription {
				t.Fatalf("ListByPlan() = %+v", listed)
			}
			checkDue("stored", listed[0].DueDate)
		})
	}
}

func TestTaskUpdateErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  func(t *testing.T, env taskEnv) string
		in      UpdateTaskInput
		wantErr error
	}{
		{"blank title", func(t *testing.T, env taskEnv) string {
			return env.createTasks(t, env.project(t, "Alpha").ID, "x")[0].ID
		}, UpdateTaskInput{Title: strPtr("  ")}, ErrInvalidInput},
		{"no fields", func(t *testing.T, env taskEnv) string {
			return env.createTasks(t, env.project(t, "Alpha").ID, "x")[0].ID
		}, UpdateTaskInput{}, ErrInvalidInput},
		{"unknown task", func(t *testing.T, env taskEnv) string { return model.NewID() },
			UpdateTaskInput{Title: strPtr("x")}, ErrNotFound},
		{"deleted task", func(t *testing.T, env taskEnv) string {
			task := env.createTasks(t, env.project(t, "Alpha").ID, "x")[0]
			if err := env.tasks.Delete(context.Background(), env.session, task.ID); err != nil {
				t.Fatal(err)
			}
			return task.ID
		}, UpdateTaskInput{Title: strPtr("x")}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTaskEnv(t)
			id := tt.target(t, env)
			if _, err := env.tasks.Update(context.Background(), env.session, id, tt.in); !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskUpdateOtherWorkspace(t *testing.T) {
	env := newTaskEnv(t)
	task := env.createTasks(t, env.project(t, "Alpha").ID, "x")[0]
	stranger := env.session
	stranger.WorkspaceID = model.NewID()

	if _, err := env.tasks.Update(context.Background(), stranger, task.ID, UpdateTaskInput{Title: strPtr("y")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
}
