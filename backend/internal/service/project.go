package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

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
	UpdateStatusPosition(ctx context.Context, q database.DBTX, workspaceID, statusID, pos, now string) error
	UpdateStatus(ctx context.Context, q database.DBTX, workspaceID, statusID, title, description, color, now string) error
	Update(ctx context.Context, q database.DBTX, p model.Project, now string) error
	SoftDelete(ctx context.Context, q database.DBTX, workspaceID, projectID, now string) error
	UpdatePlan(ctx context.Context, q database.DBTX, workspaceID, planID, name, description, now string) error
	SoftDeletePlan(ctx context.Context, q database.DBTX, workspaceID, planID, now string) error
	SoftDeleteStatus(ctx context.Context, q database.DBTX, workspaceID, statusID, now string) error
	PlanNameTaken(ctx context.Context, q database.DBTX, workspaceID, projectID, teamID, name, excludePlanID string) (bool, error)
	CountWholeProjectPlans(ctx context.Context, q database.DBTX, workspaceID, projectID string) (int, error)
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var defaultStatuses = []struct {
	title       string
	description string
	color       string
	category    model.StatusCategory
}{
	{"To Do", "Tasks that are planned but not started yet.", "#9ca3af", model.StatusToDo},
	{"In Progress", "Tasks that are currently being worked on.", "#3b82f6", model.StatusInProgress},
	{"Done", "Finished tasks.", "#22c55e", model.StatusDone},
}

type CreateProjectInput struct {
	Name        string
	Description string
	Icon        string
}

type CreatePlanInput struct {
	Name        string
	Description string
}

type CreateStatusInput struct {
	Title       string
	Description string
	Color       string
	Category    model.StatusCategory
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
	icon, err := normalizeIcon(in.Icon)
	if err != nil {
		return model.Project{}, err
	}

	project := model.Project{
		ID:          model.NewID(),
		WorkspaceID: workspaceID,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		Icon:        icon,
	}

	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
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
			Description: d.description,
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
		Description: strings.TrimSpace(in.Description),
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

// MoveStatus places a status at index among the other statuses of its plan;
// out-of-range values are clamped. Only the moved row is written: its position
// becomes the midpoint of its new neighbours.
func (s *ProjectService) MoveStatus(ctx context.Context, workspaceID, projectID, planID, statusID string, index int) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requirePlan(ctx, tx, workspaceID, projectID, planID); err != nil {
			return err
		}
		statuses, err := s.repo.ListStatuses(ctx, tx, workspaceID, planID)
		if err != nil {
			return err
		}
		found := false
		positions := make([]string, 0, len(statuses))
		for _, st := range statuses {
			if st.ID == statusID {
				found = true
				continue
			}
			positions = append(positions, st.Position)
		}
		if !found {
			return fmt.Errorf("%w: status", ErrNotFound)
		}

		index = min(max(index, 0), len(positions))
		var lo, hi string
		if index > 0 {
			lo = positions[index-1]
		}
		if index < len(positions) {
			hi = positions[index]
		}
		pos, err := position.Between(lo, hi)
		if err != nil {
			return fmt.Errorf("%w: cannot place status between %q and %q", ErrConflict, lo, hi)
		}
		return s.repo.UpdateStatusPosition(ctx, tx, workspaceID, statusID, pos, nowUTC())
	})
}

// UpdateStatusInput carries the fields to change; nil fields are kept. The
// category is not editable here: changing it would have to guard the plan's
// last done-category status.
type UpdateStatusInput struct {
	Title       *string
	Description *string
	Color       *string
}

// UpdateStatus renames, redescribes and/or recolors a status of the plan.
func (s *ProjectService) UpdateStatus(ctx context.Context, workspaceID, projectID, planID, statusID string, in UpdateStatusInput) (model.Status, error) {
	if in.Title == nil && in.Description == nil && in.Color == nil {
		return model.Status{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		return model.Status{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if in.Color != nil && !colorPattern.MatchString(*in.Color) {
		return model.Status{}, fmt.Errorf("%w: color must be #rrggbb", ErrInvalidInput)
	}

	var status model.Status
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requirePlan(ctx, tx, workspaceID, projectID, planID); err != nil {
			return err
		}
		statuses, err := s.repo.ListStatuses(ctx, tx, workspaceID, planID)
		if err != nil {
			return err
		}
		found := false
		for _, st := range statuses {
			if st.ID == statusID {
				status, found = st, true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: status", ErrNotFound)
		}
		if in.Title != nil {
			status.Title = strings.TrimSpace(*in.Title)
		}
		if in.Description != nil {
			status.Description = strings.TrimSpace(*in.Description)
		}
		if in.Color != nil {
			status.Color = strings.ToLower(*in.Color)
		}
		return s.repo.UpdateStatus(ctx, tx, workspaceID, statusID, status.Title, status.Description, status.Color, nowUTC())
	})
	if err != nil {
		return model.Status{}, err
	}
	return status, nil
}

// DeleteStatus soft-deletes a status of the plan together with the tasks in
// it. The plan's last done-category status cannot be deleted, because a task
// counts as finished only while it sits in a done-category status.
func (s *ProjectService) DeleteStatus(ctx context.Context, workspaceID, projectID, planID, statusID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requirePlan(ctx, tx, workspaceID, projectID, planID); err != nil {
			return err
		}
		statuses, err := s.repo.ListStatuses(ctx, tx, workspaceID, planID)
		if err != nil {
			return err
		}
		var target *model.Status
		doneCount := 0
		for i, st := range statuses {
			if st.ID == statusID {
				target = &statuses[i]
			}
			if st.Category == model.StatusDone {
				doneCount++
			}
		}
		if target == nil {
			return fmt.Errorf("%w: status", ErrNotFound)
		}
		if target.Category == model.StatusDone && doneCount <= 1 {
			return fmt.Errorf("%w: plan needs at least one done status", ErrConflict)
		}
		return s.repo.SoftDeleteStatus(ctx, tx, workspaceID, statusID, nowUTC())
	})
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
	// Icon set to "" removes the icon.
	Icon *string
}

// UpdatePlanInput carries the fields to change; nil fields are kept.
type UpdatePlanInput struct {
	Name        *string
	Description *string
}

// Update changes the name, description and/or icon of a project.
func (s *ProjectService) Update(ctx context.Context, workspaceID, projectID string, in UpdateProjectInput) (model.Project, error) {
	if in.Name == nil && in.Description == nil && in.Icon == nil {
		return model.Project{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return model.Project{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	var icon string
	if in.Icon != nil {
		var err error
		if icon, err = normalizeIcon(*in.Icon); err != nil {
			return model.Project{}, err
		}
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
		if in.Icon != nil {
			project.Icon = icon
		}
		return s.repo.Update(ctx, tx, project, nowUTC())
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

// maxIconRunes bounds a project icon. A single emoji can span several code
// points (skin tones, ZWJ sequences, flags), so the limit is generous, but it
// keeps the icon from turning into a second name.
const maxIconRunes = 8

// normalizeIcon trims the icon and rejects anything that is not a short run
// of non-ASCII symbols. An empty result means "no icon".
func normalizeIcon(raw string) (string, error) {
	icon := strings.TrimSpace(raw)
	if icon == "" {
		return "", nil
	}
	if !utf8.ValidString(icon) || utf8.RuneCountInString(icon) > maxIconRunes {
		return "", fmt.Errorf("%w: icon", ErrInvalidInput)
	}
	for _, r := range icon {
		if r < utf8.RuneSelf || unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			return "", fmt.Errorf("%w: icon", ErrInvalidInput)
		}
	}
	return icon, nil
}
