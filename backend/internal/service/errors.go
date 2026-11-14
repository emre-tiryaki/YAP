package service

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	// ErrStale means the caller changed an older revision than the stored one.
	ErrStale = errors.New("stale revision")
	// ErrTooLarge means a request body exceeded the allowed size.
	ErrTooLarge = errors.New("too large")
	// ErrFileMissing means a resource's bytes are gone from the disk.
	ErrFileMissing = errors.New("file missing on disk")
)
