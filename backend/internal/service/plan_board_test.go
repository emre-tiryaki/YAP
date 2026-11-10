package service

import (
	"context"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func (e taskEnv) statuses(t *testing.T, projectID, planID string) []model.Status {
	t.Helper()
	statuses, err := e.projects.Statuses(context.Background(), e.session.WorkspaceID, projectID, planID)
	if err != nil {
		t.Fatalf("Statuses() error = %v", err)
	}
	return statuses
}

func (e taskEnv) taskTitles(t *testing.T, projectID, planID, statusID string) []string {
	t.Helper()
	tasks, err := e.tasks.ListByPlan(context.Background(), e.session, projectID, planID)
	if err != nil {
		t.Fatalf("ListByPlan() error = %v", err)
	}
	titles := []string{}
	for _, task := range tasks {
		if task.StatusID == statusID {
			titles = append(titles, task.Title)
		}
	}
	return titles
}

func (e taskEnv) createTasks(t *testing.T, projectID string, titles ...string) []model.Task {
	t.Helper()
	tasks := make([]model.Task, 0, len(titles))
	for _, title := range titles {
		task, err := e.tasks.Create(context.Background(), e.session, projectID, CreateTaskInput{Title: title})
		if err != nil {
			t.Fatalf("Create(%q) error = %v", title, err)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEveryNewPlanHasDoneStatus(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	second, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Second"})
	if err != nil {
		t.Fatal(err)
	}

	for _, planID := range []string{env.planOf(t, project.ID), second.ID} {
		hasDone := false
		for _, s := range env.statuses(t, project.ID, planID) {
			hasDone = hasDone || s.Category == model.StatusDone
		}
		if !hasDone {
			t.Errorf("plan %s has no done status", planID)
		}
	}
}

func TestGetPlanCountsDoneTasks(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)
	tasks := env.createTasks(t, project.ID, "a", "b", "c")
	done := env.statuses(t, project.ID, planID)[2]

	if err := env.tasks.Move(ctx, env.session, tasks[0].ID, MoveTaskInput{StatusID: done.ID}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if err := env.tasks.Delete(ctx, env.session, tasks[1].ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	plan, err := env.projects.GetPlan(ctx, env.session.WorkspaceID, project.ID, planID)
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if plan.Name != "Main" || plan.TaskCount != 2 || plan.DoneCount != 1 {
		t.Errorf("plan = %+v, want Main with 2 tasks, 1 done", plan)
	}
}

func TestGetPlanNotFound(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	other := env.project(t, "Beta")

	tests := []struct {
		name, projectID, planID string
	}{
		{"missing plan", project.ID, "missing"},
		{"plan of another project", project.ID, env.planOf(t, other.ID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.projects.GetPlan(ctx, env.session.WorkspaceID, tt.projectID, tt.planID)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("GetPlan() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestCreateStatusAppendsAfterLastColumn(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)

	created, err := env.projects.CreateStatus(ctx, env.session.WorkspaceID, project.ID, planID, CreateStatusInput{
		Title: "  Review  ", Color: "#A855F7", Category: model.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("CreateStatus() error = %v", err)
	}
	if created.Title != "Review" || created.Color != "#a855f7" {
		t.Errorf("status = %+v, want trimmed title and lowercase color", created)
	}

	statuses := env.statuses(t, project.ID, planID)
	titles := []string{}
	for _, s := range statuses {
		titles = append(titles, s.Title)
	}
	if want := []string{"To Do", "In Progress", "Done", "Review"}; !equalStrings(titles, want) {
		t.Errorf("statuses = %v, want %v", titles, want)
	}
}

func TestCreateStatusValidation(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)

	tests := []struct {
		name   string
		planID string
		in     CreateStatusInput
		want   error
	}{
		{"empty title", planID, CreateStatusInput{Title: " ", Color: "#000000", Category: model.StatusToDo}, ErrInvalidInput},
		{"bad color", planID, CreateStatusInput{Title: "x", Color: "red", Category: model.StatusToDo}, ErrInvalidInput},
		{"bad category", planID, CreateStatusInput{Title: "x", Color: "#000000", Category: "blocked"}, ErrInvalidInput},
		{"missing plan", "missing", CreateStatusInput{Title: "x", Color: "#000000", Category: model.StatusToDo}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.projects.CreateStatus(ctx, env.session.WorkspaceID, project.ID, tt.planID, tt.in)
			if !errors.Is(err, tt.want) {
				t.Errorf("CreateStatus() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestTaskCreateInExplicitStatus(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)
	done := env.statuses(t, project.ID, planID)[2]

	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "x", PlanID: planID, StatusID: done.ID})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.StatusID != done.ID || task.StatusCategory != model.StatusDone {
		t.Errorf("task = %+v, want done status", task)
	}
}

func TestTaskCreateRejectsStatusOfAnotherPlan(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	second, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	foreign := env.statuses(t, project.ID, second.ID)[0]

	_, err = env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "x", StatusID: foreign.ID})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Create() error = %v, want ErrInvalidInput", err)
	}
}

func TestTaskMove(t *testing.T) {
	tests := []struct {
		name        string
		task        int // index into a, b, c (all start in To Do)
		status      int // index into To Do, In Progress, Done
		index       int
		wantTodo    []string
		wantTarget  []string
		wantDoneCat bool
	}{
		{"to front of same status", 2, 0, 0, []string{"c", "a", "b"}, nil, false},
		{"to middle of same status", 0, 0, 1, []string{"b", "a", "c"}, nil, false},
		{"to end of same status", 0, 0, 2, []string{"b", "c", "a"}, nil, false},
		{"index clamped to end", 0, 0, 99, []string{"b", "c", "a"}, nil, false},
		{"negative index clamped to front", 1, 0, -5, []string{"b", "a", "c"}, nil, false},
		{"into another status", 1, 1, 0, []string{"a", "c"}, []string{"b"}, false},
		{"into done counts as done", 0, 2, 0, []string{"b", "c"}, []string{"a"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			project := env.project(t, "Alpha")
			planID := env.planOf(t, project.ID)
			statuses := env.statuses(t, project.ID, planID)
			tasks := env.createTasks(t, project.ID, "a", "b", "c")
			target := statuses[tt.status]

			err := env.tasks.Move(ctx, env.session, tasks[tt.task].ID, MoveTaskInput{StatusID: target.ID, Index: tt.index})
			if err != nil {
				t.Fatalf("Move() error = %v", err)
			}
			if got := env.taskTitles(t, project.ID, planID, statuses[0].ID); !equalStrings(got, tt.wantTodo) {
				t.Errorf("todo = %v, want %v", got, tt.wantTodo)
			}
			if tt.status != 0 {
				if got := env.taskTitles(t, project.ID, planID, target.ID); !equalStrings(got, tt.wantTarget) {
					t.Errorf("target = %v, want %v", got, tt.wantTarget)
				}
			}
			plan, err := env.projects.GetPlan(ctx, env.session.WorkspaceID, project.ID, planID)
			if err != nil {
				t.Fatal(err)
			}
			wantDone := 0
			if tt.wantDoneCat {
				wantDone = 1
			}
			if plan.DoneCount != wantDone {
				t.Errorf("DoneCount = %d, want %d", plan.DoneCount, wantDone)
			}
		})
	}
}

func TestTaskMoveRepeatedlyBetweenSameNeighbours(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)
	todo := env.statuses(t, project.ID, planID)[0]
	tasks := env.createTasks(t, project.ID, "a", "b", "c")

	// Ping-pong b and c into the slot right after a many times; every move
	// must find room between the neighbours.
	for i := 0; i < 50; i++ {
		task := tasks[1+i%2]
		if err := env.tasks.Move(ctx, env.session, task.ID, MoveTaskInput{StatusID: todo.ID, Index: 1}); err != nil {
			t.Fatalf("move %d: %v", i, err)
		}
	}
	if got, want := env.taskTitles(t, project.ID, planID, todo.ID), []string{"a", "c", "b"}; !equalStrings(got, want) {
		t.Errorf("todo = %v, want %v", got, want)
	}
}

func TestTaskMoveErrors(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	second, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	foreign := env.statuses(t, project.ID, second.ID)[0]
	task := env.createTasks(t, project.ID, "a")[0]
	deleted := env.createTasks(t, project.ID, "gone")[0]
	if err := env.tasks.Delete(ctx, env.session, deleted.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		taskID string
		in     MoveTaskInput
		want   error
	}{
		{"missing status id", task.ID, MoveTaskInput{}, ErrInvalidInput},
		{"status of another plan", task.ID, MoveTaskInput{StatusID: foreign.ID}, ErrInvalidInput},
		{"missing task", "missing", MoveTaskInput{StatusID: foreign.ID}, ErrNotFound},
		{"deleted task", deleted.ID, MoveTaskInput{StatusID: foreign.ID}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := env.tasks.Move(ctx, env.session, tt.taskID, tt.in); !errors.Is(err, tt.want) {
				t.Errorf("Move() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestTaskDeleteSoftDeletesSubtasksAndAssignments(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)
	tasks := env.createTasks(t, project.ID, "parent", "child", "grandchild", "other")
	for _, link := range [][2]int{{1, 0}, {2, 1}} {
		if _, err := env.db.Exec(`UPDATE tasks SET parent_id = ? WHERE id = ?`, tasks[link[1]].ID, tasks[link[0]].ID); err != nil {
			t.Fatal(err)
		}
	}

	if err := env.tasks.Delete(ctx, env.session, tasks[0].ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	listed, err := env.tasks.ListByPlan(ctx, env.session, project.ID, planID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != tasks[3].ID {
		t.Errorf("remaining tasks = %+v, want only %q", listed, "other")
	}

	for i, task := range tasks {
		var deletedAt, removedAt *string
		if err := env.db.QueryRow(`SELECT deleted_at FROM tasks WHERE id = ?`, task.ID).Scan(&deletedAt); err != nil {
			t.Fatal(err)
		}
		if err := env.db.QueryRow(`SELECT removed_at FROM task_assignees WHERE task_id = ?`, task.ID).Scan(&removedAt); err != nil {
			t.Fatal(err)
		}
		wantDeleted := i < 3
		if (deletedAt != nil) != wantDeleted || (removedAt != nil) != wantDeleted {
			t.Errorf("%s: deleted_at = %v, removed_at = %v, want set = %v", task.Title, deletedAt, removedAt, wantDeleted)
		}
	}

	if err := env.tasks.Delete(ctx, env.session, tasks[0].ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete() error = %v, want ErrNotFound", err)
	}
}

func TestListByPlanNotFound(t *testing.T) {
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	if _, err := env.tasks.ListByPlan(context.Background(), env.session, project.ID, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("ListByPlan() error = %v, want ErrNotFound", err)
	}
}
