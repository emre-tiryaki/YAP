package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	maxAIConnectionNameLength = 100
	maxAIBaseURLLength        = 2000
	aiModelsTimeout           = 15 * time.Second
)

// AIKeyStore keeps the API keys of connections outside the database.
type AIKeyStore interface {
	Get(connectionID string) (string, error)
	Set(connectionID, key string) error
	Delete(connectionID string) error
}

// LLM is the part of the provider router the AI services use.
type LLM interface {
	Stream(ctx context.Context, req llm.Request, onDelta func(string)) error
	Models(ctx context.Context, ep llm.Endpoint) ([]string, error)
}

type AIConnectionRepository interface {
	ListConnections(ctx context.Context, q database.DBTX, workspaceID, userID string) ([]model.AIConnection, error)
	GetConnection(ctx context.Context, q database.DBTX, workspaceID, userID, id string) (model.AIConnection, error)
	InsertConnection(ctx context.Context, q database.DBTX, c model.AIConnection, now string) error
	UpdateConnection(ctx context.Context, q database.DBTX, workspaceID, id, name, baseURL string, hasKey bool, now string) error
	SoftDeleteConnection(ctx context.Context, q database.DBTX, workspaceID, id, now string) error
}

type aiPreset struct {
	provider    model.AIProvider
	baseURL     string
	keyRequired bool
}

// aiPresets is the single source of presets; the frontend mirrors it.
var aiPresets = map[model.AIPreset]aiPreset{
	model.AIPresetOllama:     {model.AIProviderOpenAICompatible, "http://localhost:11434/v1", false},
	model.AIPresetLMStudio:   {model.AIProviderOpenAICompatible, "http://localhost:1234/v1", false},
	model.AIPresetLlamaCpp:   {model.AIProviderOpenAICompatible, "http://localhost:8081/v1", false},
	model.AIPresetCustom:     {model.AIProviderOpenAICompatible, "", false},
	model.AIPresetOpenAI:     {model.AIProviderOpenAICompatible, "https://api.openai.com/v1", true},
	model.AIPresetOpenRouter: {model.AIProviderOpenAICompatible, "https://openrouter.ai/api/v1", true},
	model.AIPresetGemini:     {model.AIProviderOpenAICompatible, "https://generativelanguage.googleapis.com/v1beta/openai/", true},
	model.AIPresetGroq:       {model.AIProviderOpenAICompatible, "https://api.groq.com/openai/v1", true},
	model.AIPresetMistral:    {model.AIProviderOpenAICompatible, "https://api.mistral.ai/v1", true},
	model.AIPresetAnthropic:  {model.AIProviderAnthropic, "https://api.anthropic.com", true},
}

type CreateAIConnectionInput struct {
	Name    string
	Preset  model.AIPreset
	BaseURL string
	APIKey  string
}

// UpdateAIConnectionInput changes only the fields that are set. An APIKey of
// "" removes the key; nil keeps it.
type UpdateAIConnectionInput struct {
	Name    *string
	BaseURL *string
	APIKey  *string
}

type TestAIConnectionInput struct {
	Preset  model.AIPreset
	BaseURL string
	APIKey  string
}

type AIConnectionService struct {
	db   *sql.DB
	repo AIConnectionRepository
	keys AIKeyStore
	llm  LLM
}

func NewAIConnectionService(db *sql.DB, repo AIConnectionRepository, keys AIKeyStore, l LLM) *AIConnectionService {
	return &AIConnectionService{db: db, repo: repo, keys: keys, llm: l}
}

func normalizeAIBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || utf8.RuneCountInString(raw) > maxAIBaseURLLength {
		return "", fmt.Errorf("%w: base url", ErrInvalidInput)
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", fmt.Errorf("%w: base url", ErrInvalidInput)
	}
	return raw, nil
}

func normalizeAIConnectionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxAIConnectionNameLength {
		return "", fmt.Errorf("%w: connection name", ErrInvalidInput)
	}
	return name, nil
}

func (s *AIConnectionService) List(ctx context.Context, session model.Session) ([]model.AIConnection, error) {
	return s.repo.ListConnections(ctx, s.db, session.WorkspaceID, session.UserID)
}

func (s *AIConnectionService) get(ctx context.Context, session model.Session, id string) (model.AIConnection, error) {
	c, err := s.repo.GetConnection(ctx, s.db, session.WorkspaceID, session.UserID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AIConnection{}, ErrNotFound
	}
	return c, err
}

func (s *AIConnectionService) Create(ctx context.Context, session model.Session, in CreateAIConnectionInput) (model.AIConnection, error) {
	name, err := normalizeAIConnectionName(in.Name)
	if err != nil {
		return model.AIConnection{}, err
	}
	preset, ok := aiPresets[in.Preset]
	if !ok {
		return model.AIConnection{}, fmt.Errorf("%w: preset", ErrInvalidInput)
	}
	baseURL, err := normalizeAIBaseURL(in.BaseURL)
	if err != nil {
		return model.AIConnection{}, err
	}
	if preset.keyRequired && in.APIKey == "" {
		return model.AIConnection{}, fmt.Errorf("%w: api key is required", ErrInvalidInput)
	}
	c := model.AIConnection{
		ID: model.NewID(), WorkspaceID: session.WorkspaceID, UserID: session.UserID,
		Name: name, Provider: preset.provider, Preset: in.Preset, BaseURL: baseURL, HasKey: in.APIKey != "",
	}
	now := nowUTC()
	if err := s.repo.InsertConnection(ctx, s.db, c, now); err != nil {
		return model.AIConnection{}, err
	}
	if in.APIKey != "" {
		if err := s.keys.Set(c.ID, in.APIKey); err != nil {
			if derr := s.repo.SoftDeleteConnection(ctx, s.db, c.WorkspaceID, c.ID, nowUTC()); derr != nil {
				slog.Error("roll back ai connection", "error", derr)
			}
			return model.AIConnection{}, err
		}
	}
	return s.get(ctx, session, c.ID)
}

func (s *AIConnectionService) Update(ctx context.Context, session model.Session, id string, in UpdateAIConnectionInput) (model.AIConnection, error) {
	c, err := s.get(ctx, session, id)
	if err != nil {
		return model.AIConnection{}, err
	}
	name, baseURL := c.Name, c.BaseURL
	if in.Name != nil {
		if name, err = normalizeAIConnectionName(*in.Name); err != nil {
			return model.AIConnection{}, err
		}
	}
	if in.BaseURL != nil {
		if baseURL, err = normalizeAIBaseURL(*in.BaseURL); err != nil {
			return model.AIConnection{}, err
		}
	}
	hasKey := c.HasKey
	if in.APIKey != nil {
		hasKey = *in.APIKey != ""
		if !hasKey && aiPresets[c.Preset].keyRequired {
			return model.AIConnection{}, fmt.Errorf("%w: api key is required", ErrInvalidInput)
		}
		if hasKey {
			if err := s.keys.Set(c.ID, *in.APIKey); err != nil {
				return model.AIConnection{}, err
			}
		}
	}
	if err := s.repo.UpdateConnection(ctx, s.db, c.WorkspaceID, c.ID, name, baseURL, hasKey, nowUTC()); err != nil {
		return model.AIConnection{}, err
	}
	if in.APIKey != nil && !hasKey {
		if err := s.keys.Delete(c.ID); err != nil {
			slog.Error("delete ai key", "error", err)
		}
	}
	return s.get(ctx, session, c.ID)
}

func (s *AIConnectionService) Delete(ctx context.Context, session model.Session, id string) error {
	c, err := s.get(ctx, session, id)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDeleteConnection(ctx, s.db, c.WorkspaceID, c.ID, nowUTC()); err != nil {
		return err
	}
	if err := s.keys.Delete(c.ID); err != nil {
		slog.Error("delete ai key", "error", err)
	}
	return nil
}

// Models lists the models of a saved connection.
func (s *AIConnectionService) Models(ctx context.Context, session model.Session, id string) ([]string, error) {
	_, ep, err := s.endpoint(ctx, session, id)
	if err != nil {
		return nil, err
	}
	return s.listModels(ctx, ep)
}

// TestUnsaved lists the models of connection settings that are not saved.
func (s *AIConnectionService) TestUnsaved(ctx context.Context, in TestAIConnectionInput) ([]string, error) {
	preset, ok := aiPresets[in.Preset]
	if !ok {
		return nil, fmt.Errorf("%w: preset", ErrInvalidInput)
	}
	baseURL, err := normalizeAIBaseURL(in.BaseURL)
	if err != nil {
		return nil, err
	}
	if preset.keyRequired && in.APIKey == "" {
		return nil, fmt.Errorf("%w: api key is required", ErrInvalidInput)
	}
	return s.listModels(ctx, llm.Endpoint{Provider: llm.Provider(preset.provider), BaseURL: baseURL, APIKey: in.APIKey})
}

func (s *AIConnectionService) listModels(ctx context.Context, ep llm.Endpoint) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, aiModelsTimeout)
	defer cancel()
	models, err := s.llm.Models(ctx, ep)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: timed out", ErrAIUnreachable)
		}
		return nil, err
	}
	if models == nil {
		models = []string{}
	}
	return models, nil
}

// endpoint resolves a connection and its key. An unknown or removed
// connection is ErrAIConnectionMissing; a required key that is not stored is
// ErrAIKeyMissing.
func (s *AIConnectionService) endpoint(ctx context.Context, session model.Session, id string) (model.AIConnection, llm.Endpoint, error) {
	if id == "" {
		return model.AIConnection{}, llm.Endpoint{}, ErrAIConnectionMissing
	}
	c, err := s.repo.GetConnection(ctx, s.db, session.WorkspaceID, session.UserID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AIConnection{}, llm.Endpoint{}, ErrAIConnectionMissing
	}
	if err != nil {
		return model.AIConnection{}, llm.Endpoint{}, err
	}
	ep := llm.Endpoint{Provider: llm.Provider(c.Provider), BaseURL: c.BaseURL}
	if c.HasKey {
		key, err := s.keys.Get(c.ID)
		switch {
		case errors.Is(err, model.ErrNoCredential):
		case err != nil:
			return model.AIConnection{}, llm.Endpoint{}, err
		default:
			ep.APIKey = key
		}
	}
	if ep.APIKey == "" && aiPresets[c.Preset].keyRequired {
		return model.AIConnection{}, llm.Endpoint{}, ErrAIKeyMissing
	}
	return c, ep, nil
}
