package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestSessionBootstrapCreatesPersonalWorkspace(t *testing.T) {
	ctx := context.Background()
	db, session := newTestDB(t)

	if session.DisplayName != "tester" {
		t.Errorf("DisplayName = %q, want tester", session.DisplayName)
	}
	if session.WorkspaceName != personalWorkspaceName {
		t.Errorf("WorkspaceName = %q, want %q", session.WorkspaceName, personalWorkspaceName)
	}
	for name, id := range map[string]string{
		"UserID": session.UserID, "WorkspaceID": session.WorkspaceID, "MembershipID": session.MembershipID,
	} {
		if len(id) != 36 {
			t.Errorf("%s = %q, want a 36 character id", name, id)
		}
	}

	counts := map[string]int{"users": 1, "workspaces": 1, "memberships": 1}
	for table, want := range counts {
		var n int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("%s rows = %d, want %d", table, n, want)
		}
	}

	var kind, role string
	err := db.QueryRowContext(ctx,
		`SELECT w.kind, m.role FROM workspaces w JOIN memberships m ON m.workspace_id = w.id WHERE w.id = ?`,
		session.WorkspaceID,
	).Scan(&kind, &role)
	if err != nil {
		t.Fatal(err)
	}
	if kind != model.KindPersonal || role != model.RoleOwner {
		t.Errorf("workspace kind/role = %q/%q, want personal/owner", kind, role)
	}
}

func TestSessionBootstrapExistingUserWithoutWorkspace(t *testing.T) {
	ctx := context.Background()
	db, err := database.New(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	repo := repository.NewSessionRepository()
	user := model.User{ID: model.NewID(), DisplayName: "original"}
	if err := repo.InserUser(ctx, db, user, nowUTC()); err != nil {
		t.Fatal(err)
	}

	session, err := NewSessionService(db, repo).Bootstrap(ctx, "ignored")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if session.UserID != user.ID || session.DisplayName != "original" {
		t.Errorf("session = %+v, want existing user to be reused", session)
	}
	if session.WorkspaceID == "" || session.MembershipID == "" {
		t.Errorf("session = %+v, want workspace and membership to be created", session)
	}

	var users int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Errorf("users = %d, want 1", users)
	}
}

func TestSessionBootstrapFailsOnClosedDatabase(t *testing.T) {
	db, _ := newTestDB(t)
	db.Close()

	_, err := NewSessionService(db, repository.NewSessionRepository()).Bootstrap(context.Background(), "x")
	if err == nil {
		t.Fatal("Bootstrap() expected an error on a closed database")
	}
}