// Package storage keeps resource file bytes on the local disk. Paths are given
// as segments relative to the root; the service layer decides the layout.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const tempDir = ".tmp"

var errBadSegment = errors.New("invalid path segment")

type FS struct {
	root string
}

func NewFS(root string) (*FS, error) {
	if err := os.MkdirAll(filepath.Join(root, tempDir), 0o755); err != nil {
		return nil, fmt.Errorf("create resources root: %w", err)
	}
	return &FS{root: root}, nil
}

// abs joins the segments under the root. Each segment must be a single path
// element, so nothing can escape the root.
func (s *FS) abs(path []string) (string, error) {
	if len(path) == 0 {
		return "", errBadSegment
	}
	for _, seg := range path {
		if seg == "" || seg == "." || seg == ".." ||
			strings.ContainsAny(seg, "/\\\x00") {
			return "", fmt.Errorf("%w: %q", errBadSegment, seg)
		}
	}
	return filepath.Join(append([]string{s.root}, path...)...), nil
}

// Stage copies r into a temporary file and reports its size and whether it
// is text.
func (s *FS) Stage(ctx context.Context, r io.Reader) (model.StagedFile, error) {
	f, err := os.CreateTemp(filepath.Join(s.root, tempDir), "upload-*")
	if err != nil {
		return model.StagedFile{}, fmt.Errorf("create temp file: %w", err)
	}
	det := &textDetector{valid: true}
	n, err := io.Copy(io.MultiWriter(f, det), ctxReader{ctx, r})
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(f.Name()) // best effort: the copy already failed
		return model.StagedFile{}, fmt.Errorf("stage upload: %w", err)
	}
	return model.StagedFile{TempPath: f.Name(), Size: n, IsText: det.done(), Head: det.head}, nil
}

// Discard removes a staged file that was not placed.
func (s *FS) Discard(staged model.StagedFile) {
	if staged.TempPath != "" {
		_ = os.Remove(staged.TempPath) // already placed or never written
	}
}

// Place moves a staged file to path, replacing a file already there.
func (s *FS) Place(staged model.StagedFile, path []string) error {
	dst, err := s.abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.Rename(staged.TempPath, dst); err != nil {
		return fmt.Errorf("place file: %w", err)
	}
	return nil
}

// Open opens the file at path for reading.
func (s *FS) Open(path []string) (*os.File, error) {
	p, err := s.abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *FS) MakeDir(path []string) error {
	p, err := s.abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return fmt.Errorf("make dir: %w", err)
	}
	return nil
}

// Move renames from to to. A missing source is reported as fs.ErrNotExist.
func (s *FS) Move(from, to []string) error {
	src, err := s.abs(from)
	if err != nil {
		return err
	}
	dst, err := s.abs(to)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(src); err != nil {
		return fmt.Errorf("move: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("move: %w", err)
	}
	return nil
}

// Copy copies a file or a whole directory from from to to.
func (s *FS) Copy(from, to []string) error {
	src, err := s.abs(from)
	if err != nil {
		return err
	}
	dst, err := s.abs(to)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(src); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil // links and devices are not resources
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close() // the copy error is the one worth reporting
		return fmt.Errorf("copy file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}

// Remove deletes the file or directory tree at path; a missing path is fine.
func (s *FS) Remove(path []string) error {
	p, err := s.abs(path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(p); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

const headSize = 512

// textDetector checks a byte stream for valid UTF-8 without NUL bytes. A rune
// split across two writes is carried over to the next write.
type textDetector struct {
	valid    bool
	leftover []byte
	head     []byte
}

func (d *textDetector) Write(p []byte) (int, error) {
	if len(d.head) < headSize {
		d.head = append(d.head, p[:min(len(p), headSize-len(d.head))]...)
	}
	if !d.valid {
		return len(p), nil
	}
	buf := append(d.leftover, p...)
	cut := len(buf)
	for i := len(buf) - 1; i >= 0 && i >= len(buf)-utf8.UTFMax; i-- {
		if utf8.RuneStart(buf[i]) {
			if !utf8.FullRune(buf[i:]) {
				cut = i
			}
			break
		}
	}
	chunk := buf[:cut]
	if !utf8.Valid(chunk) || strings.IndexByte(string(chunk), 0) >= 0 {
		d.valid = false
	}
	d.leftover = append([]byte(nil), buf[cut:]...)
	return len(p), nil
}

func (d *textDetector) done() bool {
	return d.valid && len(d.leftover) == 0
}
