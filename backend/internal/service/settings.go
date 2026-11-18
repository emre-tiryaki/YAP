package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	settingAutosaveEnabled = "resources.autosave_enabled"
	settingAutosaveDelayMs = "resources.autosave_delay_ms"

	settingHideDone     = "tasks.hide_done"
	settingDoneToBottom = "tasks.done_to_bottom"

	settingStatusNamePrefix = "statuses.default_name."

	settingNotifyDueSoon  = "notifications.due_soon"
	settingNotifyReminder = "notifications.reminder"
	settingNotifyOverdue  = "notifications.overdue"
	settingNotifyComments = "notifications.comments"
	settingNotifySound    = "notifications.sound"

	settingRepoAutoUpdate = "repositories.auto_update"

	minAutosaveDelayMs = 300
	maxAutosaveDelayMs = 10_000

	maxStatusNameLength = 100
)

// Reminder is how long before a task's due date the due-soon notification
// is shown.
type Reminder string

const (
	ReminderHour Reminder = "1h"
	ReminderDay  Reminder = "1d"
	ReminderWeek Reminder = "1w"
)

func validReminder(r Reminder) bool {
	return r == ReminderHour || r == ReminderDay || r == ReminderWeek
}

var statusCategories = []model.StatusCategory{model.StatusToDo, model.StatusInProgress, model.StatusDone}

// StatusNames are the titles given to the default statuses of a new plan, by
// category.
type StatusNames struct {
	ToDo       string
	InProgress string
	Done       string
}

func (n StatusNames) get(c model.StatusCategory) string {
	switch c {
	case model.StatusToDo:
		return n.ToDo
	case model.StatusInProgress:
		return n.InProgress
	default:
		return n.Done
	}
}

func (n *StatusNames) set(c model.StatusCategory, name string) {
	switch c {
	case model.StatusToDo:
		n.ToDo = name
	case model.StatusInProgress:
		n.InProgress = name
	default:
		n.Done = name
	}
}

// NotificationSettings choose which reminders the app shows.
type NotificationSettings struct {
	DueSoon  bool
	Reminder Reminder
	Overdue  bool
	Comments bool
	Sound    bool
}

// UserSettings are preferences that follow the user to every device.
type UserSettings struct {
	AutosaveEnabled bool
	AutosaveDelayMs int
	// HideDone hides completed tasks on plan boards and lists.
	HideDone bool
	// DoneToBottom shows completed tasks after the open ones of a status.
	DoneToBottom  bool
	StatusNames   StatusNames
	Notifications NotificationSettings
	// RepoAutoUpdate keeps GitHub repositories up to date in the background
	// while the app runs and GitHub can be reached.
	RepoAutoUpdate bool
}

func defaultUserSettings() UserSettings {
	return UserSettings{
		AutosaveEnabled: true,
		AutosaveDelayMs: 1000,
		StatusNames:     StatusNames{ToDo: "To Do", InProgress: "In Progress", Done: "Done"},
		Notifications: NotificationSettings{
			DueSoon:  true,
			Reminder: ReminderDay,
			Overdue:  true,
		},
		RepoAutoUpdate: true,
	}
}

// UpdateSettingsInput changes only the fields that are set.
type UpdateSettingsInput struct {
	AutosaveEnabled *bool
	AutosaveDelayMs *int
	HideDone        *bool
	DoneToBottom    *bool
	// StatusNames holds the names to change by category.
	StatusNames map[model.StatusCategory]string

	NotifyDueSoon  *bool
	NotifyReminder *Reminder
	NotifyOverdue  *bool
	NotifyComments *bool
	NotifySound    *bool

	RepoAutoUpdate *bool
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
	return loadUserSettings(ctx, s.db, s.repo, session.UserID)
}

func loadUserSettings(ctx context.Context, q database.DBTX, repo SettingsRepository, userID string) (UserSettings, error) {
	values, err := repo.All(ctx, q, userID)
	if err != nil {
		return UserSettings{}, err
	}
	return parseUserSettings(values), nil
}

// readSetting decodes a stored JSON value into dst when it is present, valid
// and accepted by ok; otherwise dst keeps its default.
func readSetting[T any](values map[string]string, key string, dst *T, ok func(T) bool) {
	raw, found := values[key]
	if !found {
		return
	}
	var v T
	if json.Unmarshal([]byte(raw), &v) != nil {
		return
	}
	if ok != nil && !ok(v) {
		return
	}
	*dst = v
}

func parseUserSettings(values map[string]string) UserSettings {
	out := defaultUserSettings()
	readSetting(values, settingAutosaveEnabled, &out.AutosaveEnabled, nil)
	readSetting(values, settingAutosaveDelayMs, &out.AutosaveDelayMs, validDelay)
	readSetting(values, settingHideDone, &out.HideDone, nil)
	readSetting(values, settingDoneToBottom, &out.DoneToBottom, nil)
	for _, c := range statusCategories {
		name := out.StatusNames.get(c)
		readSetting(values, settingStatusNamePrefix+string(c), &name, func(v string) bool {
			_, err := normalizeStatusName(v)
			return err == nil
		})
		out.StatusNames.set(c, strings.TrimSpace(name))
	}
	n := &out.Notifications
	readSetting(values, settingNotifyDueSoon, &n.DueSoon, nil)
	readSetting(values, settingNotifyReminder, &n.Reminder, validReminder)
	readSetting(values, settingNotifyOverdue, &n.Overdue, nil)
	readSetting(values, settingNotifyComments, &n.Comments, nil)
	readSetting(values, settingNotifySound, &n.Sound, nil)
	readSetting(values, settingRepoAutoUpdate, &out.RepoAutoUpdate, nil)
	return out
}

func validDelay(ms int) bool {
	return ms >= minAutosaveDelayMs && ms <= maxAutosaveDelayMs
}

func normalizeStatusName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: status name is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(name) > maxStatusNameLength {
		return "", fmt.Errorf("%w: status name is too long", ErrInvalidInput)
	}
	return name, nil
}

// settingValues encodes the set fields of in as stored JSON values by key,
// rejecting invalid ones.
func settingValues(in UpdateSettingsInput) (map[string]any, error) {
	values := map[string]any{}
	if in.AutosaveEnabled != nil {
		values[settingAutosaveEnabled] = *in.AutosaveEnabled
	}
	if in.AutosaveDelayMs != nil {
		if !validDelay(*in.AutosaveDelayMs) {
			return nil, fmt.Errorf("%w: autosave delay", ErrInvalidInput)
		}
		values[settingAutosaveDelayMs] = *in.AutosaveDelayMs
	}
	if in.HideDone != nil {
		values[settingHideDone] = *in.HideDone
	}
	if in.DoneToBottom != nil {
		values[settingDoneToBottom] = *in.DoneToBottom
	}
	for c, name := range in.StatusNames {
		if !validCategory(c) {
			return nil, fmt.Errorf("%w: status category", ErrInvalidInput)
		}
		normalized, err := normalizeStatusName(name)
		if err != nil {
			return nil, err
		}
		values[settingStatusNamePrefix+string(c)] = normalized
	}
	if in.NotifyDueSoon != nil {
		values[settingNotifyDueSoon] = *in.NotifyDueSoon
	}
	if in.NotifyReminder != nil {
		if !validReminder(*in.NotifyReminder) {
			return nil, fmt.Errorf("%w: reminder", ErrInvalidInput)
		}
		values[settingNotifyReminder] = *in.NotifyReminder
	}
	if in.NotifyOverdue != nil {
		values[settingNotifyOverdue] = *in.NotifyOverdue
	}
	if in.NotifyComments != nil {
		values[settingNotifyComments] = *in.NotifyComments
	}
	if in.NotifySound != nil {
		values[settingNotifySound] = *in.NotifySound
	}
	if in.RepoAutoUpdate != nil {
		values[settingRepoAutoUpdate] = *in.RepoAutoUpdate
	}
	return values, nil
}

func putSettings(ctx context.Context, q database.DBTX, repo SettingsRepository, userID string, values map[string]any, now string) error {
	for key, v := range values {
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("encode setting %s: %w", key, err)
		}
		if err := repo.Put(ctx, q, userID, key, string(raw), now); err != nil {
			return err
		}
	}
	return nil
}

// resetUserSettings writes the default of every setting.
func resetUserSettings(ctx context.Context, q database.DBTX, repo SettingsRepository, userID, now string) error {
	d := defaultUserSettings()
	n := d.Notifications
	values, err := settingValues(UpdateSettingsInput{
		AutosaveEnabled: &d.AutosaveEnabled,
		AutosaveDelayMs: &d.AutosaveDelayMs,
		HideDone:        &d.HideDone,
		DoneToBottom:    &d.DoneToBottom,
		StatusNames: map[model.StatusCategory]string{
			model.StatusToDo:       d.StatusNames.ToDo,
			model.StatusInProgress: d.StatusNames.InProgress,
			model.StatusDone:       d.StatusNames.Done,
		},
		NotifyDueSoon:  &n.DueSoon,
		NotifyReminder: &n.Reminder,
		NotifyOverdue:  &n.Overdue,
		NotifyComments: &n.Comments,
		NotifySound:    &n.Sound,
		RepoAutoUpdate: &d.RepoAutoUpdate,
	})
	if err != nil {
		return err
	}
	return putSettings(ctx, q, repo, userID, values, now)
}

func (s *SettingsService) Update(ctx context.Context, session model.Session, in UpdateSettingsInput) (UserSettings, error) {
	values, err := settingValues(in)
	if err != nil {
		return UserSettings{}, err
	}
	var out UserSettings
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := putSettings(ctx, tx, s.repo, session.UserID, values, nowUTC()); err != nil {
			return err
		}
		var err error
		out, err = loadUserSettings(ctx, tx, s.repo, session.UserID)
		return err
	})
	if err != nil {
		return UserSettings{}, err
	}
	return out, nil
}
