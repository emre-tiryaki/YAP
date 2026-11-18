package repofetch

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// Git runs the git program. The zero value is not usable; use NewGit.
type Git struct {
	bin string
}

// NewGit finds git on the PATH. Without it every call returns
// model.ErrGitMissing.
func NewGit() *Git {
	bin, err := exec.LookPath("git")
	if err != nil {
		return &Git{}
	}
	return &Git{bin: bin}
}

func (g *Git) Available() bool {
	return g.bin != ""
}

// run executes git with the token as an HTTP header. The token goes through
// the environment, so it is neither on the command line nor written to
// .git/config.
func (g *Git) run(ctx context.Context, dir, token string, args ...string) (string, error) {
	if g.bin == "" {
		return "", model.ErrGitMissing
	}
	// No credential helper or prompt may get in the way of a background
	// fetch.
	base := []string{"-c", "credential.helper=", "-c", "core.askPass="}
	cmd := exec.CommandContext(ctx, g.bin, append(base, args...)...)
	cmd.Dir = dir
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "LC_ALL=C")
	if token != "" {
		basic := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
		env = append(env,
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=http.extraHeader",
			"GIT_CONFIG_VALUE_0=Authorization: Basic "+basic,
		)
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", classify(args[0], stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// classify maps git's complaints to the errors the app tells apart.
func classify(command, stderr string, err error) error {
	msg := strings.ToLower(stderr)
	switch {
	case strings.Contains(msg, "could not resolve host"),
		strings.Contains(msg, "failed to connect"),
		strings.Contains(msg, "connection timed out"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "operation timed out"):
		return fmt.Errorf("%w: git %s", model.ErrGitHubUnavailable, command)
	case strings.Contains(msg, "authentication failed"),
		strings.Contains(msg, "could not read username"),
		strings.Contains(msg, "terminal prompts disabled"):
		return fmt.Errorf("%w: git %s", model.ErrGitHubUnauthorized, command)
	case strings.Contains(msg, "not found"),
		strings.Contains(msg, "does not appear to be a git repository"),
		strings.Contains(msg, "couldn't find remote ref"):
		return fmt.Errorf("%w: git %s", model.ErrGitHubNotFound, command)
	}
	// stderr can name the URL but never the token, which only lives in the
	// environment.
	return fmt.Errorf("git %s: %w: %s", command, err, strings.TrimSpace(stderr))
}

// Clone makes a full clone into dst, which must not exist. The remote stays
// the plain URL: the token is not saved in the clone, so later pulls of a
// private repository use the user's own git credentials.
func (g *Git) Clone(ctx context.Context, cloneURL, token, dst string) error {
	_, err := g.run(ctx, "", token, "clone", "--quiet", "--", cloneURL, dst)
	return err
}

// Fetcher unpacks the copies the app shows and clones repositories for the
// user.
type Fetcher struct {
	git    *Git
	limits Limits
}

func NewFetcher(git *Git, limits Limits) *Fetcher {
	return &Fetcher{git: git, limits: limits}
}

func (f *Fetcher) GitAvailable() bool {
	return f.git.Available()
}

func (f *Fetcher) Unzip(ctx context.Context, src io.ReaderAt, size int64, dst string) error {
	return Unzip(ctx, src, size, dst, f.limits)
}

func (f *Fetcher) Clone(ctx context.Context, cloneURL, token, dst string) error {
	return f.git.Clone(ctx, cloneURL, token, dst)
}
