package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubStatusNotConfigured(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(t, http.MethodGet, "/api/github", "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[githubStatusResponse](t, rec)
	if got.Configured || got.GitAvailable || got.Account != nil {
		t.Fatalf("status = %+v", got)
	}
	if body := rec.Body.String(); body != "{\"configured\":false,\"git_available\":false,\"account\":null}\n" {
		t.Fatalf("body = %s", body)
	}

	wantErrorCode(t, e.do(t, http.MethodPost, "/api/github/connect", ""), http.StatusServiceUnavailable, "github_not_configured")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/github/connect/poll", ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, e.do(t, http.MethodGet, "/api/github/repos", ""), http.StatusConflict, "github_not_connected")
	wantStatus(t, e.do(t, http.MethodDelete, "/api/github/account", ""), http.StatusNoContent)
}

func TestRepositoryEndpoints(t *testing.T) {
	e := newTestEnv(t)
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/resources/repositories", `{"full_name":"octo/hello"}`),
		http.StatusConflict, "github_not_connected")
	// The method choice is gone: clones go to a folder the user picks.
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/resources/repositories", `{"full_name":"octo/hello","method":"git"}`),
		http.StatusBadRequest, "invalid_input")

	rec := e.do(t, http.MethodPost, "/api/resources/folders", `{"name":"Plain"}`)
	wantStatus(t, rec, http.StatusCreated)
	folder := decode[map[string]any](t, rec)
	if folder["is_repository"] != false {
		t.Fatalf("folder = %v", folder)
	}
	id := folder["id"].(string)
	for _, path := range []string{"/repository", "/repository/entries", "/repository/content?path=a"} {
		wantErrorCode(t, e.do(t, http.MethodGet, "/api/resources/"+id+path, ""), http.StatusNotFound, "not_found")
	}
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/resources/"+id+"/repository/sync", ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, e.do(t, http.MethodGet, "/api/resources/"+id+"/repository/clones", ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/resources/"+id+"/repository/clones", `{"parent_path":"/tmp","name":"x"}`),
		http.StatusConflict, "git_missing")
}

func TestCloneNeedsJSON(t *testing.T) {
	e := newTestEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/api/resources/x/repository/clones", strings.NewReader(`{"parent_path":"/tmp","name":"x"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusUnsupportedMediaType)
}

func TestLocalFoldersEndpoint(t *testing.T) {
	e := newTestEnv(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "code"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec := e.do(t, http.MethodGet, "/api/local-folders?path="+url.QueryEscape(dir), "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[localFolderResponse](t, rec)
	if got.Path != dir || len(got.Folders) != 1 || got.Folders[0] != "code" || got.Parent == nil {
		t.Fatalf("folders = %+v", got)
	}
	wantErrorCode(t, e.do(t, http.MethodGet, "/api/local-folders?path=relative", ""), http.StatusBadRequest, "invalid_input")
}
