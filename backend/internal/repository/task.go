package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type TaskRepository struct{}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (*TaskRepository) FirstStatusID(ctx context.Context, q database.DBTX, workspaceID, planID string, category model.StatusCategory) (string, error) {
	var id string
	err := q.QueryRowContext(ctx,
		`SELECT id FROM statuses
		 WHERE workspace_id = ? AND plan_id = ? AND category = ? AND deleted_at IS NULL
		 ORDER BY position
		 LIMIT 1`,
		workspaceID, planID, string(category),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("first status: %w", err)
	}
	return id, nil
}

func (*TaskRepository) LastPosition(ctx context.Context, q database.DBTX, workspaceID, planID, statusID string) (string, error) {
	var pos string
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), '') FROM tasks
		 WHERE workspace_id = ? AND plan_id = ? AND status_id = ?`,
		workspaceID, planID, statusID,
	).Scan(&pos)
	if err != nil {
		return "", fmt.Errorf("last position: %w", err)
	}
	return pos, nil
}

func (*TaskRepository) Insert(ctx context.Context, q database.DBTX, t model.Task, now string) error {
	var dueDate any
	if t.DueDate != nil {
		dueDate = t.DueDate.UTC().Format(time.RFC3339)
	}

	_, err := q.ExecContext(ctx,
		`INSERT INTO tasks
			(id, workspace_id, project_id, plan_id, status_id, created_by, title, description, position, due_date, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.WorkspaceID, t.ProjectID, t.PlanID, t.StatusID, t.CreatedBy, t.Title,
		nullIfEmpty(t.Description), t.Position, dueDate, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (*TaskRepository) InsertAssignee(ctx context.Context, q database.DBTX, workspaceID, taskID, membershipID, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_assignees (workspace_id, task_id, membership_id, assigned_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		workspaceID, taskID, membershipID, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert task assignee: %w", err)
	}
	return nil
}

func (*TaskRepository) ListUpcoming(ctx context.Context, q database.DBTX, workspaceID, membershipID string, limit int) ([]model.Task, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.title, t.project_id, t.plan_id, p.name, t.due_date, s.category
		 FROM task_assignees ta
		 JOIN tasks t ON t.id = ta.task_id AND t.workspace_id = ta.workspace_id
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE ta.workspace_id = ? AND ta.membership_id = ? AND ta.removed_at IS NULL
		   AND t.deleted_at IS NULL AND p.deleted_at IS NULL
		   AND t.due_date IS NOT NULL AND s.category <> ?
		 ORDER BY t.due_date, t.id
		 LIMIT ?`,
		workspaceID, membershipID, string(model.StatusDone), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list upcoming tasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t := model.Task{WorkspaceID: workspaceID}
		var dueDate, category string
		if err := rows.Scan(&t.ID, &t.Title, &t.ProjectID, &t.PlanID, &t.ProjectName, &dueDate, &category); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339, dueDate)
		if err != nil {
			return nil, fmt.Errorf("parse due date: %w", err)
		}
		t.DueDate = &parsed
		t.StatusCategory = model.StatusCategory(category)
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list upcoming tasks: %w", err)
	}
	return tasks, nil
}