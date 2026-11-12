package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const maxProfileNameRunes = 100

type WorkspaceRepository interface {
	Profile(ctx context.Context, q database.DBTX, userID, workspaceID string) (model.Profile, error)
	UpdateDisplayName(ctx context.Context, q database.DBTX, userID, name, now string) error
	UpdateName(ctx context.Context, q database.DBTX, workspaceID, name, now string) error
}

type WorkspaceService struct {
	db   *sql.DB
	repo WorkspaceRepository
}

func NewWorkspaceService(db *sql.DB, repo WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{db: db, repo: repo}
}

// UpdateProfileInput carries the fields to change; nil fields are kept.
type UpdateProfileInput struct {
	DisplayName   *string
	WorkspaceName *string
}

func (s *WorkspaceService) Get(ctx context.Context, session model.Session) (model.Profile, error) {
	p, err := s.repo.Profile(ctx, s.db, session.UserID, session.WorkspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Profile{}, fmt.Errorf("%w: workspace", ErrNotFound)
	}
	return p, err
}

// Update renames the session's user and/or workspace.
func (s *WorkspaceService) Update(ctx context.Context, session model.Session, in UpdateProfileInput) (model.Profile, error) {
	if in.DisplayName == nil && in.WorkspaceName == nil {
		return model.Profile{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	displayName, err := normalizeProfileName(in.DisplayName, "user name")
	if err != nil {
		return model.Profile{}, err
	}
	workspaceName, err := normalizeProfileName(in.WorkspaceName, "workspace name")
	if err != nil {
		return model.Profile{}, err
	}

	var profile model.Profile
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowUTC()
		if in.DisplayName != nil {
			if err := s.repo.UpdateDisplayName(ctx, tx, session.UserID, displayName, now); err != nil {
				return err
			}
		}
		if in.WorkspaceName != nil {
			if err := s.repo.UpdateName(ctx, tx, session.WorkspaceID, workspaceName, now); err != nil {
				return err
			}
		}
		var err error
		profile, err = s.repo.Profile(ctx, tx, session.UserID, session.WorkspaceID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: workspace", ErrNotFound)
		}
		return err
	})
	if err != nil {
		return model.Profile{}, err
	}
	return profile, nil
}

func normalizeProfileName(name *string, field string) (string, error) {
	if name == nil {
		return "", nil
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidInput, field)
	}
	if utf8.RuneCountInString(trimmed) > maxProfileNameRunes {
		return "", fmt.Errorf("%w: %s is too long", ErrInvalidInput, field)
	}
	return trimmed, nil
}
