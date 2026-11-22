package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// Activity returns a task's activity log, oldest first.
func (s *TaskService) Activity(ctx context.Context, session model.Session, taskID string) ([]model.TaskEvent, error) {
	if _, err := s.getTask(ctx, s.db, session.WorkspaceID, taskID); err != nil {
		return nil, err
	}
	return s.activity.ListEvents(ctx, s.db, session.WorkspaceID, taskID)
}

// Relations returns the relations of a task, each as the task reads it.
func (s *TaskService) Relations(ctx context.Context, session model.Session, taskID string) ([]model.TaskRelation, error) {
	if _, err := s.getTask(ctx, s.db, session.WorkspaceID, taskID); err != nil {
		return nil, err
	}
	return s.activity.ListRelations(ctx, s.db, session.WorkspaceID, taskID)
}

// AddRelation makes taskID block, relate to or duplicate relatedID. Both
// events (one per task) are logged in the same transaction.
func (s *TaskService) AddRelation(ctx context.Context, session model.Session, taskID, relatedID string, kind model.RelationKind) error {
	relatedID = strings.TrimSpace(relatedID)
	if !kind.Valid() {
		return fmt.Errorf("%w: relation kind", ErrInvalidInput)
	}
	if relatedID == "" || relatedID == taskID {
		return fmt.Errorf("%w: a task cannot be related to itself", ErrInvalidInput)
	}
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		projectID, title, err := s.activity.TaskSummary(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task", ErrNotFound)
			}
			return err
		}
		_, relatedTitle, err := s.activity.TaskSummary(ctx, tx, session.WorkspaceID, relatedID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: related task", ErrNotFound)
			}
			return err
		}
		exists, err := s.activity.RelationBetween(ctx, tx, session.WorkspaceID, taskID, relatedID)
		if err != nil {
			return err
		}
		if exists {
			return ErrRelationExists
		}
		now := nowUTC()
		if err := s.activity.InsertRelation(ctx, tx, session.WorkspaceID, projectID, taskID, relatedID, kind, now); err != nil {
			return err
		}
		mine, theirs := relationViews(kind)
		if err := s.recordEvent(ctx, tx, session, taskID, model.EventRelationAdded,
			map[string]string{"kind": string(mine), "task": relatedTitle}, now); err != nil {
			return err
		}
		return s.recordEvent(ctx, tx, session, relatedID, model.EventRelationAdded,
			map[string]string{"kind": string(theirs), "task": title}, now)
	})
}

// RemoveRelation closes the relation between two tasks, whichever of them
// holds it.
func (s *TaskService) RemoveRelation(ctx context.Context, session model.Session, taskID, relatedID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		_, title, err := s.activity.TaskSummary(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task", ErrNotFound)
			}
			return err
		}
		_, relatedTitle, err := s.activity.TaskSummary(ctx, tx, session.WorkspaceID, relatedID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: related task", ErrNotFound)
			}
			return err
		}
		kind, holder, err := s.activity.RelationKindBetween(ctx, tx, session.WorkspaceID, taskID, relatedID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: relation", ErrNotFound)
			}
			return err
		}
		now := nowUTC()
		if _, err := s.activity.RemoveRelation(ctx, tx, session.WorkspaceID, taskID, relatedID, now); err != nil {
			return err
		}
		mine, theirs := relationViews(kind)
		if holder != taskID {
			mine, theirs = theirs, mine
		}
		if err := s.recordEvent(ctx, tx, session, taskID, model.EventRelationRemoved,
			map[string]string{"kind": string(mine), "task": relatedTitle}, now); err != nil {
			return err
		}
		return s.recordEvent(ctx, tx, session, relatedID, model.EventRelationRemoved,
			map[string]string{"kind": string(theirs), "task": title}, now)
	})
}

// relationViews names a stored relation for its holder and for the other
// task.
func relationViews(kind model.RelationKind) (holder, other model.RelationView) {
	switch kind {
	case model.RelationBlocks:
		return model.ViewBlocking, model.ViewBlockedBy
	case model.RelationDuplicates:
		return model.ViewDuplicateOf, model.ViewDuplicatedBy
	default:
		return model.ViewRelates, model.ViewRelates
	}
}
