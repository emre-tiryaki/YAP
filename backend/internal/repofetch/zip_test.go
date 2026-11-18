package repofetch

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type entry struct {
	name    string
	content string
	mode    fs.FileMode
}

func archive(t *testing.T, entries []entry) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.mode != 0 {
			h.SetMode(e.mode)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func unzip(t *testing.T, entries []entry, limits Limits) (string, error) {
	t.Helper()
	dst := t.TempDir()
	r := archive(t, entries)
	return dst, Unzip(context.Background(), r, r.Size(), dst, limits)
}

func TestUnzipDropsTopFolder(t *testing.T) {
	dst, err := unzip(t, []entry{
		{name: "octo-hello-abc/"},
		{name: "octo-hello-abc/README.md", content: "hi"},
		{name: "octo-hello-abc/src/deep/main.go", content: "package main"},
		{name: "octo-hello-abc/empty/"},
	}, DefaultLimits)
	if err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}
	for name, want := range map[string]string{"README.md": "hi", "src/deep/main.go": "package main"} {
		got, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v", name, got, err)
		}
	}
	if fi, err := os.Stat(filepath.Join(dst, "empty")); err != nil || !fi.IsDir() {
		t.Errorf("empty folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "octo-hello-abc")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("top folder was kept")
	}
}

func TestUnzipSkipsUnsafeEntries(t *testing.T) {
	outside := t.TempDir()
	dst, err := unzip(t, []entry{
		{name: "top/../../escape.txt", content: "x"},
		{name: "top/a/../../escape2.txt", content: "x"},
		{name: "/abs.txt", content: "x"},
		{name: "top/back\\slash.txt", content: "x"},
		{name: "top/link", content: outside, mode: fs.ModeSymlink | 0o777},
		{name: "top/ok.txt", content: "ok"},
	}, DefaultLimits)
	if err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "ok.txt" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("unpacked %v, want only ok.txt", names)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dst), "escape.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("an entry escaped the target")
	}
}

func TestUnzipLimits(t *testing.T) {
	tests := []struct {
		name   string
		limits Limits
	}{
		{"too many bytes", Limits{MaxBytes: 5, MaxFiles: 10}},
		{"too many files", Limits{MaxBytes: 1 << 20, MaxFiles: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := unzip(t, []entry{{name: "top/a", content: "1234"}, {name: "top/b", content: "5678"}}, tt.limits)
			if !errors.Is(err, model.ErrRepositoryTooLarge) {
				t.Fatalf("Unzip() error = %v, want too large", err)
			}
		})
	}
}

func TestGitClassify(t *testing.T) {
	tests := []struct {
		stderr string
		want   error
	}{
		{"fatal: unable to access 'https://github.com/a/b.git/': Could not resolve host: github.com", model.ErrGitHubUnavailable},
		{"fatal: could not read Username for 'https://github.com': terminal prompts disabled", model.ErrGitHubUnauthorized},
		{"remote: Repository not found.\nfatal: repository 'https://github.com/a/b.git/' not found", model.ErrGitHubNotFound},
		{"warning: Remote branch dev not found in upstream origin", model.ErrGitHubNotFound},
	}
	for _, tt := range tests {
		if err := classify("clone", tt.stderr, errors.New("exit status 128")); !errors.Is(err, tt.want) {
			t.Errorf("classify(%q) = %v, want %v", tt.stderr, err, tt.want)
		}
	}
	if err := classify("clone", "fatal: something else", errors.New("exit status 1")); errors.Is(err, model.ErrGitHubNotFound) {
		t.Errorf("unknown failure classified as not found")
	}
}

func TestGitMissing(t *testing.T) {
	g := &Git{}
	if g.Available() {
		t.Fatal("zero Git claims to be available")
	}
	if err := g.Clone(context.Background(), "x", "", t.TempDir()+"/c"); !errors.Is(err, model.ErrGitMissing) {
		t.Fatalf("Clone() error = %v", err)
	}
}
