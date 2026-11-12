package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func ptr(s string) *string { return &s }

func TestWorkspaceUpdate(t *testing.T) {
	tests := []struct {
		name          string
		in            UpdateProfileInput
		wantErr       error
		wantUser      string
		wantWorkspace string
	}{
		{name: "user name only", in: UpdateProfileInput{DisplayName: ptr("  Ada  ")}, wantUser: "Ada", wantWorkspace: "Personal"},
		{name: "workspace name only", in: UpdateProfileInput{WorkspaceName: ptr(" Home ")}, wantUser: "tester", wantWorkspace: "Home"},
		{name: "both", in: UpdateProfileInput{DisplayName: ptr("Ada"), WorkspaceName: ptr("Home")}, wantUser: "Ada", wantWorkspace: "Home"},
		{name: "nothing", in: UpdateProfileInput{}, wantErr: ErrInvalidInput},
		{name: "blank user name", in: UpdateProfileInput{DisplayName: ptr("   ")}, wantErr: ErrInvalidInput},
		{name: "blank workspace name", in: UpdateProfileInput{WorkspaceName: ptr("")}, wantErr: ErrInvalidInput},
		{name: "too long", in: UpdateProfileInput{DisplayName: ptr(strings.Repeat("ş", maxProfileNameRunes+1))}, wantErr: ErrInvalidInput},
		{name: "max length", in: UpdateProfileInput{DisplayName: ptr(strings.Repeat("ş", maxProfileNameRunes))}, wantUser: strings.Repeat("ş", maxProfileNameRunes), wantWorkspace: "Personal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			db, session := newTestDB(t)
			svc := NewWorkspaceService(db, repository.NewWorkspaceRepository())

			got, err := svc.Update(ctx, session, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
				}
				// A rejected update leaves both names untouched.
				p, err := svc.Get(ctx, session)
				if err != nil {
					t.Fatalf("Get() error = %v", err)
				}
				if p.DisplayName != "tester" || p.WorkspaceName != "Personal" {
					t.Errorf("after rejected update profile = %+v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if got.DisplayName != tt.wantUser || got.WorkspaceName != tt.wantWorkspace || got.WorkspaceID != session.WorkspaceID {
				t.Errorf("Update() = %+v", got)
			}
			reloaded, err := svc.Get(ctx, session)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if reloaded != got {
				t.Errorf("Get() = %+v, want %+v", reloaded, got)
			}
		})
	}
}

func TestWorkspaceRenameSurvivesBootstrap(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)
	svc := NewWorkspaceService(db, repository.NewWorkspaceRepository())
	if _, err := svc.Update(ctx, session, UpdateProfileInput{DisplayName: ptr("Ada")}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	// Bootstrap must not overwrite a renamed user with the configured name.
	again, err := NewSessionService(db, repository.NewSessionRepository()).Bootstrap(ctx, "tester")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if again.DisplayName != "Ada" {
		t.Errorf("DisplayName after re-bootstrap = %q, want Ada", again.DisplayName)
	}
}
