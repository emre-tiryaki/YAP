package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type LabelRepository struct{}

func NewLabelRepository() *LabelRepository {
	return &LabelRepository{}
}

// List returns the live labels of a project in display order.
func (*LabelRepository) List(ctx context.Context, q database.DBTX, workspaceID, projectID string) ([]model.Label, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, name, color, position FROM labels
		 WHERE workspace_id = ? AND project_id = ? AND deleted_at IS NULL
		 ORDER BY position, id`,
		workspaceID, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	defer rows.Close()

	labels := []model.Label{}
	for rows.Next() {
		l := model.Label{WorkspaceID: workspaceID, ProjectID: projectID}
		if err := rows.Scan(&l.ID, &l.Name, &l.Color, &l.Position); err != nil {
			return nil, fmt.Errorf("scan label: %w", err)
		}
		labels = append(labels, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	return labels, nil
}

func (*LabelRepository) Insert(ctx context.Context, q database.DBTX, l model.Label, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO labels (id, workspace_id, project_id, name, color, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.WorkspaceID, l.ProjectID, l.Name, l.Color, l.Position, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert label: %w", err)
	}
	return nil
}

// Get returns a live label.
func (*LabelRepository) Get(ctx context.Context, q database.DBTX, workspaceID, labelID string) (model.Label, error) {
	l := model.Label{ID: labelID, WorkspaceID: workspaceID}
	err := q.QueryRowContext(ctx,
		`SELECT project_id, name, color, position FROM labels
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		labelID, workspaceID,
	).Scan(&l.ProjectID, &l.Name, &l.Color, &l.Position)
	if err != nil {
		return model.Label{}, fmt.Errorf("get label: %w", err)
	}
	return l, nil
}

// LastPosition returns the highest label position of the project, or "".
func (*LabelRepository) LastPosition(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error) {
	var pos string
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), '') FROM labels
		 WHERE workspace_id = ? AND project_id = ? AND deleted_at IS NULL`,
		workspaceID, projectID,
	).Scan(&pos)
	if err != nil {
		return "", fmt.Errorf("last label position: %w", err)
	}
	return pos, nil
}

// NameTaken reports whether another live label of the project has the name.
func (*LabelRepository) NameTaken(ctx context.Context, q database.DBTX, workspaceID, projectID, name, excludeLabelID string) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM labels
		 WHERE workspace_id = ? AND project_id = ? AND name = ? AND id <> ? AND deleted_at IS NULL`,
		workspaceID, projectID, name, excludeLabelID,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("label name taken: %w", err)
	}
	return n > 0, nil
}

func (*LabelRepository) Update(ctx context.Context, q database.DBTX, workspaceID, labelID, name, color, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE labels SET name = ?, color = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		name, color, now, labelID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update label: %w", err)
	}
	return nil
}

// SoftDelete marks the label deleted and removes it from every task, so each
// affected row carries its own deletion record.
func (*LabelRepository) SoftDelete(ctx context.Context, q database.DBTX, workspaceID, labelID, now string) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE task_labels SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND label_id = ? AND removed_at IS NULL`,
		now, now, workspaceID, labelID,
	); err != nil {
		return fmt.Errorf("remove label from tasks: %w", err)
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE labels SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		now, now, labelID, workspaceID,
	); err != nil {
		return fmt.Errorf("delete label: %w", err)
	}
	return nil
}

// placeholders returns "?, ?, ?" for n values.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
