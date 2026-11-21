package service

import (
	"context"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func TestTaskCreatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority model.TaskPriority
		want     model.TaskPriority
		wantErr  error
	}{
		{"defaults to none", "", model.PriorityNone, nil},
		{"urgent", model.PriorityUrgent, model.PriorityUrgent, nil},
		{"low", model.PriorityLow, model.PriorityLow, nil},
		{"unknown", "critical", "", ErrInvalidPriority},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			project := env.project(t, "Backend")

			task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T", Priority: tt.priority})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("error %v does not wrap ErrInvalidInput", err)
				}
				return
			}
			if task.Priority != tt.want {
				t.Errorf("Priority = %q, want %q", task.Priority, tt.want)
			}
			got, err := env.tasks.ListByPlan(ctx, env.session, project.ID, task.PlanID)
			if err != nil || len(got) != 1 || got[0].Priority != tt.want {
				t.Errorf("ListByPlan() = %+v (err %v), want priority %q", got, err, tt.want)
			}
		})
	}
}

func TestTaskUpdatePriority(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for _, priority := range []model.TaskPriority{
		model.PriorityUrgent, model.PriorityHigh, model.PriorityMedium, model.PriorityLow, model.PriorityNone,
	} {
		p := priority
		updated, err := env.tasks.Update(ctx, env.session, task.ID, UpdateTaskInput{Priority: &p})
		if err != nil {
			t.Fatalf("Update(%q) error = %v", priority, err)
		}
		if updated.Priority != priority {
			t.Errorf("Update(%q) returned priority %q", priority, updated.Priority)
		}
		var stored string
		if err := env.db.QueryRow(`SELECT priority FROM tasks WHERE id = ?`, task.ID).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		if stored != string(priority) {
			t.Errorf("stored priority = %q, want %q", stored, priority)
		}
	}
}

func TestTaskUpdateRejectsInvalidPriority(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T", Priority: model.PriorityHigh})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	bad := model.TaskPriority("critical")
	if _, err := env.tasks.Update(ctx, env.session, task.ID, UpdateTaskInput{Priority: &bad}); !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("Update() error = %v, want ErrInvalidPriority", err)
	}
	var stored string
	if err := env.db.QueryRow(`SELECT priority FROM tasks WHERE id = ?`, task.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != string(model.PriorityHigh) {
		t.Errorf("stored priority = %q, want it unchanged (high)", stored)
	}
}

func TestUpdateOtherFieldsKeepsPriority(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Backend")
	task, err := env.tasks.Create(ctx, env.session, project.ID, CreateTaskInput{Title: "T", Priority: model.PriorityUrgent})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	title := "renamed"
	updated, err := env.tasks.Update(ctx, env.session, task.ID, UpdateTaskInput{Title: &title})
	if err != nil || updated.Priority != model.PriorityUrgent {
		t.Errorf("Update() = %+v (err %v), want priority urgent kept", updated, err)
	}
}
