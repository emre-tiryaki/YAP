package repofetch

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func needGit(t *testing.T) *Git {
	t.Helper()
	g := NewGit()
	if g.Available() {
		return g
	}
	if os.Getenv("CI") != "" {
		t.Fatal("git is required in CI")
	}
	t.Skip("git is not installed")
	return nil
}

func sourceRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
			"GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "--quiet", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "--quiet", "-m", "init")
	return dir
}

func TestGitCloneLeavesNoToken(t *testing.T) {
	g := needGit(t)
	src := sourceRepo(t)
	dst := filepath.Join(t.TempDir(), "clone")
	if err := g.Clone(context.Background(), src, "secret-token", dst); err != nil {
		t.Fatalf("Clone() error = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dst, "README.md")); err != nil || string(got) != "# hi" {
		t.Fatalf("README = %q, %v", got, err)
	}
	config, err := os.ReadFile(filepath.Join(dst, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(config), "secret-token") || strings.Contains(string(config), "extraHeader") {
		t.Errorf("token leaked into .git/config:\n%s", config)
	}
}

func TestGitCloneFailures(t *testing.T) {
	g := needGit(t)
	t.Run("missing repository", func(t *testing.T) {
		err := g.Clone(context.Background(), filepath.Join(t.TempDir(), "nope"), "secret-token", filepath.Join(t.TempDir(), "c"))
		if err == nil {
			t.Fatal("Clone() of a missing repository returned no error")
		}
		if strings.Contains(err.Error(), "secret-token") {
			t.Errorf("error leaks the token: %v", err)
		}
	})
	t.Run("destination exists", func(t *testing.T) {
		src := sourceRepo(t)
		dst := t.TempDir()
		if err := os.WriteFile(filepath.Join(dst, "x"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := g.Clone(context.Background(), src, "", dst); err == nil {
			t.Fatal("Clone() into a non-empty folder returned no error")
		}
	})
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := g.Clone(ctx, sourceRepo(t), "", filepath.Join(t.TempDir(), "c")); !errors.Is(err, context.Canceled) {
			t.Fatalf("Clone() error = %v, want context.Canceled", err)
		}
	})
}

func TestFetcherDelegates(t *testing.T) {
	f := NewFetcher(&Git{}, DefaultLimits)
	if f.GitAvailable() {
		t.Error("GitAvailable() = true without a git binary")
	}
	if err := f.Clone(context.Background(), "x", "", filepath.Join(t.TempDir(), "c")); !errors.Is(err, model.ErrGitMissing) {
		t.Errorf("Clone() without git error = %v", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("top/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("hello"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := f.Unzip(context.Background(), bytes.NewReader(buf.Bytes()), int64(buf.Len()), dst); err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(got) != "hello" {
		t.Fatalf("a.txt = %q, %v", got, err)
	}
	if g := NewGit(); NewFetcher(g, DefaultLimits).GitAvailable() != g.Available() {
		t.Error("GitAvailable() disagrees with Git.Available()")
	}
}
