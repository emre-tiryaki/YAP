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
	"labels":          `SELECT workspace_id = ?, deleted_at IS NULL FROM labels WHERE id = ?`,
	"task_events":     `SELECT workspace_id = ?, deleted_at IS NULL FROM task_events WHERE id = ?`,
	"resource_events": `SELECT workspace_id = ?, deleted_at IS NULL FROM resource_events WHERE id = ?`,
	"ai_connections":  `SELECT workspace_id = ?, deleted_at IS NULL FROM ai_connections WHERE id = ?`,
	"ai_chats":        `SELECT workspace_id = ?, deleted_at IS NULL FROM ai_chats WHERE id = ?`,
	"ai_messages":     `SELECT workspace_id = ?, deleted_at IS NULL FROM ai_messages WHERE id = ?`,
	// Attachments have no deletion mark: a present row is live.
	"ai_message_attachments": `SELECT workspace_id = ?, 1 FROM ai_message_attachments WHERE id = ?`,
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
		`INSERT INTO projects (id, workspace_id, name, description, icon, identifier, created_at, updated_at, archived_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET name = excluded.name, description = excluded.description,
		     icon = excluded.icon, identifier = excluded.identifier, created_at = excluded.created_at,
		     updated_at = excluded.updated_at, archived_at = excluded.archived_at, deleted_at = NULL
		   WHERE projects.workspace_id = excluded.workspace_id`,
		p.ID, workspaceID, p.Name, p.Description, p.Icon, p.Identifier, p.CreatedAt, now, p.ArchivedAt,
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

// LiveIdentifiers returns the identifiers of the workspace's live projects.
func (*AccountRepository) LiveIdentifiers(ctx context.Context, q database.DBTX, workspaceID string) (map[string]bool, error) {
	return (&ProjectRepository{}).Identifiers(ctx, q, workspaceID)
}

// TaskNumber returns the number of a task row that exists (live or deleted),
// or 0 when there is no such row or it has none.
func (*AccountRepository) TaskNumber(ctx context.Context, q database.DBTX, taskID string) (int, error) {
	var number sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT number FROM tasks WHERE id = ?`, taskID).Scan(&number)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("task number: %w", err)
	}
	return int(number.Int64), nil
}

// ReserveTaskNumber takes the next task number of a project.
func (*AccountRepository) ReserveTaskNumber(ctx context.Context, q database.DBTX, workspaceID, projectID, now string) (int, error) {
	return (&TaskRepository{}).ReserveNumber(ctx, q, workspaceID, projectID, now)
}

func (*AccountRepository) InsertTask(ctx context.Context, q database.DBTX, workspaceID, createdBy string, t model.ImportTask, number int, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO tasks (id, workspace_id, project_id, plan_id, status_id, parent_id, created_by, title, description,
		                    position, due_date, start_date, priority, number, completed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET project_id = excluded.project_id, plan_id = excluded.plan_id,
		     status_id = excluded.status_id, parent_id = excluded.parent_id, created_by = excluded.created_by,
		     title = excluded.title, description = excluded.description, position = excluded.position,
		     due_date = excluded.due_date, start_date = excluded.start_date, priority = excluded.priority, completed_at = excluded.completed_at, created_at = excluded.created_at,
		     updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE tasks.workspace_id = excluded.workspace_id`,
		t.ID, workspaceID, t.ProjectID, t.PlanID, t.StatusID, t.ParentID, createdBy, t.Title, t.Description,
		t.Position, t.DueDate, t.StartDate, t.Priority, number, t.CompletedAt, t.CreatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import task: %w", err)
	}
	return nil
}

// LabelIDByName returns the id of the live label of a project with the name.
func (*AccountRepository) LabelIDByName(ctx context.Context, q database.DBTX, workspaceID, projectID, name string) (string, bool, error) {
	var id string
	err := q.QueryRowContext(ctx,
		`SELECT id FROM labels WHERE workspace_id = ? AND project_id = ? AND name = ? AND deleted_at IS NULL`,
		workspaceID, projectID, name,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("label by name: %w", err)
	}
	return id, true, nil
}

// InsertLabel adds a label, bringing a deleted one back.
func (*AccountRepository) InsertLabel(ctx context.Context, q database.DBTX, workspaceID string, l model.ImportLabel, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO labels (id, workspace_id, project_id, name, color, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET project_id = excluded.project_id, name = excluded.name, color = excluded.color,
		     position = excluded.position, created_at = excluded.created_at, updated_at = excluded.updated_at,
		     deleted_at = NULL
		   WHERE labels.workspace_id = excluded.workspace_id`,
		l.ID, workspaceID, l.ProjectID, l.Name, l.Color, l.Position, l.CreatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import label: %w", err)
	}
	return nil
}

// InsertTaskLabel labels a task, reopening a label that was removed.
func (*AccountRepository) InsertTaskLabel(ctx context.Context, q database.DBTX, workspaceID, projectID string, l model.ImportTaskLabel, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_labels (workspace_id, project_id, task_id, label_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (task_id, label_id) DO UPDATE SET removed_at = NULL, updated_at = excluded.updated_at`,
		workspaceID, projectID, l.TaskID, l.LabelID, l.CreatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("import task label: %w", err)
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

// InsertTaskEvent adds an exported event of a task, or brings a deleted one
// back.
func (*AccountRepository) InsertTaskEvent(ctx context.Context, q database.DBTX, workspaceID, actorID string, e model.ImportTaskEvent) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_events (id, workspace_id, task_id, actor_id, action, details, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET task_id = excluded.task_id, actor_id = excluded.actor_id,
		     action = excluded.action, details = excluded.details,
		     created_at = excluded.created_at, updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE task_events.workspace_id = excluded.workspace_id`,
		e.ID, workspaceID, e.TaskID, actorID, e.Action, e.Details, e.CreatedAt, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("import task event: %w", err)
	}
	return nil
}

// InsertTaskRelation adds an exported relation unless the two tasks already
// have a live one in either direction; a removed one is reopened.
func (*AccountRepository) InsertTaskRelation(ctx context.Context, q database.DBTX, workspaceID string, r model.ImportTaskRelation, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_relations (workspace_id, project_id, task_id, related_task_id, kind, created_at, updated_at)
		 SELECT t.workspace_id, t.project_id, ?, ?, ?, ?, ?
		   FROM tasks t
		  WHERE t.id = ? AND t.workspace_id = ?
		    AND NOT EXISTS (SELECT 1 FROM task_relations x
		                     WHERE x.workspace_id = t.workspace_id AND x.removed_at IS NULL
		                       AND ((x.task_id = ? AND x.related_task_id = ?) OR (x.task_id = ? AND x.related_task_id = ?)))
		 ON CONFLICT (task_id, related_task_id) DO UPDATE SET
		     kind = excluded.kind, project_id = excluded.project_id,
		     created_at = excluded.created_at, updated_at = excluded.updated_at, removed_at = NULL
		   WHERE task_relations.workspace_id = excluded.workspace_id`,
		r.TaskID, r.RelatedTaskID, r.Kind, r.CreatedAt, now,
		r.TaskID, workspaceID,
		r.TaskID, r.RelatedTaskID, r.RelatedTaskID, r.TaskID,
	)
	if err != nil {
		return fmt.Errorf("import task relation: %w", err)
	}
	return nil
}

// The AI inserts add a row, or bring back a deleted row of the same
// workspace with the same id. Connections come back without a key.
func (*AccountRepository) InsertAIConnection(ctx context.Context, q database.DBTX, workspaceID, userID string, c model.ImportAIConnection) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_connections (id, workspace_id, user_id, name, provider, preset, base_url, has_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET name = excluded.name, provider = excluded.provider, preset = excluded.preset,
		     base_url = excluded.base_url, has_key = 0, created_at = excluded.created_at,
		     updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE ai_connections.workspace_id = excluded.workspace_id`,
		c.ID, workspaceID, userID, c.Name, c.Provider, c.Preset, c.BaseURL, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("import ai connection: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertAIChat(ctx context.Context, q database.DBTX, workspaceID, userID string, c model.ImportAIChat) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_chats (id, workspace_id, user_id, title, connection_id, model, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET title = excluded.title, connection_id = excluded.connection_id,
		     model = excluded.model, created_at = excluded.created_at, updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE ai_chats.workspace_id = excluded.workspace_id`,
		c.ID, workspaceID, userID, c.Title, c.ConnectionID, c.Model, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("import ai chat: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertAIMessage(ctx context.Context, q database.DBTX, workspaceID string, m model.ImportAIMessage) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_messages (id, workspace_id, chat_id, role, content, status, error_code, connection_id, model, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET chat_id = excluded.chat_id, role = excluded.role, content = excluded.content,
		     status = excluded.status, error_code = excluded.error_code, connection_id = excluded.connection_id,
		     model = excluded.model, created_at = excluded.created_at, updated_at = excluded.updated_at, deleted_at = NULL
		   WHERE ai_messages.workspace_id = excluded.workspace_id`,
		m.ID, workspaceID, m.ChatID, m.Role, m.Content, m.Status, m.ErrorCode, m.ConnectionID, m.Model, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("import ai message: %w", err)
	}
	return nil
}

func (*AccountRepository) InsertAIAttachment(ctx context.Context, q database.DBTX, workspaceID string, a model.ImportAIAttachment) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_message_attachments (id, workspace_id, message_id, position, kind, target_id, path, label, last_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, workspaceID, a.MessageID, a.Position, a.Kind, a.TargetID, a.Path, a.Label, a.LastStatus, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("import ai attachment: %w", err)
	}
	return nil
}
