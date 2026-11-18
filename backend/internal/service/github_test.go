package service

import (
	"context"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type githubEnv struct {
	api     *fakeGitHub
	creds   *memCreds
	svc     *GitHubService
	session model.Session
	clock   *time.Time
	ctx     context.Context
}

func newGitHubEnv(t *testing.T) githubEnv {
	t.Helper()
	api, creds := newFakeGitHub(), newMemCreds()
	svc := NewGitHubService(api, creds, true)
	clock := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return clock }
	e := githubEnv{api: api, creds: creds, svc: svc, session: model.Session{UserID: "user-1"}, clock: &clock, ctx: context.Background()}
	svc.now = func() time.Time { return *e.clock }
	return e
}

func (e githubEnv) advance(d time.Duration) {
	*e.clock = e.clock.Add(d)
}

func TestGitHubStatus(t *testing.T) {
	e := newGitHubEnv(t)
	st, err := e.svc.Status(e.ctx, e.session)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !st.Configured || !st.GitAvailable || st.Account != nil {
		t.Fatalf("status = %+v, want configured, git, no account", st)
	}

	e.creds.err = model.ErrSecretStoreUnavailable
	_, err = e.svc.Status(e.ctx, e.session)
	wantErr(t, err, ErrKeychainUnavailable)
}

func TestGitHubConnectNeedsClientID(t *testing.T) {
	e := newGitHubEnv(t)
	e.api.configured = false
	_, err := e.svc.StartConnect(e.ctx, e.session)
	wantErr(t, err, ErrGitHubNotConfigured)
}

func TestGitHubDeviceFlow(t *testing.T) {
	e := newGitHubEnv(t)
	e.api.pollAnswers = []error{model.ErrDeviceAuthPending, model.ErrDeviceSlowDown, nil}

	_, err := e.svc.PollConnect(e.ctx, e.session)
	wantErr(t, err, ErrNotFound)

	flow, err := e.svc.StartConnect(e.ctx, e.session)
	if err != nil {
		t.Fatalf("StartConnect() error = %v", err)
	}
	if flow.UserCode != "ABCD-1234" || flow.Interval != 5*time.Second {
		t.Fatalf("flow = %+v", flow)
	}

	steps := []struct {
		name      string
		advance   time.Duration
		wantState ConnectState
		wantPolls int
	}{
		{"too soon does not ask GitHub", time.Second, ConnectPending, 0},
		{"pending", 5 * time.Second, ConnectPending, 1},
		{"slow down", 5 * time.Second, ConnectPending, 2},
		// The interval grew to 10s.
		{"waits the longer interval", 5 * time.Second, ConnectPending, 2},
		{"connected", 5 * time.Second, ConnectConnected, 3},
	}
	for _, step := range steps {
		e.advance(step.advance)
		res, err := e.svc.PollConnect(e.ctx, e.session)
		if err != nil {
			t.Fatalf("%s: PollConnect() error = %v", step.name, err)
		}
		if res.State != step.wantState || e.api.polls != step.wantPolls {
			t.Fatalf("%s: state %s after %d polls, want %s after %d", step.name, res.State, e.api.polls, step.wantState, step.wantPolls)
		}
		if res.State == ConnectConnected && (res.Account == nil || res.Account.User.Login != "octo") {
			t.Fatalf("%s: account = %+v", step.name, res.Account)
		}
	}

	cred, err := e.creds.Get(e.session.UserID)
	if err != nil || cred.Token != "token-1" || cred.Scopes != "repo" {
		t.Fatalf("stored credential = %+v, %v", cred, err)
	}
	_, err = e.svc.PollConnect(e.ctx, e.session)
	wantErr(t, err, ErrNotFound)

	if err := e.svc.Disconnect(e.ctx, e.session); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if _, err := e.creds.Get(e.session.UserID); err != model.ErrNoCredential {
		t.Fatalf("credential after disconnect: %v", err)
	}
}

func TestGitHubDeviceFlowEnds(t *testing.T) {
	tests := []struct {
		name   string
		answer error
		after  time.Duration
		want   ConnectState
	}{
		{"denied", model.ErrDeviceDenied, 5 * time.Second, ConnectDenied},
		{"expired at GitHub", model.ErrDeviceExpired, 5 * time.Second, ConnectExpired},
		{"expired here", nil, 16 * time.Minute, ConnectExpired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newGitHubEnv(t)
			e.api.pollAnswers = []error{tt.answer}
			if _, err := e.svc.StartConnect(e.ctx, e.session); err != nil {
				t.Fatal(err)
			}
			e.advance(tt.after)
			res, err := e.svc.PollConnect(e.ctx, e.session)
			if err != nil || res.State != tt.want {
				t.Fatalf("PollConnect() = %v, %v; want %s", res.State, err, tt.want)
			}
			if _, err := e.creds.Get(e.session.UserID); err != model.ErrNoCredential {
				t.Fatalf("a credential was stored: %v", err)
			}
		})
	}
}

func TestGitHubReposForgetsRejectedToken(t *testing.T) {
	e := newGitHubEnv(t)
	_, err := e.svc.Repos(e.ctx, e.session)
	wantErr(t, err, ErrGitHubNotConnected)

	e.creds.m[e.session.UserID] = model.GitHubCredential{Token: "old"}
	e.api.tokens = map[string]bool{"new": true}
	_, err = e.svc.Repos(e.ctx, e.session)
	wantErr(t, err, ErrGitHubNotConnected)
	if _, err := e.creds.Get(e.session.UserID); err != model.ErrNoCredential {
		t.Fatalf("rejected token was kept: %v", err)
	}

	e.creds.m[e.session.UserID] = model.GitHubCredential{Token: "new"}
	e.api.setErr(model.ErrGitHubUnavailable)
	_, err = e.svc.Repos(e.ctx, e.session)
	wantErr(t, err, ErrGitHubUnavailable)
	if _, err := e.creds.Get(e.session.UserID); err != nil {
		t.Fatalf("token dropped while offline: %v", err)
	}
}
