package service

import (
	"errors"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
	// ErrIdentifierTaken is a conflict: another project already uses the
	// identifier.
	ErrIdentifierTaken = fmt.Errorf("%w: identifier", ErrConflict)
	// ErrLabelNameTaken is a conflict: the project already has a label with
	// that name.
	ErrLabelNameTaken = fmt.Errorf("%w: label name", ErrConflict)
	// ErrInvalidPriority is an invalid input: the priority is not a known one.
	ErrInvalidPriority = fmt.Errorf("%w: priority", ErrInvalidInput)
	// ErrInvalidDateRange is an invalid input: the start date is after the
	// due date.
	ErrInvalidDateRange = fmt.Errorf("%w: start date after due date", ErrInvalidInput)
	// ErrRelationExists is a conflict: the two tasks already have a relation.
	ErrRelationExists = fmt.Errorf("%w: relation", ErrConflict)
	ErrForbidden       = errors.New("forbidden")
	// ErrStale means the caller changed an older revision than the stored one.
	ErrStale = errors.New("stale revision")
	// ErrTooLarge means a request body exceeded the allowed size.
	ErrTooLarge = errors.New("too large")
	// ErrFileMissing means a resource's bytes are gone from the disk.
	ErrFileMissing = errors.New("file missing on disk")
	// ErrReadOnly means the change would write into a repository folder,
	// whose content comes from GitHub.
	ErrReadOnly = errors.New("read only")

	// ErrGitHubNotConfigured means no OAuth client id is set.
	ErrGitHubNotConfigured = errors.New("github not configured")
	// ErrGitHubClientInvalid means GitHub does not accept the client id, or
	// the OAuth app has the device flow turned off.
	ErrGitHubClientInvalid = model.ErrGitHubClientInvalid
	ErrDeviceFlowDisabled  = model.ErrDeviceFlowDisabled
	// ErrGitHubNotConnected means the user has no (valid) GitHub token.
	ErrGitHubNotConnected = errors.New("github not connected")
	// ErrGitHubUnavailable means GitHub could not be reached.
	ErrGitHubUnavailable = model.ErrGitHubUnavailable
	// ErrKeychainUnavailable means the OS keychain cannot be used.
	ErrKeychainUnavailable = model.ErrSecretStoreUnavailable
	// ErrGitMissing means the git program is not installed.
	ErrGitMissing = model.ErrGitMissing
)
