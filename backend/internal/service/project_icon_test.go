package service

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeIcon(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"blank", "   ", "", false},
		{"simple emoji", "🚀", "🚀", false},
		{"trimmed", "  📚 ", "📚", false},
		{"variation selector", "❤️", "❤️", false},
		{"skin tone", "👍🏽", "👍🏽", false},
		{"zwj sequence", "👩‍💻", "👩‍💻", false},
		{"flag", "🇹🇷", "🇹🇷", false},
		{"ascii text", "ab", "", true},
		{"emoji with text", "🚀x", "", true},
		{"non-ascii letters", "çğ", "", true},
		{"inner space", "🚀 🚀", "", true},
		{"too long", "🚀🚀🚀🚀🚀🚀🚀🚀🚀", "", true},
		{"invalid utf8", "\xff", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeIcon(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("normalizeIcon(%q) error = %v, want ErrInvalidInput", tt.in, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("normalizeIcon(%q) = %q, %v, want %q", tt.in, got, err, tt.want)
			}
		})
	}
}

func TestProjectIcon(t *testing.T) {
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
			created, err := env.projects.Create(ctx, env.session.WorkspaceID, CreateProjectInput{Name: "Alpha", Icon: tt.create})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if created.Icon != tt.create {
				t.Errorf("Create().Icon = %q, want %q", created.Icon, tt.create)
			}
			if tt.update != nil {
				got, err := env.projects.Update(ctx, env.session.WorkspaceID, created.ID, UpdateProjectInput{Icon: tt.update})
				if err != nil {
					t.Fatalf("Update() error = %v", err)
				}
				if got.Icon != tt.wantIcon || got.Name != "Alpha" {
					t.Errorf("Update() = %+v, want icon %q and name kept", got, tt.wantIcon)
				}
			}

			stored, err := env.projects.Get(ctx, env.session.WorkspaceID, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Icon != tt.wantIcon {
				t.Errorf("Get().Icon = %q, want %q", stored.Icon, tt.wantIcon)
			}
			list, err := env.projects.List(ctx, env.session.WorkspaceID)
			if err != nil {
				t.Fatal(err)
			}
			if len(list) != 1 || list[0].Icon != tt.wantIcon {
				t.Errorf("List() = %+v, want one project with icon %q", list, tt.wantIcon)
			}
			// An empty icon is stored as NULL, like other optional text.
			if tt.wantIcon == "" {
				if v := env.nullableColumn(t, `SELECT icon FROM projects WHERE id = ?`, created.ID); v != nil {
					t.Errorf("stored icon = %q, want NULL", *v)
				}
			}
		})
	}
}

func TestProjectIconInvalid(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	if _, err := env.projects.Create(ctx, env.session.WorkspaceID, CreateProjectInput{Name: "Alpha", Icon: "abc"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Create() error = %v, want ErrInvalidInput", err)
	}
	p := env.project(t, "Beta")
	if _, err := env.projects.Update(ctx, env.session.WorkspaceID, p.ID, UpdateProjectInput{Icon: strPtr("abc")}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Update() error = %v, want ErrInvalidInput", err)
	}
}
