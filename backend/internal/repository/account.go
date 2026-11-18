package repository

import (
	"context"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type AccountRepository struct{}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{}
}

// exportQueries select the live rows of a user's workspace, table by table.
// They use SELECT * on purpose: an export carries every column, including
// ones added by later migrations. Each query takes the user id and the
// workspace id, in the order given by its args.
var exportQueries = []struct {
	table string
	query string
	args  func(userID, workspaceID string) []any
}{
	{"users", `SELECT * FROM users WHERE id = ?`, byUser},
	{"workspaces", `SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL`, byWorkspace},
	{"memberships", `SELECT * FROM memberships WHERE workspace_id = ? AND left_at IS NULL ORDER BY id`, byWorkspace},
	{"teams", `SELECT * FROM teams WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	{"team_members", `SELECT * FROM team_members WHERE workspace_id = ? AND removed_at IS NULL ORDER BY team_id, membership_id`, byWorkspace},
	{"projects", `SELECT * FROM projects WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	{"project_teams", `SELECT * FROM project_teams WHERE workspace_id = ? AND removed_at IS NULL ORDER BY project_id, team_id`, byWorkspace},
	{"plans", `SELECT * FROM plans WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	{"statuses", `SELECT * FROM statuses WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY plan_id, position, id`, byWorkspace},
	{"tasks", `SELECT * FROM tasks WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	{"task_assignees", `SELECT * FROM task_assignees WHERE workspace_id = ? AND removed_at IS NULL ORDER BY task_id, membership_id`, byWorkspace},
	{"task_comments", `SELECT * FROM task_comments WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	// Trashed items are left out, like their bytes, and so are their links.
	{"resources", `SELECT * FROM resources WHERE workspace_id = ? AND trashed_at IS NULL AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	{"task_resources", `SELECT * FROM task_resources tr
		 WHERE tr.workspace_id = ? AND tr.removed_at IS NULL
		   AND EXISTS (SELECT 1 FROM resources r
		                WHERE r.id = tr.resource_id AND r.workspace_id = tr.workspace_id
		                  AND r.trashed_at IS NULL AND r.deleted_at IS NULL)
		 ORDER BY tr.task_id, tr.resource_id`, byWorkspace},
	{"resource_events", `SELECT * FROM resource_events WHERE workspace_id = ? AND deleted_at IS NULL ORDER BY id`, byWorkspace},
	// Which repository fills a folder, not its files: every device fetches
	// its own copy. The local checkout state is left out.
	{"resource_repositories", `SELECT rr.* FROM resource_repositories rr
		 JOIN resources r ON r.id = rr.resource_id AND r.workspace_id = rr.workspace_id
		 WHERE rr.workspace_id = ? AND r.trashed_at IS NULL AND r.deleted_at IS NULL
		 ORDER BY rr.resource_id`, byWorkspace},
	{"user_settings", `SELECT * FROM user_settings WHERE user_id = ? ORDER BY key`, byUser},
}

func byUser(userID, _ string) []any           { return []any{userID} }
func byWorkspace(_, workspaceID string) []any { return []any{workspaceID} }

// Export reads the live rows of everything the user owns in the workspace.
// Values are kept as stored: TEXT as string, INTEGER as int64, NULL as nil.
func (*AccountRepository) Export(ctx context.Context, q database.DBTX, userID, workspaceID string) ([]model.ExportTable, error) {
	tables := make([]model.ExportTable, 0, len(exportQueries))
	for _, eq := range exportQueries {
		rows, err := dumpRows(ctx, q, eq.query, eq.args(userID, workspaceID)...)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", eq.table, err)
		}
		tables = append(tables, model.ExportTable{Name: eq.table, Rows: rows})
	}
	return tables, nil
}

func dumpRows(ctx context.Context, q database.DBTX, query string, args ...any) ([]map[string]any, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			if b, ok := values[i].([]byte); ok {
				values[i] = string(b)
			}
			row[col] = values[i]
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// resetStatements soft-delete the content of a workspace. Each row gets its
// own deletion record so synchronization can carry the reset to other
// devices. Memberships stay: the owner keeps the workspace.
var resetStatements = []struct {
	table string
	query string
}{
	{"task_comments", `UPDATE task_comments SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	{"task_assignees", `UPDATE task_assignees SET removed_at = ?, updated_at = ? WHERE workspace_id = ? AND removed_at IS NULL`},
	{"task_resources", `UPDATE task_resources SET removed_at = ?, updated_at = ? WHERE workspace_id = ? AND removed_at IS NULL`},
	{"tasks", `UPDATE tasks SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	{"statuses", `UPDATE statuses SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	{"plans", `UPDATE plans SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	{"project_teams", `UPDATE project_teams SET removed_at = ?, updated_at = ? WHERE workspace_id = ? AND removed_at IS NULL`},
	{"projects", `UPDATE projects SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	{"team_members", `UPDATE team_members SET removed_at = ?, updated_at = ? WHERE workspace_id = ? AND removed_at IS NULL`},
	{"teams", `UPDATE teams SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	// Live and trashed resources alike become purged tombstones.
	{"resources", `UPDATE resources SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
	{"resource_events", `UPDATE resource_events SET deleted_at = ?, updated_at = ? WHERE workspace_id = ? AND deleted_at IS NULL`},
}

// Reset soft-deletes all content of the workspace.
func (*AccountRepository) Reset(ctx context.Context, q database.DBTX, workspaceID, now string) error {
	for _, st := range resetStatements {
		if _, err := q.ExecContext(ctx, st.query, now, now, workspaceID); err != nil {
			return fmt.Errorf("reset %s: %w", st.table, err)
		}
	}
	return nil
}
