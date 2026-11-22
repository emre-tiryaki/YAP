package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

type labelEnv struct {
	taskEnv
	labels *LabelService
}

func newLabelEnv(t *testing.T) labelEnv {
	t.Helper()
	env := newTaskEnv(t)
	return labelEnv{taskEnv: env, labels: NewLabelService(env.db, env.projectRepo, repository.NewLabelRepository())}
}

func TestLabelCreateValidation(t *testing.T) {
	ctx := context.Background()
	env := newLabelEnv(t)
	project := env.project(t, "Backend")
	other := env.project(t, "Web")
	if _, err := env.labels.Create(ctx, env.session, project.ID, CreateLabelInput{Name: "bug", Color: "red"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name    string
		project string
		input   CreateLabelInput
		wantErr error
	}{
		{"valid", project.ID, CreateLabelInput{Name: " feature ", Color: "blue"}, nil},
		{"same name in another project", other.ID, CreateLabelInput{Name: "bug", Color: "red"}, nil},
		{"duplicate name", project.ID, CreateLabelInput{Name: "bug", Color: "teal"}, ErrLabelNameTaken},
		{"empty name", project.ID, CreateLabelInput{Name: "  ", Color: "red"}, ErrInvalidInput},
		{"long name", project.ID, CreateLabelInput{Name: string(make([]rune, 51)) + "x", Color: "red"}, ErrInvalidInput},
		{"unknown colour", project.ID, CreateLabelInput{Name: "x", Color: "purple"}, ErrInvalidInput},
		{"unknown project", "missing", CreateLabelInput{Name: "x", Color: "red"}, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.labels.Create(ctx, env.session, tt.project, tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
	if !errors.Is(ErrLabelNameTaken, ErrConflict) {
		t.Error("ErrLabelNameTaken must be a conflict (HTTP 409)")
	}

	list, err := env.labels.List(ctx, env.session, project.ID)
	if err != nil || len(list) != 2 || list[0].Name != "bug" || list[1].Name != "feature" {
		t.Errorf("List() = %+v (err %v), want bug then feature", list, err)
	}
}

func TestLabelUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	env := newLabelEnv(t)
	project := env.project(t, "Backend")
	bug, _ := env.labels.Create(ctx, env.session, project.ID, CreateLabelInput{Name: "bug", Color: "red"})
	feature, _ := env.labels.Create(ctx, env.session, project.ID, CreateLabelInput{Name: "feature", Color: "blue"})
	str := func(s string) *string { return &s }

	if got, err := env.labels.Update(ctx, env.session, bug.ID, UpdateLabelInput{Name: str("defect"), Color: str("pink")}); err != nil || got.Name != "defect" || got.Color != "pink" {
		t.Errorf("Update() = %+v (err %v), want defect/pink", got, err)
	}
	if _, err := env.labels.Update(ctx, env.session, bug.ID, UpdateLabelInput{Name: str("feature")}); !errors.Is(err, ErrLabelNameTaken) {
		t.Errorf("rename to a taken name error = %v, want ErrLabelNameTaken", err)
	}
	if _, err := env.labels.Update(ctx, env.session, bug.ID, UpdateLabelInput{Color: str("neon")}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("unknown colour error = %v, want ErrInvalidInput", err)
	}
	if _, err := env.labels.Update(ctx, env.session, bug.ID, UpdateLabelInput{}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("empty update error = %v, want ErrInvalidInput", err)
	}

	if err := env.labels.Delete(ctx, env.session, feature.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := env.labels.Delete(ctx, env.session, feature.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete() error = %v, want ErrNotFound", err)
	}
	// The name can be used again.
	if _, err := env.labels.Create(ctx, env.session, project.ID, CreateLabelInput{Name: "feature", Color: "teal"}); err != nil {
		t.Errorf("Create() after delete error = %v", err)
	}
}

func (e labelEnv) label(t *testing.T, projectID, name string) string {
	t.Helper()
	l, err := e.labels.Create(context.Background(), e.session, projectID, CreateLabelInput{Name: name, Color: "gray"})
	if err != nil {
		t.Fatalf("Create label error = %v", err)
	}
	return l.ID
}

func (e labelEnv) setLabels(t *testing.T, taskID string, ids ...string) error {
	t.Helper()
	_, err := e.tasks.Update(context.Background(), e.session, taskID, UpdateTaskInput{LabelIDs: &ids})
	return err
}

func (e labelEnv) taskLabels(t *testing.T, projectID, planID, taskID string) []string {
	t.Helper()
	tasks, err := e.tasks.ListByPlan(context.Background(), e.session, projectID, planID)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if task.ID == taskID {
			return task.LabelIDs
		}
	}
	t.Fatalf("task %s not listed", taskID)
	return nil
}

func TestTaskLabelsReplaceTheSet(t *testing.T) {
	ctx := context.Background()
	env := newLabelEnv(t)
	project := env.project(t, "Backend")
	a, b, c := env.label(t, project.ID, "a"), env.label(t, project.ID, "b"), env.label(t, project.ID, "c")
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}

	if err := env.setLabels(t, task.ID, a, b); err != nil {
		t.Fatalf("set {a,b} error = %v", err)
	}
	if got := env.taskLabels(t, project.ID, task.PlanID, task.ID); !slices.Equal(got, []string{a, b}) {
		t.Errorf("labels = %v, want [a b]", got)
	}
	// Replacing: b stays, a is removed, c is added; duplicates are ignored.
	if err := env.setLabels(t, task.ID, b, c, c); err != nil {
		t.Fatalf("set {b,c} error = %v", err)
	}
	if got := env.taskLabels(t, project.ID, task.PlanID, task.ID); !slices.Equal(got, []string{b, c}) {
		t.Errorf("labels = %v, want [b c]", got)
	}
	// The removed link is kept as a row with removed_at, and comes back.
	var removed int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM task_labels WHERE task_id = ? AND label_id = ? AND removed_at IS NOT NULL`, task.ID, a).Scan(&removed); err != nil || removed != 1 {
		t.Errorf("removed a-links = %d (err %v), want 1", removed, err)
	}
	if err := env.setLabels(t, task.ID, a); err != nil {
		t.Fatalf("set {a} error = %v", err)
	}
	if got := env.taskLabels(t, project.ID, task.PlanID, task.ID); !slices.Equal(got, []string{a}) {
		t.Errorf("labels = %v, want [a]", got)
	}
	var rows int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM task_labels WHERE task_id = ? AND label_id = ?`, task.ID, a).Scan(&rows); err != nil || rows != 1 {
		t.Errorf("a-link rows = %d (err %v), want the same row reused", rows, err)
	}
	if err := env.setLabels(t, task.ID); err != nil {
		t.Fatalf("clear error = %v", err)
	}
	if got := env.taskLabels(t, project.ID, task.PlanID, task.ID); len(got) != 0 {
		t.Errorf("labels = %v, want none", got)
	}
}

func TestTaskLabelsMustBelongToTheProject(t *testing.T) {
	ctx := context.Background()
	env := newLabelEnv(t)
	project := env.project(t, "Backend")
	other := env.project(t, "Web")
	foreign := env.label(t, other.ID, "foreign")
	gone := env.label(t, project.ID, "gone")
	if err := env.labels.Delete(ctx, env.session, gone); err != nil {
		t.Fatal(err)
	}
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}

	for name, ids := range map[string][]string{"another project's label": {foreign}, "deleted label": {gone}, "unknown label": {"missing"}} {
		if err := env.setLabels(t, task.ID, ids...); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestDeletingALabelRemovesItFromTasks(t *testing.T) {
	ctx := context.Background()
	env := newLabelEnv(t)
	project := env.project(t, "Backend")
	keep, drop := env.label(t, project.ID, "keep"), env.label(t, project.ID, "drop")
	task, _ := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T"})
	if err := env.setLabels(t, task.ID, keep, drop); err != nil {
		t.Fatal(err)
	}
	if err := env.labels.Delete(ctx, env.session, drop); err != nil {
		t.Fatal(err)
	}
	if got := env.taskLabels(t, project.ID, task.PlanID, task.ID); !slices.Equal(got, []string{keep}) {
		t.Errorf("labels = %v, want [keep]", got)
	}
	var open int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM task_labels WHERE label_id = ? AND removed_at IS NULL`, drop).Scan(&open); err != nil || open != 0 {
		t.Errorf("open links of the deleted label = %d (err %v), want 0", open, err)
	}
}

func TestDeletingTasksPlansAndProjectsClosesTheirLabelLinks(t *testing.T) {
	ctx := context.Background()
	env := newLabelEnv(t)
	project := env.project(t, "Backend")
	label := env.label(t, project.ID, "bug")
	task, _ := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T"})
	sub, err := env.tasks.CreateSubtask(ctx, env.session, task.ID, "S")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.setLabels(t, task.ID, label); err != nil {
		t.Fatal(err)
	}
	if err := env.setLabels(t, sub.ID, label); err != nil {
		t.Fatal(err)
	}
	open := func() int {
		var n int
		if err := env.db.QueryRow(`SELECT COUNT(*) FROM task_labels WHERE removed_at IS NULL`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if open() != 2 {
		t.Fatalf("open links = %d, want 2", open())
	}
	if err := env.tasks.Delete(ctx, env.session, task.ID); err != nil {
		t.Fatal(err)
	}
	if open() != 0 {
		t.Errorf("open links after deleting the task = %d, want 0 (subtask included)", open())
	}

	other, _ := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "O"})
	if err := env.setLabels(t, other.ID, label); err != nil {
		t.Fatal(err)
	}
	if err := env.projects.Delete(ctx, env.session.WorkspaceID, project.ID); err != nil {
		t.Fatal(err)
	}
	if open() != 0 {
		t.Errorf("open links after deleting the project = %d, want 0", open())
	}
	var live int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM labels WHERE deleted_at IS NULL`).Scan(&live); err != nil || live != 0 {
		t.Errorf("live labels after deleting the project = %d (err %v), want 0", live, err)
	}
}
