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

// notNestedSubtask keeps a task `t` unless it is a subtask living in the same
// plan as its parent; those are shown inside the parent instead of on the
// board. A subtask in another plan (a team's share of the work) stays visible.
const notNestedSubtask = `NOT EXISTS (
	SELECT 1 FROM tasks parent
	 WHERE parent.id = t.parent_id AND parent.workspace_id = t.workspace_id
	   AND parent.plan_id = t.plan_id AND parent.deleted_at IS NULL)`

// taskCounts selects the direct subtask, finished subtask and comment counts
// of a task `t`.
// taskBlocked is true while a live task that is not completed blocks the task.
const taskBlocked = `EXISTS (SELECT 1 FROM task_relations r
	JOIN tasks b ON b.id = r.task_id AND b.workspace_id = r.workspace_id
	WHERE r.related_task_id = t.id AND r.workspace_id = t.workspace_id
	  AND r.kind = 'blocks' AND r.removed_at IS NULL
	  AND b.deleted_at IS NULL AND b.completed_at IS NULL)`

const taskCounts = `(SELECT COUNT(*) FROM tasks c
	 JOIN statuses cs ON cs.id = c.status_id AND cs.plan_id = c.plan_id
	 WHERE c.parent_id = t.id AND c.workspace_id = t.workspace_id
	   AND c.deleted_at IS NULL AND cs.deleted_at IS NULL),
	(SELECT COUNT(*) FROM tasks c
	 JOIN statuses cs ON cs.id = c.status_id AND cs.plan_id = c.plan_id
	 WHERE c.parent_id = t.id AND c.workspace_id = t.workspace_id
	   AND c.deleted_at IS NULL AND cs.deleted_at IS NULL AND c.completed_at IS NOT NULL),
	(SELECT COUNT(*) FROM task_comments tc
	 WHERE tc.task_id = t.id AND tc.workspace_id = t.workspace_id AND tc.deleted_at IS NULL)`

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
	dueDate := formatTime(t.DueDate)

	_, err := q.ExecContext(ctx,
		`INSERT INTO tasks
			(id, workspace_id, project_id, plan_id, status_id, parent_id, created_by, title, description, position, due_date, start_date, priority, number, completed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.WorkspaceID, t.ProjectID, t.PlanID, t.StatusID, nullIfEmpty(t.ParentID), t.CreatedBy, t.Title,
		nullIfEmpty(t.Description), t.Position, dueDate, formatTime(t.StartDate), priorityOrNone(t.Priority), nullIfZero(t.Number), formatTime(t.CompletedAt), now, now,
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

// ListUpcoming returns the member's open (not completed) tasks: dated ones first by
// due date, then the undated ones.
func (*TaskRepository) ListUpcoming(ctx context.Context, q database.DBTX, workspaceID, membershipID string, limit int) ([]model.Task, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.title, t.description, t.project_id, t.plan_id, t.status_id, t.position,
			p.name, t.due_date, s.category, t.completed_at, t.priority, COALESCE(t.number, 0),
			t.created_at, t.updated_at, t.start_date, `+taskBlocked+`
		 FROM task_assignees ta
		 JOIN tasks t ON t.id = ta.task_id AND t.workspace_id = ta.workspace_id
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE ta.workspace_id = ? AND ta.membership_id = ? AND ta.removed_at IS NULL
		   AND t.deleted_at IS NULL AND p.deleted_at IS NULL AND s.deleted_at IS NULL
		   AND t.completed_at IS NULL
		   AND `+notNestedSubtask+`
		 ORDER BY t.due_date IS NULL, t.due_date, t.id
		 LIMIT ?`,
		workspaceID, membershipID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list upcoming tasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t := model.Task{WorkspaceID: workspaceID}
		var description, dueDate, completedAt, startDate sql.NullString
		var category, priority, createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.Title, &description, &t.ProjectID, &t.PlanID, &t.StatusID, &t.Position,
			&t.ProjectName, &dueDate, &category, &completedAt, &priority, &t.Number, &createdAt, &updatedAt, &startDate, &t.Blocked); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Description = description.String
		t.StatusCategory = model.StatusCategory(category)
		t.Priority = model.TaskPriority(priority)
		if err := scanTaskTimes(&t, dueDate, completedAt); err != nil {
			return nil, err
		}
		if err := scanTaskStamps(&t, createdAt, updatedAt, startDate); err != nil {
			return nil, err
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
			t.position, t.due_date, s.category, t.completed_at, t.priority, COALESCE(t.number, 0), t.created_at, t.updated_at, t.start_date, `+taskBlocked+`, `+taskCounts+`
		 FROM tasks t
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE t.workspace_id = ? AND t.plan_id = ?
		   AND t.deleted_at IS NULL AND s.deleted_at IS NULL
		   AND `+notNestedSubtask+`
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
		var description, dueDate, completedAt, startDate sql.NullString
		var category, priority, createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.StatusID, &t.CreatedBy, &t.Title, &description,
			&t.Position, &dueDate, &category, &completedAt, &priority, &t.Number, &createdAt, &updatedAt, &startDate, &t.Blocked, &t.SubtaskCount, &t.DoneSubtaskCount, &t.CommentCount); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Description = description.String
		t.StatusCategory = model.StatusCategory(category)
		t.Priority = model.TaskPriority(priority)
		if err := scanTaskTimes(&t, dueDate, completedAt); err != nil {
			return nil, err
		}
		if err := scanTaskStamps(&t, createdAt, updatedAt, startDate); err != nil {
			return nil, err
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
// deleted, together with their comments, and closes their open assignments.
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
		tree+` UPDATE task_resources SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND removed_at IS NULL AND task_id IN (SELECT id FROM tree)`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("close task resource links: %w", err)
	}
	_, err = q.ExecContext(ctx,
		tree+` UPDATE task_labels SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND removed_at IS NULL AND task_id IN (SELECT id FROM tree)`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("close task labels: %w", err)
	}
	_, err = q.ExecContext(ctx,
		tree+` UPDATE task_comments SET deleted_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND deleted_at IS NULL AND task_id IN (SELECT id FROM tree)`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("delete task comments: %w", err)
	}
	_, err = q.ExecContext(ctx,
		tree+` UPDATE task_events SET deleted_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND deleted_at IS NULL AND task_id IN (SELECT id FROM tree)`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("delete task events: %w", err)
	}
	_, err = q.ExecContext(ctx,
		tree+` UPDATE task_relations SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND removed_at IS NULL
		   AND (task_id IN (SELECT id FROM tree) OR related_task_id IN (SELECT id FROM tree))`,
		taskID, workspaceID, workspaceID, now, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("close task relations: %w", err)
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
	var description, dueDate, completedAt, parentID, startDate sql.NullString
	var category, priority, createdAt, updatedAt string
	err := q.QueryRowContext(ctx,
		`SELECT t.project_id, p.name, t.plan_id, t.status_id, t.parent_id, t.created_by, t.title,
			t.description, t.position, t.due_date, s.category, t.completed_at, t.priority, COALESCE(t.number, 0), t.created_at, t.updated_at, t.start_date, `+taskBlocked+`, `+taskCounts+`
		 FROM tasks t
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE t.id = ? AND t.workspace_id = ? AND t.deleted_at IS NULL`,
		taskID, workspaceID,
	).Scan(&t.ProjectID, &t.ProjectName, &t.PlanID, &t.StatusID, &parentID, &t.CreatedBy, &t.Title,
		&description, &t.Position, &dueDate, &category, &completedAt, &priority, &t.Number, &createdAt, &updatedAt, &startDate, &t.Blocked, &t.SubtaskCount, &t.DoneSubtaskCount, &t.CommentCount)
	if err != nil {
		return model.Task{}, fmt.Errorf("get task: %w", err)
	}
	t.ParentID = parentID.String
	t.Description = description.String
	t.StatusCategory = model.StatusCategory(category)
	t.Priority = model.TaskPriority(priority)
	if err := scanTaskTimes(&t, dueDate, completedAt); err != nil {
		return model.Task{}, err
	}
	if err := scanTaskStamps(&t, createdAt, updatedAt, startDate); err != nil {
		return model.Task{}, err
	}
	return t, nil
}

// Update overwrites the title, description, due date and priority of a
// non-deleted task.
func (*TaskRepository) Update(ctx context.Context, q database.DBTX, workspaceID, taskID, title, description string, dueDate, startDate *time.Time, priority model.TaskPriority, now string) error {
	due := formatTime(dueDate)
	_, err := q.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, due_date = ?, start_date = ?, priority = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		title, nullIfEmpty(description), due, formatTime(startDate), priorityOrNone(priority), now, taskID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

// ListSubtasks returns the non-deleted direct subtasks of a task, oldest first.
func (*TaskRepository) ListSubtasks(ctx context.Context, q database.DBTX, workspaceID, parentID string) ([]model.Task, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.project_id, t.plan_id, t.status_id, t.created_by, t.title, t.description,
			t.position, t.due_date, s.category, t.completed_at, COALESCE(t.number, 0), t.priority
		 FROM tasks t
		 JOIN statuses s ON s.id = t.status_id AND s.plan_id = t.plan_id
		 WHERE t.workspace_id = ? AND t.parent_id = ?
		   AND t.deleted_at IS NULL AND s.deleted_at IS NULL
		 ORDER BY t.created_at, t.id`,
		workspaceID, parentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list subtasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t := model.Task{WorkspaceID: workspaceID, ParentID: parentID}
		var description, dueDate, completedAt sql.NullString
		var category, priority string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.PlanID, &t.StatusID, &t.CreatedBy, &t.Title, &description,
			&t.Position, &dueDate, &category, &completedAt, &t.Number, &priority); err != nil {
			return nil, fmt.Errorf("scan subtask: %w", err)
		}
		t.Priority = model.TaskPriority(priority)
		t.Description = description.String
		t.StatusCategory = model.StatusCategory(category)
		if err := scanTaskTimes(&t, dueDate, completedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list subtasks: %w", err)
	}
	return tasks, nil
}

// SetCompletedAt marks a non-deleted task finished at completedAt, or reopens
// it when completedAt is nil. Its status and position are left alone.
func (*TaskRepository) SetCompletedAt(ctx context.Context, q database.DBTX, workspaceID, taskID string, completedAt *time.Time, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE tasks SET completed_at = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		formatTime(completedAt), now, taskID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("set task completed_at: %w", err)
	}
	return nil
}

// ReserveNumber takes the next task number of the project and advances the
// counter. Call it in the same transaction that inserts the task.
func (*TaskRepository) ReserveNumber(ctx context.Context, q database.DBTX, workspaceID, projectID, now string) (int, error) {
	var next int
	err := q.QueryRowContext(ctx,
		`UPDATE projects SET next_task_number = next_task_number + 1, updated_at = ?
		 WHERE id = ? AND workspace_id = ?
		 RETURNING next_task_number - 1`,
		now, projectID, workspaceID,
	).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("reserve task number: %w", err)
	}
	return next, nil
}

// nullIfZero stores an unset number as NULL.
func nullIfZero(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

// priorityOrNone stores an unset priority as 'none'.
func priorityOrNone(p model.TaskPriority) string {
	if p == "" {
		return string(model.PriorityNone)
	}
	return string(p)
}

// formatTime renders an optional time as a UTC RFC 3339 string, or NULL.
func formatTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC().Format(time.RFC3339)
}

// scanTaskTimes parses the optional due date and completion time of a task.
func scanTaskTimes(t *model.Task, dueDate, completedAt sql.NullString) error {
	for _, f := range []struct {
		name string
		src  sql.NullString
		dst  **time.Time
	}{
		{"due date", dueDate, &t.DueDate},
		{"completed_at", completedAt, &t.CompletedAt},
	} {
		if !f.src.Valid {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, f.src.String)
		if err != nil {
			return fmt.Errorf("parse %s: %w", f.name, err)
		}
		*f.dst = &parsed
	}
	return nil
}

// scanTaskStamps parses the created_at, updated_at and start_date columns of
// a task.
func scanTaskStamps(t *model.Task, createdAt, updatedAt string, startDate sql.NullString) error {
	var err error
	if startDate.Valid {
		start, err := time.Parse(time.RFC3339, startDate.String)
		if err != nil {
			return fmt.Errorf("parse start date: %w", err)
		}
		t.StartDate = &start
	}
	if t.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return fmt.Errorf("parse created_at: %w", err)
	}
	if t.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return fmt.Errorf("parse updated_at: %w", err)
	}
	return nil
}

// labelChunk bounds the ids of one IN (...) list.
const labelChunk = 400

// LabelIDsByTask returns the live label ids of the given tasks, in label order.
func (*TaskRepository) LabelIDsByTask(ctx context.Context, q database.DBTX, workspaceID string, taskIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	for start := 0; start < len(taskIDs); start += labelChunk {
		chunk := taskIDs[start:min(start+labelChunk, len(taskIDs))]
		args := []any{workspaceID}
		for _, id := range chunk {
			args = append(args, id)
		}
		rows, err := q.QueryContext(ctx,
			`SELECT tl.task_id, tl.label_id
			 FROM task_labels tl
			 JOIN labels l ON l.id = tl.label_id AND l.workspace_id = tl.workspace_id
			 WHERE tl.workspace_id = ? AND tl.removed_at IS NULL AND l.deleted_at IS NULL
			   AND tl.task_id IN (`+placeholders(len(chunk))+`)
			 ORDER BY l.position, l.id`,
			args...,
		)
		if err != nil {
			return nil, fmt.Errorf("task label ids: %w", err)
		}
		for rows.Next() {
			var taskID, labelID string
			if err := rows.Scan(&taskID, &labelID); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan task label: %w", err)
			}
			out[taskID] = append(out[taskID], labelID)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("task label ids: %w", err)
		}
	}
	return out, nil
}

// CountProjectLabels returns how many of the ids are live labels of the project.
func (*TaskRepository) CountProjectLabels(ctx context.Context, q database.DBTX, workspaceID, projectID string, labelIDs []string) (int, error) {
	if len(labelIDs) == 0 {
		return 0, nil
	}
	args := []any{workspaceID, projectID}
	for _, id := range labelIDs {
		args = append(args, id)
	}
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM labels
		 WHERE workspace_id = ? AND project_id = ? AND deleted_at IS NULL AND id IN (`+placeholders(len(labelIDs))+`)`,
		args...,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count project labels: %w", err)
	}
	return n, nil
}

// SetLabels makes labelIDs the task's label set: labels no longer in it are
// marked removed, new ones are added, and one that comes back has its
// removed_at cleared.
func (*TaskRepository) SetLabels(ctx context.Context, q database.DBTX, workspaceID, projectID, taskID string, labelIDs []string, now string) error {
	removeArgs := []any{now, now, workspaceID, taskID}
	remove := `UPDATE task_labels SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND task_id = ? AND removed_at IS NULL`
	if len(labelIDs) > 0 {
		remove += ` AND label_id NOT IN (` + placeholders(len(labelIDs)) + `)`
		for _, id := range labelIDs {
			removeArgs = append(removeArgs, id)
		}
	}
	if _, err := q.ExecContext(ctx, remove, removeArgs...); err != nil {
		return fmt.Errorf("remove task labels: %w", err)
	}
	for _, id := range labelIDs {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO task_labels (workspace_id, project_id, task_id, label_id, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (task_id, label_id) DO UPDATE SET removed_at = NULL, updated_at = excluded.updated_at
			   WHERE task_labels.removed_at IS NOT NULL`,
			workspaceID, projectID, taskID, id, now, now,
		); err != nil {
			return fmt.Errorf("add task label: %w", err)
		}
	}
	return nil
}
