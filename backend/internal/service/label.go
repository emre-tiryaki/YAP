package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/position"
)

const maxLabelNameRunes = 50

type LabelRepository interface {
	List(ctx context.Context, q database.DBTX, workspaceID, projectID string) ([]model.Label, error)
	Insert(ctx context.Context, q database.DBTX, l model.Label, now string) error
	Get(ctx context.Context, q database.DBTX, workspaceID, labelID string) (model.Label, error)
	LastPosition(ctx context.Context, q database.DBTX, workspaceID, projectID string) (string, error)
	NameTaken(ctx context.Context, q database.DBTX, workspaceID, projectID, name, excludeLabelID string) (bool, error)
	Update(ctx context.Context, q database.DBTX, workspaceID, labelID, name, color, now string) error
	SoftDelete(ctx context.Context, q database.DBTX, workspaceID, labelID, now string) error
}

type CreateLabelInput struct {
	Name  string
	Color string
}

// UpdateLabelInput carries the fields to change; nil fields are kept.
type UpdateLabelInput struct {
	Name  *string
	Color *string
}

type LabelService struct {
	db       *sql.DB
	projects ProjectRepository
	labels   LabelRepository
}

func NewLabelService(db *sql.DB, projects ProjectRepository, labels LabelRepository) *LabelService {
	return &LabelService{db: db, projects: projects, labels: labels}
}

func validLabelColor(color string) bool {
	return slices.Contains(model.LabelColors, color)
}

func normalizeLabelName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxLabelNameRunes {
		return "", fmt.Errorf("%w: label name must be 1 to %d characters", ErrInvalidInput, maxLabelNameRunes)
	}
	return name, nil
}

// List returns the labels of a project in display order.
func (s *LabelService) List(ctx context.Context, session model.Session, projectID string) ([]model.Label, error) {
	if err := s.requireProject(ctx, s.db, session.WorkspaceID, projectID); err != nil {
		return nil, err
	}
	return s.labels.List(ctx, s.db, session.WorkspaceID, projectID)
}

// Create adds a label at the end of the project's labels. The name is unique
// within the project and the colour is one of model.LabelColors.
func (s *LabelService) Create(ctx context.Context, session model.Session, projectID string, in CreateLabelInput) (model.Label, error) {
	name, err := normalizeLabelName(in.Name)
	if err != nil {
		return model.Label{}, err
	}
	if !validLabelColor(in.Color) {
		return model.Label{}, fmt.Errorf("%w: label color", ErrInvalidInput)
	}

	label := model.Label{ID: model.NewID(), WorkspaceID: session.WorkspaceID, ProjectID: projectID, Name: name, Color: in.Color}
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requireProject(ctx, tx, session.WorkspaceID, projectID); err != nil {
			return err
		}
		if taken, err := s.labels.NameTaken(ctx, tx, session.WorkspaceID, projectID, name, ""); err != nil {
			return err
		} else if taken {
			return ErrLabelNameTaken
		}
		last, err := s.labels.LastPosition(ctx, tx, session.WorkspaceID, projectID)
		if err != nil {
			return err
		}
		label.Position = position.After(last)
		return s.labels.Insert(ctx, tx, label, nowUTC())
	})
	if err != nil {
		return model.Label{}, err
	}
	return label, nil
}

// Update renames or recolours a label.
func (s *LabelService) Update(ctx context.Context, session model.Session, labelID string, in UpdateLabelInput) (model.Label, error) {
	if in.Name == nil && in.Color == nil {
		return model.Label{}, fmt.Errorf("%w: nothing to update", ErrInvalidInput)
	}
	var name string
	if in.Name != nil {
		var err error
		if name, err = normalizeLabelName(*in.Name); err != nil {
			return model.Label{}, err
		}
	}
	if in.Color != nil && !validLabelColor(*in.Color) {
		return model.Label{}, fmt.Errorf("%w: label color", ErrInvalidInput)
	}

	var label model.Label
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		if label, err = s.getLabel(ctx, tx, session.WorkspaceID, labelID); err != nil {
			return err
		}
		if in.Name != nil && name != label.Name {
			if taken, err := s.labels.NameTaken(ctx, tx, session.WorkspaceID, label.ProjectID, name, labelID); err != nil {
				return err
			} else if taken {
				return ErrLabelNameTaken
			}
			label.Name = name
		}
		if in.Color != nil {
			label.Color = *in.Color
		}
		return s.labels.Update(ctx, tx, session.WorkspaceID, labelID, label.Name, label.Color, nowUTC())
	})
	if err != nil {
		return model.Label{}, err
	}
	return label, nil
}

// Delete soft-deletes a label and removes it from its tasks.
func (s *LabelService) Delete(ctx context.Context, session model.Session, labelID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.getLabel(ctx, tx, session.WorkspaceID, labelID); err != nil {
			return err
		}
		return s.labels.SoftDelete(ctx, tx, session.WorkspaceID, labelID, nowUTC())
	})
}

func (s *LabelService) getLabel(ctx context.Context, q database.DBTX, workspaceID, labelID string) (model.Label, error) {
	label, err := s.labels.Get(ctx, q, workspaceID, labelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Label{}, fmt.Errorf("%w: label", ErrNotFound)
		}
		return model.Label{}, err
	}
	return label, nil
}

func (s *LabelService) requireProject(ctx context.Context, q database.DBTX, workspaceID, projectID string) error {
	if _, err := s.projects.Name(ctx, q, workspaceID, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: project", ErrNotFound)
		}
		return err
	}
	return nil
}
