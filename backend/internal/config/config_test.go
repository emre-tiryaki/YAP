package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolate runs the test in dir with a clean environment for every variable
// Load reads.
func isolate(t *testing.T, dir string) {
	t.Helper()
	for _, k := range []string{"DB_PATH", "RESOURCES_PATH", "BACKEND_PORT", "DISPLAY_NAME", "GITHUB_CLIENT_ID"} {
		old, had := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		})
	}
	t.Chdir(dir)
}

func realDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefaultsWithoutEnvFile(t *testing.T) {
	dir := realDir(t)
	// A .git marker stops the walk, so a stray .env above the temp dir is
	// never picked up.
	write(t, filepath.Join(dir, ".git", "HEAD"), "")
	isolate(t, dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPath != filepath.Join(dir, "app.db") {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.ResourcesPath != filepath.Join(dir, "resources") {
		t.Errorf("ResourcesPath = %q", cfg.ResourcesPath)
	}
	if cfg.Port != "8080" || cfg.DisplayName == "" || cfg.GitHubClientID != "" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoadReadsEnvFileFoundAbove(t *testing.T) {
	root := realDir(t)
	write(t, filepath.Join(root, ".git", "HEAD"), "")
	write(t, filepath.Join(root, ".env"),
		"DB_PATH=data/yap.db\nBACKEND_PORT=9090\nDISPLAY_NAME=Ada\nGITHUB_CLIENT_ID=cid\n")
	sub := filepath.Join(root, "backend", "cmd")
	write(t, filepath.Join(sub, "keep"), "")
	isolate(t, sub)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Relative paths resolve against the .env directory, not the cwd.
	if cfg.DBPath != filepath.Join(root, "data", "yap.db") {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.ResourcesPath != filepath.Join(root, "data", "resources") {
		t.Errorf("ResourcesPath = %q", cfg.ResourcesPath)
	}
	if cfg.Port != "9090" || cfg.DisplayName != "Ada" || cfg.GitHubClientID != "cid" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoadResourcesPathOverride(t *testing.T) {
	root := realDir(t)
	write(t, filepath.Join(root, ".git", "HEAD"), "")
	abs := filepath.Join(realDir(t), "files")
	write(t, filepath.Join(root, ".env"), "RESOURCES_PATH="+abs+"\n")
	isolate(t, root)
	cfg, err := Load()
	if err != nil || cfg.ResourcesPath != abs {
		t.Fatalf("absolute: %q, %v", cfg.ResourcesPath, err)
	}

	write(t, filepath.Join(root, ".env"), "RESOURCES_PATH=rel/files\n")
	isolate(t, root)
	cfg, err = Load()
	if err != nil || cfg.ResourcesPath != filepath.Join(root, "rel", "files") {
		t.Fatalf("relative: %q, %v", cfg.ResourcesPath, err)
	}
}

func TestLoadEnvironmentBeatsEnvFile(t *testing.T) {
	root := realDir(t)
	write(t, filepath.Join(root, ".git", "HEAD"), "")
	write(t, filepath.Join(root, ".env"), "BACKEND_PORT=1111\n")
	isolate(t, root)
	t.Setenv("BACKEND_PORT", "2222")
	cfg, err := Load()
	if err != nil || cfg.Port != "2222" {
		t.Fatalf("Port = %q, %v", cfg.Port, err)
	}
}

func TestFindUpStopsAtRepoRoot(t *testing.T) {
	outer := realDir(t)
	write(t, filepath.Join(outer, ".env"), "x=1\n")
	repo := filepath.Join(outer, "repo")
	write(t, filepath.Join(repo, ".git", "HEAD"), "")
	sub := filepath.Join(repo, "a", "b")
	write(t, filepath.Join(sub, "keep"), "")
	t.Chdir(sub)
	if p, ok := findUp(".env"); ok {
		t.Fatalf("findUp() = %q, want none past the .git directory", p)
	}
	write(t, filepath.Join(repo, ".env"), "x=1\n")
	if p, ok := findUp(".env"); !ok || p != filepath.Join(repo, ".env") {
		t.Fatalf("findUp() = %q, %v", p, ok)
	}
}

func TestResolve(t *testing.T) {
	if got := resolve("/base", "/abs/x"); got != "/abs/x" {
		t.Errorf("absolute = %q", got)
	}
	if got := resolve("/base", "./x/y"); got != "/base/x/y" {
		t.Errorf("relative = %q", got)
	}
}

func TestLoadRejectsBrokenEnvFile(t *testing.T) {
	root := realDir(t)
	write(t, filepath.Join(root, ".git", "HEAD"), "")
	write(t, filepath.Join(root, ".env"), "KEY=\"unterminated\n")
	isolate(t, root)
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want a parse error")
	}
}
