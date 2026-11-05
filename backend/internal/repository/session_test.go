package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func TestSessionFirstUser(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewSessionRepository()

	if _, err := repo.FirstUser(ctx, db); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("FirstUser() on empty db error = %v, want sql.ErrNoRows", err)
	}

	first := model.User{ID: model.NewID(), DisplayName: "first"}
	if err := repo.InserUser(ctx, db, first, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	second := model.User{ID: model.NewID(), DisplayName: "second"}
	if err := repo.InserUser(ctx, db, second, "2026-01-02T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FirstUser(ctx, db)
	if err != nil {
		t.Fatalf("FirstUser() error = %v", err)
	}
	if got != first {
		t.Errorf("FirstUser() = %+v, want %+v", got, first)
	}
}

func TestSessionInsertUserDuplicateID(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewSessionRepository()

	u := model.User{ID: model.NewID(), DisplayName: "dup"}
	if err := repo.InserUser(ctx, db, u, baseNow); err != nil {
		t.Fatal(err)
	}
	if err := repo.InserUser(ctx, db, u, baseNow); err == nil {
		t.Fatal("InserUser() expected an error for a duplicate id")
	}
}

func TestSessionInsertWorkspaceRejectsInvalidKind(t *testing.T) {
	db := newTestDB(t)
	err := NewSessionRepository().InsertWorkspace(context.Background(), db, model.NewID(), "bogus", "x", baseNow)
	if err == nil {
		t.Fatal("InsertWorkspace() expected an error for an invalid kind")
	}
}

func TestSessionInsertMembershipRules(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewSessionRepository()
	s := seedWorkspace(t, db, "owner")

	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"duplicate membership", "member", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.InsertMembership(ctx, db, model.NewID(), s.WorkspaceID, s.UserID, tt.role, baseNow)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertMembership() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	other := model.User{ID: model.NewID(), DisplayName: "other"}
	if err := repo.InserUser(ctx, db, other, baseNow); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertMembership(ctx, db, model.NewID(), s.WorkspaceID, other.ID, "superuser", baseNow); err == nil {
		t.Error("InsertMembership() expected an error for an invalid role")
	}
}

func TestSessionPersonalSession(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewSessionRepository()

	user := model.User{ID: model.NewID(), DisplayName: "Ada"}
	if err := repo.InserUser(ctx, db, user, baseNow); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.PersonalSession(ctx, db, user); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("PersonalSession() without workspace error = %v, want sql.ErrNoRows", err)
	}

	s := seedWorkspaceFor(t, db, user.ID, "Personal")
	got, err := repo.PersonalSession(ctx, db, user)
	if err != nil {
		t.Fatalf("PersonalSession() error = %v", err)
	}
	want := model.Session{
		UserID:        user.ID,
		DisplayName:   "Ada",
		WorkspaceID:   s.WorkspaceID,
		WorkspaceName: "Personal",
		MembershipID:  s.MembershipID,
	}
	if got != want {
		t.Errorf("PersonalSession() = %+v, want %+v", got, want)
	}
}

func TestSessionPersonalSessionIgnoresInactiveWorkspaces(t *testing.T) {
	ctx := context.Background()
	repo := NewSessionRepository()

	tests := []struct {
		name   string
		mutate func(t *testing.T, dbExec func(string, ...any), s workspaceSeed)
	}{
		{"left membership", func(t *testing.T, dbExec func(string, ...any), s workspaceSeed) {
			dbExec(`UPDATE memberships SET left_at = ? WHERE id = ?`, baseNow, s.MembershipID)
		}},
		{"deleted workspace", func(t *testing.T, dbExec func(string, ...any), s workspaceSeed) {
			dbExec(`UPDATE workspaces SET deleted_at = ? WHERE id = ?`, baseNow, s.WorkspaceID)
		}},
		{"team workspace", func(t *testing.T, dbExec func(string, ...any), s workspaceSeed) {
			dbExec(`UPDATE workspaces SET kind = 'team' WHERE id = ?`, s.WorkspaceID)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			s := seedWorkspace(t, db, "Personal")
			tt.mutate(t, func(q string, a ...any) { mustExec(t, db, q, a...) }, s)

			_, err := repo.PersonalSession(ctx, db, model.User{ID: s.UserID, DisplayName: "x"})
			if !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("PersonalSession() error = %v, want sql.ErrNoRows", err)
			}
		})
	}
}