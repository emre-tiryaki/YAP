package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// maxCommentLength is counted in code points.
const maxCommentLength = 10_000

type CommentRepository interface {
	Insert(ctx context.Context, q database.DBTX, c model.Comment, now string) error
	ListByTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.Comment, error)
	Get(ctx context.Context, q database.DBTX, workspaceID, commentID string) (model.Comment, error)
	UpdateBody(ctx context.Context, q database.DBTX, workspaceID, commentID, body, now string) error
	SoftDelete(ctx context.Context, q database.DBTX, workspaceID, commentID, now string) error
}

// CommentTaskLookup is the part of the task repository comments need: making
// sure the commented task exists and is not deleted.
type CommentTaskLookup interface {
	Placement(ctx context.Context, q database.DBTX, workspaceID, taskID string) (planID, statusID string, err error)
}

type CommentService struct {
	db       *sql.DB
	tasks    CommentTaskLookup
	comments CommentRepository
}

func NewCommentService(db *sql.DB, tasks CommentTaskLookup, comments CommentRepository) *CommentService {
	return &CommentService{db: db, tasks: tasks, comments: comments}
}

func normalizeCommentBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("%w: body is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(body) > maxCommentLength {
		return "", fmt.Errorf("%w: body is too long", ErrInvalidInput)
	}
	return body, nil
}

func (s *CommentService) requireTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) error {
	if _, _, err := s.tasks.Placement(ctx, q, workspaceID, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: task", ErrNotFound)
		}
		return err
	}
	return nil
}

func (s *CommentService) List(ctx context.Context, session model.Session, taskID string) ([]model.Comment, error) {
	if err := s.requireTask(ctx, s.db, session.WorkspaceID, taskID); err != nil {
		return nil, err
	}
	return s.comments.ListByTask(ctx, s.db, session.WorkspaceID, taskID)
}

func (s *CommentService) Create(ctx context.Context, session model.Session, taskID, body string) (model.Comment, error) {
	body, err := normalizeCommentBody(body)
	if err != nil {
		return model.Comment{}, err
	}

	var comment model.Comment
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requireTask(ctx, tx, session.WorkspaceID, taskID); err != nil {
			return err
		}
		id := model.NewID()
		if err := s.comments.Insert(ctx, tx, model.Comment{
			ID:          id,
			WorkspaceID: session.WorkspaceID,
			TaskID:      taskID,
			AuthorID:    session.MembershipID,
			Body:        body,
		}, nowUTC()); err != nil {
			return err
		}
		comment, err = s.comments.Get(ctx, tx, session.WorkspaceID, id)
		return err
	})
	if err != nil {
		return model.Comment{}, err
	}
	return comment, nil
}

// authored loads a comment and checks that the session's member wrote it.
func (s *CommentService) authored(ctx context.Context, q database.DBTX, session model.Session, commentID string) (model.Comment, error) {
	comment, err := s.comments.Get(ctx, q, session.WorkspaceID, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Comment{}, fmt.Errorf("%w: comment", ErrNotFound)
		}
		return model.Comment{}, err
	}
	if comment.AuthorID != session.MembershipID {
		return model.Comment{}, fmt.Errorf("%w: only the author can change a comment", ErrForbidden)
	}
	return comment, nil
}

// Update replaces the body of a comment; only its author may do so.
func (s *CommentService) Update(ctx context.Context, session model.Session, commentID, body string) (model.Comment, error) {
	body, err := normalizeCommentBody(body)
	if err != nil {
		return model.Comment{}, err
	}

	var comment model.Comment
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		comment, err = s.authored(ctx, tx, session, commentID)
		if err != nil {
			return err
		}
		now := nowUTC()
		if err := s.comments.UpdateBody(ctx, tx, session.WorkspaceID, commentID, body, now); err != nil {
			return err
		}
		comment.Body = body
		comment.UpdatedAt, err = time.Parse(time.RFC3339, now)
		return err
	})
	if err != nil {
		return model.Comment{}, err
	}
	return comment, nil
}

// Delete soft-deletes a comment; only its author may do so.
func (s *CommentService) Delete(ctx context.Context, session model.Session, commentID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.authored(ctx, tx, session, commentID); err != nil {
			return err
		}
		return s.comments.SoftDelete(ctx, tx, session.WorkspaceID, commentID, nowUTC())
	})
}
