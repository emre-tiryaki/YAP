// Package github is a small client for the parts of the GitHub API the app
// uses: the OAuth device flow, the signed-in user, repositories, branch heads
// and source archives. Errors are reported with the sentinels in model, so
// callers can tell "offline" from "not allowed" without importing this
// package.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	DefaultAPIURL = "https://api.github.com"
	DefaultWebURL = "https://github.com"
	// Scope asks for read access to private repositories too. GitHub has no
	// read-only scope for private repositories of an OAuth app.
	Scope = "repo"

	maxRepoPages = 10
	perPage      = 100
)

type Client struct {
	http     *http.Client
	apiURL   string
	webURL   string
	clientID string
}

// New returns a client for github.com. clientID is the OAuth app's client
// id; it is not a secret.
func New(clientID string) *Client {
	return NewWithURLs(clientID, DefaultAPIURL, DefaultWebURL, &http.Client{Timeout: 30 * time.Second})
}

// NewWithURLs points the client somewhere else, for tests.
func NewWithURLs(clientID, apiURL, webURL string, hc *http.Client) *Client {
	return &Client{http: hc, apiURL: strings.TrimRight(apiURL, "/"), webURL: strings.TrimRight(webURL, "/"), clientID: clientID}
}

// Configured reports whether an OAuth client id is set.
func (c *Client) Configured() bool {
	return c.clientID != ""
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	res, err := c.http.Do(req)
	if err != nil {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, ctxErr
		}
		// DNS failures, refused connections and timeouts all mean "offline"
		// for the user.
		return nil, fmt.Errorf("%w: %v", model.ErrGitHubUnavailable, err)
	}
	return res, nil
}

// checkStatus turns an error answer into a sentinel error and closes its
// body.
func checkStatus(res *http.Response) error {
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10)) // drain for connection reuse
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return model.ErrGitHubUnauthorized
	case res.StatusCode == http.StatusNotFound:
		return model.ErrGitHubNotFound
	case res.StatusCode == http.StatusTooManyRequests,
		// Primary limits set X-RateLimit-Remaining, secondary (abuse)
		// limits only Retry-After.
		res.StatusCode == http.StatusForbidden && (res.Header.Get("X-RateLimit-Remaining") == "0" || res.Header.Get("Retry-After") != ""):
		return fmt.Errorf("%w: rate limited", model.ErrGitHubUnavailable)
	case res.StatusCode == http.StatusForbidden:
		// Forbidden without a rate limit: SSO enforcement or a blocked
		// repository. The token cannot see it.
		return fmt.Errorf("%w: forbidden", model.ErrGitHubNotFound)
	case res.StatusCode >= 500:
		return fmt.Errorf("%w: status %d", model.ErrGitHubUnavailable, res.StatusCode)
	default:
		return fmt.Errorf("github: unexpected status %d", res.StatusCode)
	}
}

func (c *Client) apiRequest(ctx context.Context, token, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func (c *Client) getJSON(ctx context.Context, token, path string, dst any) (http.Header, error) {
	req, err := c.apiRequest(ctx, token, path)
	if err != nil {
		return nil, err
	}
	res, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(res); err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(io.LimitReader(res.Body, 16<<20)).Decode(dst); err != nil {
		return nil, fmt.Errorf("github: decode %s: %w", path, err)
	}
	return res.Header, nil
}

func (c *Client) postForm(ctx context.Context, path string, form url.Values, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := c.do(req)
	if err != nil {
		return err
	}
	// The OAuth endpoints explain a refusal in a JSON "error" field, also
	// with 400 and 401; the caller reads it.
	if res.StatusCode != http.StatusBadRequest && res.StatusCode != http.StatusUnauthorized {
		if err := checkStatus(res); err != nil {
			return err
		}
	}
	defer res.Body.Close()
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(dst); err != nil {
		return fmt.Errorf("github: decode %s: %w", path, err)
	}
	return nil
}

// StartDeviceFlow asks GitHub for a code the user enters in the browser.
func (c *Client) StartDeviceFlow(ctx context.Context) (model.DeviceCode, error) {
	var body struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
	}
	form := url.Values{"client_id": {c.clientID}, "scope": {Scope}}
	if err := c.postForm(ctx, "/login/device/code", form, &body); err != nil {
		// An unknown client id is answered with a plain 404.
		if errors.Is(err, model.ErrGitHubNotFound) {
			return model.DeviceCode{}, model.ErrGitHubClientInvalid
		}
		return model.DeviceCode{}, err
	}
	switch {
	case body.Error == "device_flow_disabled":
		return model.DeviceCode{}, model.ErrDeviceFlowDisabled
	case body.Error == "incorrect_client_credentials", body.Error == "unauthorized_client":
		return model.DeviceCode{}, model.ErrGitHubClientInvalid
	case body.Error != "" || body.DeviceCode == "":
		return model.DeviceCode{}, fmt.Errorf("github: device code: %s", body.Error)
	}
	return model.DeviceCode{
		DeviceCode:      body.DeviceCode,
		UserCode:        body.UserCode,
		VerificationURI: body.VerificationURI,
		ExpiresIn:       time.Duration(body.ExpiresIn) * time.Second,
		Interval:        time.Duration(max(body.Interval, 1)) * time.Second,
	}, nil
}

// PollDeviceToken asks whether the user has approved the device code. While
// they have not it returns model.ErrDeviceAuthPending (or ErrDeviceSlowDown
// with the new interval).
func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (token, scopes string, interval time.Duration, err error) {
	var body struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		Interval    int    `json:"interval"`
	}
	form := url.Values{
		"client_id":   {c.clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	if err := c.postForm(ctx, "/login/oauth/access_token", form, &body); err != nil {
		return "", "", 0, err
	}
	switch body.Error {
	case "":
		if body.AccessToken == "" {
			return "", "", 0, errors.New("github: empty access token")
		}
		return body.AccessToken, body.Scope, 0, nil
	case "authorization_pending":
		return "", "", 0, model.ErrDeviceAuthPending
	case "slow_down":
		return "", "", time.Duration(body.Interval) * time.Second, model.ErrDeviceSlowDown
	case "expired_token":
		return "", "", 0, model.ErrDeviceExpired
	case "access_denied":
		return "", "", 0, model.ErrDeviceDenied
	case "device_flow_disabled":
		return "", "", 0, model.ErrDeviceFlowDisabled
	case "incorrect_client_credentials", "unauthorized_client":
		return "", "", 0, model.ErrGitHubClientInvalid
	default:
		return "", "", 0, fmt.Errorf("github: device token: %s", body.Error)
	}
}

type userDTO struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// User returns the owner of the token.
func (c *Client) User(ctx context.Context, token string) (model.GitHubUser, error) {
	var u userDTO
	if _, err := c.getJSON(ctx, token, "/user", &u); err != nil {
		return model.GitHubUser{}, err
	}
	return model.GitHubUser{ID: u.ID, Login: u.Login, Name: u.Name, AvatarURL: u.AvatarURL}, nil
}

type repoDTO struct {
	ID            int64   `json:"id"`
	FullName      string  `json:"full_name"`
	Description   *string `json:"description"`
	HTMLURL       string  `json:"html_url"`
	CloneURL      string  `json:"clone_url"`
	DefaultBranch string  `json:"default_branch"`
	Private       bool    `json:"private"`
	PushedAt      *string `json:"pushed_at"`
}

func (d repoDTO) toModel() model.GitHubRepo {
	r := model.GitHubRepo{
		ID: d.ID, FullName: d.FullName, HTMLURL: d.HTMLURL, CloneURL: d.CloneURL,
		DefaultBranch: d.DefaultBranch, Private: d.Private,
	}
	if d.Description != nil {
		r.Description = *d.Description
	}
	if d.PushedAt != nil {
		if t, err := time.Parse(time.RFC3339, *d.PushedAt); err == nil {
			r.PushedAt = &t
		}
	}
	return r
}

// Repos lists the repositories the token's owner can see (owned,
// collaborator and organization ones), most recently updated first, up to
// 1000.
func (c *Client) Repos(ctx context.Context, token string) ([]model.GitHubRepo, error) {
	out := []model.GitHubRepo{}
	for page := 1; page <= maxRepoPages; page++ {
		var batch []repoDTO
		path := "/user/repos?sort=updated&affiliation=owner,collaborator,organization_member" +
			"&per_page=" + strconv.Itoa(perPage) + "&page=" + strconv.Itoa(page)
		if _, err := c.getJSON(ctx, token, path, &batch); err != nil {
			return nil, err
		}
		for _, d := range batch {
			out = append(out, d.toModel())
		}
		if len(batch) < perPage {
			break
		}
	}
	return out, nil
}

// validFullName accepts "owner/name" with the characters GitHub allows.
func validFullName(fullName string) bool {
	owner, name, ok := strings.Cut(fullName, "/")
	if !ok || owner == "" || name == "" || strings.Trim(owner, ".") == "" || strings.Trim(name, ".") == "" {
		return false
	}
	for _, r := range owner + name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

// Repo looks a repository up by "owner/name". token may be empty for a
// public repository.
func (c *Client) Repo(ctx context.Context, token, fullName string) (model.GitHubRepo, error) {
	if !validFullName(fullName) {
		return model.GitHubRepo{}, model.ErrGitHubNotFound
	}
	var d repoDTO
	if _, err := c.getJSON(ctx, token, "/repos/"+fullName, &d); err != nil {
		return model.GitHubRepo{}, err
	}
	return d.toModel(), nil
}

// RepoByID looks a repository up by its id, which survives renames and
// transfers.
func (c *Client) RepoByID(ctx context.Context, token string, id int64) (model.GitHubRepo, error) {
	var d repoDTO
	if _, err := c.getJSON(ctx, token, "/repositories/"+strconv.FormatInt(id, 10), &d); err != nil {
		return model.GitHubRepo{}, err
	}
	return d.toModel(), nil
}

// BranchHead returns the commit a branch points at.
func (c *Client) BranchHead(ctx context.Context, token, fullName, branch string) (string, error) {
	if !validFullName(fullName) {
		return "", model.ErrGitHubNotFound
	}
	var body struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if _, err := c.getJSON(ctx, token, "/repos/"+fullName+"/branches/"+escapeSegments(branch), &body); err != nil {
		return "", err
	}
	if body.Commit.SHA == "" {
		return "", errors.New("github: branch without a commit")
	}
	return body.Commit.SHA, nil
}

// CommitsPerPage is how many commits one page of history holds.
const CommitsPerPage = 50

// Commits returns a page (from 1) of a branch's history, newest first.
func (c *Client) Commits(ctx context.Context, token, fullName, branch string, page int) ([]model.GitHubCommit, error) {
	if !validFullName(fullName) {
		return nil, model.ErrGitHubNotFound
	}
	q := url.Values{"sha": {branch}, "per_page": {strconv.Itoa(CommitsPerPage)}, "page": {strconv.Itoa(page)}}
	var body []struct {
		SHA     string `json:"sha"`
		HTMLURL string `json:"html_url"`
		Commit  struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		Author *struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
		} `json:"author"`
		Parents []struct {
			SHA string `json:"sha"`
		} `json:"parents"`
	}
	if _, err := c.getJSON(ctx, token, "/repos/"+fullName+"/commits?"+q.Encode(), &body); err != nil {
		return nil, err
	}
	out := make([]model.GitHubCommit, 0, len(body))
	for _, d := range body {
		cm := model.GitHubCommit{
			SHA: d.SHA, Message: d.Commit.Message, AuthorName: d.Commit.Author.Name, HTMLURL: d.HTMLURL,
			Parents: make([]string, 0, len(d.Parents)),
		}
		if t, err := time.Parse(time.RFC3339, d.Commit.Author.Date); err == nil {
			cm.Date = t
		}
		if d.Author != nil {
			cm.AuthorLogin, cm.AvatarURL = d.Author.Login, d.Author.AvatarURL
		}
		for _, p := range d.Parents {
			cm.Parents = append(cm.Parents, p.SHA)
		}
		out = append(out, cm)
	}
	return out, nil
}

// escapeSegments escapes a branch name but keeps its slashes, which GitHub
// expects as they are.
func escapeSegments(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// Zipball streams the source archive of a commit. The caller closes it.
func (c *Client) Zipball(ctx context.Context, token, fullName, sha string) (io.ReadCloser, error) {
	if !validFullName(fullName) {
		return nil, model.ErrGitHubNotFound
	}
	req, err := c.apiRequest(ctx, token, "/repos/"+fullName+"/zipball/"+url.PathEscape(sha))
	if err != nil {
		return nil, err
	}
	// Archives can be large; only the connection setup has a timeout.
	hc := *c.http
	hc.Timeout = 0
	res, err := (&Client{http: &hc}).do(req)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(res); err != nil {
		return nil, err
	}
	return res.Body, nil
}
