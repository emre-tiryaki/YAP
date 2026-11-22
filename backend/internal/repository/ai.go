package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// aiChatListLimit caps the chat list.
const aiChatListLimit = 500

type AIRepository struct{}

func NewAIRepository() *AIRepository {
	return &AIRepository{}
}

func parseStampErr(s, field string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", field, err)
	}
	return t, nil
}

func nullable(s string) any {
	return nullIfEmpty(s)
}

func stringArgs(ids []string) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

// --- connections ---

const selectAIConnection = `SELECT id, user_id, name, provider, preset, base_url, has_key, created_at, updated_at
	 FROM ai_connections`

func scanAIConnection(row rowScanner, workspaceID string) (model.AIConnection, error) {
	c := model.AIConnection{WorkspaceID: workspaceID}
	var hasKey int
	var created, updated string
	if err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Provider, &c.Preset, &c.BaseURL, &hasKey, &created, &updated); err != nil {
		return model.AIConnection{}, err
	}
	c.HasKey = hasKey == 1
	var err error
	if c.CreatedAt, err = parseStampErr(created, "created_at"); err != nil {
		return model.AIConnection{}, err
	}
	if c.UpdatedAt, err = parseStampErr(updated, "updated_at"); err != nil {
		return model.AIConnection{}, err
	}
	return c, nil
}

func (*AIRepository) ListConnections(ctx context.Context, q database.DBTX, workspaceID, userID string) ([]model.AIConnection, error) {
	rows, err := q.QueryContext(ctx,
		selectAIConnection+` WHERE workspace_id = ? AND user_id = ? AND deleted_at IS NULL ORDER BY created_at, id`,
		workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list ai connections: %w", err)
	}
	return scanAll(rows, "list ai connections", func(r *sql.Rows) (model.AIConnection, error) {
		return scanAIConnection(r, workspaceID)
	})
}

// GetConnection returns a live connection or sql.ErrNoRows.
func (*AIRepository) GetConnection(ctx context.Context, q database.DBTX, workspaceID, userID, id string) (model.AIConnection, error) {
	c, err := scanAIConnection(q.QueryRowContext(ctx,
		selectAIConnection+` WHERE workspace_id = ? AND user_id = ? AND id = ? AND deleted_at IS NULL`,
		workspaceID, userID, id), workspaceID)
	if err != nil {
		return model.AIConnection{}, err
	}
	return c, nil
}

func (*AIRepository) InsertConnection(ctx context.Context, q database.DBTX, c model.AIConnection, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_connections (id, workspace_id, user_id, name, provider, preset, base_url, has_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.UserID, c.Name, c.Provider, c.Preset, c.BaseURL, boolInt(c.HasKey), now, now)
	if err != nil {
		return fmt.Errorf("insert ai connection: %w", err)
	}
	return nil
}

func (*AIRepository) UpdateConnection(ctx context.Context, q database.DBTX, workspaceID, id, name, baseURL string, hasKey bool, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_connections SET name = ?, base_url = ?, has_key = ?, updated_at = ?
		 WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		name, baseURL, boolInt(hasKey), now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("update ai connection: %w", err)
	}
	return nil
}

func (*AIRepository) SoftDeleteConnection(ctx context.Context, q database.DBTX, workspaceID, id, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_connections SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("delete ai connection: %w", err)
	}
	return nil
}

// LiveConnectionIDs returns which of ids are live connections of the workspace.
func (*AIRepository) LiveConnectionIDs(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]bool, error) {
	return liveIDs(ctx, q, `SELECT id FROM ai_connections WHERE workspace_id = ? AND deleted_at IS NULL AND id IN (`, workspaceID, ids)
}

// --- chats ---

const selectAIChat = `SELECT id, user_id, title, COALESCE(connection_id, ''), model, created_at, updated_at FROM ai_chats`

func scanAIChat(row rowScanner, workspaceID string) (model.AIChat, error) {
	c := model.AIChat{WorkspaceID: workspaceID}
	var created, updated string
	if err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.ConnectionID, &c.Model, &created, &updated); err != nil {
		return model.AIChat{}, err
	}
	var err error
	if c.CreatedAt, err = parseStampErr(created, "created_at"); err != nil {
		return model.AIChat{}, err
	}
	if c.UpdatedAt, err = parseStampErr(updated, "updated_at"); err != nil {
		return model.AIChat{}, err
	}
	return c, nil
}

// ListChats returns the live chats of a user, most recently active first.
// needle must already be folded with database.Fold; empty means no filter.
func (*AIRepository) ListChats(ctx context.Context, q database.DBTX, workspaceID, userID, needle string) ([]model.AIChat, error) {
	query := selectAIChat + ` WHERE workspace_id = ? AND user_id = ? AND deleted_at IS NULL`
	args := []any{workspaceID, userID}
	if needle != "" {
		query += ` AND instr(` + fold + `(title), ?) > 0`
		args = append(args, needle)
	}
	query += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, aiChatListLimit)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ai chats: %w", err)
	}
	return scanAll(rows, "list ai chats", func(r *sql.Rows) (model.AIChat, error) {
		return scanAIChat(r, workspaceID)
	})
}

// GetChat returns a live chat of the user or sql.ErrNoRows.
func (*AIRepository) GetChat(ctx context.Context, q database.DBTX, workspaceID, userID, id string) (model.AIChat, error) {
	return scanAIChat(q.QueryRowContext(ctx,
		selectAIChat+` WHERE workspace_id = ? AND user_id = ? AND id = ? AND deleted_at IS NULL`,
		workspaceID, userID, id), workspaceID)
}

func (*AIRepository) InsertChat(ctx context.Context, q database.DBTX, c model.AIChat, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_chats (id, workspace_id, user_id, title, connection_id, model, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.UserID, c.Title, nullable(c.ConnectionID), c.Model, now, now)
	if err != nil {
		return fmt.Errorf("insert ai chat: %w", err)
	}
	return nil
}

func (*AIRepository) UpdateChatTitle(ctx context.Context, q database.DBTX, workspaceID, id, title, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_chats SET title = ?, updated_at = ? WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		title, now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("rename ai chat: %w", err)
	}
	return nil
}

func (*AIRepository) UpdateChatModel(ctx context.Context, q database.DBTX, workspaceID, id, connectionID, modelName, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_chats SET connection_id = ?, model = ?, updated_at = ? WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		nullable(connectionID), modelName, now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("update ai chat model: %w", err)
	}
	return nil
}

func (*AIRepository) TouchChat(ctx context.Context, q database.DBTX, workspaceID, id, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_chats SET updated_at = ? WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("touch ai chat: %w", err)
	}
	return nil
}

func (*AIRepository) SoftDeleteChat(ctx context.Context, q database.DBTX, workspaceID, id, now string) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE ai_messages SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND chat_id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, id); err != nil {
		return fmt.Errorf("delete ai messages: %w", err)
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE ai_chats SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, id); err != nil {
		return fmt.Errorf("delete ai chat: %w", err)
	}
	return nil
}

// --- messages ---

const selectAIMessage = `SELECT id, chat_id, role, content, status, COALESCE(error_code, ''), COALESCE(connection_id, ''), COALESCE(model, ''), created_at, updated_at
	 FROM ai_messages`

func scanAIMessage(row rowScanner, workspaceID string) (model.AIMessage, error) {
	m := model.AIMessage{WorkspaceID: workspaceID}
	var created, updated string
	if err := row.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &m.Status, &m.ErrorCode, &m.ConnectionID, &m.Model, &created, &updated); err != nil {
		return model.AIMessage{}, err
	}
	var err error
	if m.CreatedAt, err = parseStampErr(created, "created_at"); err != nil {
		return model.AIMessage{}, err
	}
	if m.UpdatedAt, err = parseStampErr(updated, "updated_at"); err != nil {
		return model.AIMessage{}, err
	}
	return m, nil
}

func (*AIRepository) ListMessages(ctx context.Context, q database.DBTX, workspaceID, chatID string) ([]model.AIMessage, error) {
	rows, err := q.QueryContext(ctx,
		selectAIMessage+` WHERE workspace_id = ? AND chat_id = ? AND deleted_at IS NULL ORDER BY created_at, id`,
		workspaceID, chatID)
	if err != nil {
		return nil, fmt.Errorf("list ai messages: %w", err)
	}
	return scanAll(rows, "list ai messages", func(r *sql.Rows) (model.AIMessage, error) {
		return scanAIMessage(r, workspaceID)
	})
}

func (*AIRepository) InsertMessage(ctx context.Context, q database.DBTX, m model.AIMessage, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_messages (id, workspace_id, chat_id, role, content, status, error_code, connection_id, model, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.WorkspaceID, m.ChatID, m.Role, m.Content, m.Status, nullable(m.ErrorCode), nullable(m.ConnectionID), nullable(m.Model), now, now)
	if err != nil {
		return fmt.Errorf("insert ai message: %w", err)
	}
	return nil
}

func (*AIRepository) UpdateMessageContent(ctx context.Context, q database.DBTX, workspaceID, id, content, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_messages SET content = ?, updated_at = ? WHERE workspace_id = ? AND id = ?`,
		content, now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("update ai message: %w", err)
	}
	return nil
}

func (*AIRepository) FinishMessage(ctx context.Context, q database.DBTX, workspaceID, id, content string, status model.AIMessageStatus, errorCode, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_messages SET content = ?, status = ?, error_code = ?, updated_at = ? WHERE workspace_id = ? AND id = ?`,
		content, status, nullable(errorCode), now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("finish ai message: %w", err)
	}
	return nil
}

func (*AIRepository) SoftDeleteMessage(ctx context.Context, q database.DBTX, workspaceID, id, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE ai_messages SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, id)
	if err != nil {
		return fmt.Errorf("delete ai message: %w", err)
	}
	return nil
}

// MarkStreamingInterrupted turns every answer a restart left half-written
// into an interrupted one (all workspaces) and returns how many there were.
func (*AIRepository) MarkStreamingInterrupted(ctx context.Context, q database.DBTX, now string) (int64, error) {
	res, err := q.ExecContext(ctx,
		`UPDATE ai_messages SET status = 'interrupted', updated_at = ? WHERE status = 'streaming'`, now)
	if err != nil {
		return 0, fmt.Errorf("mark ai messages interrupted: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark ai messages interrupted: %w", err)
	}
	return n, nil
}

// --- attachments ---

func (*AIRepository) InsertAttachment(ctx context.Context, q database.DBTX, a model.AIAttachment, now string) error {
	status := a.LastStatus
	if status == "" {
		status = model.AIAttachIncluded
	}
	_, err := q.ExecContext(ctx,
		`INSERT INTO ai_message_attachments (id, workspace_id, message_id, position, kind, target_id, path, label, last_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.WorkspaceID, a.MessageID, a.Position, a.Kind, a.TargetID, nullable(a.Path), a.Label, status, now, now)
	if err != nil {
		return fmt.Errorf("insert ai attachment: %w", err)
	}
	return nil
}

// ListAttachments returns the attachments of the given messages by message id,
// each list in position order. Rows of deleted messages are left out.
func (*AIRepository) ListAttachments(ctx context.Context, q database.DBTX, workspaceID string, messageIDs []string) (map[string][]model.AIAttachment, error) {
	out := map[string][]model.AIAttachment{}
	if len(messageIDs) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx,
		`SELECT a.id, a.message_id, a.position, a.kind, a.target_id, COALESCE(a.path, ''), a.label, a.last_status, a.created_at, a.updated_at
		 FROM ai_message_attachments a
		 JOIN ai_messages m ON m.id = a.message_id AND m.workspace_id = a.workspace_id
		 WHERE a.workspace_id = ? AND m.deleted_at IS NULL AND a.message_id IN (`+placeholders(len(messageIDs))+`)
		 ORDER BY a.message_id, a.position`,
		append([]any{workspaceID}, stringArgs(messageIDs)...)...)
	if err != nil {
		return nil, fmt.Errorf("list ai attachments: %w", err)
	}
	list, err := scanAll(rows, "list ai attachments", func(r *sql.Rows) (model.AIAttachment, error) {
		a := model.AIAttachment{WorkspaceID: workspaceID}
		var created, updated string
		if err := r.Scan(&a.ID, &a.MessageID, &a.Position, &a.Kind, &a.TargetID, &a.Path, &a.Label, &a.LastStatus, &created, &updated); err != nil {
			return a, err
		}
		var err error
		if a.CreatedAt, err = parseStampErr(created, "created_at"); err != nil {
			return a, err
		}
		a.UpdatedAt, err = parseStampErr(updated, "updated_at")
		return a, err
	})
	if err != nil {
		return nil, err
	}
	for _, a := range list {
		out[a.MessageID] = append(out[a.MessageID], a)
	}
	return out, nil
}

func (*AIRepository) SetAttachmentStatuses(ctx context.Context, q database.DBTX, workspaceID string, statuses map[string]model.AIAttachmentStatus, now string) error {
	for id, status := range statuses {
		if _, err := q.ExecContext(ctx,
			`UPDATE ai_message_attachments SET last_status = ?, updated_at = ? WHERE workspace_id = ? AND id = ?`,
			status, now, workspaceID, id); err != nil {
			return fmt.Errorf("set ai attachment status: %w", err)
		}
	}
	return nil
}

// --- existence helpers for attachment chips ---

func liveIDs(ctx context.Context, q database.DBTX, prefix, workspaceID string, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx, prefix+placeholders(len(ids))+`)`, append([]any{workspaceID}, stringArgs(ids)...)...)
	if err != nil {
		return nil, fmt.Errorf("live ids: %w", err)
	}
	found, err := scanAll(rows, "live ids", func(r *sql.Rows) (string, error) {
		var id string
		err := r.Scan(&id)
		return id, err
	})
	if err != nil {
		return nil, err
	}
	for _, id := range found {
		out[id] = true
	}
	return out, nil
}

// LiveTaskPlacements returns where each live task sits; tasks that are
// deleted, or whose plan or project is deleted, are left out.
func (*AIRepository) LiveTaskPlacements(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]model.AITaskPlacement, error) {
	out := map[string]model.AITaskPlacement{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.project_id, t.plan_id
		 FROM tasks t
		 JOIN plans pl ON pl.id = t.plan_id AND pl.workspace_id = t.workspace_id
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 WHERE t.workspace_id = ? AND t.deleted_at IS NULL AND pl.deleted_at IS NULL AND p.deleted_at IS NULL
		   AND t.id IN (`+placeholders(len(ids))+`)`,
		append([]any{workspaceID}, stringArgs(ids)...)...)
	if err != nil {
		return nil, fmt.Errorf("live tasks: %w", err)
	}
	type row struct {
		id string
		p  model.AITaskPlacement
	}
	list, err := scanAll(rows, "live tasks", func(r *sql.Rows) (row, error) {
		var v row
		err := r.Scan(&v.id, &v.p.ProjectID, &v.p.PlanID)
		return v, err
	})
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		out[v.id] = v.p
	}
	return out, nil
}

// LivePlanProjects returns the project id of each live plan.
func (*AIRepository) LivePlanProjects(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx,
		`SELECT pl.id, pl.project_id
		 FROM plans pl
		 JOIN projects p ON p.id = pl.project_id AND p.workspace_id = pl.workspace_id
		 WHERE pl.workspace_id = ? AND pl.deleted_at IS NULL AND p.deleted_at IS NULL
		   AND pl.id IN (`+placeholders(len(ids))+`)`,
		append([]any{workspaceID}, stringArgs(ids)...)...)
	if err != nil {
		return nil, fmt.Errorf("live plans: %w", err)
	}
	type row struct{ id, project string }
	list, err := scanAll(rows, "live plans", func(r *sql.Rows) (row, error) {
		var v row
		err := r.Scan(&v.id, &v.project)
		return v, err
	})
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		out[v.id] = v.project
	}
	return out, nil
}

func (*AIRepository) LiveProjectIDs(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]bool, error) {
	return liveIDs(ctx, q, `SELECT id FROM projects WHERE workspace_id = ? AND deleted_at IS NULL AND id IN (`, workspaceID, ids)
}

// LiveResourceIDs returns the ids that are neither purged nor in the trash.
func (*AIRepository) LiveResourceIDs(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]bool, error) {
	return liveIDs(ctx, q, `SELECT id FROM resources WHERE workspace_id = ? AND deleted_at IS NULL AND trashed_at IS NULL AND id IN (`, workspaceID, ids)
}

// --- reset ---

// SoftDeleteAllChats deletes every chat of the user and their messages.
func (*AIRepository) SoftDeleteAllChats(ctx context.Context, q database.DBTX, workspaceID, userID, now string) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE ai_messages SET deleted_at = ?, updated_at = ?
		 WHERE workspace_id = ? AND deleted_at IS NULL
		   AND chat_id IN (SELECT id FROM ai_chats WHERE workspace_id = ? AND user_id = ?)`,
		now, now, workspaceID, workspaceID, userID); err != nil {
		return fmt.Errorf("delete all ai messages: %w", err)
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE ai_chats SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND user_id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, userID); err != nil {
		return fmt.Errorf("delete all ai chats: %w", err)
	}
	return nil
}

// SoftDeleteAllConnections deletes the user's connections and returns their
// ids so the caller can remove the keys from the keychain.
func (*AIRepository) SoftDeleteAllConnections(ctx context.Context, q database.DBTX, workspaceID, userID, now string) ([]string, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id FROM ai_connections WHERE workspace_id = ? AND user_id = ? AND deleted_at IS NULL`, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list ai connections: %w", err)
	}
	ids, err := scanAll(rows, "list ai connections", func(r *sql.Rows) (string, error) {
		var id string
		err := r.Scan(&id)
		return id, err
	})
	if err != nil {
		return nil, err
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE ai_connections SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND user_id = ? AND deleted_at IS NULL`,
		now, now, workspaceID, userID); err != nil {
		return nil, fmt.Errorf("delete all ai connections: %w", err)
	}
	return ids, nil
}
