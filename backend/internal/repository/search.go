package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// The fold function makes matching case-insensitive beyond ASCII; needle must
// already be folded with database.Fold. instr is used instead of LIKE so the
// needle needs no escaping. Name matches that start with the needle come
// first, then shorter names.
const fold = database.FoldFunction

type SearchRepository struct{}

func NewSearchRepository() *SearchRepository {
	return &SearchRepository{}
}

func (*SearchRepository) SearchProjects(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchProject, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, name, COALESCE(icon, '')
		 FROM projects
		 WHERE workspace_id = ? AND deleted_at IS NULL
		   AND instr(`+fold+`(name), ?) > 0
		 ORDER BY instr(`+fold+`(name), ?) <> 1, length(name), name, id
		 LIMIT ?`,
		workspaceID, needle, needle, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search projects: %w", err)
	}
	return scanAll(rows, "search projects", func(r *sql.Rows) (model.SearchProject, error) {
		var p model.SearchProject
		err := r.Scan(&p.ID, &p.Name, &p.Icon)
		return p, err
	})
}

func (*SearchRepository) SearchPlans(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchPlan, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT pl.id, pl.project_id, p.name, pl.name, COALESCE(pl.icon, '')
		 FROM plans pl
		 JOIN projects p ON p.id = pl.project_id AND p.workspace_id = pl.workspace_id
		 WHERE pl.workspace_id = ? AND pl.deleted_at IS NULL AND p.deleted_at IS NULL
		   AND instr(`+fold+`(pl.name), ?) > 0
		 ORDER BY instr(`+fold+`(pl.name), ?) <> 1, length(pl.name), pl.name, pl.id
		 LIMIT ?`,
		workspaceID, needle, needle, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search plans: %w", err)
	}
	return scanAll(rows, "search plans", func(r *sql.Rows) (model.SearchPlan, error) {
		var p model.SearchPlan
		err := r.Scan(&p.ID, &p.ProjectID, &p.ProjectName, &p.Name, &p.Icon)
		return p, err
	})
}

// SearchTasks matches task titles and descriptions; title matches rank first.
func (*SearchRepository) SearchTasks(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchTask, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.id, t.project_id, p.name, t.plan_id, pl.name,
		        COALESCE(t.parent_id, ''), t.title, t.completed_at IS NOT NULL
		 FROM tasks t
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN plans pl ON pl.id = t.plan_id AND pl.project_id = t.project_id
		 LEFT JOIN tasks parent ON parent.id = t.parent_id
		 WHERE t.workspace_id = ? AND t.deleted_at IS NULL
		   AND p.deleted_at IS NULL AND pl.deleted_at IS NULL
		   AND (t.parent_id IS NULL OR parent.deleted_at IS NULL)
		   AND (instr(`+fold+`(t.title), ?) > 0
		        OR instr(`+fold+`(COALESCE(t.description, '')), ?) > 0)
		 ORDER BY instr(`+fold+`(t.title), ?) = 0, instr(`+fold+`(t.title), ?) <> 1,
		          t.completed_at IS NOT NULL, length(t.title), t.title, t.id
		 LIMIT ?`,
		workspaceID, needle, needle, needle, needle, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	return scanAll(rows, "search tasks", func(r *sql.Rows) (model.SearchTask, error) {
		var t model.SearchTask
		err := r.Scan(&t.ID, &t.ProjectID, &t.ProjectName, &t.PlanID, &t.PlanName,
			&t.ParentID, &t.Title, &t.Done)
		return t, err
	})
}

// TaskByKey finds the live task with the given project identifier and number.
func (*SearchRepository) TaskByKey(ctx context.Context, q database.DBTX, workspaceID, identifier string, number int) (model.SearchTask, bool, error) {
	var t model.SearchTask
	err := q.QueryRowContext(ctx,
		`SELECT t.id, t.project_id, p.name, t.plan_id, pl.name,
		        COALESCE(t.parent_id, ''), t.title, t.completed_at IS NOT NULL
		 FROM tasks t
		 JOIN projects p ON p.id = t.project_id AND p.workspace_id = t.workspace_id
		 JOIN plans pl ON pl.id = t.plan_id AND pl.project_id = t.project_id
		 LEFT JOIN tasks parent ON parent.id = t.parent_id
		 WHERE t.workspace_id = ? AND p.identifier = ? AND t.number = ?
		   AND t.deleted_at IS NULL AND p.deleted_at IS NULL AND pl.deleted_at IS NULL
		   AND (t.parent_id IS NULL OR parent.deleted_at IS NULL)`,
		workspaceID, identifier, number,
	).Scan(&t.ID, &t.ProjectID, &t.ProjectName, &t.PlanID, &t.PlanName, &t.ParentID, &t.Title, &t.Done)
	if errors.Is(err, sql.ErrNoRows) {
		return model.SearchTask{}, false, nil
	}
	if err != nil {
		return model.SearchTask{}, false, fmt.Errorf("task by key: %w", err)
	}
	return t, true, nil
}

// SearchResources matches the names of live (not trashed, not purged) files
// and folders.
func (*SearchRepository) SearchResources(ctx context.Context, q database.DBTX, workspaceID, needle string, limit int) ([]model.SearchResource, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT r.id, COALESCE(r.parent_id, ''), COALESCE(parent.name, ''),
		        r.kind, r.name, COALESCE(r.mime_type, ''), r.is_text
		 FROM resources r
		 LEFT JOIN resources parent ON parent.id = r.parent_id
		 WHERE r.workspace_id = ? AND r.trashed_at IS NULL AND r.deleted_at IS NULL
		   AND instr(`+fold+`(r.name), ?) > 0
		 ORDER BY instr(`+fold+`(r.name), ?) <> 1, length(r.name), r.name, r.id
		 LIMIT ?`,
		workspaceID, needle, needle, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search resources: %w", err)
	}
	return scanAll(rows, "search resources", func(r *sql.Rows) (model.SearchResource, error) {
		var res model.SearchResource
		var kind string
		err := r.Scan(&res.ID, &res.ParentID, &res.ParentName, &kind, &res.Name, &res.MimeType, &res.IsText)
		res.Kind = model.ResourceKind(kind)
		return res, err
	})
}

func scanAll[T any](rows *sql.Rows, op string, scan func(*sql.Rows) (T, error)) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return out, nil
}
