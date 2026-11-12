package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/position"
)

const upcomingTaskLimit = 50

type TaskRepository interface {
	FirstStatusID(ctx context.Context, q database.DBTX, workspaceID, planID string, category model.StatusCategory) (string, error)
	LastPosition(ctx context.Context, q database.DBTX, workspaceID, planID, statusID string) (string, error)
	Insert(ctx context.Context, q database.DBTX, t model.Task, now string) error
	InsertAssignee(ctx context.Context, q database.DBTX, workspaceID, taskID, membershipID, now string) error
	ListUpcoming(ctx context.Context, q database.DBTX, workspaceID, membershipID string, limit int) ([]model.Task, error)
	ListByPlan(ctx context.Context, q database.DBTX, workspaceID, planID string) ([]model.Task, error)
	Placement(ctx context.Context, q database.DBTX, workspaceID, taskID string) (planID, statusID string, err error)
	StatusCategory(ctx context.Context, q database.DBTX, workspaceID, planID, statusID string) (model.StatusCategory, error)
	PositionsInStatus(ctx context.Context, q database.DBTX, workspaceID, planID, statusID, excludeTaskID string) ([]string, error)
	UpdatePlacement(ctx context.Context, q database.DBTX, workspaceID, taskID, statusID, pos, now string) error
	SoftDeleteTree(ctx context.Context, q database.DBTX, workspaceID, taskID, now string) error
	Get(ctx context.Context, q database.DBTX, workspaceID, taskID string) (model.Task, error)
	Update(ctx context.Context, q database.DBTX, workspaceID, taskID, title, description string, dueDate *time.Time, now string) error
	ListSubtasks(ctx context.Context, q database.DBTX, workspaceID, parentID string) ([]model.Task, error)
	SetCompletedAt(ctx context.Context, q database.DBTX, workspaceID, taskID string, completedAt *time.Time, now string) error
}

type CreateTaskInput struct {
	Title       string
	Description string
	DueDate     *time.Time
	PlanID      string
	// StatusID is optional; empty means the plan's first todo status.
	StatusID string
}

type MoveTaskInput struct {
	StatusID string
	// Index is the task's position among the other tasks of the target
	// status; out-of-range values are clamped.
	Index int
}

type TaskService struct {
	db       *sql.DB
	projects ProjectRepository
	tasks    TaskRepository
}

func NewTaskService(db *sql.DB, projects ProjectRepository, tasks TaskRepository) *TaskService {
	return &TaskService{db: db, projects: projects, tasks: tasks}
}

func (s *TaskService) Upcoming(ctx context.Context, session model.Session) ([]model.Task, error) {
	return s.tasks.ListUpcoming(ctx, s.db, session.WorkspaceID, session.MembershipID, upcomingTaskLimit)
}

func (s *TaskService) Create(ctx context.Context, session model.Session, projectID string, in CreateTaskInput) (model.Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.Task{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	task := model.Task{
		ID:          model.NewID(),
		WorkspaceID: session.WorkspaceID,
		ProjectID:   projectID,
		CreatedBy:   session.MembershipID,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		DueDate:     in.DueDate,
	}

	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		projectName, err := s.projects.Name(ctx, tx, session.WorkspaceID, projectID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: project", ErrNotFound)
			}
			return err
		}

		planID, err := s.resolvePlanID(ctx, tx, session.WorkspaceID, projectID, strings.TrimSpace(in.PlanID))
		if err != nil {
			return err
		}

		statusID, category, err := s.resolveStatus(ctx, tx, session.WorkspaceID, planID, strings.TrimSpace(in.StatusID))
		if err != nil {
			return err
		}
		last, err := s.tasks.LastPosition(ctx, tx, session.WorkspaceID, planID, statusID)
		if err != nil {
			return err
		}

		task.ProjectName = projectName
		task.PlanID = planID
		task.StatusID = statusID
		task.StatusCategory = category
		task.Position = position.After(last)

		now := nowUTC()
		if err := s.tasks.Insert(ctx, tx, task, now); err != nil {
			return err
		}
		return s.tasks.InsertAssignee(ctx, tx, session.WorkspaceID, task.ID, session.MembershipID, now)
	})
	if err != nil {
		return model.Task{}, err
	}
	return task, nil
}

func (s *TaskService) resolvePlanID(ctx context.Context, q database.DBTX, workspaceID, projectID, planID string) (string, error) {
	var id string
	var err error
	if planID == "" {
		id, err = s.projects.DefaultPlanID(ctx, q, workspaceID, projectID)
	} else {
		id, err = s.projects.PlanID(ctx, q, workspaceID, projectID, planID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%w: plan", ErrNotFound)
		}
		return "", err
	}
	return id, nil
}
func (s *TaskService) resolveStatus(ctx context.Context, q database.DBTX, workspaceID, planID, statusID string) (string, model.StatusCategory, error) {
	if statusID == "" {
		id, err := s.tasks.FirstStatusID(ctx, q, workspaceID, planID, model.StatusToDo)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", fmt.Errorf("%w: plan has no todo status", ErrNotFound)
			}
			return "", "", err
		}
		return id, model.StatusToDo, nil
	}
	category, err := s.tasks.StatusCategory(ctx, q, workspaceID, planID, statusID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("%w: status is not in the plan", ErrInvalidInput)
		}
		return "", "", err
	}
	return statusID, category, nil
}

func (s *TaskService) ListByPlan(ctx context.Context, session model.Session, projectID, planID string) ([]model.Task, error) {
	if _, err := s.resolvePlanID(ctx, s.db, session.WorkspaceID, projectID, planID); err != nil {
		return nil, err
	}
	return s.tasks.ListByPlan(ctx, s.db, session.WorkspaceID, planID)
}

// Move places a task at in.Index inside in.StatusID of its own plan. Only the
// moved row is written; its position is the midpoint of its new neighbours.
func (s *TaskService) Move(ctx context.Context, session model.Session, taskID string, in MoveTaskInput) error {
	statusID := strings.TrimSpace(in.StatusID)
	if statusID == "" {
		return fmt.Errorf("%w: status_id is required", ErrInvalidInput)
	}
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		planID, _, err := s.tasks.Placement(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task", ErrNotFound)
			}
			return err
		}
		if _, _, err := s.resolveStatus(ctx, tx, session.WorkspaceID, planID, statusID); err != nil {
			return err
		}

		positions, err := s.tasks.PositionsInStatus(ctx, tx, session.WorkspaceID, planID, statusID, taskID)
		if err != nil {
			return err
		}
		index := min(max(in.Index, 0), len(positions))
		var lo, hi string
		if index > 0 {
			lo = positions[index-1]
		}
		if index < len(positions) {
			hi = positions[index]
		}
		pos, err := position.Between(lo, hi)
		if err != nil {
			return fmt.Errorf("%w: cannot place task between %q and %q", ErrConflict, lo, hi)
		}
		return s.tasks.UpdatePlacement(ctx, tx, session.WorkspaceID, taskID, statusID, pos, nowUTC())
	})
}

// Delete soft-deletes a task together with its subtasks and closes their
// assignments, so every affected row carries its own deletion record.
func (s *TaskService) Delete(ctx context.Context, session model.Session, taskID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, _, err := s.tasks.Placement(ctx, tx, session.WorkspaceID, taskID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task", ErrNotFound)
			}
			return err
		}
		return s.tasks.SoftDeleteTree(ctx, tx, session.WorkspaceID, taskID, nowUTC())
	})
}

// UpdateTaskInput carries the fields to change; nil fields are kept. The due
// date is changed only when DueDateSet is true, and a nil DueDate clears it.
type UpdateTaskInput struct {
	Title       *string
	Description *string
	DueDate     *time.Time
	DueDateSet  bool
}

// Update changes the title, description and/or due date of a task. Its
// placement (plan, status, position) is left alone; use Move for that.
func (s *TaskService) Update(ctx context.Context, session model.Session, taskID string, in UpdateTaskInput) (model.Task, error) {
	if in.Title == nil && in.Description == nil && !in.DueDateSet {
		return model.Task{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		return model.Task{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	var task model.Task
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		task, err = s.tasks.Get(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task", ErrNotFound)
			}
			return err
		}
		if in.Title != nil {
			task.Title = strings.TrimSpace(*in.Title)
		}
		if in.Description != nil {
			task.Description = strings.TrimSpace(*in.Description)
		}
		if in.DueDateSet {
			task.DueDate = nil
			if in.DueDate != nil {
				due := in.DueDate.UTC()
				task.DueDate = &due
			}
		}
		return s.tasks.Update(ctx, tx, session.WorkspaceID, taskID, task.Title, task.Description, task.DueDate, nowUTC())
	})
	if err != nil {
		return model.Task{}, err
	}
	return task, nil
}

// CreateSubtask adds a subtask to a top-level task. It goes into the parent's
// plan, in the plan's first todo status. Subtasks are one level deep: a
// subtask cannot have subtasks of its own, which also rules out cycles.
func (s *TaskService) CreateSubtask(ctx context.Context, session model.Session, parentID, title string) (model.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	task := model.Task{
		ID:          model.NewID(),
		WorkspaceID: session.WorkspaceID,
		ParentID:    parentID,
		CreatedBy:   session.MembershipID,
		Title:       title,
	}
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		parent, err := s.getTask(ctx, tx, session.WorkspaceID, parentID)
		if err != nil {
			return err
		}
		if parent.ParentID != "" {
			return fmt.Errorf("%w: a subtask cannot have subtasks", ErrInvalidInput)
		}

		statusID, category, err := s.resolveStatus(ctx, tx, session.WorkspaceID, parent.PlanID, "")
		if err != nil {
			return err
		}
		last, err := s.tasks.LastPosition(ctx, tx, session.WorkspaceID, parent.PlanID, statusID)
		if err != nil {
			return err
		}

		task.ProjectID = parent.ProjectID
		task.ProjectName = parent.ProjectName
		task.PlanID = parent.PlanID
		task.StatusID = statusID
		task.StatusCategory = category
		task.Position = position.After(last)

		now := nowUTC()
		if err := s.tasks.Insert(ctx, tx, task, now); err != nil {
			return err
		}
		return s.tasks.InsertAssignee(ctx, tx, session.WorkspaceID, task.ID, session.MembershipID, now)
	})
	if err != nil {
		return model.Task{}, err
	}
	return task, nil
}

// Subtasks lists the direct subtasks of a task, oldest first.
func (s *TaskService) Subtasks(ctx context.Context, session model.Session, parentID string) ([]model.Task, error) {
	if _, err := s.getTask(ctx, s.db, session.WorkspaceID, parentID); err != nil {
		return nil, err
	}
	return s.tasks.ListSubtasks(ctx, s.db, session.WorkspaceID, parentID)
}

// SetDone marks a task finished, or reopens it. Finishing is independent of
// the board: the task keeps its status and position, and moving a task into
// a done-category status does not finish it. A task already in the wanted
// state is left untouched.
func (s *TaskService) SetDone(ctx context.Context, session model.Session, taskID string, done bool) (model.Task, error) {
	var task model.Task
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		task, err = s.getTask(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			return err
		}
		if (task.CompletedAt != nil) == done {
			return nil
		}

		now := nowUTC()
		var completedAt *time.Time
		if done {
			parsed, err := time.Parse(time.RFC3339, now)
			if err != nil {
				return fmt.Errorf("parse now: %w", err)
			}
			completedAt = &parsed
		}
		if err := s.tasks.SetCompletedAt(ctx, tx, session.WorkspaceID, taskID, completedAt, now); err != nil {
			return err
		}
		task.CompletedAt = completedAt
		return nil
	})
	if err != nil {
		return model.Task{}, err
	}
	return task, nil
}

func (s *TaskService) getTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) (model.Task, error) {
	task, err := s.tasks.Get(ctx, q, workspaceID, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Task{}, fmt.Errorf("%w: task", ErrNotFound)
		}
		return model.Task{}, err
	}
	return task, nil
}
