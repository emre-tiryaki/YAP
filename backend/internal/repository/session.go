package repository

import (
	"context"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type SessionRepository struct{}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{}
}

func (*SessionRepository) FirstUser(ctx context.Context, q database.DBTX) (model.User, error) {
	var u model.User
	err := q.QueryRowContext(ctx,
		`SELECT id, display_name FROM users ORDER BY created_at LIMIT 1`,
	).Scan(&u.ID, &u.DisplayName)
	if err != nil {
		return model.User{}, fmt.Errorf("first user: %w", err)
	}
	return u, nil
}

func (*SessionRepository) InserUser(ctx context.Context, q database.DBTX, u model.User, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO users (id, display_name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		u.ID, u.DisplayName, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (*SessionRepository) InsertWorkspace(ctx context.Context, q database.DBTX, id, kind, name, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO workspaces (id, kind, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, kind, name, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert workspace: %w", err)
	}
	return nil
}

func (*SessionRepository) InsertMembership(ctx context.Context, q database.DBTX, id, workspaceID, userID, role, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, workspaceID, userID, role, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert membership: %w", err)
	}
	return nil
}

func (*SessionRepository) PersonalSession(ctx context.Context, q database.DBTX, u model.User) (model.Session, error) {
	s := model.Session{UserID: u.ID, DisplayName: u.DisplayName}
	err := q.QueryRowContext(ctx,
		`SELECT w.id, w.name, m.id
		 FROM workspaces w
		 JOIN memberships m ON m.workspace_id = w.id AND m.user_id = ? AND m.left_at IS NULL
		 WHERE w.kind = ? AND w.deleted_at IS NULL
		 LIMIT 1`,
		u.ID, model.KindPersonal,
	).Scan(&s.WorkspaceID, &s.WorkspaceName, &s.MembershipID)
	if err != nil {
		return model.Session{}, fmt.Errorf("personal session: %w", err)
	}
	return s, nil
}
