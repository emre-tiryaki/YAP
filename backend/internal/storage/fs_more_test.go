package storage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNewFSCannotCreateRoot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFS(filepath.Join(file, "root")); err == nil {
		t.Fatal("NewFS() under a file returned no error")
	}
}

func TestPathAndMakeDir(t *testing.T) {
	s := newTestFS(t)
	if err := s.MakeDir([]string{"ws", "a", "b"}); err != nil {
		t.Fatalf("MakeDir() error = %v", err)
	}
	p, err := s.Path([]string{"ws", "a", "b"})
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if info, err := os.Stat(p); err != nil || !info.IsDir() {
		t.Fatalf("Stat(%s) = %v, %v", p, info, err)
	}
	// Making a directory twice is fine.
	if err := s.MakeDir([]string{"ws", "a", "b"}); err != nil {
		t.Errorf("second MakeDir() error = %v", err)
	}
	for _, bad := range [][]string{nil, {""}, {".."}, {"a/b"}, {"a", "."}, {"a\\b"}} {
		if _, err := s.Path(bad); !errors.Is(err, errBadSegment) {
			t.Errorf("Path(%q) error = %v", bad, err)
		}
		if err := s.MakeDir(bad); !errors.Is(err, errBadSegment) {
			t.Errorf("MakeDir(%q) error = %v", bad, err)
		}
	}
}

func TestMoveAndOpenErrors(t *testing.T) {
	s := newTestFS(t)
	if err := s.Move([]string{"ws", "missing"}, []string{"ws", "x"}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Move(missing) error = %v, want fs.ErrNotExist", err)
	}
	if err := s.MakeDir([]string{"ws", "src"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Move([]string{"ws", "src"}, []string{"ws", ".."}); !errors.Is(err, errBadSegment) {
		t.Errorf("Move(bad target) error = %v", err)
	}
	if err := s.Move([]string{"..", "src"}, []string{"ws", "x"}); !errors.Is(err, errBadSegment) {
		t.Errorf("Move(bad source) error = %v", err)
	}
	// Moving creates the target's parent folders.
	if err := s.Move([]string{"ws", "src"}, []string{"ws", "deep", "er", "dst"}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if got, _ := s.DirNames([]string{"ws", "deep", "er"}); !slices.Equal(got, []string{"dst"}) {
		t.Errorf("DirNames() = %v", got)
	}
	if _, err := s.Open([]string{"ws", "nothing"}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Open(missing) error = %v", err)
	}
	if _, err := s.Open([]string{"ws", ".."}); !errors.Is(err, errBadSegment) {
		t.Errorf("Open(bad) error = %v", err)
	}
	if err := s.Copy([]string{"ws", "nothing"}, []string{"ws", "copy"}); err == nil {
		t.Error("Copy(missing) returned no error")
	}
	if err := s.Remove([]string{"ws", ".."}); !errors.Is(err, errBadSegment) {
		t.Errorf("Remove(bad) error = %v", err)
	}
}

func TestTempDirAndTempFile(t *testing.T) {
	s := newTestFS(t)
	dir, err := s.TempDir()
	if err != nil {
		t.Fatalf("TempDir() error = %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("temp dir: %v, %v", info, err)
	}
	f, err := s.TempFile()
	if err != nil {
		t.Fatalf("TempFile() error = %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	// Both live inside the tree, so renaming them into place stays on one
	// file system.
	root, _ := filepath.Abs(s.root)
	for _, p := range []string{dir, f.Name()} {
		if !strings.HasPrefix(p, root+string(filepath.Separator)) {
			t.Errorf("%s is outside the root %s", p, root)
		}
	}
}

func writeTree(t *testing.T, dir string, files map[string]string) {
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
}

func TestInstallDir(t *testing.T) {
	s := newTestFS(t)
	target := []string{"ws", "repos", "r1"}

	first, err := s.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	writeTree(t, first, map[string]string{"a.txt": "one", "old-only.txt": "x"})
	if err := s.InstallDir(first, target); err != nil {
		t.Fatalf("InstallDir() into nothing error = %v", err)
	}
	dst, _ := s.Path(target)
	if got, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(got) != "one" {
		t.Fatalf("a.txt = %q, %v", got, err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Errorf("source still exists after install: %v", err)
	}

	second, err := s.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	writeTree(t, second, map[string]string{"a.txt": "two"})
	if err := s.InstallDir(second, target); err != nil {
		t.Fatalf("InstallDir() over an old copy error = %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "a.txt")); string(got) != "two" {
		t.Errorf("a.txt = %q, want the new copy", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "old-only.txt")); !os.IsNotExist(err) {
		t.Errorf("a file of the old copy survived the replace: %v", err)
	}
	if _, err := os.Stat(second + ".old"); !os.IsNotExist(err) {
		t.Errorf("old copy was not cleaned up: %v", err)
	}
}

func TestInstallDirPutsTheOldCopyBackOnFailure(t *testing.T) {
	s := newTestFS(t)
	target := []string{"ws", "r"}
	old, _ := s.TempDir()
	writeTree(t, old, map[string]string{"keep.txt": "old"})
	if err := s.InstallDir(old, target); err != nil {
		t.Fatal(err)
	}
	// A source that does not exist makes the second rename fail.
	missing := filepath.Join(t.TempDir(), "nope")
	if err := s.InstallDir(missing, target); err == nil {
		t.Fatal("InstallDir() of a missing source returned no error")
	}
	dst, _ := s.Path(target)
	if got, err := os.ReadFile(filepath.Join(dst, "keep.txt")); err != nil || string(got) != "old" {
		t.Fatalf("old copy after failed install = %q, %v", got, err)
	}
	if err := s.InstallDir(missing, []string{".."}); !errors.Is(err, errBadSegment) {
		t.Errorf("InstallDir(bad path) error = %v", err)
	}
}

func TestDirNames(t *testing.T) {
	s := newTestFS(t)
	if got, err := s.DirNames([]string{"ws", "none"}); err != nil || got != nil {
		t.Errorf("DirNames(missing) = %v, %v; want nil, nil", got, err)
	}
	for _, n := range []string{"b", "a"} {
		if err := s.MakeDir([]string{"ws", n}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.DirNames([]string{"ws"})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("DirNames() = %v", got)
	}
	if _, err := s.DirNames([]string{".."}); !errors.Is(err, errBadSegment) {
		t.Errorf("DirNames(bad) error = %v", err)
	}
}

func TestStageCanceledAndPlace(t *testing.T) {
	s := newTestFS(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Stage(ctx, strings.NewReader("data")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Stage(canceled) error = %v", err)
	}
	// The failed upload leaves no temp file behind.
	if left, _ := os.ReadDir(filepath.Join(s.root, tempDir)); len(left) != 0 {
		t.Errorf("temp dir holds %d leftovers after a failed stage", len(left))
	}

	staged, err := s.Stage(context.Background(), strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Place(staged, []string{"ws", "sub", "f.txt"}); err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if err := s.Place(staged, []string{"ws", ".."}); !errors.Is(err, errBadSegment) {
		t.Errorf("Place(bad path) error = %v", err)
	}
	f, err := s.Open([]string{"ws", "sub", "f.txt"})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if string(got) != "hello" {
		t.Errorf("placed content = %q", got)
	}
}
