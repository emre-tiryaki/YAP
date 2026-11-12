package repository

import (
	"context"
	"database/sql"
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

// ListUpcoming returns the member's open (not done) tasks: dated ones first by
// due date, then the undated ones.
func (*TaskRepository) ListUpcoming(ctx context.Context, q database.DBTX, workspaceID, membershipID string, limit int) ([]model.Task, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.title, t.description, t.project_id, t.plan_id, t.status_id, t.position,
			p.name, t.due_date, s.category
		 FROM task_assignees ta
		 JOIN tasks t ON t.id = ta.task_id AND t.workspace_id = ta.workspace_id
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE ta.workspace_id = ? AND ta.membership_id = ? AND ta.removed_at IS NULL
		   AND t.deleted_at IS NULL AND p.deleted_at IS NULL AND s.deleted_at IS NULL
		   AND s.category <> ?
		 ORDER BY t.due_date IS NULL, t.due_date, t.id
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
		var description, dueDate sql.NullString
		var category string
		if err := rows.Scan(&t.ID, &t.Title, &description, &t.ProjectID, &t.PlanID, &t.StatusID, &t.Position,
			&t.ProjectName, &dueDate, &category); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Description = description.String
		t.StatusCategory = model.StatusCategory(category)
		if dueDate.Valid {
			parsed, err := time.Parse(time.RFC3339, dueDate.String)
			if err != nil {
				return nil, fmt.Errorf("parse due date: %w", err)
			}
			t.DueDate = &parsed
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list upcoming tasks: %w", err)
	}
	return tasks, nil
}

func (*TaskRepository) ListByPlan(ctx context.Context, q database.DBTX, workspaceID, planID string) ([]model.Task, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.project_id, t.status_id, t.created_by, t.title, t.description,
			t.position, t.due_date, s.category
		 FROM tasks t
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE t.workspace_id = ? AND t.plan_id = ?
		   AND t.deleted_at IS NULL AND s.deleted_at IS NULL
		 ORDER BY s.position, s.id, t.position, t.id`,
		workspaceID, planID,
	)
	if err != nil {
		return nil, fmt.Errorf("list plan tasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t := model.Task{WorkspaceID: workspaceID, PlanID: planID}
		var description, dueDate sql.NullString
		var category string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.StatusID, &t.CreatedBy, &t.Title, &description,
			&t.Position, &dueDate, &category); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Description = description.String
		t.StatusCategory = model.StatusCategory(category)
		if dueDate.Valid {
			parsed, err := time.Parse(time.RFC3339, dueDate.String)
			if err != nil {
				return nil, fmt.Errorf("parse due date: %w", err)
			}
			t.DueDate = &parsed
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list plan tasks: %w", err)
	}
	return tasks, nil
}

// Placement returns the plan and status of a non-deleted task.
func (*TaskRepository) Placement(ctx context.Context, q database.DBTX, workspaceID, taskID string) (planID, statusID string, err error) {
	err = q.QueryRowContext(ctx,
		`SELECT plan_id, status_id FROM tasks
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		taskID, workspaceID,
	).Scan(&planID, &statusID)
	if err != nil {
		return "", "", fmt.Errorf("task placement: %w", err)
	}
	return planID, statusID, nil
}

// StatusCategory returns the category of a non-deleted status of the plan.
func (*TaskRepository) StatusCategory(ctx context.Context, q database.DBTX, workspaceID, planID, statusID string) (model.StatusCategory, error) {
	var category string
	err := q.QueryRowContext(ctx,
		`SELECT category FROM statuses
		 WHERE id = ? AND workspace_id = ? AND plan_id = ? AND deleted_at IS NULL`,
		statusID, workspaceID, planID,
	).Scan(&category)
	if err != nil {
		return "", fmt.Errorf("status category: %w", err)
	}
	return model.StatusCategory(category), nil
}

// PositionsInStatus lists the ordered positions of the non-deleted tasks in a
// status, leaving out excludeTaskID (the task being moved).
func (*TaskRepository) PositionsInStatus(ctx context.Context, q database.DBTX, workspaceID, planID, statusID, excludeTaskID string) ([]string, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT position FROM tasks
		 WHERE workspace_id = ? AND plan_id = ? AND status_id = ?
		   AND deleted_at IS NULL AND id <> ?
		 ORDER BY position, id`,
		workspaceID, planID, statusID, excludeTaskID,
	)
	if err != nil {
		return nil, fmt.Errorf("positions in status: %w", err)
	}
	defer rows.Close()

	positions := []string{}
	for rows.Next() {
		var pos string
		if err := rows.Scan(&pos); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, pos)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("positions in status: %w", err)
	}
	return positions, nil
}

func (*TaskRepository) UpdatePlacement(ctx context.Context, q database.DBTX, workspaceID, taskID, statusID, pos, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE tasks SET status_id = ?, position = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		statusID, pos, now, taskID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update task placement: %w", err)
	}
	return nil
}

// SoftDeleteTree marks the task and all of its non-deleted descendants as
// deleted and closes their open assignments.
func (*TaskRepository) SoftDeleteTree(ctx context.Context, q database.DBTX, workspaceID, taskID, now string) error {
	const tree = `WITH RECURSIVE tree(id) AS (
			SELECT id FROM tasks WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL
			UNION
			SELECT t.id FROM tasks t JOIN tree ON t.parent_id = tree.id
			 WHERE t.workspace_id = ? AND t.deleted_at IS NULL
		)`
	_, err := q.ExecContext(ctx,
		tree+` UPDATE task_assignees SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND removed_at IS NULL AND task_id IN (SELECT id FROM tree)`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("close task assignees: %w", err)
	}
	_, err = q.ExecContext(ctx,
		tree+` UPDATE tasks SET deleted_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND id IN (SELECT id FROM tree)`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("delete task tree: %w", err)
	}
	return nil
}

// Get returns a non-deleted task with its project name and status category.
func (*TaskRepository) Get(ctx context.Context, q database.DBTX, workspaceID, taskID string) (model.Task, error) {
	t := model.Task{ID: taskID, WorkspaceID: workspaceID}
	var description, dueDate sql.NullString
	var category string
	err := q.QueryRowContext(ctx,
		`SELECT t.project_id, p.name, t.plan_id, t.status_id, t.created_by, t.title,
			t.description, t.position, t.due_date, s.category
		 FROM tasks t
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE t.id = ? AND t.workspace_id = ? AND t.deleted_at IS NULL`,
		taskID, workspaceID,
	).Scan(&t.ProjectID, &t.ProjectName, &t.PlanID, &t.StatusID, &t.CreatedBy, &t.Title,
		&description, &t.Position, &dueDate, &category)
	if err != nil {
		return model.Task{}, fmt.Errorf("get task: %w", err)
	}
	t.Description = description.String
	t.StatusCategory = model.StatusCategory(category)
	if dueDate.Valid {
		parsed, err := time.Parse(time.RFC3339, dueDate.String)
		if err != nil {
			return model.Task{}, fmt.Errorf("parse due date: %w", err)
		}
		t.DueDate = &parsed
	}
	return t, nil
}

// Update overwrites the title, description and due date of a non-deleted task.
func (*TaskRepository) Update(ctx context.Context, q database.DBTX, workspaceID, taskID, title, description string, dueDate *time.Time, now string) error {
	var due any
	if dueDate != nil {
		due = dueDate.UTC().Format(time.RFC3339)
	}
	_, err := q.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, due_date = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		title, nullIfEmpty(description), due, now, taskID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}
