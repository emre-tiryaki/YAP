package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewWithURLs("client-1", srv.URL, srv.URL, srv.Client()), srv
}

func TestDeviceFlow(t *testing.T) {
	answers := []string{
		`{"error":"authorization_pending"}`,
		`{"error":"slow_down","interval":10}`,
		`{"access_token":"gho_x","scope":"repo","token_type":"bearer"}`,
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.Form.Get("client_id") != "client-1" {
			t.Errorf("form = %v, %v", r.Form, err)
		}
		switch r.URL.Path {
		case "/login/device/code":
			if r.Form.Get("scope") != "repo" {
				t.Errorf("scope = %q", r.Form.Get("scope"))
			}
			_, _ = io.WriteString(w, `{"device_code":"dc","user_code":"AB-12","verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`)
		case "/login/oauth/access_token":
			if r.Form.Get("device_code") != "dc" {
				t.Errorf("device_code = %q", r.Form.Get("device_code"))
			}
			_, _ = io.WriteString(w, answers[0])
			answers = answers[1:]
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	code, err := c.StartDeviceFlow(ctx)
	if err != nil || code.UserCode != "AB-12" || code.Interval != 5*time.Second || code.ExpiresIn != 15*time.Minute {
		t.Fatalf("StartDeviceFlow() = %+v, %v", code, err)
	}
	if _, _, _, err := c.PollDeviceToken(ctx, "dc"); !errors.Is(err, model.ErrDeviceAuthPending) {
		t.Fatalf("poll 1 error = %v", err)
	}
	if _, _, interval, err := c.PollDeviceToken(ctx, "dc"); !errors.Is(err, model.ErrDeviceSlowDown) || interval != 10*time.Second {
		t.Fatalf("poll 2 = %v, %v", interval, err)
	}
	token, scopes, _, err := c.PollDeviceToken(ctx, "dc")
	if err != nil || token != "gho_x" || scopes != "repo" {
		t.Fatalf("poll 3 = %q %q %v", token, scopes, err)
	}
}

func TestStatusErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		header map[string]string
		want   error
	}{
		{"unauthorized", http.StatusUnauthorized, nil, model.ErrGitHubUnauthorized},
		{"not found", http.StatusNotFound, nil, model.ErrGitHubNotFound},
		{"rate limited", http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "0"}, model.ErrGitHubUnavailable},
		{"too many requests", http.StatusTooManyRequests, nil, model.ErrGitHubUnavailable},
		{"secondary rate limit", http.StatusForbidden, map[string]string{"Retry-After": "60"}, model.ErrGitHubUnavailable},
		{"forbidden", http.StatusForbidden, nil, model.ErrGitHubNotFound},
		{"server error", http.StatusBadGateway, nil, model.ErrGitHubUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.header {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.status)
			})
			if _, err := c.User(context.Background(), "t"); !errors.Is(err, tt.want) {
				t.Fatalf("User() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestOfflineIsUnavailable(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	srv.Close()
	if _, err := c.Repos(context.Background(), "t"); !errors.Is(err, model.ErrGitHubUnavailable) {
		t.Fatalf("Repos() error = %v", err)
	}
}

func TestRepoRequests(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		switch r.URL.EscapedPath() {
		case "/repos/octo/hello":
			_, _ = io.WriteString(w, `{"id":42,"full_name":"octo/hello","description":null,"html_url":"https://github.com/octo/hello",
				"clone_url":"https://github.com/octo/hello.git","default_branch":"main","private":true,"pushed_at":"2026-11-01T10:00:00Z"}`)
		case "/repos/octo/hello/branches/feature/x%20y":
			_, _ = io.WriteString(w, `{"name":"feature/x y","commit":{"sha":"abc"}}`)
		case "/user/repos":
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":1,"full_name":"a/b","default_branch":"main"}]`)
				return
			}
			t.Errorf("asked for page %s after a short page", r.URL.Query().Get("page"))
		default:
			t.Errorf("unexpected path %s", r.URL.EscapedPath())
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	repo, err := c.Repo(ctx, "tok", "octo/hello")
	if err != nil || repo.ID != 42 || !repo.Private || repo.DefaultBranch != "main" || repo.PushedAt == nil {
		t.Fatalf("Repo() = %+v, %v", repo, err)
	}
	sha, err := c.BranchHead(ctx, "tok", "octo/hello", "feature/x y")
	if err != nil || sha != "abc" {
		t.Fatalf("BranchHead() = %q, %v", sha, err)
	}
	repos, err := c.Repos(ctx, "tok")
	if err != nil || len(repos) != 1 {
		t.Fatalf("Repos() = %v, %v", repos, err)
	}
	for _, bad := range []string{"octo", "../x", "octo/hello/extra", "octo/he llo"} {
		if _, err := c.Repo(ctx, "tok", bad); !errors.Is(err, model.ErrGitHubNotFound) {
			t.Errorf("Repo(%q) error = %v", bad, err)
		}
	}
}

func TestDeviceFlowClientProblems(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"unknown client id", http.StatusNotFound, `{"error":"Not Found"}`, model.ErrGitHubClientInvalid},
		{"device flow off", http.StatusBadRequest, `{"error":"device_flow_disabled"}`, model.ErrDeviceFlowDisabled},
		{"bad credentials", http.StatusUnauthorized, `{"error":"incorrect_client_credentials"}`, model.ErrGitHubClientInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			})
			if _, err := c.StartDeviceFlow(context.Background()); !errors.Is(err, tt.want) {
				t.Fatalf("StartDeviceFlow() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCommits(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/repos/octo/hello/commits" || q.Get("sha") != "main" || q.Get("page") != "2" || q.Get("per_page") != "50" {
			t.Errorf("request = %s", r.URL)
		}
		_, _ = io.WriteString(w, `[
			{"sha":"m","html_url":"https://github.com/octo/hello/commit/m",
			 "commit":{"message":"Merge branch 'x'\n\nbody","author":{"name":"Octo","date":"2026-11-01T10:00:00Z"}},
			 "author":{"login":"octo","avatar_url":"https://a/1"},"parents":[{"sha":"a"},{"sha":"b"}]},
			{"sha":"a","commit":{"message":"init","author":{"name":"Ghost","date":"2026-10-01T10:00:00Z"}},
			 "author":null,"parents":[]}
		]`)
	})
	got, err := c.Commits(context.Background(), "", "octo/hello", "main", 2)
	if err != nil || len(got) != 2 {
		t.Fatalf("Commits() = %+v, %v", got, err)
	}
	m := got[0]
	if m.SHA != "m" || len(m.Parents) != 2 || m.Parents[1] != "b" || m.AuthorLogin != "octo" || m.AuthorName != "Octo" ||
		!m.Date.Equal(time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)) || m.Message != "Merge branch 'x'\n\nbody" {
		t.Fatalf("merge commit = %+v", m)
	}
	if got[1].AuthorLogin != "" || got[1].Parents == nil || len(got[1].Parents) != 0 {
		t.Fatalf("root commit = %+v", got[1])
	}
}
