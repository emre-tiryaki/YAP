package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestFS(t *testing.T) *FS {
	t.Helper()
	s, err := NewFS(t.TempDir())
	if err != nil {
		t.Fatalf("NewFS() error = %v", err)
	}
	return s
}

// chunked hands out one byte per Read, so multi-byte runes are split.
type chunked struct{ r io.Reader }

func (c chunked) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return c.r.Read(p)
}

func TestStageDetectsText(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{"ascii", []byte("hello"), true},
		{"multibyte split across reads", []byte("çğüşİ 日本"), true},
		{"empty", nil, true},
		{"nul byte", []byte("a\x00b"), false},
		{"invalid utf8", []byte{0xff, 0xfe, 'a'}, false},
		{"truncated rune", []byte("ab\xe6\x97"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestFS(t)
			staged, err := s.Stage(context.Background(), chunked{bytes.NewReader(tt.content)})
			if err != nil {
				t.Fatalf("Stage() error = %v", err)
			}
			defer s.Discard(staged)
			if staged.IsText != tt.want {
				t.Errorf("IsText = %v, want %v", staged.IsText, tt.want)
			}
			if staged.Size != int64(len(tt.content)) {
				t.Errorf("Size = %d, want %d", staged.Size, len(tt.content))
			}
		})
	}
}

func TestPlaceMoveCopyRemove(t *testing.T) {
	s := newTestFS(t)
	ctx := context.Background()

	staged, err := s.Stage(ctx, strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}
	if err := s.Place(staged, []string{"ws", "a", "f.txt"}); err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if err := s.Copy([]string{"ws", "a"}, []string{"ws", "b"}); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}
	if err := s.Move([]string{"ws", "a"}, []string{"ws", ".trash", "x", "a"}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	f, err := s.Open([]string{"ws", "b", "f.txt"})
	if err != nil {
		t.Fatalf("Open() copy error = %v", err)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if string(got) != "data" {
		t.Errorf("copied content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(s.root, "ws", ".trash", "x", "a", "f.txt")); err != nil {
		t.Errorf("moved file missing: %v", err)
	}
	if err := s.Remove([]string{"ws", ".trash", "x"}); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if err := s.Move([]string{"ws", "missing"}, []string{"ws", "c"}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Move() missing source error = %v, want fs.ErrNotExist", err)
	}
}

func TestPathSegmentsCannotEscapeRoot(t *testing.T) {
	s := newTestFS(t)
	for _, path := range [][]string{{".."}, {"ws", "../x"}, {"ws", "a/b"}, {""}, {}} {
		if err := s.MakeDir(path); err == nil {
			t.Errorf("MakeDir(%q) error = nil, want error", path)
		}
	}
}
