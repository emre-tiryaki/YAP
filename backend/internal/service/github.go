package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// GitHubAPI is the part of the GitHub client the services use. An empty
// token makes an anonymous request.
type GitHubAPI interface {
	Configured() bool
	StartDeviceFlow(ctx context.Context) (model.DeviceCode, error)
	PollDeviceToken(ctx context.Context, deviceCode string) (token, scopes string, interval time.Duration, err error)
	User(ctx context.Context, token string) (model.GitHubUser, error)
	Repos(ctx context.Context, token string) ([]model.GitHubRepo, error)
	Repo(ctx context.Context, token, fullName string) (model.GitHubRepo, error)
	RepoByID(ctx context.Context, token string, id int64) (model.GitHubRepo, error)
	BranchHead(ctx context.Context, token, fullName, branch string) (string, error)
	Zipball(ctx context.Context, token, fullName, sha string) (io.ReadCloser, error)
	Commits(ctx context.Context, token, fullName, branch string, page int) ([]model.GitHubCommit, error)
}

// CredentialStore keeps GitHub tokens outside the database (the OS
// keychain). Get returns model.ErrNoCredential when there is none.
type CredentialStore interface {
	Get(userID string) (model.GitHubCredential, error)
	Set(userID string, c model.GitHubCredential) error
	Delete(userID string) error
}

// GitHubStatus is what the settings page shows about the connection.
type GitHubStatus struct {
	// Configured is false when no OAuth client id is set; nothing can be
	// connected then.
	Configured   bool
	GitAvailable bool
	// Account is nil while not connected.
	Account *GitHubAccount
}

type GitHubAccount struct {
	User        model.GitHubUser
	Scopes      string
	ConnectedAt time.Time
}

// DeviceFlow is shown to the user while connecting: they open
// VerificationURI and enter UserCode.
type DeviceFlow struct {
	UserCode        string
	VerificationURI string
	ExpiresAt       time.Time
	Interval        time.Duration
}

type ConnectState string

const (
	ConnectPending   ConnectState = "pending"
	ConnectConnected ConnectState = "connected"
	ConnectExpired   ConnectState = "expired"
	ConnectDenied    ConnectState = "denied"
)

type ConnectResult struct {
	State   ConnectState
	Account *GitHubAccount
}

type deviceFlow struct {
	code      model.DeviceCode
	expiresAt time.Time
	nextPoll  time.Time
	interval  time.Duration
}

type GitHubService struct {
	api          GitHubAPI
	creds        CredentialStore
	gitAvailable bool
	now          func() time.Time

	mu sync.Mutex
	// flows holds the device flow each user is in the middle of. The device
	// code stays here; the browser only sees the user code.
	flows map[string]*deviceFlow
}

func NewGitHubService(api GitHubAPI, creds CredentialStore, gitAvailable bool) *GitHubService {
	return &GitHubService{api: api, creds: creds, gitAvailable: gitAvailable, now: time.Now, flows: map[string]*deviceFlow{}}
}

func toAccount(c model.GitHubCredential) *GitHubAccount {
	return &GitHubAccount{User: c.User, Scopes: c.Scopes, ConnectedAt: c.ConnectedAt}
}

func (s *GitHubService) Status(_ context.Context, session model.Session) (GitHubStatus, error) {
	st := GitHubStatus{Configured: s.api.Configured(), GitAvailable: s.gitAvailable}
	c, err := s.creds.Get(session.UserID)
	switch {
	case err == nil:
		st.Account = toAccount(c)
	case errors.Is(err, model.ErrNoCredential):
	default:
		return GitHubStatus{}, err
	}
	return st, nil
}

// StartConnect begins a device flow, replacing one already running.
func (s *GitHubService) StartConnect(ctx context.Context, session model.Session) (DeviceFlow, error) {
	if !s.api.Configured() {
		return DeviceFlow{}, ErrGitHubNotConfigured
	}
	code, err := s.api.StartDeviceFlow(ctx)
	if err != nil {
		return DeviceFlow{}, err
	}
	now := s.now()
	f := &deviceFlow{code: code, expiresAt: now.Add(code.ExpiresIn), nextPoll: now.Add(code.Interval), interval: code.Interval}
	s.mu.Lock()
	s.flows[session.UserID] = f
	s.mu.Unlock()
	return DeviceFlow{UserCode: code.UserCode, VerificationURI: code.VerificationURI, ExpiresAt: f.expiresAt, Interval: f.interval}, nil
}

// PollConnect checks whether the user has approved the device flow. Polls
// that come sooner than GitHub allows answer "pending" without asking it.
func (s *GitHubService) PollConnect(ctx context.Context, session model.Session) (ConnectResult, error) {
	s.mu.Lock()
	f, ok := s.flows[session.UserID]
	var flow deviceFlow
	if ok {
		flow = *f
	}
	s.mu.Unlock()
	if !ok {
		return ConnectResult{}, fmt.Errorf("%w: no connection in progress", ErrNotFound)
	}
	now := s.now()
	if !now.Before(flow.expiresAt) {
		s.endFlow(session.UserID, f)
		return ConnectResult{State: ConnectExpired}, nil
	}
	if now.Before(flow.nextPoll) {
		return ConnectResult{State: ConnectPending}, nil
	}

	token, scopes, interval, err := s.api.PollDeviceToken(ctx, flow.code.DeviceCode)
	switch {
	case errors.Is(err, model.ErrDeviceAuthPending), errors.Is(err, model.ErrDeviceSlowDown):
		s.mu.Lock()
		if errors.Is(err, model.ErrDeviceSlowDown) {
			f.interval = max(interval, f.interval+5*time.Second)
		}
		f.nextPoll = now.Add(f.interval)
		s.mu.Unlock()
		return ConnectResult{State: ConnectPending}, nil
	case errors.Is(err, model.ErrDeviceExpired):
		s.endFlow(session.UserID, f)
		return ConnectResult{State: ConnectExpired}, nil
	case errors.Is(err, model.ErrDeviceDenied):
		s.endFlow(session.UserID, f)
		return ConnectResult{State: ConnectDenied}, nil
	case err != nil:
		return ConnectResult{}, err
	}

	user, err := s.api.User(ctx, token)
	if err != nil {
		return ConnectResult{}, err
	}
	c := model.GitHubCredential{Token: token, Scopes: scopes, User: user, ConnectedAt: now.UTC().Truncate(time.Second)}
	if err := s.creds.Set(session.UserID, c); err != nil {
		return ConnectResult{}, err
	}
	s.endFlow(session.UserID, f)
	return ConnectResult{State: ConnectConnected, Account: toAccount(c)}, nil
}

// endFlow forgets a flow unless a newer one has replaced it.
func (s *GitHubService) endFlow(userID string, f *deviceFlow) {
	s.mu.Lock()
	if s.flows[userID] == f {
		delete(s.flows, userID)
	}
	s.mu.Unlock()
}

// Disconnect forgets the token. It stays valid on GitHub until the user
// revokes the app there; revoking needs the app's client secret, which a
// local app does not have.
func (s *GitHubService) Disconnect(_ context.Context, session model.Session) error {
	s.mu.Lock()
	delete(s.flows, session.UserID)
	s.mu.Unlock()
	return s.creds.Delete(session.UserID)
}

// token returns the user's token, or ErrGitHubNotConnected.
func (s *GitHubService) token(session model.Session) (string, error) {
	c, err := s.creds.Get(session.UserID)
	if errors.Is(err, model.ErrNoCredential) {
		return "", ErrGitHubNotConnected
	}
	if err != nil {
		return "", err
	}
	return c.Token, nil
}

// apiError translates a GitHub error for the caller. A rejected token is
// forgotten, so the user is asked to connect again.
func (s *GitHubService) apiError(session model.Session, token string, err error) error {
	switch {
	case errors.Is(err, model.ErrGitHubUnauthorized):
		if token != "" {
			s.forget(session, token)
		}
		return fmt.Errorf("%w: token rejected", ErrGitHubNotConnected)
	case errors.Is(err, model.ErrGitHubNotFound):
		return fmt.Errorf("%w: repository", ErrNotFound)
	}
	return err
}

// forget deletes the stored credential if it still holds the rejected token.
func (s *GitHubService) forget(session model.Session, token string) {
	c, err := s.creds.Get(session.UserID)
	if err != nil || c.Token != token {
		return
	}
	// A keychain failure here only means the user sees the rejection again
	// on the next call.
	_ = s.creds.Delete(session.UserID)
}

// Repos lists the repositories the connected account can see.
func (s *GitHubService) Repos(ctx context.Context, session model.Session) ([]model.GitHubRepo, error) {
	token, err := s.token(session)
	if err != nil {
		return nil, err
	}
	repos, err := s.api.Repos(ctx, token)
	if err != nil {
		return nil, s.apiError(session, token, err)
	}
	return repos, nil
}
