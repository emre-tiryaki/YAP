package llm

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// noKey is sent when a local server needs no key; the SDK would otherwise
// fall back to the OPENAI_API_KEY environment variable.
const noKey = "yap-no-key"

func (r *Router) openAIClient(ep Endpoint) openai.Client {
	key := ep.APIKey
	if key == "" {
		key = noKey
	}
	return openai.NewClient(
		option.WithBaseURL(ep.BaseURL),
		option.WithAPIKey(key),
		option.WithHTTPClient(r.HTTPClient),
		option.WithMaxRetries(1),
	)
}

func (r *Router) streamOpenAI(ctx context.Context, req Request, onDelta func(string)) error {
	client := r.openAIClient(req.Endpoint)
	msgs := []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(req.System)}
	for _, m := range req.Messages {
		if m.Role == RoleAssistant {
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		} else {
			msgs = append(msgs, openai.UserMessage(m.Content))
		}
	}
	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: msgs,
	})
	defer stream.Close()
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) == 0 {
			continue
		}
		if text := chunk.Choices[0].Delta.Content; text != "" {
			onDelta(text)
		}
	}
	if err := stream.Err(); err != nil {
		return err
	}
	return ctx.Err()
}

func (r *Router) modelsOpenAI(ctx context.Context, ep Endpoint) ([]string, error) {
	client := r.openAIClient(ep)
	pager := client.Models.ListAutoPaging(ctx)
	var ids []string
	for pager.Next() {
		ids = append(ids, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
