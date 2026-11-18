package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repofetch"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/storage"
)

type repoEnv struct {
	resourceEnv
	api      *fakeGitHub
	creds    *memCreds
	github   *GitHubService
	repos    *RepositoryService
	settings *SettingsService
	clock    *time.Time
}

func newRepoEnvWith(t *testing.T, git *repofetch.Git) repoEnv {
	t.Helper()
	db, session := newTestDB(t)
	root := t.TempDir()
	store, err := storage.NewFS(root)
	if err != nil {
		t.Fatalf("storage.NewFS() error = %v", err)
	}
	resources := NewResourceService(db, repository.NewResourceRepository(), repository.NewTaskRepository(), store)
	api, creds := newFakeGitHub(), newMemCreds()
	creds.m[session.UserID] = model.GitHubCredential{Token: "token-1"}
	fetcher := repofetch.NewFetcher(git, repofetch.DefaultLimits)
	gh := NewGitHubService(api, creds, fetcher.GitAvailable())
	settingsRepo := repository.NewSettingsRepository()
	repos := NewRepositoryService(db, repository.NewRepoRepository(), resources, gh, settingsRepo, store, fetcher)
	clock := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	e := repoEnv{
		resourceEnv: resourceEnv{db: db, session: session, svc: resources, root: root, ctx: context.Background()},
		api:         api, creds: creds, github: gh, repos: repos,
		settings: NewSettingsService(db, settingsRepo), clock: &clock,
	}
	repos.now = func() time.Time { return *e.clock }
	t.Cleanup(repos.Wait)
	return e
}

func newRepoEnv(t *testing.T) repoEnv {
	t.Helper()
	return newRepoEnvWith(t, repofetch.NewGit())
}

func (e repoEnv) advance(d time.Duration) {
	*e.clock = e.clock.Add(d)
}

var helloRepo = model.GitHubRepo{
	ID: 42, FullName: "octo/hello", HTMLURL: "https://github.com/octo/hello", DefaultBranch: "main",
}

func (e repoEnv) addHello(t *testing.T, parentID string) model.Resource {
	t.Helper()
	e.api.addRepo(helloRepo, "sha-1", makeZip(t, "octo-hello-sha1", map[string]string{
		"README.md":   "# Hello",
		"src/main.go": "package main",
		"logo.bin":    "\x00\x01\x02",
	}))
	r, err := e.repos.Add(e.ctx, e.session, AddRepositoryInput{
		ParentID: parentID, FullName: "octo/hello", OnConflict: ConflictFail,
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	e.repos.Wait()
	return r
}

func (e repoEnv) get(t *testing.T, id string) model.Repository {
	t.Helper()
	r, err := e.repos.Get(e.ctx, e.session, id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return r
}

func (e repoEnv) entryNames(t *testing.T, id, dir string) []string {
	t.Helper()
	entries, err := e.repos.Entries(e.ctx, e.session, id, dir)
	if err != nil {
		t.Fatalf("Entries(%q) error = %v", dir, err)
	}
	names := []string{}
	for _, en := range entries {
		names = append(names, en.Name)
	}
	return names
}

func (e repoEnv) readRepoFile(t *testing.T, id, p string) string {
	t.Helper()
	_, f, err := e.repos.OpenFile(e.ctx, e.session, id, p)
	if err != nil {
		t.Fatalf("OpenFile(%q) error = %v", p, err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAddRepositoryFetchesZip(t *testing.T) {
	e := newRepoEnv(t)
	docs := e.folder(t, "", "Docs")
	r := e.addHello(t, docs.ID)

	if r.Name != "hello" || r.Kind != model.ResourceFolder || !r.IsRepository || r.ParentID != docs.ID {
		t.Fatalf("resource = %+v", r)
	}
	repo := e.get(t, r.ID)
	if repo.State != model.CheckoutReady || repo.CommitSHA != "sha-1" || repo.Error != "" ||
		repo.FetchedAt == nil || repo.CheckedAt == nil || repo.Branch != "main" {
		t.Fatalf("repository = %+v", repo)
	}
	if got := e.entryNames(t, r.ID, ""); !slices.Equal(got, []string{"src", "logo.bin", "README.md"}) {
		t.Fatalf("top entries = %v", got)
	}
	if got := e.entryNames(t, r.ID, "src"); !slices.Equal(got, []string{"main.go"}) {
		t.Fatalf("src entries = %v", got)
	}
	if got := e.readRepoFile(t, r.ID, "src/main.go"); got != "package main" {
		t.Fatalf("main.go = %q", got)
	}

	entries, _ := e.repos.Entries(e.ctx, e.session, r.ID, "")
	for _, en := range entries {
		switch en.Name {
		case "README.md":
			if !en.IsText || en.Path != "README.md" || en.Kind != model.ResourceFile {
				t.Errorf("README.md = %+v", en)
			}
		case "logo.bin":
			if en.IsText {
				t.Errorf("binary file counted as text")
			}
		}
	}

	// The copy lives apart from the tree; the folder's own directory stays
	// empty.
	wantExists(t, e.disk(repoDirName, r.ID, "README.md"), true)
	wantExists(t, e.disk("Docs", "hello", "README.md"), false)

	acts := e.actions(t, EventQuery{ResourceID: r.ID})
	if !equalActions(acts, []model.ResourceAction{model.ResourceCreated}) {
		t.Fatalf("events = %v", acts)
	}
}

func TestAddRepositoryRules(t *testing.T) {
	tests := []struct {
		name  string
		setup func(e repoEnv)
		in    AddRepositoryInput
		want  error
	}{
		{"replace is not for folders", nil, AddRepositoryInput{FullName: "octo/hello", OnConflict: ConflictReplace}, ErrInvalidInput},
		{"empty name", nil, AddRepositoryInput{FullName: " "}, ErrInvalidInput},
		{"not connected", func(e repoEnv) { delete(e.creds.m, e.session.UserID) },
			AddRepositoryInput{FullName: "octo/hello"}, ErrGitHubNotConnected},
		{"unknown repository", nil, AddRepositoryInput{FullName: "octo/nope"}, ErrNotFound},
		{"offline", func(e repoEnv) { e.api.setErr(model.ErrGitHubUnavailable) },
			AddRepositoryInput{FullName: "octo/hello"}, ErrGitHubUnavailable},
		{"name taken", func(e repoEnv) { _, _ = e.svc.CreateFolder(e.ctx, e.session, "", "hello") },
			AddRepositoryInput{FullName: "octo/hello"}, ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newRepoEnv(t)
			e.api.addRepo(helloRepo, "sha-1", makeZip(t, "top", map[string]string{"a": "b"}))
			if tt.setup != nil {
				tt.setup(e)
			}
			if tt.in.OnConflict == "" {
				tt.in.OnConflict = ConflictFail
			}
			_, err := e.repos.Add(e.ctx, e.session, tt.in)
			wantErr(t, err, tt.want)
		})
	}

	t.Run("keep both", func(t *testing.T) {
		e := newRepoEnv(t)
		e.api.addRepo(helloRepo, "sha-1", makeZip(t, "top", map[string]string{"a": "b"}))
		e.folder(t, "", "hello")
		r, err := e.repos.Add(e.ctx, e.session, AddRepositoryInput{FullName: "octo/hello", OnConflict: ConflictKeepBoth})
		if err != nil || r.Name != "hello (1)" {
			t.Fatalf("Add() = %q, %v", r.Name, err)
		}
	})
}

func TestRepositoryFolderIsReadOnly(t *testing.T) {
	e := newRepoEnv(t)
	outer := e.folder(t, "", "Outer")
	r := e.addHello(t, outer.ID)
	other := e.folder(t, "", "Other")
	file := e.upload(t, "", "notes.txt", "hi")

	tests := []struct {
		name string
		do   func() error
	}{
		{"create folder inside", func() error { _, err := e.svc.CreateFolder(e.ctx, e.session, r.ID, "x"); return err }},
		{"create file inside", func() error { _, err := e.svc.CreateFile(e.ctx, e.session, r.ID, "x.txt"); return err }},
		{"upload inside", func() error {
			_, err := e.svc.Upload(e.ctx, e.session, UploadInput{ParentID: r.ID, Name: "x", OnConflict: ConflictFail}, strings.NewReader("x"))
			return err
		}},
		{"move a file inside", func() error {
			_, err := e.svc.Move(e.ctx, e.session, file.ID, TransferInput{ParentID: r.ID, OnConflict: ConflictFail})
			return err
		}},
		{"copy the repository", func() error {
			_, err := e.svc.Copy(e.ctx, e.session, r.ID, TransferInput{ParentID: other.ID, OnConflict: ConflictFail})
			return err
		}},
		{"copy a folder holding a repository", func() error {
			_, err := e.svc.Copy(e.ctx, e.session, outer.ID, TransferInput{ParentID: other.ID, OnConflict: ConflictFail})
			return err
		}},
		{"add a repository inside", func() error {
			_, err := e.repos.Add(e.ctx, e.session, AddRepositoryInput{ParentID: r.ID, FullName: "octo/hello", OnConflict: ConflictKeepBoth})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, tt.do(), ErrReadOnly)
		})
	}

	// The folder itself can be renamed, moved and trashed like any folder;
	// its copy stays where it is.
	if _, err := e.svc.Rename(e.ctx, e.session, r.ID, "Hello repo"); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if _, err := e.svc.Move(e.ctx, e.session, r.ID, TransferInput{ParentID: other.ID, OnConflict: ConflictFail}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if got := e.readRepoFile(t, r.ID, "README.md"); got != "# Hello" {
		t.Fatalf("README after move = %q", got)
	}
}

func TestRepositoryNameIsReserved(t *testing.T) {
	e := newRepoEnv(t)
	_, err := e.svc.CreateFolder(e.ctx, e.session, "", ".YAP-repos")
	wantErr(t, err, ErrInvalidInput)
}

func TestRepositoryPathsStayInside(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	if err := os.WriteFile(filepath.Join(e.root, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(e.root, "secret.txt"), e.disk(repoDirName, r.ID, "link")); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"../secret.txt", "src/../../x", "a\\b", "src//main.go"} {
		_, err := e.repos.Entries(e.ctx, e.session, r.ID, p)
		wantErr(t, err, ErrInvalidInput)
	}
	_, _, err := e.repos.OpenFile(e.ctx, e.session, r.ID, "link")
	wantErr(t, err, ErrNotFound)
	_, _, err = e.repos.OpenFile(e.ctx, e.session, r.ID, "src")
	wantErr(t, err, ErrInvalidInput)
	_, err = e.repos.Entries(e.ctx, e.session, r.ID, "README.md")
	wantErr(t, err, ErrInvalidInput)
	if names := e.entryNames(t, r.ID, ""); slices.Contains(names, "link") {
		t.Fatalf("link listed: %v", names)
	}
}

func TestRepositoryUpdates(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	if n := e.api.zipballCount(); n != 1 {
		t.Fatalf("zipballs after add = %d", n)
	}

	// Nothing new: checked, not downloaded again.
	e.advance(time.Minute)
	if _, err := e.repos.Sync(e.ctx, e.session, r.ID); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	e.repos.Wait()
	repo := e.get(t, r.ID)
	if n := e.api.zipballCount(); n != 1 || !repo.CheckedAt.Equal(*e.clock) {
		t.Fatalf("zipballs = %d, checked_at = %v", n, repo.CheckedAt)
	}

	// Offline: the copy stays and the error says why it may be outdated.
	e.api.setErr(model.ErrGitHubUnavailable)
	e.advance(time.Minute)
	if _, err := e.repos.Sync(e.ctx, e.session, r.ID); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	e.repos.Wait()
	repo = e.get(t, r.ID)
	if repo.State != model.CheckoutReady || repo.Error != model.CheckoutOffline || repo.CommitSHA != "sha-1" ||
		repo.CheckedAt.Equal(*e.clock) || !repo.AttemptedAt.Equal(*e.clock) {
		t.Fatalf("offline repository = %+v", repo)
	}
	if got := e.readRepoFile(t, r.ID, "README.md"); got != "# Hello" {
		t.Fatalf("README while offline = %q", got)
	}

	// Back online with a new commit (and a rename on GitHub).
	e.api.setErr(nil)
	renamed := helloRepo
	renamed.FullName, renamed.HTMLURL = "octo/hello-world", "https://github.com/octo/hello-world"
	e.api.mu.Lock()
	delete(e.api.repos, helloRepo.FullName)
	e.api.mu.Unlock()
	e.api.addRepo(renamed, "sha-2", makeZip(t, "octo-hello-sha2", map[string]string{"README.md": "# Hello v2"}))
	e.advance(time.Minute)
	if _, err := e.repos.Sync(e.ctx, e.session, r.ID); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	e.repos.Wait()
	repo = e.get(t, r.ID)
	if repo.State != model.CheckoutReady || repo.CommitSHA != "sha-2" || repo.Error != "" || repo.FullName != "octo/hello-world" {
		t.Fatalf("updated repository = %+v", repo)
	}
	if got := e.readRepoFile(t, r.ID, "README.md"); got != "# Hello v2" {
		t.Fatalf("README after update = %q", got)
	}
	if names := e.entryNames(t, r.ID, ""); !slices.Equal(names, []string{"README.md"}) {
		t.Fatalf("old files left behind: %v", names)
	}
}

func TestRepositoryFetchErrors(t *testing.T) {
	tests := []struct {
		name    string
		private bool
		token   bool
		err     error
		want    model.CheckoutError
	}{
		{"offline", false, true, model.ErrGitHubUnavailable, model.CheckoutOffline},
		{"token rejected", false, true, model.ErrGitHubUnauthorized, model.CheckoutUnauthorized},
		{"gone", false, true, model.ErrGitHubNotFound, model.CheckoutNotFound},
		{"private without account", true, false, nil, model.CheckoutUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newRepoEnv(t)
			repo := helloRepo
			repo.Private = tt.private
			// No archive yet, so the first fetch leaves the copy pending.
			e.api.addRepo(repo, "sha-1", nil)
			r, err := e.repos.Add(e.ctx, e.session, AddRepositoryInput{FullName: "octo/hello", OnConflict: ConflictFail})
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
			e.repos.Wait()

			e.api.setHead(repo.ID, "sha-1", makeZip(t, "top", map[string]string{"a": "b"}))
			e.api.setErr(tt.err)
			if !tt.token {
				delete(e.creds.m, e.session.UserID)
			}
			e.advance(time.Minute)
			if _, err := e.repos.Sync(e.ctx, e.session, r.ID); err != nil {
				t.Fatalf("Sync() error = %v", err)
			}
			e.repos.Wait()
			got := e.get(t, r.ID)
			if got.State != model.CheckoutPending || got.Error != tt.want {
				t.Fatalf("state %s, error %q; want pending, %q", got.State, got.Error, tt.want)
			}
			_, err = e.repos.Entries(e.ctx, e.session, r.ID, "")
			wantErr(t, err, ErrFileMissing)
			if tt.want == model.CheckoutUnauthorized && tt.token {
				if _, err := e.creds.Get(e.session.UserID); err != model.ErrNoCredential {
					t.Fatalf("rejected token kept: %v", err)
				}
			}
		})
	}
}

func TestRepositoryTick(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	e.api.setHead(helloRepo.ID, "sha-2", makeZip(t, "top", map[string]string{"README.md": "v2"}))

	check := func(name string, advance time.Duration, wantSHA string) {
		t.Helper()
		e.advance(advance)
		if err := e.repos.Tick(e.ctx, e.session); err != nil {
			t.Fatalf("%s: Tick() error = %v", name, err)
		}
		if got := e.get(t, r.ID).CommitSHA; got != wantSHA {
			t.Fatalf("%s: commit = %s, want %s", name, got, wantSHA)
		}
	}
	check("not due yet", 10*time.Minute, "sha-1")
	check("due", 5*time.Minute, "sha-2")

	off := false
	if _, err := e.settings.Update(e.ctx, e.session, UpdateSettingsInput{RepoAutoUpdate: &off}); err != nil {
		t.Fatal(err)
	}
	e.api.setHead(helloRepo.ID, "sha-3", makeZip(t, "top", map[string]string{"README.md": "v3"}))
	check("automatic updates off", time.Hour, "sha-2")

	// A manual sync still works with automatic updates off.
	if _, err := e.repos.Sync(e.ctx, e.session, r.ID); err != nil {
		t.Fatal(err)
	}
	e.repos.Wait()
	if got := e.get(t, r.ID).CommitSHA; got != "sha-3" {
		t.Fatalf("manual sync commit = %s", got)
	}

	// Trashed repositories are left alone and cannot be synced.
	if err := e.svc.MoveToTrash(e.ctx, e.session, r.ID); err != nil {
		t.Fatal(err)
	}
	_, err := e.repos.Sync(e.ctx, e.session, r.ID)
	wantErr(t, err, ErrNotFound)
}

func TestRepositoryTickRetriesPendingAndCollectsGarbage(t *testing.T) {
	e := newRepoEnv(t)
	off := false
	if _, err := e.settings.Update(e.ctx, e.session, UpdateSettingsInput{RepoAutoUpdate: &off}); err != nil {
		t.Fatal(err)
	}
	e.api.addRepo(helloRepo, "sha-1", makeZip(t, "top", map[string]string{"a": "b"}))
	r, err := e.repos.Add(e.ctx, e.session, AddRepositoryInput{FullName: "octo/hello", OnConflict: ConflictFail})
	if err != nil {
		t.Fatal(err)
	}
	e.repos.Wait()
	if got := e.get(t, r.ID); got.State != model.CheckoutReady {
		t.Fatalf("state = %s", got.State)
	}

	// A new repository whose first fetch failed is retried even with
	// automatic updates off.
	e.api.addRepo(model.GitHubRepo{ID: 43, FullName: "octo/other", HTMLURL: "https://github.com/octo/other", DefaultBranch: "main"}, "o-1", nil)
	other, err := e.repos.Add(e.ctx, e.session, AddRepositoryInput{FullName: "octo/other", OnConflict: ConflictFail})
	if err != nil {
		t.Fatal(err)
	}
	e.repos.Wait()
	if got := e.get(t, other.ID); got.State != model.CheckoutPending || got.Error != model.CheckoutNotFound {
		t.Fatalf("other = %s / %s", got.State, got.Error)
	}
	e.api.setHead(43, "o-1", makeZip(t, "top", map[string]string{"x": "y"}))
	e.advance(repoRetryInterval)
	if err := e.repos.Tick(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	if got := e.get(t, other.ID); got.State != model.CheckoutReady {
		t.Fatalf("other after retry = %s / %s", got.State, got.Error)
	}

	// Purging the folder lets the next tick remove its copy.
	if err := e.svc.MoveToTrash(e.ctx, e.session, r.ID); err != nil {
		t.Fatal(err)
	}
	wantExists(t, e.disk(repoDirName, r.ID), true)
	if err := e.svc.Purge(e.ctx, e.session, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.repos.Tick(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	wantExists(t, e.disk(repoDirName, r.ID), false)
	wantExists(t, e.disk(repoDirName, other.ID), true)
}

func TestRepositoryRecoverClearsFetching(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	repo := e.get(t, r.ID)
	repo.State = model.CheckoutFetching
	if err := repository.NewRepoRepository().SaveCheckout(e.ctx, e.db, repo, e.repos.nowString()); err != nil {
		t.Fatal(err)
	}
	if err := e.repos.recover(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	if got := e.get(t, r.ID).State; got != model.CheckoutReady {
		t.Fatalf("state = %s", got)
	}
}

// gitRepo makes a local repository with one commit on main and returns its
// path and head.
func gitRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
			"GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "--quiet", "--initial-branch=main")
	commit(t, dir, run, files)
	return dir, run("rev-parse", "HEAD")
}

func commit(t *testing.T, dir string, run func(...string) string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "--quiet", "-m", "change")
}

func TestRepositoryCloneToFolder(t *testing.T) {
	git := repofetch.NewGit()
	if !git.Available() {
		t.Skip("git is not installed")
	}
	e := newRepoEnvWith(t, git)
	r := e.addHello(t, "")
	src, head := gitRepo(t, map[string]string{"README.md": "# Git", "pkg/a.go": "package pkg"})
	upstream := helloRepo
	upstream.CloneURL = src
	e.api.addRepo(upstream, head, nil)
	parent := t.TempDir()

	c, err := e.repos.Clone(e.ctx, e.session, r.ID, CloneInput{ParentPath: parent, Name: "hello"})
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}
	target := filepath.Join(parent, "hello")
	if c.Path != target || !c.Exists {
		t.Fatalf("clone = %+v", c)
	}
	got, err := os.ReadFile(filepath.Join(target, "README.md"))
	if err != nil || string(got) != "# Git" {
		t.Fatalf("README in clone = %q, %v", got, err)
	}
	wantExists(t, filepath.Join(target, ".git"), true)
	// The token is not left in the clone's config.
	config, err := os.ReadFile(filepath.Join(target, ".git", "config"))
	if err != nil || strings.Contains(string(config), "token-1") || strings.Contains(string(config), "extraHeader") {
		t.Fatalf("clone config = %s, %v", config, err)
	}

	// Background work never touches the user's clone.
	if err := os.WriteFile(filepath.Join(target, "mine.txt"), []byte("work"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.api.setHead(helloRepo.ID, "sha-2", makeZip(t, "top", map[string]string{"README.md": "v2"}))
	e.advance(time.Hour)
	if err := e.repos.Tick(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	wantExists(t, filepath.Join(target, "mine.txt"), true)

	clones, err := e.repos.Clones(e.ctx, e.session, r.ID)
	if err != nil || len(clones) != 1 || clones[0].Path != target || !clones[0].Exists {
		t.Fatalf("Clones() = %+v, %v", clones, err)
	}
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if clones, _ := e.repos.Clones(e.ctx, e.session, r.ID); len(clones) != 1 || clones[0].Exists {
		t.Fatalf("deleted clone = %+v", clones)
	}
	if err := e.repos.ForgetClone(e.ctx, e.session, r.ID, c.ID); err != nil {
		t.Fatalf("ForgetClone() error = %v", err)
	}
	wantErr(t, e.repos.ForgetClone(e.ctx, e.session, r.ID, c.ID), ErrNotFound)
	if clones, _ := e.repos.Clones(e.ctx, e.session, r.ID); len(clones) != 0 {
		t.Fatalf("forgotten clone still listed: %+v", clones)
	}
}

func TestRepositoryCloneRules(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, "taken"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(parent, "file.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		in   CloneInput
		want error
	}{
		{"relative folder", CloneInput{ParentPath: "code", Name: "hello"}, ErrInvalidInput},
		{"missing folder", CloneInput{ParentPath: filepath.Join(parent, "nope"), Name: "hello"}, ErrNotFound},
		{"not a folder", CloneInput{ParentPath: file, Name: "hello"}, ErrInvalidInput},
		{"bad name", CloneInput{ParentPath: parent, Name: "../escape"}, ErrInvalidInput},
		{"name taken", CloneInput{ParentPath: parent, Name: "taken"}, ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.repos.Clone(e.ctx, e.session, r.ID, tt.in)
			wantErr(t, err, tt.want)
		})
	}

	t.Run("git missing", func(t *testing.T) {
		e := newRepoEnvWith(t, &repofetch.Git{})
		r := e.addHello(t, "")
		_, err := e.repos.Clone(e.ctx, e.session, r.ID, CloneInput{ParentPath: parent, Name: "x"})
		wantErr(t, err, ErrGitMissing)
	})

	t.Run("offline leaves nothing behind", func(t *testing.T) {
		e.api.setErr(model.ErrGitHubUnavailable)
		defer e.api.setErr(nil)
		_, err := e.repos.Clone(e.ctx, e.session, r.ID, CloneInput{ParentPath: parent, Name: "fresh"})
		wantErr(t, err, ErrGitHubUnavailable)
		wantExists(t, filepath.Join(parent, "fresh"), false)
	})
}

func TestLocalFolders(t *testing.T) {
	e := newRepoEnv(t)
	dir := t.TempDir()
	for _, name := range []string{"b", "A", ".hidden"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := e.repos.LocalFolders(e.ctx, dir)
	if err != nil {
		t.Fatalf("LocalFolders() error = %v", err)
	}
	if got.Path != dir || got.Parent != filepath.Dir(dir) || !slices.Equal(got.Folders, []string{"A", "b"}) || got.Home == "" {
		t.Fatalf("LocalFolders() = %+v", got)
	}
	if home, err := e.repos.LocalFolders(e.ctx, ""); err != nil || home.Path != home.Home {
		t.Fatalf("LocalFolders(\"\") = %+v, %v", home, err)
	}
	if root, err := e.repos.LocalFolders(e.ctx, "/"); err != nil || root.Parent != "" {
		t.Fatalf("LocalFolders(/) = %+v, %v", root, err)
	}
	_, err = e.repos.LocalFolders(e.ctx, "relative")
	wantErr(t, err, ErrInvalidInput)
	_, err = e.repos.LocalFolders(e.ctx, filepath.Join(dir, "missing"))
	wantErr(t, err, ErrNotFound)
}

func TestOldGitCopyIsReplacedByZip(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	// A copy from before clones moved out of the app.
	if err := os.Mkdir(e.disk(repoDirName, r.ID, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repos.Entries(e.ctx, e.session, r.ID, ".git"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Entries(.git) error = %v", err)
	}
	e.advance(time.Minute)
	if _, err := e.repos.Sync(e.ctx, e.session, r.ID); err != nil {
		t.Fatal(err)
	}
	e.repos.Wait()
	wantExists(t, e.disk(repoDirName, r.ID, ".git"), false)
	if n := e.api.zipballCount(); n != 2 {
		t.Fatalf("zipballs = %d, want the copy downloaded again", n)
	}
}

func TestRepositoryExportAndImport(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	acct := accountEnv{
		resourceEnv: e.resourceEnv,
		svc: NewAccountService(e.db, repository.NewAccountRepository(), repository.NewResourceRepository(),
			repository.NewWorkspaceRepository(), repository.NewSettingsRepository(), e.svc.store, "tester"),
		projects: NewProjectService(e.db, repository.NewProjectRepository(), repository.NewSettingsRepository()),
	}
	archive := acct.exportBytes(t)

	var rows []any
	rewrite(t, archive, func(doc map[string]any) { rows = tableRows(doc, "resource_repositories") })
	if len(rows) != 1 {
		t.Fatalf("exported repositories = %v", rows)
	}
	row := rows[0].(map[string]any)
	if _, hasMethod := row["method"]; row["full_name"] != "octo/hello" || hasMethod {
		t.Fatalf("exported row = %v", row)
	}
	if _, ok := row["commit_sha"]; ok {
		t.Fatalf("the local checkout state was exported: %v", row)
	}

	if err := acct.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	acct.importBytes(t, archive)
	got, ok := acct.resourcePaths(t)["hello"]
	if !ok || !got.IsRepository || got.ID != r.ID {
		t.Fatalf("imported folder = %+v (found %v)", got, ok)
	}
	if repo := e.get(t, r.ID); repo.State != model.CheckoutPending || repo.FullName != "octo/hello" {
		t.Fatalf("imported repository = %+v", repo)
	}
	if err := e.repos.Tick(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	if got := e.readRepoFile(t, r.ID, "README.md"); got != "# Hello" {
		t.Fatalf("README after import = %q", got)
	}

	// Nothing may be imported into a repository folder.
	bad := rewrite(t, archive, func(doc map[string]any) {
		tables := doc["tables"].(map[string]any)
		tables["resources"] = append(tableRows(doc, "resources"), map[string]any{
			"id": model.NewID(), "parent_id": r.ID, "kind": "folder", "name": "inside", "revision": 1,
			"created_at": "2026-11-18T12:00:00Z",
		})
	})
	if _, err := acct.svc.Import(e.ctx, e.session, bytes.NewReader(bad)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Import() with a resource inside a repository error = %v", err)
	}
}

func TestRepositoryCommits(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	e.api.mu.Lock()
	e.api.commits = [][]model.GitHubCommit{
		{{SHA: "c2", Parents: []string{"c1"}, Message: "second"}, {SHA: "c1", Parents: []string{}, Message: "first"}},
	}
	e.api.mu.Unlock()

	got, err := e.repos.Commits(e.ctx, e.session, r.ID, 1)
	if err != nil || len(got) != 2 || got[0].SHA != "c2" || got[1].Parents == nil {
		t.Fatalf("Commits(1) = %+v, %v", got, err)
	}
	if got, err := e.repos.Commits(e.ctx, e.session, r.ID, 2); err != nil || len(got) != 0 {
		t.Fatalf("Commits(2) = %+v, %v", got, err)
	}
	for _, page := range []int{0, maxCommitPage + 1} {
		_, err := e.repos.Commits(e.ctx, e.session, r.ID, page)
		wantErr(t, err, ErrInvalidInput)
	}

	e.api.setErr(model.ErrGitHubUnavailable)
	_, err = e.repos.Commits(e.ctx, e.session, r.ID, 1)
	wantErr(t, err, ErrGitHubUnavailable)
	e.api.setErr(nil)

	// A private repository needs the account.
	private := helloRepo
	private.Private = true
	e.api.addRepo(private, "sha-1", nil)
	delete(e.creds.m, e.session.UserID)
	_, err = e.repos.Commits(e.ctx, e.session, r.ID, 1)
	wantErr(t, err, ErrGitHubNotConnected)
}
