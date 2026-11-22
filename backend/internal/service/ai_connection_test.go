package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func newAIConnService(t *testing.T) (*AIConnectionService, *memAIKeys, *fakeLLM, model.Session) {
	t.Helper()
	db, session := newTestDB(t)
	keys, fake := newMemAIKeys(), &fakeLLM{models: []string{"a", "b"}}
	return NewAIConnectionService(db, repository.NewAIRepository(), keys, fake), keys, fake, session
}

func TestAIConnectionCreateValidation(t *testing.T) {
	svc, keys, _, session := newAIConnService(t)
	tests := []struct {
		name string
		in   CreateAIConnectionInput
	}{
		{"blank name", CreateAIConnectionInput{Name: " ", Preset: model.AIPresetOllama, BaseURL: "http://localhost:11434/v1"}},
		{"long name", CreateAIConnectionInput{Name: strings.Repeat("x", 101), Preset: model.AIPresetOllama, BaseURL: "http://localhost/v1"}},
		{"unknown preset", CreateAIConnectionInput{Name: "x", Preset: "nope", BaseURL: "http://localhost/v1"}},
		{"bad scheme", CreateAIConnectionInput{Name: "x", Preset: model.AIPresetCustom, BaseURL: "ftp://localhost"}},
		{"no host", CreateAIConnectionInput{Name: "x", Preset: model.AIPresetCustom, BaseURL: "http://"}},
		{"empty url", CreateAIConnectionInput{Name: "x", Preset: model.AIPresetCustom}},
		{"long url", CreateAIConnectionInput{Name: "x", Preset: model.AIPresetCustom, BaseURL: "http://h/" + strings.Repeat("a", 2000)}},
		{"key required", CreateAIConnectionInput{Name: "x", Preset: model.AIPresetOpenAI, BaseURL: "https://api.openai.com/v1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Create(context.Background(), session, tt.in); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("Create() error = %v, want ErrInvalidInput", err)
			}
		})
	}
	if len(keys.keys) != 0 {
		t.Errorf("keys stored = %v, want none", keys.keys)
	}
}

func TestAIConnectionLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, keys, fake, session := newAIConnService(t)

	c, err := svc.Create(ctx, session, CreateAIConnectionInput{Name: " OpenAI ", Preset: model.AIPresetOpenAI, BaseURL: " https://api.openai.com/v1 ", APIKey: "sk-1"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "OpenAI" || c.Provider != model.AIProviderOpenAICompatible || !c.HasKey || c.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Create() = %+v", c)
	}
	if keys.keys[c.ID] != "sk-1" {
		t.Errorf("stored key = %q", keys.keys[c.ID])
	}

	models, err := svc.Models(ctx, session, c.ID)
	if err != nil || len(models) != 2 || fake.endpoint[0].APIKey != "sk-1" {
		t.Errorf("Models() = %v (err %v), endpoint %+v", models, err, fake.endpoint)
	}

	// Required key cannot be removed; nil keeps it; a new value replaces it.
	empty := ""
	if _, err := svc.Update(ctx, session, c.ID, UpdateAIConnectionInput{APIKey: &empty}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("remove required key error = %v", err)
	}
	name, newKey := "Work", "sk-2"
	c, err = svc.Update(ctx, session, c.ID, UpdateAIConnectionInput{Name: &name, APIKey: &newKey})
	if err != nil || c.Name != "Work" || keys.keys[c.ID] != "sk-2" {
		t.Errorf("Update() = %+v (err %v), key %q", c, err, keys.keys[c.ID])
	}

	if err := svc.Delete(ctx, session, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := keys.keys[c.ID]; ok {
		t.Error("key still stored after delete")
	}
	if list, err := svc.List(ctx, session); err != nil || len(list) != 0 {
		t.Errorf("List() = %v (err %v)", list, err)
	}
	if _, err := svc.Models(ctx, session, c.ID); !errors.Is(err, ErrAIConnectionMissing) {
		t.Errorf("Models() after delete error = %v", err)
	}
	if err := svc.Delete(ctx, session, c.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete() error = %v", err)
	}
}

func TestAIConnectionOptionalKeyCanBeRemoved(t *testing.T) {
	ctx := context.Background()
	svc, keys, _, session := newAIConnService(t)
	c, err := svc.Create(ctx, session, CreateAIConnectionInput{Name: "Local", Preset: model.AIPresetOllama, BaseURL: "http://localhost:11434/v1", APIKey: "k"})
	if err != nil || !c.HasKey {
		t.Fatalf("Create() = %+v (err %v)", c, err)
	}
	empty := ""
	c, err = svc.Update(ctx, session, c.ID, UpdateAIConnectionInput{APIKey: &empty})
	if err != nil || c.HasKey {
		t.Errorf("Update() = %+v (err %v), want no key", c, err)
	}
	if _, ok := keys.keys[c.ID]; ok {
		t.Error("key still stored")
	}
}

func TestAIConnectionKeychainFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	svc, keys, _, session := newAIConnService(t)
	keys.err = ErrKeychainUnavailable
	_, err := svc.Create(ctx, session, CreateAIConnectionInput{Name: "x", Preset: model.AIPresetOpenAI, BaseURL: "https://api.openai.com/v1", APIKey: "k"})
	if !errors.Is(err, ErrKeychainUnavailable) {
		t.Fatalf("Create() error = %v", err)
	}
	if list, _ := svc.List(ctx, session); len(list) != 0 {
		t.Errorf("List() = %v, want rolled back", list)
	}
}

func TestAIConnectionEndpointKeyMissing(t *testing.T) {
	ctx := context.Background()
	svc, keys, _, session := newAIConnService(t)
	c, err := svc.Create(ctx, session, CreateAIConnectionInput{Name: "x", Preset: model.AIPresetOpenAI, BaseURL: "https://api.openai.com/v1", APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	delete(keys.keys, c.ID)
	if _, _, err := svc.endpoint(ctx, session, c.ID); !errors.Is(err, ErrAIKeyMissing) {
		t.Errorf("endpoint() error = %v, want ErrAIKeyMissing", err)
	}
	if _, _, err := svc.endpoint(ctx, session, "nope"); !errors.Is(err, ErrAIConnectionMissing) {
		t.Errorf("endpoint(unknown) error = %v", err)
	}
	keys.err = ErrKeychainUnavailable
	if _, _, err := svc.endpoint(ctx, session, c.ID); !errors.Is(err, ErrKeychainUnavailable) {
		t.Errorf("endpoint() with broken keychain error = %v", err)
	}
}

func TestAIConnectionTestUnsaved(t *testing.T) {
	ctx := context.Background()
	svc, _, fake, _ := newAIConnService(t)
	models, err := svc.TestUnsaved(ctx, TestAIConnectionInput{Preset: model.AIPresetAnthropic, BaseURL: "https://api.anthropic.com", APIKey: "k"})
	if err != nil || len(models) != 2 || fake.endpoint[0].Provider != llm.ProviderAnthropic {
		t.Errorf("TestUnsaved() = %v (err %v), endpoint %+v", models, err, fake.endpoint)
	}
	if _, err := svc.TestUnsaved(ctx, TestAIConnectionInput{Preset: model.AIPresetAnthropic, BaseURL: "https://api.anthropic.com"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("TestUnsaved() without key error = %v", err)
	}
	fake.err = ErrAIUnreachable
	if _, err := svc.TestUnsaved(ctx, TestAIConnectionInput{Preset: model.AIPresetOllama, BaseURL: "http://localhost:1/v1"}); !errors.Is(err, ErrAIUnreachable) {
		t.Errorf("TestUnsaved() error = %v, want ErrAIUnreachable", err)
	}
}
