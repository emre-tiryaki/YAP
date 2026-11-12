package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

type commentEnv struct {
	taskEnv
	comments *CommentService
	repo     *repository.CommentRepository
}

func newCommentEnv(t *testing.T) commentEnv {
	t.Helper()
	env := newTaskEnv(t)
	repo := repository.NewCommentRepository()
	return commentEnv{
		taskEnv:  env,
		comments: NewCommentService(env.db, repository.NewTaskRepository(), repo),
		repo:     repo,
	}
}

func (e commentEnv) comment(t *testing.T, taskID, body string) model.Comment {
	t.Helper()
	c, err := e.comments.Create(context.Background(), e.session, taskID, body)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", body, err)
	}
	return c
}

// foreignComment stores a comment written by another member of the workspace.
func (e commentEnv) foreignComment(t *testing.T, taskID string) model.Comment {
	t.Helper()
	now := nowUTC()
	userID, membershipID := model.NewID(), model.NewID()
	if _, err := e.db.Exec(`INSERT INTO users (id, display_name, created_at, updated_at) VALUES (?, 'other', ?, ?)`, userID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(`INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at) VALUES (?, ?, ?, 'member', ?, ?)`,
		membershipID, e.session.WorkspaceID, userID, now, now); err != nil {
		t.Fatal(err)
	}
	c := model.Comment{ID: model.NewID(), WorkspaceID: e.session.WorkspaceID, TaskID: taskID, AuthorID: membershipID, Body: "theirs"}
	if err := e.repo.Insert(context.Background(), e.db, c, now); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCommentCreateAndList(t *testing.T) {
	ctx := context.Background()
	env := newCommentEnv(t)
	project := env.project(t, "Alpha")
	task := env.createTasks(t, project.ID, "task")[0]

	first := env.comment(t, task.ID, "  first\nline two  ")
	if first.Body != "first\nline two" || first.TaskID != task.ID || first.AuthorID != env.session.MembershipID ||
		first.AuthorName != env.session.DisplayName || first.CreatedAt.IsZero() {
		t.Errorf("Create() = %+v", first)
	}
	second := env.comment(t, task.ID, "second")

	got, err := env.comments.List(ctx, env.session, task.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Errorf("List() = %+v, want [first, second]", got)
	}

	planID := env.planOf(t, project.ID)
	listed, _ := env.boardTask(t, project.ID, planID, task.ID)
	if listed.CommentCount != 2 {
		t.Errorf("CommentCount = %d, want 2", listed.CommentCount)
	}
}

func TestCommentCreateErrors(t *testing.T) {
	env := newCommentEnv(t)
	project := env.project(t, "Alpha")
	task := env.createTasks(t, project.ID, "task")[0]
	gone := env.createTasks(t, project.ID, "gone")[0]
	if err := env.tasks.Delete(context.Background(), env.session, gone.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		taskID string
		body   string
		want   error
	}{
		{"blank body", task.ID, " \n ", ErrInvalidInput},
		{"too long", task.ID, strings.Repeat("ç", maxCommentLength+1), ErrInvalidInput},
		{"unknown task", model.NewID(), "hi", ErrNotFound},
		{"deleted task", gone.ID, "hi", ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.comments.Create(context.Background(), env.session, tt.taskID, tt.body)
			if !errors.Is(err, tt.want) {
				t.Errorf("Create() error = %v, want %v", err, tt.want)
			}
		})
	}

	if _, err := env.comments.Create(context.Background(), env.session, task.ID, strings.Repeat("ç", maxCommentLength)); err != nil {
		t.Errorf("Create() at the length limit error = %v", err)
	}
}

func TestCommentUpdate(t *testing.T) {
	ctx := context.Background()
	env := newCommentEnv(t)
	project := env.project(t, "Alpha")
	task := env.createTasks(t, project.ID, "task")[0]
	own := env.comment(t, task.ID, "draft")
	theirs := env.foreignComment(t, task.ID)

	got, err := env.comments.Update(ctx, env.session, own.ID, " final ")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Body != "final" || got.ID != own.ID || got.UpdatedAt.Before(own.UpdatedAt) {
		t.Errorf("Update() = %+v", got)
	}
	stored := env.nullableColumn(t, `SELECT body FROM task_comments WHERE id = ?`, own.ID)
	if stored == nil || *stored != "final" {
		t.Errorf("stored body = %v, want final", stored)
	}

	tests := []struct {
		name      string
		commentID string
		body      string
		want      error
	}{
		{"blank body", own.ID, "  ", ErrInvalidInput},
		{"unknown comment", model.NewID(), "x", ErrNotFound},
		{"another member's comment", theirs.ID, "x", ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := env.comments.Update(ctx, env.session, tt.commentID, tt.body); !errors.Is(err, tt.want) {
				t.Errorf("Update() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCommentDelete(t *testing.T) {
	ctx := context.Background()
	env := newCommentEnv(t)
	project := env.project(t, "Alpha")
	task := env.createTasks(t, project.ID, "task")[0]
	own := env.comment(t, task.ID, "mine")
	theirs := env.foreignComment(t, task.ID)

	if err := env.comments.Delete(ctx, env.session, theirs.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("Delete(theirs) error = %v, want ErrForbidden", err)
	}
	if err := env.comments.Delete(ctx, env.session, own.ID); err != nil {
		t.Fatalf("Delete(own) error = %v", err)
	}
	if env.nullableColumn(t, `SELECT deleted_at FROM task_comments WHERE id = ?`, own.ID) == nil {
		t.Error("deleted_at not set, want a soft delete")
	}
	if err := env.comments.Delete(ctx, env.session, own.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete() error = %v, want ErrNotFound", err)
	}

	got, err := env.comments.List(ctx, env.session, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != theirs.ID {
		t.Errorf("List() = %+v, want only the other member's comment", got)
	}
}

// Every way of deleting a task also marks its comments (and its subtasks'
// comments) as deleted, so each row carries its own deletion record.
func TestCommentsDeletedWithTheirTask(t *testing.T) {
	tests := []struct {
		name   string
		delete func(t *testing.T, env commentEnv, project model.Project, planID string, task model.Task)
	}{
		{"task", func(t *testing.T, env commentEnv, _ model.Project, _ string, task model.Task) {
			if err := env.tasks.Delete(context.Background(), env.session, task.ID); err != nil {
				t.Fatal(err)
			}
		}},
		{"status", func(t *testing.T, env commentEnv, project model.Project, planID string, task model.Task) {
			if err := env.projects.DeleteStatus(context.Background(), env.session.WorkspaceID, project.ID, planID, task.StatusID); err != nil {
				t.Fatal(err)
			}
		}},
		{"plan", func(t *testing.T, env commentEnv, project model.Project, planID string, _ model.Task) {
			env.createPlan(t, project.ID, "Spare")
			if err := env.projects.DeletePlan(context.Background(), env.session.WorkspaceID, project.ID, planID); err != nil {
				t.Fatal(err)
			}
		}},
		{"project", func(t *testing.T, env commentEnv, project model.Project, _ string, _ model.Task) {
			if err := env.projects.Delete(context.Background(), env.session.WorkspaceID, project.ID); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newCommentEnv(t)
			project := env.project(t, "Alpha")
			planID := env.planOf(t, project.ID)
			task := env.createTasks(t, project.ID, "task")[0]
			sub := env.subtask(t, task.ID, "sub")
			onTask := env.comment(t, task.ID, "on task")
			onSub := env.comment(t, sub.ID, "on subtask")

			tt.delete(t, env, project, planID, task)

			for _, c := range []model.Comment{onTask, onSub} {
				if env.nullableColumn(t, `SELECT deleted_at FROM task_comments WHERE id = ?`, c.ID) == nil {
					t.Errorf("comment %q not deleted", c.Body)
				}
			}
		})
	}
}
