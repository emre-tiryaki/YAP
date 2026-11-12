package repository

import (
	"context"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type WorkspaceRepository struct{}

func NewWorkspaceRepository() *WorkspaceRepository {
	return &WorkspaceRepository{}
}

// Profile loads the current display name of the user and the name of the
// workspace, both of which can change after the session was bootstrapped.
func (*WorkspaceRepository) Profile(ctx context.Context, q database.DBTX, userID, workspaceID string) (model.Profile, error) {
	p := model.Profile{WorkspaceID: workspaceID}
	err := q.QueryRowContext(ctx,
		`SELECT u.display_name, w.name
		 FROM users u, workspaces w
		 WHERE u.id = ? AND w.id = ? AND w.deleted_at IS NULL`,
		userID, workspaceID,
	).Scan(&p.DisplayName, &p.WorkspaceName)
	if err != nil {
		return model.Profile{}, fmt.Errorf("profile: %w", err)
	}
	return p, nil
}

func (*WorkspaceRepository) UpdateDisplayName(ctx context.Context, q database.DBTX, userID, name, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE users SET display_name = ?, updated_at = ? WHERE id = ?`,
		name, now, userID,
	)
	if err != nil {
		return fmt.Errorf("update display name: %w", err)
	}
	return nil
}

func (*WorkspaceRepository) UpdateName(ctx context.Context, q database.DBTX, workspaceID, name, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE workspaces SET name = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		name, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update workspace name: %w", err)
	}
	return nil
}
