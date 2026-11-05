package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type ProjectRepository struct{}

func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{}
}

func (*ProjectRepository) List(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Project, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT p.id, p.name, p.description,
			(SELECT COUNT(*) FROM tasks t
				WHERE t.project_id = p.id
				AND t.workspace_id = p.workspace_id
				AND t.deleted_at IS NULL)
		 FROM projects p
		 WHERE p.workspace_id = ? AND p.deleted_at IS NULL AND p.archived_at IS NULL
		 ORDER BY p.created_at DESC, p.id DESC`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list projectS: %w", err)
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		p := model.Project{WorkspaceID: workspaceID}
		var description sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &description, &p.TaskCount); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.Description = description.String
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (*ProjectRepository) Insert(ctx context.Context, q database.DBTX, p model.Project, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO projects (id, workspace_id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.WorkspaceID, p.Name, nullIfEmpty(p.Description), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (*ProjectRepository) InsertStatus(ctx context.Context, q database.DBTX, s model.Status, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO statuses (id, workspace_id, project_id, title, color, category, position, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.WorkspaceID, s.ProjectID, s.Title, s.Color, string(s.Category), s.Position, now,
	)
	if err != nil {
		return fmt.Errorf("insert status: %w", err)
	}
	return nil
}

func (*ProjectRepository) Name(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error) {
	var name string
	err := q.QueryRowContext(ctx,
		`SELECT name FROM projects WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		projectID, workspaceID,
	).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("project name: %w", err)
	}
	return name, nil
}
