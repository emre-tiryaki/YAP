// Package llm talks to language-model providers. Every OpenAI-compatible
// endpoint (Ollama, LM Studio, llama.cpp, OpenAI, OpenRouter, Gemini, Groq,
// Mistral) goes through the OpenAI SDK; Anthropic has its own adapter.
package llm

import (
	"context"
	"fmt"
	"net/http"
	"sort"
)

type Provider string

const (
	ProviderOpenAICompatible Provider = "openai_compatible"
	ProviderAnthropic        Provider = "anthropic"
)

// Endpoint says where and how to reach a provider. An empty APIKey is valid
// for local servers.
type Endpoint struct {
	Provider Provider
	BaseURL  string
	APIKey   string
}

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type Request struct {
	Endpoint Endpoint
	Model    string
	System   string
	Messages []Message
}

// Router sends requests to the adapter of an endpoint's provider. It builds a
// client per call, so it holds no per-connection state.
type Router struct {
	HTTPClient *http.Client
}

func NewRouter() *Router {
	return &Router{HTTPClient: &http.Client{}}
}

// Stream sends req and calls onDelta with every piece of answer text. It
// returns nil when the provider finished the answer.
func (r *Router) Stream(ctx context.Context, req Request, onDelta func(string)) error {
	switch req.Endpoint.Provider {
	case ProviderOpenAICompatible:
		return mapError(r.streamOpenAI(ctx, req, onDelta))
	case ProviderAnthropic:
		return mapError(r.streamAnthropic(ctx, req, onDelta))
	default:
		return fmt.Errorf("unknown provider %q", req.Endpoint.Provider)
	}
}

// Models lists the model ids the endpoint offers, sorted ascending.
func (r *Router) Models(ctx context.Context, ep Endpoint) ([]string, error) {
	var (
		ids []string
		err error
	)
	switch ep.Provider {
	case ProviderOpenAICompatible:
		ids, err = r.modelsOpenAI(ctx, ep)
	case ProviderAnthropic:
		ids, err = r.modelsAnthropic(ctx, ep)
	default:
		return nil, fmt.Errorf("unknown provider %q", ep.Provider)
	}
	if err != nil {
		return nil, mapError(err)
	}
	sort.Strings(ids)
	return ids, nil
}
