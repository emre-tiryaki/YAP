package service

import (
	"context"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func (e taskEnv) subtask(t *testing.T, parentID, title string) model.Task {
	t.Helper()
	task, err := e.tasks.CreateSubtask(context.Background(), e.session, parentID, title)
	if err != nil {
		t.Fatalf("CreateSubtask(%q) error = %v", title, err)
	}
	return task
}

func (e taskEnv) boardTask(t *testing.T, projectID, planID, taskID string) (model.Task, bool) {
	t.Helper()
	tasks, err := e.tasks.ListByPlan(context.Background(), e.session, projectID, planID)
	if err != nil {
		t.Fatalf("ListByPlan() error = %v", err)
	}
	for _, task := range tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return model.Task{}, false
}

func TestCreateSubtask(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	parent := env.createTasks(t, project.ID, "parent")[0]

	sub := env.subtask(t, parent.ID, "  child  ")
	if sub.Title != "child" || sub.ParentID != parent.ID || sub.PlanID != parent.PlanID ||
		sub.ProjectID != project.ID || sub.StatusCategory != model.StatusToDo {
		t.Errorf("CreateSubtask() = %+v, want child of %s in plan %s", sub, parent.ID, parent.PlanID)
	}
	stored := env.nullableColumn(t, `SELECT parent_id FROM tasks WHERE id = ?`, sub.ID)
	if stored == nil || *stored != parent.ID {
		t.Errorf("stored parent_id = %v, want %s", stored, parent.ID)
	}
	assignee := env.nullableColumn(t, `SELECT membership_id FROM task_assignees WHERE task_id = ?`, sub.ID)
	if assignee == nil || *assignee != env.session.MembershipID {
		t.Errorf("assignee = %v, want %s", assignee, env.session.MembershipID)
	}

	subtasks, err := env.tasks.Subtasks(ctx, env.session, parent.ID)
	if err != nil {
		t.Fatalf("Subtasks() error = %v", err)
	}
	if len(subtasks) != 1 || subtasks[0].ID != sub.ID {
		t.Errorf("Subtasks() = %+v, want [%s]", subtasks, sub.ID)
	}
}

func TestCreateSubtaskErrors(t *testing.T) {
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	parent := env.createTasks(t, project.ID, "parent")[0]
	child := env.subtask(t, parent.ID, "child")
	deleted := env.createTasks(t, project.ID, "gone")[0]
	if err := env.tasks.Delete(context.Background(), env.session, deleted.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		parentID string
		title    string
		want     error
	}{
		{"blank title", parent.ID, "   ", ErrInvalidInput},
		{"unknown parent", model.NewID(), "x", ErrNotFound},
		{"deleted parent", deleted.ID, "x", ErrNotFound},
		{"nested subtask", child.ID, "x", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.tasks.CreateSubtask(context.Background(), env.session, tt.parentID, tt.title)
			if !errors.Is(err, tt.want) {
				t.Errorf("CreateSubtask() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSubtasksOnBoard(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	planID := env.planOf(t, project.ID)
	parent := env.createTasks(t, project.ID, "parent")[0]
	open := env.subtask(t, parent.ID, "open")
	done := env.subtask(t, parent.ID, "done")
	if _, err := env.tasks.SetDone(ctx, env.session, done.ID, true); err != nil {
		t.Fatal(err)
	}
	removed := env.subtask(t, parent.ID, "removed")
	if err := env.tasks.Delete(ctx, env.session, removed.ID); err != nil {
		t.Fatal(err)
	}

	// A subtask in another plan of the project is that plan's share of the work.
	other := env.createPlan(t, project.ID, "Other")
	foreign, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "foreign", PlanID: other.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.db.Exec(`UPDATE tasks SET parent_id = ? WHERE id = ?`, parent.ID, foreign.ID); err != nil {
		t.Fatal(err)
	}

	got, ok := env.boardTask(t, project.ID, planID, parent.ID)
	if !ok {
		t.Fatal("parent missing from board")
	}
	if got.SubtaskCount != 3 || got.DoneSubtaskCount != 1 {
		t.Errorf("counts = %d/%d, want 1 done of 3 (deleted one left out)", got.DoneSubtaskCount, got.SubtaskCount)
	}
	if _, ok := env.boardTask(t, project.ID, planID, open.ID); ok {
		t.Error("same-plan subtask listed on the board, want it nested in the parent")
	}
	if _, ok := env.boardTask(t, project.ID, other.ID, foreign.ID); !ok {
		t.Error("subtask in another plan missing from that plan's board")
	}

	upcoming, err := env.tasks.Upcoming(ctx, env.session)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range upcoming {
		if task.ID == open.ID {
			t.Error("same-plan subtask listed in upcoming tasks")
		}
	}
}

func TestSetDone(t *testing.T) {
	tests := []struct {
		name         string
		startDone    bool
		done         bool
		wantCategory model.StatusCategory
		wantMoved    bool
	}{
		{"complete open task", false, true, model.StatusDone, true},
		{"reopen finished task", true, false, model.StatusToDo, true},
		{"complete finished task is a no-op", true, true, model.StatusDone, false},
		{"reopen open task is a no-op", false, false, model.StatusToDo, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			project := env.project(t, "Alpha")
			planID := env.planOf(t, project.ID)
			statuses := env.statuses(t, project.ID, planID)
			// Another finished task, so a completed task lands after it.
			env.createTasks(t, project.ID, "earlier")
			earlier, _ := env.tasks.ListByPlan(ctx, env.session, project.ID, planID)
			if _, err := env.tasks.SetDone(ctx, env.session, earlier[0].ID, true); err != nil {
				t.Fatal(err)
			}

			task := env.createTasks(t, project.ID, "task")[0]
			if tt.startDone {
				if _, err := env.tasks.SetDone(ctx, env.session, task.ID, true); err != nil {
					t.Fatal(err)
				}
			}
			before := env.nullableColumn(t, `SELECT status_id || '|' || position FROM tasks WHERE id = ?`, task.ID)

			got, err := env.tasks.SetDone(ctx, env.session, task.ID, tt.done)
			if err != nil {
				t.Fatalf("SetDone() error = %v", err)
			}
			if got.StatusCategory != tt.wantCategory {
				t.Errorf("category = %q, want %q", got.StatusCategory, tt.wantCategory)
			}
			wantStatus := statuses[0].ID
			if tt.wantCategory == model.StatusDone {
				wantStatus = statuses[len(statuses)-1].ID
			}
			if got.StatusID != wantStatus {
				t.Errorf("status = %s, want %s", got.StatusID, wantStatus)
			}
			after := env.nullableColumn(t, `SELECT status_id || '|' || position FROM tasks WHERE id = ?`, task.ID)
			if moved := *before != *after; moved != tt.wantMoved {
				t.Errorf("moved = %v (%s -> %s), want %v", moved, *before, *after, tt.wantMoved)
			}
			if tt.done && tt.wantMoved {
				titles := env.taskTitles(t, project.ID, planID, wantStatus)
				if !equalStrings(titles, []string{"earlier", "task"}) {
					t.Errorf("done column = %v, want task appended last", titles)
				}
			}
		})
	}
}

func TestSetDoneUnknownTask(t *testing.T) {
	env := newTaskEnv(t)
	_, err := env.tasks.SetDone(context.Background(), env.session, model.NewID(), true)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("SetDone() error = %v, want ErrNotFound", err)
	}
}
