package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type CommentRepository struct{}

func NewCommentRepository() *CommentRepository {
	return &CommentRepository{}
}

func (*CommentRepository) Insert(ctx context.Context, q database.DBTX, c model.Comment, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO task_comments (id, workspace_id, task_id, author_id, body, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.WorkspaceID, c.TaskID, c.AuthorID, c.Body, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

const selectComment = `SELECT c.id, c.task_id, c.author_id, u.display_name, c.body, c.created_at, c.updated_at
	 FROM task_comments c
	 JOIN memberships m ON m.id = c.author_id AND m.workspace_id = c.workspace_id
	 JOIN users u ON u.id = m.user_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanComment(row rowScanner, workspaceID string) (model.Comment, error) {
	c := model.Comment{WorkspaceID: workspaceID}
	var createdAt, updatedAt string
	if err := row.Scan(&c.ID, &c.TaskID, &c.AuthorID, &c.AuthorName, &c.Body, &createdAt, &updatedAt); err != nil {
		return model.Comment{}, err
	}
	var err error
	if c.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return model.Comment{}, fmt.Errorf("parse created_at: %w", err)
	}
	if c.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return model.Comment{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return c, nil
}

// ListByTask returns the non-deleted comments of a task, oldest first.
func (*CommentRepository) ListByTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.Comment, error) {
	rows, err := q.QueryContext(ctx,
		selectComment+`
		 WHERE c.workspace_id = ? AND c.task_id = ? AND c.deleted_at IS NULL
		 ORDER BY c.created_at, c.id`,
		workspaceID, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	comments := []model.Comment{}
	for rows.Next() {
		c, err := scanComment(rows, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return comments, nil
}

// Get returns a non-deleted comment whose task is not deleted either.
func (*CommentRepository) Get(ctx context.Context, q database.DBTX, workspaceID, commentID string) (model.Comment, error) {
	row := q.QueryRowContext(ctx,
		selectComment+`
		 JOIN tasks t ON t.id = c.task_id AND t.workspace_id = c.workspace_id
		 WHERE c.id = ? AND c.workspace_id = ? AND c.deleted_at IS NULL AND t.deleted_at IS NULL`,
		commentID, workspaceID,
	)
	c, err := scanComment(row, workspaceID)
	if err != nil {
		return model.Comment{}, fmt.Errorf("get comment: %w", err)
	}
	return c, nil
}

func (*CommentRepository) UpdateBody(ctx context.Context, q database.DBTX, workspaceID, commentID, body, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE task_comments SET body = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		body, now, commentID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	return nil
}

func (*CommentRepository) SoftDelete(ctx context.Context, q database.DBTX, workspaceID, commentID, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE task_comments SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		now, now, commentID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}
