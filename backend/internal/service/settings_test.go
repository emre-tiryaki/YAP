package service

import (
	"context"
	"errors"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestUserSettings(t *testing.T) {
	db, session := newTestDB(t)
	ctx := context.Background()
	svc := NewSettingsService(db, repository.NewSettingsRepository())

	got, err := svc.Get(ctx, session)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != defaultUserSettings() {
		t.Errorf("Get() = %+v, want defaults", got)
	}

	off, delay := false, 2500
	got, err = svc.Update(ctx, session, UpdateSettingsInput{AutosaveEnabled: &off})
	if err != nil || got.AutosaveEnabled || got.AutosaveDelayMs != 1000 {
		t.Fatalf("Update(enabled) = %+v (err %v)", got, err)
	}
	got, err = svc.Update(ctx, session, UpdateSettingsInput{AutosaveDelayMs: &delay})
	if err != nil || got.AutosaveEnabled || got.AutosaveDelayMs != 2500 {
		t.Fatalf("Update(delay) = %+v (err %v), want the earlier change kept", got, err)
	}

	for _, bad := range []int{0, 299, 10_001} {
		if _, err := svc.Update(ctx, session, UpdateSettingsInput{AutosaveDelayMs: &bad}); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Update(delay %d) error = %v, want ErrInvalidInput", bad, err)
		}
	}
}
