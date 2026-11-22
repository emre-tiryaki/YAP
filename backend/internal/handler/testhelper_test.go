package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/github"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repofetch"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
	"github.com/emre-tiryaki/YAP/backend/internal/storage"
)

type testEnv struct {
	mux     *http.ServeMux
	session model.Session
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()
	ctx := context.Background()

	db, err := database.New(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("database.New() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	session, err := service.NewSessionService(db, repository.NewSessionRepository()).Bootstrap(ctx, "tester")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	projectRepo := repository.NewProjectRepository()
	projectSvc := service.NewProjectService(db, projectRepo, repository.NewSettingsRepository())
	taskRepo := repository.NewTaskRepository()
	taskSvc := service.NewTaskService(db, projectRepo, taskRepo, repository.NewTaskActivityRepository())
	commentSvc := service.NewCommentService(db, taskRepo, repository.NewCommentRepository())
	labelSvc := service.NewLabelService(db, projectRepo, repository.NewLabelRepository())

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", HelloWorld)
	NewWorkspaceHandler(session, service.NewWorkspaceService(db, repository.NewWorkspaceRepository())).Register(mux)
	NewProjectHandler(session, projectSvc).Register(mux)
	NewTaskHandler(session, taskSvc).Register(mux)
	NewCommentHandler(session, commentSvc).Register(mux)
	NewLabelHandler(session, labelSvc).Register(mux)

	store, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewFS() error = %v", err)
	}
	resourceSvc := service.NewResourceService(db, repository.NewResourceRepository(), taskRepo, store)
	NewResourceHandler(session, resourceSvc).Register(mux)
	// GitHub is not configured and nobody is connected; no request leaves
	// the test.
	githubSvc := service.NewGitHubService(github.New(""), memCreds{}, false)
	repoSvc := service.NewRepositoryService(db, repository.NewRepoRepository(), resourceSvc, githubSvc,
		repository.NewSettingsRepository(), store, repofetch.NewFetcher(&repofetch.Git{}, repofetch.DefaultLimits))
	NewGitHubHandler(session, githubSvc).Register(mux)
	NewRepositoryHandler(session, repoSvc).Register(mux)
	NewSettingsHandler(session, service.NewSettingsService(db, repository.NewSettingsRepository())).Register(mux)
	NewAccountHandler(session, service.NewAccountService(db, repository.NewAccountRepository(), repository.NewResourceRepository(),
		repository.NewWorkspaceRepository(), repository.NewSettingsRepository(), store, "tester")).Register(mux)
	NewSearchHandler(session, service.NewSearchService(db, repository.NewSearchRepository())).Register(mux)

	return testEnv{mux: mux, session: session}
}

func (e testEnv) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, status, rec.Body.String())
	}
}

func wantErrorCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	wantStatus(t, rec, status)
	if got := decode[errorResponse](t, rec).Error; got != code {
		t.Errorf("error code = %q, want %q", got, code)
	}
}

// memCreds is a keychain that never holds a credential.
type memCreds struct{}

func (memCreds) Get(string) (model.GitHubCredential, error) {
	return model.GitHubCredential{}, model.ErrNoCredential
}
func (memCreds) Set(string, model.GitHubCredential) error { return nil }
func (memCreds) Delete(string) error                      { return nil }
