package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	settingAutosaveEnabled = "resources.autosave_enabled"
	settingAutosaveDelayMs = "resources.autosave_delay_ms"

	minAutosaveDelayMs = 300
	maxAutosaveDelayMs = 10_000
)

// UserSettings are preferences that follow the user to every device.
type UserSettings struct {
	AutosaveEnabled bool
	AutosaveDelayMs int
}

func defaultUserSettings() UserSettings {
	return UserSettings{AutosaveEnabled: true, AutosaveDelayMs: 1000}
}

// UpdateSettingsInput changes only the fields that are set.
type UpdateSettingsInput struct {
	AutosaveEnabled *bool
	AutosaveDelayMs *int
}

type SettingsRepository interface {
	All(ctx context.Context, q database.DBTX, userID string) (map[string]string, error)
	Put(ctx context.Context, q database.DBTX, userID, key, value, now string) error
}

type SettingsService struct {
	db   *sql.DB
	repo SettingsRepository
}

func NewSettingsService(db *sql.DB, repo SettingsRepository) *SettingsService {
	return &SettingsService{db: db, repo: repo}
}

// Get returns the user's settings; a missing or unreadable value falls back
// to its default.
func (s *SettingsService) Get(ctx context.Context, session model.Session) (UserSettings, error) {
	return s.get(ctx, s.db, session.UserID)
}

func (s *SettingsService) get(ctx context.Context, q database.DBTX, userID string) (UserSettings, error) {
	values, err := s.repo.All(ctx, q, userID)
	if err != nil {
		return UserSettings{}, err
	}
	out := defaultUserSettings()
	if v, ok := values[settingAutosaveEnabled]; ok {
		var b bool
		if json.Unmarshal([]byte(v), &b) == nil {
			out.AutosaveEnabled = b
		}
	}
	if v, ok := values[settingAutosaveDelayMs]; ok {
		var n int
		if json.Unmarshal([]byte(v), &n) == nil && validDelay(n) {
			out.AutosaveDelayMs = n
		}
	}
	return out, nil
}

func validDelay(ms int) bool {
	return ms >= minAutosaveDelayMs && ms <= maxAutosaveDelayMs
}

func (s *SettingsService) Update(ctx context.Context, session model.Session, in UpdateSettingsInput) (UserSettings, error) {
	if in.AutosaveDelayMs != nil && !validDelay(*in.AutosaveDelayMs) {
		return UserSettings{}, fmt.Errorf("%w: autosave delay", ErrInvalidInput)
	}
	var out UserSettings
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowUTC()
		if in.AutosaveEnabled != nil {
			if err := s.repo.Put(ctx, tx, session.UserID, settingAutosaveEnabled, strconv.FormatBool(*in.AutosaveEnabled), now); err != nil {
				return err
			}
		}
		if in.AutosaveDelayMs != nil {
			if err := s.repo.Put(ctx, tx, session.UserID, settingAutosaveDelayMs, strconv.Itoa(*in.AutosaveDelayMs), now); err != nil {
				return err
			}
		}
		var err error
		out, err = s.get(ctx, tx, session.UserID)
		return err
	})
	if err != nil {
		return UserSettings{}, err
	}
	return out, nil
}
