package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type ResourceRepository struct{}

func NewResourceRepository() *ResourceRepository {
	return &ResourceRepository{}
}

const resourceColumns = `r.id, r.parent_id, r.kind, r.name, r.mime_type, r.size, r.is_text, r.revision,
	r.created_by, r.created_at, r.updated_at, r.trashed_at, r.trashed_root_id`

func scanResource(row rowScanner, workspaceID string) (model.Resource, error) {
	r := model.Resource{WorkspaceID: workspaceID}
	var parentID, mimeType, trashedAt, trashedRootID sql.NullString
	var kind, createdAt, updatedAt string
	var isText int
	if err := row.Scan(&r.ID, &parentID, &kind, &r.Name, &mimeType, &r.Size, &isText, &r.Revision,
		&r.CreatedBy, &createdAt, &updatedAt, &trashedAt, &trashedRootID); err != nil {
		return model.Resource{}, err
	}
	r.ParentID = parentID.String
	r.Kind = model.ResourceKind(kind)
	r.MimeType = mimeType.String
	r.IsText = isText == 1
	r.TrashedRootID = trashedRootID.String
	var err error
	if r.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return model.Resource{}, fmt.Errorf("parse created_at: %w", err)
	}
	if r.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return model.Resource{}, fmt.Errorf("parse updated_at: %w", err)
	}
	if trashedAt.Valid {
		t, err := time.Parse(time.RFC3339, trashedAt.String)
		if err != nil {
			return model.Resource{}, fmt.Errorf("parse trashed_at: %w", err)
		}
		r.TrashedAt = &t
	}
	return r, nil
}

func collectResources(rows *sql.Rows, workspaceID string) ([]model.Resource, error) {
	defer rows.Close()
	out := []model.Resource{}
	for rows.Next() {
		r, err := scanResource(rows, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("scan resource: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (*ResourceRepository) Insert(ctx context.Context, q database.DBTX, r model.Resource, nameKey, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO resources (id, workspace_id, parent_id, kind, name, name_key, mime_type, size, is_text,
		                        revision, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)`,
		r.ID, r.WorkspaceID, nullIfEmpty(r.ParentID), string(r.Kind), r.Name, nameKey,
		nullIfEmpty(r.MimeType), r.Size, boolInt(r.IsText), r.CreatedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert resource: %w", err)
	}
	return nil
}

// Get returns a resource that has not been purged; trashed ones included.
func (*ResourceRepository) Get(ctx context.Context, q database.DBTX, workspaceID, id string) (model.Resource, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+resourceColumns+` FROM resources r
		 WHERE r.id = ? AND r.workspace_id = ? AND r.deleted_at IS NULL`,
		id, workspaceID,
	)
	r, err := scanResource(row, workspaceID)
	if err != nil {
		return model.Resource{}, fmt.Errorf("get resource: %w", err)
	}
	return r, nil
}

// ListLive returns every resource of the workspace that is neither trashed
// nor purged.
func (*ResourceRepository) ListLive(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Resource, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+resourceColumns+` FROM resources r
		 WHERE r.workspace_id = ? AND r.trashed_at IS NULL AND r.deleted_at IS NULL
		 ORDER BY r.name_key, r.id`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	return collectResources(rows, workspaceID)
}

// ListTrash returns the items that were moved to the trash themselves, most
// recently trashed first. Their contents are left out.
func (*ResourceRepository) ListTrash(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Resource, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+resourceColumns+` FROM resources r
		 WHERE r.workspace_id = ? AND r.trashed_root_id = r.id AND r.deleted_at IS NULL
		 ORDER BY r.trashed_at DESC, r.id`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list trash: %w", err)
	}
	return collectResources(rows, workspaceID)
}

// Ancestors returns the resource followed by its parents, up to the top of
// the tree.
func (*ResourceRepository) Ancestors(ctx context.Context, q database.DBTX, workspaceID, id string) ([]model.Resource, error) {
	rows, err := q.QueryContext(ctx,
		`WITH RECURSIVE chain(id, depth) AS (
			SELECT id, 0 FROM resources WHERE id = ? AND workspace_id = ?
			UNION ALL
			SELECT p.parent_id, chain.depth + 1 FROM resources p JOIN chain ON p.id = chain.id
			 WHERE p.workspace_id = ? AND p.parent_id IS NOT NULL
		)
		SELECT `+resourceColumns+` FROM resources r JOIN chain ON chain.id = r.id
		 WHERE r.workspace_id = ?
		 ORDER BY chain.depth`,
		id, workspaceID, workspaceID, workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("resource ancestors: %w", err)
	}
	return collectResources(rows, workspaceID)
}

// LiveSubtree returns the live resource and all of its live descendants,
// parents before children.
func (*ResourceRepository) LiveSubtree(ctx context.Context, q database.DBTX, workspaceID, id string) ([]model.Resource, error) {
	rows, err := q.QueryContext(ctx,
		`WITH RECURSIVE tree(id, depth) AS (
			SELECT id, 0 FROM resources
			 WHERE id = ? AND workspace_id = ? AND trashed_at IS NULL AND deleted_at IS NULL
			UNION ALL
			SELECT c.id, tree.depth + 1 FROM resources c JOIN tree ON c.parent_id = tree.id
			 WHERE c.workspace_id = ? AND c.trashed_at IS NULL AND c.deleted_at IS NULL
		)
		SELECT `+resourceColumns+` FROM resources r JOIN tree ON tree.id = r.id
		 WHERE r.workspace_id = ?
		 ORDER BY tree.depth, r.name_key`,
		id, workspaceID, workspaceID, workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("resource subtree: %w", err)
	}
	return collectResources(rows, workspaceID)
}

// SiblingNameKeys returns the name keys of the live items in a folder ("" for
// the top of the tree).
func (*ResourceRepository) SiblingNameKeys(ctx context.Context, q database.DBTX, workspaceID, parentID string) (map[string]string, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT name_key, id FROM resources
		 WHERE workspace_id = ? AND COALESCE(parent_id, '') = ?
		   AND trashed_at IS NULL AND deleted_at IS NULL`,
		workspaceID, parentID,
	)
	if err != nil {
		return nil, fmt.Errorf("sibling names: %w", err)
	}
	defer rows.Close()
	keys := map[string]string{}
	for rows.Next() {
		var key, id string
		if err := rows.Scan(&key, &id); err != nil {
			return nil, fmt.Errorf("scan sibling name: %w", err)
		}
		keys[key] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sibling names: %w", err)
	}
	return keys, nil
}

// Place sets the parent and name of a resource.
func (*ResourceRepository) Place(ctx context.Context, q database.DBTX, workspaceID, id, parentID, name, nameKey, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE resources SET parent_id = ?, name = ?, name_key = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		nullIfEmpty(parentID), name, nameKey, now, id, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("place resource: %w", err)
	}
	return nil
}

// UpdateContent records new file content and bumps the revision.
func (*ResourceRepository) UpdateContent(ctx context.Context, q database.DBTX, workspaceID, id, mimeType string, size int64, isText bool, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE resources SET mime_type = ?, size = ?, is_text = ?, revision = revision + 1, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND kind = 'file' AND deleted_at IS NULL`,
		nullIfEmpty(mimeType), size, boolInt(isText), now, id, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update resource content: %w", err)
	}
	return nil
}

// Trash moves the live resource and its live descendants to the trash, all
// tagged with the resource as their trash root.
func (*ResourceRepository) Trash(ctx context.Context, q database.DBTX, workspaceID, id, now string) error {
	_, err := q.ExecContext(ctx,
		`WITH RECURSIVE tree(id) AS (
			SELECT id FROM resources
			 WHERE id = ? AND workspace_id = ? AND trashed_at IS NULL AND deleted_at IS NULL
			UNION
			SELECT c.id FROM resources c JOIN tree ON c.parent_id = tree.id
			 WHERE c.workspace_id = ? AND c.trashed_at IS NULL AND c.deleted_at IS NULL
		)
		UPDATE resources SET trashed_at = ?, trashed_root_id = ?, updated_at = ?
		 WHERE workspace_id = ? AND id IN (SELECT id FROM tree)`,
		id, workspaceID, workspaceID, now, id, now, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("trash resource: %w", err)
	}
	return nil
}

// Restore takes everything trashed together with rootID out of the trash.
func (*ResourceRepository) Restore(ctx context.Context, q database.DBTX, workspaceID, rootID, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE resources SET trashed_at = NULL, trashed_root_id = NULL, updated_at = ?
		 WHERE workspace_id = ? AND trashed_root_id = ? AND deleted_at IS NULL`,
		now, workspaceID, rootID,
	)
	if err != nil {
		return fmt.Errorf("restore resource: %w", err)
	}
	return nil
}

// Purge marks everything trashed together with rootID as deleted for good.
func (*ResourceRepository) Purge(ctx context.Context, q database.DBTX, workspaceID, rootID, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE resources SET deleted_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND trashed_root_id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, rootID,
	)
	if err != nil {
		return fmt.Errorf("purge resource: %w", err)
	}
	return nil
}

// CloseLinksOfTrashGroup removes the task links of everything trashed
// together with rootID.
func (*ResourceRepository) CloseLinksOfTrashGroup(ctx context.Context, q database.DBTX, workspaceID, rootID, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE task_resources SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND removed_at IS NULL
		   AND resource_id IN (SELECT id FROM resources WHERE workspace_id = ? AND trashed_root_id = ?)`,
		now, now, workspaceID, workspaceID, rootID,
	)
	if err != nil {
		return fmt.Errorf("close resource links: %w", err)
	}
	return nil
}

// Link links a task to a resource, reopening a removed link.
func (*ResourceRepository) Link(ctx context.Context, q database.DBTX, workspaceID, taskID, resourceID, linkedBy, now string) (bool, error) {
	res, err := q.ExecContext(ctx,
		`INSERT INTO task_resources (workspace_id, task_id, resource_id, linked_by, linked_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (task_id, resource_id) DO UPDATE
		   SET removed_at = NULL, linked_by = excluded.linked_by, linked_at = excluded.linked_at,
		       updated_at = excluded.updated_at
		 WHERE task_resources.removed_at IS NOT NULL`,
		workspaceID, taskID, resourceID, linkedBy, now, now,
	)
	if err != nil {
		return false, fmt.Errorf("link resource: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("link resource: %w", err)
	}
	return n > 0, nil
}

// Unlink removes an open link and reports whether there was one.
func (*ResourceRepository) Unlink(ctx context.Context, q database.DBTX, workspaceID, taskID, resourceID, now string) (bool, error) {
	res, err := q.ExecContext(ctx,
		`UPDATE task_resources SET removed_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND task_id = ? AND resource_id = ? AND removed_at IS NULL`,
		now, now, workspaceID, taskID, resourceID,
	)
	if err != nil {
		return false, fmt.Errorf("unlink resource: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("unlink resource: %w", err)
	}
	return n > 0, nil
}

// ListByTask returns the resources linked to a task, trashed ones included.
func (*ResourceRepository) ListByTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.Resource, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+resourceColumns+` FROM resources r
		 JOIN task_resources tr ON tr.resource_id = r.id AND tr.workspace_id = r.workspace_id
		 WHERE tr.workspace_id = ? AND tr.task_id = ? AND tr.removed_at IS NULL AND r.deleted_at IS NULL
		 ORDER BY tr.linked_at, r.id`,
		workspaceID, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task resources: %w", err)
	}
	return collectResources(rows, workspaceID)
}

// ListTasks returns the non-deleted tasks a resource is linked to.
func (*ResourceRepository) ListTasks(ctx context.Context, q database.DBTX, workspaceID, resourceID string) ([]model.LinkedTask, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.title, t.project_id, p.name, t.plan_id, t.completed_at
		 FROM task_resources tr
		 JOIN tasks t ON t.id = tr.task_id AND t.workspace_id = tr.workspace_id
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 WHERE tr.workspace_id = ? AND tr.resource_id = ? AND tr.removed_at IS NULL
		   AND t.deleted_at IS NULL
		 ORDER BY tr.linked_at, t.id`,
		workspaceID, resourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list resource tasks: %w", err)
	}
	defer rows.Close()
	tasks := []model.LinkedTask{}
	for rows.Next() {
		var lt model.LinkedTask
		var completedAt sql.NullString
		if err := rows.Scan(&lt.TaskID, &lt.Title, &lt.ProjectID, &lt.ProjectName, &lt.PlanID, &completedAt); err != nil {
			return nil, fmt.Errorf("scan resource task: %w", err)
		}
		if completedAt.Valid {
			t, err := time.Parse(time.RFC3339, completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("parse completed_at: %w", err)
			}
			lt.CompletedAt = &t
		}
		tasks = append(tasks, lt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list resource tasks: %w", err)
	}
	return tasks, nil
}

func (*ResourceRepository) InsertEvent(ctx context.Context, q database.DBTX, e model.ResourceEvent, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO resource_events (id, workspace_id, resource_id, actor_id, action, task_id, details, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WorkspaceID, e.ResourceID, e.ActorID, string(e.Action), nullIfEmpty(e.TaskID), e.Details, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert resource event: %w", err)
	}
	return nil
}

const selectEvent = `SELECT e.id, e.resource_id, r.kind, e.actor_id, u.display_name, e.action, e.task_id,
	e.details, e.created_at, e.updated_at
	 FROM resource_events e
	 JOIN resources r ON r.id = e.resource_id AND r.workspace_id = e.workspace_id
	 JOIN memberships m ON m.id = e.actor_id AND m.workspace_id = e.workspace_id
	 JOIN users u ON u.id = m.user_id`

func scanEvent(row rowScanner, workspaceID string) (model.ResourceEvent, error) {
	e := model.ResourceEvent{WorkspaceID: workspaceID}
	var kind, action, createdAt, updatedAt string
	var taskID sql.NullString
	if err := row.Scan(&e.ID, &e.ResourceID, &kind, &e.ActorID, &e.ActorName, &action, &taskID,
		&e.Details, &createdAt, &updatedAt); err != nil {
		return model.ResourceEvent{}, err
	}
	e.ResourceKind = model.ResourceKind(kind)
	e.Action = model.ResourceAction(action)
	e.TaskID = taskID.String
	var err error
	if e.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return model.ResourceEvent{}, fmt.Errorf("parse created_at: %w", err)
	}
	if e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return model.ResourceEvent{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return e, nil
}

// LastEvent returns the newest event of a resource.
func (*ResourceRepository) LastEvent(ctx context.Context, q database.DBTX, workspaceID, resourceID string) (model.ResourceEvent, error) {
	row := q.QueryRowContext(ctx,
		selectEvent+`
		 WHERE e.workspace_id = ? AND e.resource_id = ?
		 ORDER BY e.id DESC LIMIT 1`,
		workspaceID, resourceID,
	)
	e, err := scanEvent(row, workspaceID)
	if err != nil {
		return model.ResourceEvent{}, fmt.Errorf("last resource event: %w", err)
	}
	return e, nil
}

// ExtendEvent folds a further edit into an open 'edited' event.
func (*ResourceRepository) ExtendEvent(ctx context.Context, q database.DBTX, workspaceID, eventID, details, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE resource_events SET details = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND action = 'edited'`,
		details, now, eventID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("extend resource event: %w", err)
	}
	return nil
}

// ListEvents returns events newest first, older than the before event id
// when it is given.
func (*ResourceRepository) ListEvents(ctx context.Context, q database.DBTX, workspaceID string, scope model.ResourceEventScope, before string, limit int) ([]model.ResourceEvent, error) {
	var (
		query string
		args  []any
	)
	switch {
	case scope.ResourceID != "":
		query = `WITH RECURSIVE tree(id) AS (
				SELECT id FROM resources WHERE id = ? AND workspace_id = ?
				UNION
				SELECT c.id FROM resources c JOIN tree ON c.parent_id = tree.id WHERE c.workspace_id = ?
			) ` + selectEvent + `
			 WHERE e.workspace_id = ? AND e.resource_id IN (SELECT id FROM tree)`
		args = []any{scope.ResourceID, workspaceID, workspaceID, workspaceID}
	case scope.ProjectID != "":
		query = `WITH RECURSIVE tree(id) AS (
				SELECT tr.resource_id FROM task_resources tr
				 JOIN tasks t ON t.id = tr.task_id AND t.workspace_id = tr.workspace_id
				 WHERE tr.workspace_id = ? AND t.project_id = ? AND tr.removed_at IS NULL AND t.deleted_at IS NULL
				UNION
				SELECT c.id FROM resources c JOIN tree ON c.parent_id = tree.id WHERE c.workspace_id = ?
			) ` + selectEvent + `
			 WHERE e.workspace_id = ?
			   AND (e.resource_id IN (SELECT id FROM tree)
			        OR e.task_id IN (SELECT id FROM tasks WHERE workspace_id = ? AND project_id = ?))`
		args = []any{workspaceID, scope.ProjectID, workspaceID, workspaceID, workspaceID, scope.ProjectID}
	default:
		query = selectEvent + ` WHERE e.workspace_id = ?`
		args = []any{workspaceID}
	}
	if before != "" {
		query += ` AND e.id < ?`
		args = append(args, before)
	}
	query += ` ORDER BY e.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list resource events: %w", err)
	}
	defer rows.Close()
	events := []model.ResourceEvent{}
	for rows.Next() {
		e, err := scanEvent(rows, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("scan resource event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list resource events: %w", err)
	}
	return events, nil
}
