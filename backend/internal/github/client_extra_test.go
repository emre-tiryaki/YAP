package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func TestPollDeviceTokenErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{"expired", `{"error":"expired_token"}`, model.ErrDeviceExpired},
		{"denied", `{"error":"access_denied"}`, model.ErrDeviceDenied},
		{"disabled", `{"error":"device_flow_disabled"}`, model.ErrDeviceFlowDisabled},
		{"bad credentials", `{"error":"incorrect_client_credentials"}`, model.ErrGitHubClientInvalid},
		{"unauthorized client", `{"error":"unauthorized_client"}`, model.ErrGitHubClientInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, tt.body) })
			if _, _, _, err := c.PollDeviceToken(context.Background(), "dc"); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}

	t.Run("unknown error", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"error":"weird"}`) })
		_, _, _, err := c.PollDeviceToken(context.Background(), "dc")
		if err == nil || errors.Is(err, model.ErrDeviceAuthPending) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("empty token", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{}`) })
		if _, _, _, err := c.PollDeviceToken(context.Background(), "dc"); err == nil {
			t.Fatal("want an error for an empty access token")
		}
	})
}

func TestRepoByID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Path != "/repositories/42" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"id":42,"full_name":"octo/renamed","default_branch":"dev","private":true}`)
	})
	ctx := context.Background()
	repo, err := c.RepoByID(ctx, "tok", 42)
	if err != nil || repo.ID != 42 || repo.FullName != "octo/renamed" || repo.DefaultBranch != "dev" || !repo.Private {
		t.Fatalf("RepoByID() = %+v, %v", repo, err)
	}
	if _, err := c.RepoByID(ctx, "tok", 7); !errors.Is(err, model.ErrGitHubNotFound) {
		t.Fatalf("RepoByID(missing) error = %v", err)
	}
}

func TestZipball(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/octo/hello/zipball/abc123":
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				t.Errorf("Authorization = %q", got)
			}
			_, _ = io.WriteString(w, "zip-bytes")
		case "/repos/octo/hello/zipball/boom":
			w.WriteHeader(http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	body, err := c.Zipball(ctx, "tok", "octo/hello", "abc123")
	if err != nil {
		t.Fatalf("Zipball() error = %v", err)
	}
	got, err := io.ReadAll(body)
	body.Close()
	if err != nil || string(got) != "zip-bytes" {
		t.Fatalf("body = %q, %v", got, err)
	}
	if _, err := c.Zipball(ctx, "tok", "octo/hello", "missing"); !errors.Is(err, model.ErrGitHubNotFound) {
		t.Fatalf("missing sha error = %v", err)
	}
	if _, err := c.Zipball(ctx, "tok", "octo/hello", "boom"); !errors.Is(err, model.ErrGitHubUnavailable) {
		t.Fatalf("server error = %v", err)
	}
	if _, err := c.Zipball(ctx, "tok", "../x", "abc"); !errors.Is(err, model.ErrGitHubNotFound) {
		t.Fatalf("bad name error = %v", err)
	}
}
