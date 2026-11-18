// Package repofetch writes the files of a repository to a directory, from a
// GitHub source archive or with the git program.
package repofetch

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// Limits bound what an archive may unpack to.
type Limits struct {
	MaxBytes int64
	MaxFiles int
}

// DefaultLimits are generous for source code and stop runaway archives.
var DefaultLimits = Limits{MaxBytes: 2 << 30, MaxFiles: 200_000}

// Unzip unpacks a GitHub source archive into dst, which must exist and be
// empty. The archive's single top folder ("owner-repo-sha/") is dropped.
// Links and special files are skipped; entries that would land outside dst
// are rejected. Exceeding the limits returns model.ErrRepositoryTooLarge.
func Unzip(ctx context.Context, src io.ReaderAt, size int64, dst string, limits Limits) error {
	zr, err := zip.NewReader(src, size)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	root, err := os.OpenRoot(dst)
	if err != nil {
		return fmt.Errorf("open target: %w", err)
	}
	defer root.Close()

	var written int64
	files := 0
	for _, f := range zr.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name, ok := entryName(f.Name)
		if !ok {
			continue
		}
		mode := f.Mode()
		switch {
		case mode.IsDir():
			if err := root.MkdirAll(name, 0o755); err != nil {
				return fmt.Errorf("make dir %q: %w", name, err)
			}
			continue
		case !mode.IsRegular():
			continue // links and devices are not copied
		}
		files++
		if files > limits.MaxFiles {
			return fmt.Errorf("%w: more than %d files", model.ErrRepositoryTooLarge, limits.MaxFiles)
		}
		if dir := path.Dir(name); dir != "." {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("make dir %q: %w", dir, err)
			}
		}
		n, err := writeEntry(root, f, name, limits.MaxBytes-written)
		written += n
		if err != nil {
			return err
		}
	}
	return nil
}

// entryName drops the archive's top folder and cleans the rest. It reports
// false for the top folder itself and for names that are not plain relative
// paths.
func entryName(raw string) (string, bool) {
	if strings.ContainsAny(raw, "\\\x00") {
		return "", false
	}
	top, rest, found := strings.Cut(raw, "/")
	if !found || top == "" || top == "." || top == ".." {
		return "", false
	}
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return "", false
	}
	clean := path.Clean(rest)
	if clean != rest || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func writeEntry(root *os.Root, f *zip.File, name string, budget int64) (int64, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, fmt.Errorf("read %q: %w", name, err)
	}
	defer rc.Close()
	out, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("create %q: %w", name, err)
	}
	// The header's size can lie; count what is actually written.
	n, err := io.Copy(out, io.LimitReader(rc, budget+1))
	closeErr := out.Close()
	if err != nil {
		return n, fmt.Errorf("write %q: %w", name, err)
	}
	if closeErr != nil {
		return n, fmt.Errorf("write %q: %w", name, closeErr)
	}
	if n > budget {
		return n, fmt.Errorf("%w: more than %d bytes", model.ErrRepositoryTooLarge, budget)
	}
	return n, nil
}
