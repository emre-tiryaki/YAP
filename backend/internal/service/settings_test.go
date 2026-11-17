package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
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

func TestUserSettingsTaskViewsAndNotifications(t *testing.T) {
	db, session := newTestDB(t)
	ctx := context.Background()
	svc := NewSettingsService(db, repository.NewSettingsRepository())

	on, off, hour := true, false, ReminderHour
	got, err := svc.Update(ctx, session, UpdateSettingsInput{
		HideDone:       &on,
		DoneToBottom:   &on,
		NotifyDueSoon:  &off,
		NotifyReminder: &hour,
		NotifyOverdue:  &off,
		NotifyComments: &on,
		NotifySound:    &on,
		StatusNames: map[model.StatusCategory]string{
			model.StatusToDo: "  Backlog ",
			model.StatusDone: "Shipped",
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	want := defaultUserSettings()
	want.HideDone, want.DoneToBottom = true, true
	want.Notifications = NotificationSettings{DueSoon: false, Reminder: ReminderHour, Overdue: false, Comments: true, Sound: true}
	want.StatusNames.ToDo, want.StatusNames.Done = "Backlog", "Shipped"
	if got != want {
		t.Errorf("Update() = %+v, want %+v", got, want)
	}
	if again, err := svc.Get(ctx, session); err != nil || again != want {
		t.Errorf("Get() = %+v (err %v), want %+v", again, err, want)
	}
}

func TestUserSettingsRejectsInvalidValues(t *testing.T) {
	db, session := newTestDB(t)
	ctx := context.Background()
	svc := NewSettingsService(db, repository.NewSettingsRepository())

	bad := Reminder("2d")
	tests := []struct {
		name string
		in   UpdateSettingsInput
	}{
		{"unknown reminder", UpdateSettingsInput{NotifyReminder: &bad}},
		{"blank status name", UpdateSettingsInput{StatusNames: map[model.StatusCategory]string{model.StatusToDo: "  "}}},
		{"long status name", UpdateSettingsInput{StatusNames: map[model.StatusCategory]string{model.StatusDone: strings.Repeat("x", 101)}}},
		{"unknown category", UpdateSettingsInput{StatusNames: map[model.StatusCategory]string{"later": "Later"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Update(ctx, session, tt.in); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("Update() error = %v, want ErrInvalidInput", err)
			}
		})
	}
	// A rejected update writes nothing, also not its valid fields.
	on := true
	if _, err := svc.Update(ctx, session, UpdateSettingsInput{HideDone: &on, NotifyReminder: &bad}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Update() error = %v, want ErrInvalidInput", err)
	}
	if got, err := svc.Get(ctx, session); err != nil || got != defaultUserSettings() {
		t.Errorf("Get() = %+v (err %v), want defaults", got, err)
	}
}

func TestUserSettingsIgnoreUnreadableStoredValues(t *testing.T) {
	db, session := newTestDB(t)
	ctx := context.Background()
	repo := repository.NewSettingsRepository()
	now := nowUTC()
	for key, value := range map[string]string{
		settingNotifyReminder:            `"3d"`,
		settingHideDone:                  `"yes"`,
		settingStatusNamePrefix + "todo": `""`,
		settingStatusNamePrefix + "done": `42`,
	} {
		if err := repo.Put(ctx, db, session.UserID, key, value, now); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewSettingsService(db, repo).Get(ctx, session)
	if err != nil || got != defaultUserSettings() {
		t.Errorf("Get() = %+v (err %v), want defaults", got, err)
	}
}

func TestNewPlansUseDefaultStatusNames(t *testing.T) {
	ctx := context.Background()
	env := newTaskEnv(t)
	settings := NewSettingsService(env.db, repository.NewSettingsRepository())
	if _, err := settings.Update(ctx, env.session, UpdateSettingsInput{StatusNames: map[model.StatusCategory]string{
		model.StatusToDo:       "Backlog",
		model.StatusInProgress: "Doing",
	}}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	titles := func(t *testing.T, projectID, planID string) string {
		t.Helper()
		statuses, err := env.projects.Statuses(ctx, env.session.WorkspaceID, projectID, planID)
		if err != nil {
			t.Fatalf("Statuses() error = %v", err)
		}
		var out []string
		for _, s := range statuses {
			out = append(out, s.Title)
		}
		return strings.Join(out, ",")
	}

	project, err := env.projects.Create(ctx, env.session.WorkspaceID, CreateProjectInput{Name: "P", UserID: env.session.UserID})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	mainPlan, err := env.projectRepo.DefaultPlanID(ctx, env.db, env.session.WorkspaceID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(t, project.ID, mainPlan); got != "Backlog,Doing,Done" {
		t.Errorf("first plan statuses = %s, want Backlog,Doing,Done", got)
	}

	plan, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Sprint", UserID: env.session.UserID})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if got := titles(t, project.ID, plan.ID); got != "Backlog,Doing,Done" {
		t.Errorf("new plan statuses = %s, want Backlog,Doing,Done", got)
	}

	// Without a user the built-in names are used.
	other, err := env.projects.CreatePlan(ctx, env.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Other"})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if got := titles(t, project.ID, other.ID); got != "To Do,In Progress,Done" {
		t.Errorf("plan without user statuses = %s, want To Do,In Progress,Done", got)
	}
}
