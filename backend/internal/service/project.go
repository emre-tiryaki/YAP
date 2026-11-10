package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/position"
)

const defaultPlanName = "Main"

type ProjectRepository interface {
	List(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Project, error)
	Insert(ctx context.Context, q database.DBTX, p model.Project, now string) error
	InsertPlan(ctx context.Context, q database.DBTX, p model.Plan, now string) error
	InsertStatus(ctx context.Context, q database.DBTX, s model.Status, now string) error
	Name(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error)
	DefaultPlanID(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error)
	PlanID(ctx context.Context, q database.DBTX, workspaceID, projectID, planID string) (string, error)
	Get(ctx context.Context, q database.DBTX, workspaceID, projectID string) (model.Project, error)
	ListPlans(ctx context.Context, q database.DBTX, workspaceID, projectID string) ([]model.Plan, error)
	PlanNameExists(ctx context.Context, q database.DBTX, workspaceID, projectID, name string) (bool, error)
}

var defaultStatuses = []struct {
	title    string
	color    string
	category model.StatusCategory
}{
	{"To Do", "#9ca3af", model.StatusToDo},
	{"In Progress", "#3b82f6", model.StatusInProgress},
	{"Done", "#22c55e", model.StatusDone},
}

type CreateProjectInput struct {
	Name        string
	Description string
}

type CreatePlanInput struct {
	Name        string
	Description string
}

type ProjectService struct {
	db   *sql.DB
	repo ProjectRepository
}

func NewProjectService(db *sql.DB, repo ProjectRepository) *ProjectService {
	return &ProjectService{db: db, repo: repo}
}

func (s *ProjectService) List(ctx context.Context, workspaceID string) ([]model.Project, error) {
	return s.repo.List(ctx, s.db, workspaceID)
}

func (s *ProjectService) Create(ctx context.Context, workspaceID string, in CreateProjectInput) (model.Project, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return model.Project{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	project := model.Project{
		ID:          model.NewID(),
		WorkspaceID: workspaceID,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
	}

	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowUTC()
		if err := s.repo.Insert(ctx, tx, project, now); err != nil {
			return err
		}

		plan := model.Plan{
			ID:          model.NewID(),
			WorkspaceID: workspaceID,
			ProjectID:   project.ID,
			Name:        defaultPlanName,
		}
		if err := s.repo.InsertPlan(ctx, tx, plan, now); err != nil {
			return err
		}

		return s.insertDefaultStatuses(ctx, tx, plan, now)
	})
	if err != nil {
		return model.Project{}, err
	}
	return project, nil
}

func (s *ProjectService) Get(ctx context.Context, workspaceID, projectID string) (model.Project, error) {
	project, err := s.repo.Get(ctx, s.db, workspaceID, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Project{}, fmt.Errorf("%w: project", ErrNotFound)
		}
		return model.Project{}, err
	}
	return project, nil
}

func (s *ProjectService) Plans(ctx context.Context, workspaceID, projectID string) ([]model.Plan, error) {
	if _, err := s.repo.Name(ctx, s.db, workspaceID, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: project", ErrNotFound)
		}
		return nil, err
	}
	return s.repo.ListPlans(ctx, s.db, workspaceID, projectID)
}

// CreatePlan adds a whole-project plan with the default statuses. Plan names
// are unique per project among non-deleted plans.
func (s *ProjectService) CreatePlan(ctx context.Context, workspaceID, projectID string, in CreatePlanInput) (model.Plan, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return model.Plan{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	plan := model.Plan{
		ID:          model.NewID(),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
	}

	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.repo.Name(ctx, tx, workspaceID, projectID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: project", ErrNotFound)
			}
			return err
		}
		exists, err := s.repo.PlanNameExists(ctx, tx, workspaceID, projectID, name)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: plan name already used", ErrConflict)
		}

		now := nowUTC()
		if err := s.repo.InsertPlan(ctx, tx, plan, now); err != nil {
			return err
		}
		return s.insertDefaultStatuses(ctx, tx, plan, now)
	})
	if err != nil {
		return model.Plan{}, err
	}
	return plan, nil
}

func (s *ProjectService) insertDefaultStatuses(ctx context.Context, tx *sql.Tx, plan model.Plan, now string) error {
	pos := ""
	for _, d := range defaultStatuses {
		pos = position.After(pos)
		status := model.Status{
			ID:          model.NewID(),
			WorkspaceID: plan.WorkspaceID,
			PlanID:      plan.ID,
			Title:       d.title,
			Color:       d.color,
			Category:    d.category,
			Position:    pos,
		}
		if err := s.repo.InsertStatus(ctx, tx, status, now); err != nil {
			return err
		}
	}
	return nil
}
