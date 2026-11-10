package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
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
	GetPlan(ctx context.Context, q database.DBTX, workspaceID, projectID, planID string) (model.Plan, error)
	ListStatuses(ctx context.Context, q database.DBTX, workspaceID, planID string) ([]model.Status, error)
	LastStatusPosition(ctx context.Context, q database.DBTX, workspaceID, planID string) (string, error)
	Update(ctx context.Context, q database.DBTX, workspaceID, projectID, name, description, now string) error
	SoftDelete(ctx context.Context, q database.DBTX, workspaceID, projectID, now string) error
	UpdatePlan(ctx context.Context, q database.DBTX, workspaceID, planID, name, description, now string) error
	SoftDeletePlan(ctx context.Context, q database.DBTX, workspaceID, planID, now string) error
	PlanNameTaken(ctx context.Context, q database.DBTX, workspaceID, projectID, teamID, name, excludePlanID string) (bool, error)
	CountWholeProjectPlans(ctx context.Context, q database.DBTX, workspaceID, projectID string) (int, error)
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

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

type CreateStatusInput struct {
	Title    string
	Color    string
	Category model.StatusCategory
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

func (s *ProjectService) GetPlan(ctx context.Context, workspaceID, projectID, planID string) (model.Plan, error) {
	plan, err := s.repo.GetPlan(ctx, s.db, workspaceID, projectID, planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Plan{}, fmt.Errorf("%w: plan", ErrNotFound)
		}
		return model.Plan{}, err
	}
	return plan, nil
}

func (s *ProjectService) Statuses(ctx context.Context, workspaceID, projectID, planID string) ([]model.Status, error) {
	if err := s.requirePlan(ctx, s.db, workspaceID, projectID, planID); err != nil {
		return nil, err
	}
	return s.repo.ListStatuses(ctx, s.db, workspaceID, planID)
}

// CreateStatus appends a status after the plan's last column. Any category is
// allowed; a task counts as finished while it sits in a done-category status.
func (s *ProjectService) CreateStatus(ctx context.Context, workspaceID, projectID, planID string, in CreateStatusInput) (model.Status, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.Status{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if !colorPattern.MatchString(in.Color) {
		return model.Status{}, fmt.Errorf("%w: color must be #rrggbb", ErrInvalidInput)
	}
	if !validCategory(in.Category) {
		return model.Status{}, fmt.Errorf("%w: category", ErrInvalidInput)
	}

	status := model.Status{
		ID:          model.NewID(),
		WorkspaceID: workspaceID,
		PlanID:      planID,
		Title:       title,
		Color:       strings.ToLower(in.Color),
		Category:    in.Category,
	}
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requirePlan(ctx, tx, workspaceID, projectID, planID); err != nil {
			return err
		}
		last, err := s.repo.LastStatusPosition(ctx, tx, workspaceID, planID)
		if err != nil {
			return err
		}
		status.Position = position.After(last)
		return s.repo.InsertStatus(ctx, tx, status, nowUTC())
	})
	if err != nil {
		return model.Status{}, err
	}
	return status, nil
}

func (s *ProjectService) requirePlan(ctx context.Context, q database.DBTX, workspaceID, projectID, planID string) error {
	if _, err := s.repo.PlanID(ctx, q, workspaceID, projectID, planID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: plan", ErrNotFound)
		}
		return err
	}
	return nil
}

func validCategory(c model.StatusCategory) bool {
	switch c {
	case model.StatusToDo, model.StatusInProgress, model.StatusDone:
		return true
	}
	return false
}

// UpdateProjectInput carries the fields to change; nil fields are kept.
type UpdateProjectInput struct {
	Name        *string
	Description *string
}

// UpdatePlanInput carries the fields to change; nil fields are kept.
type UpdatePlanInput struct {
	Name        *string
	Description *string
}

// Update changes the name and/or description of a project.
func (s *ProjectService) Update(ctx context.Context, workspaceID, projectID string, in UpdateProjectInput) (model.Project, error) {
	if in.Name == nil && in.Description == nil {
		return model.Project{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return model.Project{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	var project model.Project
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		project, err = s.repo.Get(ctx, tx, workspaceID, projectID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: project", ErrNotFound)
			}
			return err
		}
		if in.Name != nil {
			project.Name = strings.TrimSpace(*in.Name)
		}
		if in.Description != nil {
			project.Description = strings.TrimSpace(*in.Description)
		}
		return s.repo.Update(ctx, tx, workspaceID, projectID, project.Name, project.Description, nowUTC())
	})
	if err != nil {
		return model.Project{}, err
	}
	return project, nil
}

// Delete soft-deletes a project together with its plans, statuses, tasks and
// team links in one transaction.
func (s *ProjectService) Delete(ctx context.Context, workspaceID, projectID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.repo.Name(ctx, tx, workspaceID, projectID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: project", ErrNotFound)
			}
			return err
		}
		return s.repo.SoftDelete(ctx, tx, workspaceID, projectID, nowUTC())
	})
}

// UpdatePlan changes the name and/or description of a plan. The new name must
// stay unique among the non-deleted plans of the same project and team scope.
func (s *ProjectService) UpdatePlan(ctx context.Context, workspaceID, projectID, planID string, in UpdatePlanInput) (model.Plan, error) {
	if in.Name == nil && in.Description == nil {
		return model.Plan{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return model.Plan{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	var plan model.Plan
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		plan, err = s.repo.GetPlan(ctx, tx, workspaceID, projectID, planID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: plan", ErrNotFound)
			}
			return err
		}
		if in.Name != nil {
			plan.Name = strings.TrimSpace(*in.Name)
			taken, err := s.repo.PlanNameTaken(ctx, tx, workspaceID, projectID, plan.TeamID, plan.Name, planID)
			if err != nil {
				return err
			}
			if taken {
				return fmt.Errorf("%w: plan name already used", ErrConflict)
			}
		}
		if in.Description != nil {
			plan.Description = strings.TrimSpace(*in.Description)
		}
		return s.repo.UpdatePlan(ctx, tx, workspaceID, planID, plan.Name, plan.Description, nowUTC())
	})
	if err != nil {
		return model.Plan{}, err
	}
	return plan, nil
}

// DeletePlan soft-deletes a plan together with its statuses and tasks. The
// project's last whole-project plan cannot be deleted, because tasks created
// without an explicit plan need a default plan to land in.
func (s *ProjectService) DeletePlan(ctx context.Context, workspaceID, projectID, planID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		plan, err := s.repo.GetPlan(ctx, tx, workspaceID, projectID, planID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: plan", ErrNotFound)
			}
			return err
		}
		if plan.TeamID == "" {
			n, err := s.repo.CountWholeProjectPlans(ctx, tx, workspaceID, projectID)
			if err != nil {
				return err
			}
			if n <= 1 {
				return fmt.Errorf("%w: project needs at least one whole-project plan", ErrConflict)
			}
		}
		return s.repo.SoftDeletePlan(ctx, tx, workspaceID, planID, nowUTC())
	})
}
