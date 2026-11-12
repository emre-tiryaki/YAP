package service

import (
	"context"
	"errors"
	"testing"
)

func TestPlanIcon(t *testing.T) {
	tests := []struct {
		name     string
		create   string
		update   *string
		wantIcon string
	}{
		{"create with icon", "🚀", nil, "🚀"},
		{"create without icon", "", nil, ""},
		{"set icon", "", strPtr("📚"), "📚"},
		{"change icon", "🚀", strPtr("📚"), "📚"},
		{"clear icon", "🚀", strPtr(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newTaskEnv(t)
			project := env.project(t, "Alpha")
			created, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Sprint", Icon: tt.create})
			if err != nil {
				t.Fatalf("CreatePlan() error = %v", err)
			}
			if created.Icon != tt.create {
				t.Errorf("CreatePlan().Icon = %q, want %q", created.Icon, tt.create)
			}
			if tt.update != nil {
				got, err := env.projects.UpdatePlan(ctx, env.session.WorkspaceID, project.ID, created.ID, UpdatePlanInput{Icon: tt.update})
				if err != nil {
					t.Fatalf("UpdatePlan() error = %v", err)
				}
				if got.Icon != tt.wantIcon || got.Name != "Sprint" {
					t.Errorf("UpdatePlan() = %+v, want icon %q and name kept", got, tt.wantIcon)
				}
			}

			stored, err := env.projects.GetPlan(ctx, env.session.WorkspaceID, project.ID, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Icon != tt.wantIcon {
				t.Errorf("GetPlan().Icon = %q, want %q", stored.Icon, tt.wantIcon)
			}
			plans, err := env.projects.Plans(ctx, env.session.WorkspaceID, project.ID)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, p := range plans {
				if p.ID == created.ID {
					found = true
					if p.Icon != tt.wantIcon {
						t.Errorf("Plans() icon = %q, want %q", p.Icon, tt.wantIcon)
					}
				}
			}
			if !found {
				t.Errorf("Plans() = %+v, missing %s", plans, created.ID)
			}
			// An empty icon is stored as NULL, like other optional text.
			if tt.wantIcon == "" {
				if v := env.nullableColumn(t, `SELECT icon FROM plans WHERE id = ?`, created.ID); v != nil {
					t.Errorf("stored icon = %q, want NULL", *v)
				}
			}
		})
	}
}

func TestPlanIconInvalid(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	project := env.project(t, "Alpha")
	if _, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Sprint", Icon: "abc"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("CreatePlan() error = %v, want ErrInvalidInput", err)
	}
	planID := env.planOf(t, project.ID)
	if _, err := env.projects.UpdatePlan(ctx, env.session.WorkspaceID, project.ID, planID, UpdatePlanInput{Icon: strPtr("abc")}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("UpdatePlan() error = %v, want ErrInvalidInput", err)
	}
}
