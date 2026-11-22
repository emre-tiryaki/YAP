package model

import (
	"errors"
	"time"
)

// Errors reported by the llm package. They live here so the service layer can
// recognize them without importing it.
var (
	// ErrAIUnreachable means the model server could not be reached or failed.
	ErrAIUnreachable = errors.New("ai unreachable")
	// ErrAIAuthFailed means the model server rejected the API key.
	ErrAIAuthFailed = errors.New("ai auth failed")
	// ErrAIRateLimited means the model server asked to slow down.
	ErrAIRateLimited = errors.New("ai rate limited")
	// ErrAIModelNotFound means the server does not know the model (or URL).
	ErrAIModelNotFound = errors.New("ai model not found")
	// ErrAIRejected means the server refused the request or sent no answer.
	ErrAIRejected = errors.New("ai rejected")
)

type AIProvider string

const (
	AIProviderOpenAICompatible AIProvider = "openai_compatible"
	AIProviderAnthropic        AIProvider = "anthropic"
)

type AIPreset string

const (
	AIPresetOllama     AIPreset = "ollama"
	AIPresetLMStudio   AIPreset = "lmstudio"
	AIPresetLlamaCpp   AIPreset = "llamacpp"
	AIPresetOpenAI     AIPreset = "openai"
	AIPresetOpenRouter AIPreset = "openrouter"
	AIPresetGemini     AIPreset = "gemini"
	AIPresetGroq       AIPreset = "groq"
	AIPresetMistral    AIPreset = "mistral"
	AIPresetAnthropic  AIPreset = "anthropic"
	AIPresetCustom     AIPreset = "custom"
)

type AIMessageRole string

const (
	AIRoleUser      AIMessageRole = "user"
	AIRoleAssistant AIMessageRole = "assistant"
)

type AIMessageStatus string

const (
	AIStatusComplete    AIMessageStatus = "complete"
	AIStatusStreaming   AIMessageStatus = "streaming"
	AIStatusStopped     AIMessageStatus = "stopped"
	AIStatusInterrupted AIMessageStatus = "interrupted"
	AIStatusError       AIMessageStatus = "error"
)

type AIAttachmentKind string

const (
	AIAttachTask     AIAttachmentKind = "task"
	AIAttachPlan     AIAttachmentKind = "plan"
	AIAttachProject  AIAttachmentKind = "project"
	AIAttachResource AIAttachmentKind = "resource"
	AIAttachRepoFile AIAttachmentKind = "repo_file"
)

// AIAttachmentStatus says how an attachment went into the latest request.
type AIAttachmentStatus string

const (
	AIAttachIncluded   AIAttachmentStatus = "included"
	AIAttachTruncated  AIAttachmentStatus = "truncated"
	AIAttachOmitted    AIAttachmentStatus = "omitted"
	AIAttachMissing    AIAttachmentStatus = "missing"
	AIAttachUnreadable AIAttachmentStatus = "unreadable"
)

// AIConnection is a model endpoint of a user. The API key is not part of it:
// it lives in the OS keychain.
type AIConnection struct {
	ID          string
	WorkspaceID string
	UserID      string
	Name        string
	Provider    AIProvider
	Preset      AIPreset
	BaseURL     string
	HasKey      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AIChat struct {
	ID           string
	WorkspaceID  string
	UserID       string
	Title        string
	ConnectionID string // empty when none, or the connection was removed
	Model        string
	// Generating is filled by the service while an answer is being written.
	Generating bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AIMessage struct {
	ID           string
	WorkspaceID  string
	ChatID       string
	Role         AIMessageRole
	Content      string
	Status       AIMessageStatus
	ErrorCode    string
	ConnectionID string
	Model        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AIAttachment struct {
	ID          string
	WorkspaceID string
	MessageID   string
	Position    int
	Kind        AIAttachmentKind
	TargetID    string
	Path        string
	Label       string
	LastStatus  AIAttachmentStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AITaskPlacement is where a live task sits.
type AITaskPlacement struct {
	ProjectID string
	PlanID    string
}

// Rows of the AI tables as read from an export. Columns the import does not
// use (workspace_id, user_id, deleted_at) are ignored: the importing user and
// workspace own what is added.
type ImportAIConnection struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Preset    string `json:"preset"`
	BaseURL   string `json:"base_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ImportAIChat struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	ConnectionID *string `json:"connection_id"`
	Model        string  `json:"model"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type ImportAIMessage struct {
	ID           string  `json:"id"`
	ChatID       string  `json:"chat_id"`
	Role         string  `json:"role"`
	Content      string  `json:"content"`
	Status       string  `json:"status"`
	ErrorCode    *string `json:"error_code"`
	ConnectionID *string `json:"connection_id"`
	Model        *string `json:"model"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type ImportAIAttachment struct {
	ID         string  `json:"id"`
	MessageID  string  `json:"message_id"`
	Position   int     `json:"position"`
	Kind       string  `json:"kind"`
	TargetID   string  `json:"target_id"`
	Path       *string `json:"path"`
	Label      string  `json:"label"`
	LastStatus string  `json:"last_status"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}
