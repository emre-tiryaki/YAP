package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// fakeGitHub stands in for the GitHub API. Repositories are keyed by full
// name; heads map a repository id to its branch's commit and archives a
// commit to its zipball.
type fakeGitHub struct {
	mu         sync.Mutex
	configured bool
	repos      map[string]model.GitHubRepo
	heads      map[int64]string
	archives   map[string][]byte
	// err, when set, is returned by every repository call.
	err error
	// tokens are the tokens GitHub accepts; empty accepts any. Anonymous
	// calls only see public repositories.
	tokens map[string]bool

	// commits is the history returned by Commits, page after page.
	commits [][]model.GitHubCommit

	user model.GitHubUser
	// pollAnswers are returned by PollDeviceToken in order; nil means
	// success with "token-1".
	pollAnswers []error
	polls       int
	zipballs    int
}

func newFakeGitHub() *fakeGitHub {
	return &fakeGitHub{
		configured: true,
		repos:      map[string]model.GitHubRepo{},
		heads:      map[int64]string{},
		archives:   map[string][]byte{},
		user:       model.GitHubUser{ID: 7, Login: "octo", Name: "Octo Cat"},
	}
}

func (f *fakeGitHub) addRepo(r model.GitHubRepo, sha string, archive []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repos[r.FullName] = r
	f.heads[r.ID] = sha
	if archive != nil {
		f.archives[sha] = archive
	}
}

func (f *fakeGitHub) setHead(id int64, sha string, archive []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heads[id] = sha
	f.archives[sha] = archive
}

func (f *fakeGitHub) setErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeGitHub) zipballCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.zipballs
}

func (f *fakeGitHub) check(token string, r model.GitHubRepo) error {
	if f.err != nil {
		return f.err
	}
	if token != "" && len(f.tokens) > 0 && !f.tokens[token] {
		return model.ErrGitHubUnauthorized
	}
	if token == "" && r.Private {
		return model.ErrGitHubNotFound
	}
	return nil
}

func (f *fakeGitHub) Configured() bool { return f.configured }

func (f *fakeGitHub) StartDeviceFlow(context.Context) (model.DeviceCode, error) {
	return model.DeviceCode{DeviceCode: "dev-1", UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device",
		ExpiresIn: 15 * time.Minute, Interval: 5 * time.Second}, nil
}

func (f *fakeGitHub) PollDeviceToken(context.Context, string) (string, string, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.polls++
	if len(f.pollAnswers) > 0 {
		answer := f.pollAnswers[0]
		f.pollAnswers = f.pollAnswers[1:]
		if answer != nil {
			return "", "", 0, answer
		}
	}
	return "token-1", "repo", 0, nil
}

func (f *fakeGitHub) User(_ context.Context, token string) (model.GitHubUser, error) {
	return f.user, nil
}

func (f *fakeGitHub) Repos(_ context.Context, token string) ([]model.GitHubRepo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if len(f.tokens) > 0 && !f.tokens[token] {
		return nil, model.ErrGitHubUnauthorized
	}
	out := []model.GitHubRepo{}
	for _, r := range f.repos {
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeGitHub) Repo(_ context.Context, token, fullName string) (model.GitHubRepo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.repos[fullName]
	if !ok {
		return model.GitHubRepo{}, model.ErrGitHubNotFound
	}
	if err := f.check(token, r); err != nil {
		return model.GitHubRepo{}, err
	}
	return r, nil
}

func (f *fakeGitHub) RepoByID(_ context.Context, token string, id int64) (model.GitHubRepo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.repos {
		if r.ID == id {
			if err := f.check(token, r); err != nil {
				return model.GitHubRepo{}, err
			}
			return r, nil
		}
	}
	return model.GitHubRepo{}, model.ErrGitHubNotFound
}

func (f *fakeGitHub) BranchHead(_ context.Context, token, fullName, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.repos[fullName]
	if !ok {
		return "", model.ErrGitHubNotFound
	}
	if err := f.check(token, r); err != nil {
		return "", err
	}
	return f.heads[r.ID], nil
}

func (f *fakeGitHub) Zipball(_ context.Context, token, fullName, sha string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.repos[fullName]
	if !ok {
		return nil, model.ErrGitHubNotFound
	}
	if err := f.check(token, r); err != nil {
		return nil, err
	}
	archive, ok := f.archives[sha]
	if !ok {
		return nil, model.ErrGitHubNotFound
	}
	f.zipballs++
	return io.NopCloser(bytes.NewReader(archive)), nil
}

func (f *fakeGitHub) Commits(_ context.Context, token, fullName, _ string, page int) ([]model.GitHubCommit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.repos[fullName]
	if !ok {
		return nil, model.ErrGitHubNotFound
	}
	if err := f.check(token, r); err != nil {
		return nil, err
	}
	if page > len(f.commits) {
		return []model.GitHubCommit{}, nil
	}
	return f.commits[page-1], nil
}

// memCreds is an in-memory keychain.
type memCreds struct {
	mu  sync.Mutex
	m   map[string]model.GitHubCredential
	err error
}

func newMemCreds() *memCreds {
	return &memCreds{m: map[string]model.GitHubCredential{}}
}

func (c *memCreds) Get(userID string) (model.GitHubCredential, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return model.GitHubCredential{}, c.err
	}
	cred, ok := c.m[userID]
	if !ok {
		return model.GitHubCredential{}, model.ErrNoCredential
	}
	return cred, nil
}

func (c *memCreds) Set(userID string, cred model.GitHubCredential) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.m[userID] = cred
	return nil
}

func (c *memCreds) Delete(userID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	delete(c.m, userID)
	return nil
}

// makeZip builds a GitHub-like source archive: every file under one top
// folder.
func makeZip(t *testing.T, top string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create(top + "/"); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		w, err := zw.Create(top + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func wantErr(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
