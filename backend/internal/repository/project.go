package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type ProjectRepository struct{}

func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{}
}

func (*ProjectRepository) List(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Project, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT p.id, p.name, p.description,
			(SELECT COUNT(*) FROM tasks t
				WHERE t.project_id = p.id
				AND t.workspace_id = p.workspace_id
				AND t.deleted_at IS NULL)
		 FROM projects p
		 WHERE p.workspace_id = ? AND p.deleted_at IS NULL AND p.archived_at IS NULL
		 ORDER BY p.created_at DESC, p.id DESC`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		p := model.Project{WorkspaceID: workspaceID}
		var description sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &description, &p.TaskCount); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.Description = description.String
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (*ProjectRepository) Insert(ctx context.Context, q database.DBTX, p model.Project, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO projects (id, workspace_id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.WorkspaceID, p.Name, nullIfEmpty(p.Description), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (*ProjectRepository) InsertPlan(ctx context.Context, q database.DBTX, p model.Plan, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO plans (id, workspace_id, project_id, team_id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.WorkspaceID, p.ProjectID, nullIfEmpty(p.TeamID), p.Name, nullIfEmpty(p.Description), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert plan: %w", err)
	}
	return nil
}

func (*ProjectRepository) InsertStatus(ctx context.Context, q database.DBTX, s model.Status, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO statuses (id, workspace_id, plan_id, title, color, category, position, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.WorkspaceID, s.PlanID, s.Title, s.Color, string(s.Category), s.Position, now,
	)
	if err != nil {
		return fmt.Errorf("insert status: %w", err)
	}
	return nil
}

func (*ProjectRepository) Name(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error) {
	var name string
	err := q.QueryRowContext(ctx,
		`SELECT name FROM projects WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		projectID, workspaceID,
	).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("project name: %w", err)
	}
	return name, nil
}

func (*ProjectRepository) DefaultPlanID(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error) {
	var id string
	err := q.QueryRowContext(ctx,
		`SELECT id FROM plans
		 WHERE workspace_id = ? AND project_id = ? AND team_id IS NULL
		   AND deleted_at IS NULL AND archived_at IS NULL
		 ORDER BY created_at, id
		 LIMIT 1`,
		workspaceID, projectID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("default plan: %w", err)
	}
	return id, nil
}

func (*ProjectRepository) PlanID(ctx context.Context, q database.DBTX, workspaceID, projectID, planID string) (string, error) {
	var id string
	err := q.QueryRowContext(ctx,
		`SELECT id FROM plans
		 WHERE id = ? AND workspace_id = ? AND project_id = ?
		   AND deleted_at IS NULL AND archived_at IS NULL`,
		planID, workspaceID, projectID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("plan: %w", err)
	}
	return id, nil
}

func (*ProjectRepository) Get(ctx context.Context, q database.DBTX, workspaceID, projectID string) (model.Project, error) {
	p := model.Project{ID: projectID, WorkspaceID: workspaceID}
	var description sql.NullString
	err := q.QueryRowContext(ctx,
		`SELECT p.name, p.description,
			(SELECT COUNT(*) FROM tasks t
				WHERE t.project_id = p.id
				AND t.workspace_id = p.workspace_id
				AND t.deleted_at IS NULL)
		 FROM projects p
		 WHERE p.id = ? AND p.workspace_id = ? AND p.deleted_at IS NULL`,
		projectID, workspaceID,
	).Scan(&p.Name, &description, &p.TaskCount)
	if err != nil {
		return model.Project{}, fmt.Errorf("get project: %w", err)
	}
	p.Description = description.String
	return p, nil
}

func (*ProjectRepository) ListPlans(ctx context.Context, q database.DBTX, workspace_id, project_id string) ([]model.Plan, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT pl.id, pl.team_id, tm.name, pl.name, pl.description,
			(SELECT COUNT(*) FROM tasks t
				WHERE t.plan_id = pl.id
				AND t.workspace_id = pl.workspace_id
				AND t.deleted_at IS NULL)
			 FROM plans pl
			 LEFT JOIN teams tm ON tm.id = pl.team_id AND tm.workspace_id = pl.workspace_id
			 WHERE pl.workspace_id = ? AND pl.project_id = ?
				AND pl.deleted_at IS NULL AND pl.archived_at IS NULL
			 ORDER BY pl.created_at, pl.id`,
		workspace_id, project_id,
	)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	plans := []model.Plan{}
	for rows.Next() {
		pl := model.Plan{WorkspaceID: workspace_id, ProjectID: project_id}
		var teamID, teamName, description sql.NullString
		if err := rows.Scan(&pl.ID, &teamID, &teamName, &pl.Name, &description, &pl.TaskCount); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		pl.TeamID = teamID.String
		pl.TeamName = teamName.String
		pl.Description = description.String
		plans = append(plans, pl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	return plans, nil
}

// PlanNameExists reports whether a non-deleted whole-project plan with the
// given name exists. It mirrors idx_plans_project_id_team_id_name, so archived
// plans still count.
func (*ProjectRepository) PlanNameExists(ctx context.Context, q database.DBTX, workspaceID, projectID, name string) (bool, error) {
	var exists bool
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM plans
			 WHERE workspace_id = ? AND project_id = ? AND team_id IS NULL
			   AND name = ? AND deleted_at IS NULL)`,
		workspaceID, projectID, name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("plan name exists: %w", err)
	}
	return exists, nil
}
