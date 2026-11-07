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

const upcomingTaskLimit = 10

type TaskRepository interface {
	FirstStatusID(ctx context.Context, q database.DBTX, workspaceID, planID string, category model.StatusCategory) (string, error)
	LastPosition(ctx context.Context, q database.DBTX, workspaceID, planID, statusID string) (string, error)
	Insert(ctx context.Context, q database.DBTX, t model.Task, now string) error
	InsertAssignee(ctx context.Context, q database.DBTX, workspaceID, taskID, membershipID, now string) error
	ListUpcoming(ctx context.Context, q database.DBTX, workspaceID, membershipID string, limit int) ([]model.Task, error)
}

type CreateTaskInput struct {
	Title       string
	Description string
	DueDate     *time.Time
	PlanID      string
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

		statusID, err := s.tasks.FirstStatusID(ctx, tx, session.WorkspaceID, planID, model.StatusToDo)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: plan has no todo status", ErrNotFound)
			}
			return err
		}
		last, err := s.tasks.LastPosition(ctx, tx, session.WorkspaceID, planID, statusID)
		if err != nil {
			return err
		}

		task.ProjectName = projectName
		task.PlanID = planID
		task.StatusID = statusID
		task.StatusCategory = model.StatusToDo
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