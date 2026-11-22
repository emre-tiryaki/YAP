package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/testdb"
)

func newTestDB(t *testing.T) (*sql.DB, model.Session) {
	t.Helper()
	ctx := context.Background()

	db, err := database.New(ctx, testdb.Path(t))
	if err != nil {
		t.Fatalf("database.New() err = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	session, err := NewSessionService(db, repository.NewSessionRepository()).Bootstrap(ctx, "tester")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	return db, session
}
