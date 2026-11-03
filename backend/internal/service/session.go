package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const personalWorkspaceName = "Personal"

type SessionRepository interface {
	FirstUser(ctx context.Context, q database.DBTX) (model.User, error)
	InserUser(ctx context.Context, q database.DBTX, u model.User, now string) error
	InsertWorkspace(ctx context.Context, q database.DBTX, id, kind, name, now string) error
	InsertMembership(ctx context.Context, q database.DBTX, id, workspaceID, userID, role, now string) error
	PersonalSession(ctx context.Context, q database.DBTX, u model.User) (model.Session, error)
}

type SesionService struct {
	db   *sql.DB
	repo SessionRepository
}

func NewSessionService(db *sql.DB, repo SessionRepository) *SesionService {
	return  &SesionService{db: db, repo: repo}
}

// Bootstrap returns the local user's personal workspace session,
// creating the user, workspace and owner membership on first run
func (s *SesionService) Bootstrap(ctx context.Context, displayName string) (model.Session, error) {
	var session model.Session
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowUTC()

		user, err := s.repo.FirstUser(ctx, tx)
		if errors.Is(err, sql.ErrNoRows) {
			user = model.User{ID: model.NewID(), DisplayName: displayName}
			err = s.repo.InserUser(ctx, tx, user, now)
		}
		if err != nil {
			return err
		}

		session, err = s.repo.PersonalSession(ctx, tx, user)
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		workspaceID := model.NewID()
		if err := s.repo.InsertWorkspace(ctx, tx, workspaceID, model.KindPersonal, personalWorkspaceName, now); err != nil {
			return err
		}
		if err := s.repo.InsertMembership(ctx, tx, model.NewID(), workspaceID, user.ID, model.RoleOwner, now); err != nil {
			return err
		}
		session, err = s.repo.PersonalSession(ctx, tx, user)
		return err
	})
	if err != nil {
		return model.Session{}, err
	}
	return session, nil
}
