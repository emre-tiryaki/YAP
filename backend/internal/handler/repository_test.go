package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repofetch"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
	"github.com/emre-tiryaki/YAP/backend/internal/storage"
	"github.com/emre-tiryaki/YAP/backend/internal/testdb"
)

// repoAPI is a one-repository stand-in for GitHub.
type repoAPI struct {
	repo    model.GitHubRepo
	archive []byte
	commits []model.GitHubCommit
}

func (a *repoAPI) Configured() bool { return true }
func (a *repoAPI) StartDeviceFlow(context.Context) (model.DeviceCode, error) {
	return model.DeviceCode{}, nil
}
func (a *repoAPI) PollDeviceToken(context.Context, string) (string, string, time.Duration, error) {
	return "", "", 0, nil
}
func (a *repoAPI) User(context.Context, string) (model.GitHubUser, error) {
	return model.GitHubUser{ID: 1, Login: "octo"}, nil
}
func (a *repoAPI) Repos(context.Context, string) ([]model.GitHubRepo, error) {
	return []model.GitHubRepo{a.repo}, nil
}
func (a *repoAPI) Repo(_ context.Context, _, fullName string) (model.GitHubRepo, error) {
	if fullName != a.repo.FullName {
		return model.GitHubRepo{}, model.ErrGitHubNotFound
	}
	return a.repo, nil
}
func (a *repoAPI) RepoByID(context.Context, string, int64) (model.GitHubRepo, error) {
	return a.repo, nil
}
func (a *repoAPI) BranchHead(context.Context, string, string, string) (string, error) {
	return "sha-1", nil
}
func (a *repoAPI) Zipball(context.Context, string, string, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(a.archive)), nil
}
func (a *repoAPI) Commits(_ context.Context, _, _, _ string, page int) ([]model.GitHubCommit, error) {
	if page > 1 {
		return []model.GitHubCommit{}, nil
	}
	return a.commits, nil
}

type tokenCreds struct{}

func (tokenCreds) Get(string) (model.GitHubCredential, error) {
	return model.GitHubCredential{Token: "t"}, nil
}
func (tokenCreds) Set(string, model.GitHubCredential) error { return nil }
func (tokenCreds) Delete(string) error                      { return nil }

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create("octo-hello-sha1/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// repoEnv serves the repository routes against a fake GitHub that holds
// octo/hello.
type repoEnv struct {
	mux  *http.ServeMux
	svc  *service.RepositoryService
	api  *repoAPI
	test testEnv
}

func newRepoHandlerEnv(t *testing.T, git *repofetch.Git, api *repoAPI) repoEnv {
	t.Helper()
	ctx := context.Background()
	db, err := database.New(ctx, testdb.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	session, err := service.NewSessionService(db, repository.NewSessionRepository()).Bootstrap(ctx, "tester")
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	resources := service.NewResourceService(db, repository.NewResourceRepository(), repository.NewTaskRepository(), store)
	fetcher := repofetch.NewFetcher(git, repofetch.DefaultLimits)
	gh := service.NewGitHubService(api, tokenCreds{}, fetcher.GitAvailable())
	svc := service.NewRepositoryService(db, repository.NewRepoRepository(), resources, gh,
		repository.NewSettingsRepository(), store, fetcher)
	t.Cleanup(svc.Wait)
	mux := http.NewServeMux()
	NewRepositoryHandler(session, svc).Register(mux)
	NewResourceHandler(session, resources).Register(mux)
	return repoEnv{mux: mux, svc: svc, api: api, test: testEnv{mux: mux, session: session}}
}

func newHello(t *testing.T) *repoAPI {
	return &repoAPI{
		repo:    model.GitHubRepo{ID: 42, FullName: "octo/hello", HTMLURL: "https://github.com/octo/hello", DefaultBranch: "main"},
		archive: zipOf(t, map[string]string{"README.md": "# Hello", "src/main.go": "package main"}),
		commits: []model.GitHubCommit{{
			SHA: "abc", Parents: []string{"p1"}, Message: "first", AuthorName: "Octo", AuthorLogin: "octo",
			AvatarURL: "https://a/x.png", Date: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), HTMLURL: "https://github.com/octo/hello/commit/abc",
		}},
	}
}

func (e repoEnv) add(t *testing.T) resourceResponse {
	t.Helper()
	rec := e.test.do(t, "POST", "/api/resources/repositories", `{"full_name":"octo/hello"}`)
	wantStatus(t, rec, http.StatusCreated)
	res := decode[resourceResponse](t, rec)
	e.svc.Wait()
	return res
}

func TestRepositoryHappyPath(t *testing.T) {
	e := newRepoHandlerEnv(t, &repofetch.Git{}, newHello(t))
	res := e.add(t)
	base := "/api/resources/" + res.ID + "/repository"

	rec := e.test.do(t, "GET", base, "")
	wantStatus(t, rec, http.StatusOK)
	repo := decode[repositoryResponse](t, rec)
	if repo.ResourceID != res.ID || repo.GitHubID != 42 || repo.FullName != "octo/hello" || repo.Branch != "main" ||
		repo.State != "ready" || repo.CommitSHA == nil || *repo.CommitSHA != "sha-1" || repo.FetchedAt == nil ||
		repo.Error != nil || repo.Private {
		t.Fatalf("repository = %+v", repo)
	}

	rec = e.test.do(t, "GET", base+"/entries", "")
	wantStatus(t, rec, http.StatusOK)
	entries := decode[[]repoEntryResponse](t, rec)
	if len(entries) != 2 || entries[0].Name != "src" || entries[0].Kind != "folder" || entries[1].Name != "README.md" || !entries[1].IsText {
		t.Fatalf("entries = %+v", entries)
	}
	if got := decode[[]repoEntryResponse](t, e.test.do(t, "GET", base+"/entries?path=src", "")); len(got) != 1 || got[0].Path != "src/main.go" {
		t.Fatalf("sub entries = %+v", got)
	}
	wantErrorCode(t, e.test.do(t, "GET", base+"/entries?path=../etc", ""), http.StatusBadRequest, "invalid_input")

	rec = e.test.do(t, "GET", base+"/content?path=README.md", "")
	wantStatus(t, rec, http.StatusOK)
	if rec.Body.String() != "# Hello" {
		t.Errorf("content = %q", rec.Body.String())
	}
	wantErrorCode(t, e.test.do(t, "GET", base+"/content?path=nope.md", ""), http.StatusNotFound, "not_found")

	rec = e.test.do(t, "POST", base+"/sync", "")
	wantStatus(t, rec, http.StatusAccepted)
	e.svc.Wait()
}

func TestRepositoryCommitsEndpoint(t *testing.T) {
	e := newRepoHandlerEnv(t, &repofetch.Git{}, newHello(t))
	res := e.add(t)
	base := "/api/resources/" + res.ID + "/repository/commits"

	rec := e.test.do(t, "GET", base, "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[[]commitResponse](t, rec)
	if len(got) != 1 || got[0].SHA != "abc" || got[0].Message != "first" || got[0].AuthorLogin != "octo" ||
		len(got[0].Parents) != 1 || got[0].Date != "2026-01-02T03:04:05Z" {
		t.Fatalf("commits = %+v", got)
	}
	if got := decode[[]commitResponse](t, e.test.do(t, "GET", base+"?page=2", "")); len(got) != 0 {
		t.Fatalf("page 2 = %+v", got)
	}
	wantErrorCode(t, e.test.do(t, "GET", base+"?page=x", ""), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.test.do(t, "GET", "/api/resources/nope/repository/commits", ""), http.StatusNotFound, "not_found")
}

func TestRepositoryCloneEndpoints(t *testing.T) {
	git := repofetch.NewGit()
	requireGit(t, git)
	src := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = src
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
			"GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "--quiet", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("# Git"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "--quiet", "-m", "init")

	api := newHello(t)
	api.repo.CloneURL = src
	e := newRepoHandlerEnv(t, git, api)
	res := e.add(t)
	base := "/api/resources/" + res.ID + "/repository/clones"

	if got := decode[[]cloneResponse](t, e.test.do(t, "GET", base, "")); len(got) != 0 {
		t.Fatalf("clones = %+v", got)
	}
	parent := t.TempDir()
	rec := e.test.do(t, "POST", base, `{"parent_path":"`+parent+`","name":"hello"}`)
	wantStatus(t, rec, http.StatusCreated)
	clone := decode[cloneResponse](t, rec)
	if clone.Path != filepath.Join(parent, "hello") || !clone.Exists || clone.ID == "" || clone.CreatedAt == "" {
		t.Fatalf("clone = %+v", clone)
	}
	if got := decode[[]cloneResponse](t, e.test.do(t, "GET", base, "")); len(got) != 1 || got[0].ID != clone.ID {
		t.Fatalf("clones = %+v", got)
	}

	wantErrorCode(t, e.test.do(t, "DELETE", base+"/nope", ""), http.StatusNotFound, "not_found")
	wantStatus(t, e.test.do(t, "DELETE", base+"/"+clone.ID, ""), http.StatusNoContent)
	if _, err := os.Stat(clone.Path); err != nil {
		t.Errorf("forgetting a clone must leave the folder alone: %v", err)
	}
	if got := decode[[]cloneResponse](t, e.test.do(t, "GET", base, "")); len(got) != 0 {
		t.Fatalf("clones after forget = %+v", got)
	}
}

func TestRepositoryAddErrors(t *testing.T) {
	e := newRepoHandlerEnv(t, &repofetch.Git{}, newHello(t))
	tests := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"bad json", `{`, 400, "invalid_input"},
		{"empty name", `{"full_name":""}`, 400, "invalid_input"},
		{"unknown repo", `{"full_name":"octo/nope"}`, 404, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, e.test.do(t, "POST", "/api/resources/repositories", tt.body), tt.status, tt.code)
		})
	}
	e.add(t)
	wantErrorCode(t, e.test.do(t, "POST", "/api/resources/repositories", `{"full_name":"octo/hello"}`), http.StatusConflict, "conflict")
}

// requireGit skips when git is missing, except in CI where a missing git
// would hide the clone tests, so it fails there.
func requireGit(t *testing.T, git *repofetch.Git) {
	t.Helper()
	if git.Available() {
		return
	}
	if os.Getenv("CI") != "" {
		t.Fatal("git is required in CI")
	}
	t.Skip("git is not installed")
}
