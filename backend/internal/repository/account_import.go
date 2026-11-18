package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// rowStateQueries look up an id in the tables an import adds to. They
// report whether the row is in the given workspace and whether it is live.
var rowStateQueries = map[string]string{
	"projects":        `SELECT workspace_id = ?, deleted_at IS NULL FROM projects WHERE id = ?`,
	"plans":           `SELECT workspace_id = ?, deleted_at IS NULL FROM plans WHERE id = ?`,
	"statuses":        `SELECT workspace_id = ?, deleted_at IS NULL FROM statuses WHERE id = ?`,
	"tasks":           `SELECT workspace_id = ?, deleted_at IS NULL FROM tasks WHERE id = ?`,
	"task_comments":   `SELECT workspace_id = ?, deleted_at IS NULL FROM task_comments WHERE id = ?`,
	"resource_events": `SELECT workspace_id = ?, deleted_at IS NULL FROM resource_events WHERE id = ?`,
	// An item in the trash counts as taken: the import adds a live copy
	// instead of reaching into the trash.
	"resources": `SELECT workspace_id = ? AND (trashed_at IS NULL OR deleted_at IS NOT NULL), deleted_at IS NULL
		 FROM resources WHERE id = ?`,
}

// RowState says what an id stands for in table.
func (*AccountRepository) RowState(ctx context.Context, q database.DBTX, workspaceID, table, id string) (model.ImportRowState, error) {
	query, ok := rowStateQueries[table]
	if !ok {
		return "", fmt.Errorf("row state: unknown table %q", table)
	}
	var sameWorkspace, live bool
	err := q.QueryRowContext(ctx, query, workspaceID, id).Scan(&sameWorkspace, &live)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return model.RowAbsent, nil
	case err != nil:
		return "", fmt.Errorf("row state of %s: %w", table, err)
	case !sameWorkspace:
		return model.RowElsewhere, nil
	case live:
		return model.RowLive, nil
	default:
		return model.RowDeleted, nil
	}
}

// HasContent reports whether the workspace has a live project or a resource
// outside the trash.
func (*AccountRepository) HasContent(ctx context.Context, q database.DBTX, workspaceID string) (bool, error) {
	var has bool
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM projects WHERE workspace_id = ? AND deleted_at IS NULL)
		     OR EXISTS (SELECT 1 FROM resources WHERE workspace_id = ? AND trashed_at IS NULL AND deleted_at IS NULL)`,
		workspaceID, workspaceID,
	).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("has content: %w", err)
	}
	return has, nil
}

// PlanNames returns the names of the live plans of a project.
func (*AccountRepository) PlanNames(ctx context.Context, q database.DBTX, workspaceID, projectID string) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT name FROM plans WHERE workspace_id = ? AND project_id = ? AND team_id IS NULL AND deleted_at IS NULL`,
		workspaceID, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("plan names: %w", err)
	}
	defer rows.Close()
	names := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan plan name: %w", err)
		}
		names[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("plan names: %w", err)
	}
	return names, nil
}

// The Insert* methods add a row, or bring back a deleted row of the same
// workspace with the same id.
func (*AccountRepository) InsertProject(ctx context.Context, q database.DBTX, workspaceID string, p model.ImportProject, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO projects (id, workspace_id, name, description, icon, created_at, updated_at, archived_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET name = excluded.name, description = excluded.description,
		     icon = excluded.icon, created_at = excluded.created_at, updated_at = excluded.updated_at,
		     archived_at = excluded.archived_at, deleted_at = NULL
		   WHERE projects.workspace_id = excluded.workspace_id`,
		p.ID, workspaceID, p.Name, p.Description, p.Icon, p.CreatedAt, now, p.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("import project: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertPlan(ctx context.Context, q database.DBTX, workspaceID string, p model.ImportPlan, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO plans (id, workspace_id, project_id, team_id, name, description, icon, created_at, updated_at, archived_at)
		 VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET project_id = excluded.project_id, team_id = NULL, name = excluded.name,
		     description = excluded.description, icon = excluded.icon, created_at = excluded.created_at,
		     updated_at = excluded.updated_at, archived_at = excluded.archived_at, deleted_at = NULL
		   WHERE plans.workspace_id = excluded.workspace_id`,
		p.ID, workspaceID, p.ProjectID, p.Name, p.Description, p.Icon, p.CreatedAt, now, p.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("import plan: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertStatus(ctx context.Context, q database.DBTX, workspaceID string, s model.ImportStatus, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO statuses (id, workspace_id, plan_id, title, description, color, category, position, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET plan_id = excluded.plan_id, title = excluded.title,
		     description = excluded.description, color = excluded.color, category = excluded.category,
		     position = excluded.position, updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE statuses.workspace_id = excluded.workspace_id`,
		s.ID, workspaceID, s.PlanID, s.Title, s.Description, s.Color, s.Category, s.Position, now,
	)
	if err != nil {
		return fmt.Errorf("import status: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertTask(ctx context.Context, q database.DBTX, workspaceID, createdBy string, t model.ImportTask, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO tasks (id, workspace_id, project_id, plan_id, status_id, parent_id, created_by, title, description,
		                    position, due_date, completed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET project_id = excluded.project_id, plan_id = excluded.plan_id,
		     status_id = excluded.status_id, parent_id = excluded.parent_id, created_by = excluded.created_by,
		     title = excluded.title, description = excluded.description, position = excluded.position,
		     due_date = excluded.due_date, completed_at = excluded.completed_at, created_at = excluded.created_at,
		     updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE tasks.workspace_id = excluded.workspace_id`,
		t.ID, workspaceID, t.ProjectID, t.PlanID, t.StatusID, t.ParentID, createdBy, t.Title, t.Description,
		t.Position, t.DueDate, t.CompletedAt, t.CreatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import task: %w", err)
	}
	return nil
}

// InsertAssignee assigns a task, reopening an assignment that was removed.
func (*AccountRepository) InsertAssignee(ctx context.Context, q database.DBTX, workspaceID, taskID, membershipID, assignedAt, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_assignees (workspace_id, task_id, membership_id, assigned_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (task_id, membership_id) DO UPDATE SET removed_at = NULL, updated_at = excluded.updated_at`,
		workspaceID, taskID, membershipID, assignedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import assignee: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertComment(ctx context.Context, q database.DBTX, workspaceID, authorID string, c model.ImportComment) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_comments (id, workspace_id, task_id, author_id, body, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET task_id = excluded.task_id, author_id = excluded.author_id,
		     body = excluded.body, created_at = excluded.created_at, updated_at = excluded.updated_at,
		     deleted_at = NULL
		   WHERE task_comments.workspace_id = excluded.workspace_id`,
		c.ID, workspaceID, c.TaskID, authorID, c.Body, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("import comment: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertResource(ctx context.Context, q database.DBTX, workspaceID, createdBy, nameKey string, r model.ImportResource, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO resources (id, workspace_id, parent_id, kind, name, name_key, mime_type, size, is_text,
		                        revision, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET parent_id = excluded.parent_id, kind = excluded.kind, name = excluded.name,
		     name_key = excluded.name_key, mime_type = excluded.mime_type, size = excluded.size,
		     is_text = excluded.is_text, revision = excluded.revision, created_by = excluded.created_by,
		     created_at = excluded.created_at, updated_at = excluded.updated_at,
		     trashed_at = NULL, trashed_root_id = NULL, deleted_at = NULL
		   WHERE resources.workspace_id = excluded.workspace_id`,
		r.ID, workspaceID, r.ParentID, r.Kind, r.Name, nameKey, r.MimeType, r.Size, boolInt(r.IsText),
		r.Revision, createdBy, r.CreatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import resource: %w", err)
	}
	return nil
}

// InsertRepository makes an imported folder a repository folder again, with
// a copy still to be fetched.
func (*AccountRepository) InsertRepository(ctx context.Context, q database.DBTX, workspaceID, addedBy string, r model.ImportRepository, now string) error {
	repo := model.Repository{
		ResourceID: r.ResourceID, WorkspaceID: workspaceID, GitHubID: r.GitHubID, FullName: r.FullName,
		HTMLURL: r.HTMLURL, Branch: r.Branch, Private: r.IsPrivate == 1, AddedBy: addedBy,
	}
	if err := (&RepoRepository{}).Insert(ctx, q, repo, now); err != nil {
		return fmt.Errorf("import repository: %w", err)
	}
	return nil
}

// InsertTaskResource links a task and a resource, reopening a removed link.
func (*AccountRepository) InsertTaskResource(ctx context.Context, q database.DBTX, workspaceID, linkedBy string, l model.ImportTaskResource, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_resources (workspace_id, task_id, resource_id, linked_by, linked_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (task_id, resource_id) DO UPDATE SET removed_at = NULL, updated_at = excluded.updated_at`,
		workspaceID, l.TaskID, l.ResourceID, linkedBy, l.LinkedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import task resource: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertEvent(ctx context.Context, q database.DBTX, workspaceID, actorID string, e model.ImportEvent) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO resource_events (id, workspace_id, resource_id, actor_id, action, task_id, details, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET resource_id = excluded.resource_id, actor_id = excluded.actor_id,
		     action = excluded.action, task_id = excluded.task_id, details = excluded.details,
		     created_at = excluded.created_at, updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE resource_events.workspace_id = excluded.workspace_id`,
		e.ID, workspaceID, e.ResourceID, actorID, e.Action, e.TaskID, e.Details, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("import resource event: %w", err)
	}
	return nil
}
