package service

import (
	"context"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

// aiEnv wires the AI services to real repositories and a fake model.
type aiEnv struct {
	repoEnv
	projects *ProjectService
	tasks    *TaskService
	comments *CommentService
	labels   *LabelService
	aiRepo   *repository.AIRepository
	keys     *memAIKeys
	llm      *fakeLLM
	conns    *AIConnectionService
	prompt   *aiPrompt
	chats    *AIChatService
}

func newAIEnv(t *testing.T) aiEnv {
	t.Helper()
	e := aiEnv{repoEnv: newRepoEnv(t)}
	projectRepo, taskRepo := repository.NewProjectRepository(), repository.NewTaskRepository()
	activity, commentRepo := repository.NewTaskActivityRepository(), repository.NewCommentRepository()
	e.projects = NewProjectService(e.db, projectRepo, repository.NewSettingsRepository())
	e.tasks = NewTaskService(e.db, projectRepo, taskRepo, activity)
	e.comments = NewCommentService(e.db, taskRepo, commentRepo)
	e.labels = NewLabelService(e.db, projectRepo, repository.NewLabelRepository())
	e.aiRepo = repository.NewAIRepository()
	e.keys = newMemAIKeys()
	e.llm = &fakeLLM{models: []string{"m1"}}
	e.conns = NewAIConnectionService(e.db, e.aiRepo, e.keys, e.llm)
	src := AIContextSources{
		Tasks: taskRepo, Activity: activity, Comments: commentRepo, Projects: projectRepo,
		Resources: e.svc, Repos: e.repos, Placement: e.aiRepo,
	}
	e.prompt = newAIPrompt(e.db, src)
	e.chats = NewAIChatService(e.db, e.aiRepo, e.conns, e.prompt, repository.NewSettingsRepository(), e.llm)
	e.chats.Start(context.Background())
	t.Cleanup(e.chats.Wait)
	return e
}

func (e aiEnv) connection(t *testing.T) model.AIConnection {
	t.Helper()
	c, err := e.conns.Create(e.ctx, e.session, CreateAIConnectionInput{Name: "Local", Preset: model.AIPresetOllama, BaseURL: "http://localhost:11434/v1"})
	if err != nil {
		t.Fatalf("connection Create() error = %v", err)
	}
	return c
}

func (e aiEnv) project(t *testing.T, name string) model.Project {
	t.Helper()
	p, err := e.projects.Create(e.ctx, e.session.WorkspaceID, CreateProjectInput{Name: name})
	if err != nil {
		t.Fatalf("project Create() error = %v", err)
	}
	return p
}

func (e aiEnv) task(t *testing.T, projectID string, in CreateTaskInput) model.Task {
	t.Helper()
	task, err := e.tasks.Create(e.ctx, e.session, projectID, in)
	if err != nil {
		t.Fatalf("task Create() error = %v", err)
	}
	return task
}

func (e aiEnv) planOf(t *testing.T, projectID string) string {
	t.Helper()
	var id string
	if err := e.db.QueryRow(`SELECT id FROM plans WHERE project_id = ? ORDER BY created_at, id LIMIT 1`, projectID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e aiEnv) aiSettings() SettingsRepository { return repository.NewSettingsRepository() }
