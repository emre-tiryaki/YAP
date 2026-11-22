package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// TaskActivityRepository reads and writes task events and task relations.
type TaskActivityRepository struct{}

func NewTaskActivityRepository() *TaskActivityRepository {
	return &TaskActivityRepository{}
}

// InsertEvent appends an event to a task's log.
func (*TaskActivityRepository) InsertEvent(ctx context.Context, q database.DBTX, e model.TaskEvent, now string) error {
	details, err := json.Marshal(e.Details)
	if err != nil {
		return fmt.Errorf("encode task event details: %w", err)
	}
	if e.Details == nil {
		details = []byte("{}")
	}
	_, err = q.ExecContext(ctx,
		`INSERT INTO task_events (id, workspace_id, task_id, actor_id, action, details, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WorkspaceID, e.TaskID, e.ActorID, string(e.Action), string(details), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert task event: %w", err)
	}
	return nil
}

// ListEvents returns a task's events, oldest first.
func (*TaskActivityRepository) ListEvents(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.TaskEvent, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, actor_id, action, details, created_at FROM task_events
		 WHERE workspace_id = ? AND task_id = ? AND deleted_at IS NULL
		 ORDER BY created_at, id`,
		workspaceID, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task events: %w", err)
	}
	defer rows.Close()

	events := []model.TaskEvent{}
	for rows.Next() {
		e := model.TaskEvent{WorkspaceID: workspaceID, TaskID: taskID}
		var action, details, createdAt string
		if err := rows.Scan(&e.ID, &e.ActorID, &action, &details, &createdAt); err != nil {
			return nil, fmt.Errorf("scan task event: %w", err)
		}
		e.Action = model.TaskEventAction(action)
		if err := json.Unmarshal([]byte(details), &e.Details); err != nil {
			e.Details = map[string]string{}
		}
		if e.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("parse task event time: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list task events: %w", err)
	}
	return events, nil
}

// StatusTitle returns the title of a status.
func (*TaskActivityRepository) StatusTitle(ctx context.Context, q database.DBTX, workspaceID, statusID string) (string, error) {
	var title string
	err := q.QueryRowContext(ctx,
		`SELECT title FROM statuses WHERE id = ? AND workspace_id = ?`, statusID, workspaceID,
	).Scan(&title)
	if err != nil {
		return "", fmt.Errorf("status title: %w", err)
	}
	return title, nil
}

// LabelNames returns the names of the labels with the ids, by id.
func (*TaskActivityRepository) LabelNames(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]string, error) {
	names := map[string]string{}
	for _, id := range ids {
		var name string
		err := q.QueryRowContext(ctx,
			`SELECT name FROM labels WHERE id = ? AND workspace_id = ?`, id, workspaceID,
		).Scan(&name)
		if err != nil {
			return nil, fmt.Errorf("label name: %w", err)
		}
		names[id] = name
	}
	return names, nil
}

// TaskSummary returns the project and title of a live task, or sql.ErrNoRows.
func (*TaskActivityRepository) TaskSummary(ctx context.Context, q database.DBTX, workspaceID, taskID string) (projectID, title string, err error) {
	err = q.QueryRowContext(ctx,
		`SELECT project_id, title FROM tasks WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		taskID, workspaceID,
	).Scan(&projectID, &title)
	if err != nil {
		return "", "", fmt.Errorf("task summary: %w", err)
	}
	return projectID, title, nil
}

// RelationBetween reports whether a live relation links the two tasks, in
// either direction.
func (*TaskActivityRepository) RelationBetween(ctx context.Context, q database.DBTX, workspaceID, a, b string) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_relations
		 WHERE workspace_id = ? AND removed_at IS NULL
		   AND ((task_id = ? AND related_task_id = ?) OR (task_id = ? AND related_task_id = ?))`,
		workspaceID, a, b, b, a,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("relation between: %w", err)
	}
	return n > 0, nil
}

// InsertRelation stores task blocks/relates/duplicates related, reopening the
// same row when it was removed before.
func (*TaskActivityRepository) InsertRelation(ctx context.Context, q database.DBTX, workspaceID, projectID, taskID, relatedID string, kind model.RelationKind, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_relations (workspace_id, project_id, task_id, related_task_id, kind, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (task_id, related_task_id) DO UPDATE SET
		     kind = excluded.kind, project_id = excluded.project_id,
		     created_at = excluded.created_at, updated_at = excluded.updated_at, removed_at = NULL
		   WHERE task_relations.workspace_id = excluded.workspace_id AND task_relations.removed_at IS NOT NULL`,
		workspaceID, projectID, taskID, relatedID, string(kind), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert task relation: %w", err)
	}
	return nil
}

// RemoveRelation closes the live relation between two tasks, whichever side
// holds it. It reports whether one existed.
func (*TaskActivityRepository) RemoveRelation(ctx context.Context, q database.DBTX, workspaceID, a, b, now string) (bool, error) {
	res, err := q.ExecContext(ctx,
		`UPDATE task_relations SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND removed_at IS NULL
		   AND ((task_id = ? AND related_task_id = ?) OR (task_id = ? AND related_task_id = ?))`,
		now, now, workspaceID, a, b, b, a,
	)
	if err != nil {
		return false, fmt.Errorf("remove task relation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("remove task relation: %w", err)
	}
	return n > 0, nil
}

// RelationKindBetween returns how the two tasks are related, as stored, and
// which of them holds the relation (the id of the task_id side).
func (*TaskActivityRepository) RelationKindBetween(ctx context.Context, q database.DBTX, workspaceID, a, b string) (kind model.RelationKind, holder string, err error) {
	var k string
	err = q.QueryRowContext(ctx,
		`SELECT kind, task_id FROM task_relations
		 WHERE workspace_id = ? AND removed_at IS NULL
		   AND ((task_id = ? AND related_task_id = ?) OR (task_id = ? AND related_task_id = ?))`,
		workspaceID, a, b, b, a,
	).Scan(&k, &holder)
	if err != nil {
		return "", "", fmt.Errorf("relation kind: %w", err)
	}
	return model.RelationKind(k), holder, nil
}

// ListRelations returns the live relations of a task, read from the task's
// side, with the other task's summary. Relations to deleted tasks are left out.
func (*TaskActivityRepository) ListRelations(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.TaskRelation, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT r.kind, r.task_id = ?, o.id, o.title, COALESCE(o.number, 0), o.project_id, o.plan_id, o.completed_at
		 FROM task_relations r
		 JOIN tasks o ON o.workspace_id = r.workspace_id
		             AND o.id = CASE WHEN r.task_id = ? THEN r.related_task_id ELSE r.task_id END
		 WHERE r.workspace_id = ? AND r.removed_at IS NULL AND o.deleted_at IS NULL
		   AND (r.task_id = ? OR r.related_task_id = ?)
		 ORDER BY r.created_at, o.id`,
		taskID, taskID, workspaceID, taskID, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task relations: %w", err)
	}
	defer rows.Close()

	out := []model.TaskRelation{}
	for rows.Next() {
		var rel model.TaskRelation
		var kind string
		var holds bool
		var completedAt sql.NullString
		if err := rows.Scan(&kind, &holds, &rel.TaskID, &rel.Title, &rel.Number, &rel.ProjectID, &rel.PlanID, &completedAt); err != nil {
			return nil, fmt.Errorf("scan task relation: %w", err)
		}
		switch model.RelationKind(kind) {
		case model.RelationBlocks:
			rel.View = model.ViewBlockedBy
			if holds {
				rel.View = model.ViewBlocking
			}
		case model.RelationDuplicates:
			rel.View = model.ViewDuplicatedBy
			if holds {
				rel.View = model.ViewDuplicateOf
			}
		default:
			rel.View = model.ViewRelates
		}
		if completedAt.Valid {
			t, err := time.Parse(time.RFC3339, completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("parse completed_at: %w", err)
			}
			rel.CompletedAt = &t
		}
		out = append(out, rel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list task relations: %w", err)
	}
	return out, nil
}
