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
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
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
	projectSvc := service.NewProjectService(db, projectRepo)
	taskRepo := repository.NewTaskRepository()
	taskSvc := service.NewTaskService(db, projectRepo, taskRepo)
	commentSvc := service.NewCommentService(db, taskRepo, repository.NewCommentRepository())

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", HelloWorld)
	NewWorkspaceHandler(session).Register(mux)
	NewProjectHandler(session, projectSvc).Register(mux)
	NewTaskHandler(session, taskSvc).Register(mux)
	NewCommentHandler(session, commentSvc).Register(mux)

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