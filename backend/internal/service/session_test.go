package service

import (
	"context"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestSessionBootstrapIsIdempotent(t *testing.T) {
	db, first := newTestDB(t)

	second, err := NewSessionService(db, repository.NewSessionRepository()).Bootstrap(context.Background(), "other")
	if err != nil {
		t.Fatalf("second bootstrap() error = %v", err)
	}
	if first != second {
		t.Errorf("session changed: first = %+v, second = %+v", first, second)
	}
}