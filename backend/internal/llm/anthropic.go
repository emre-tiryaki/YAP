package llm

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicMaxTokens = 8192

func (r *Router) anthropicClient(ep Endpoint) anthropic.Client {
	return anthropic.NewClient(
		option.WithBaseURL(ep.BaseURL),
		option.WithAPIKey(ep.APIKey),
		option.WithHTTPClient(r.HTTPClient),
		option.WithMaxRetries(1),
	)
}

// anthropicMessages merges adjacent same-role messages: the API wants the
// roles to alternate and a dropped answer can leave two user turns in a row.
func anthropicMessages(msgs []Message) []anthropic.MessageParam {
	var out []anthropic.MessageParam
	var lastRole Role
	var texts []string
	flush := func() {
		if len(texts) == 0 {
			return
		}
		block := anthropic.NewTextBlock(strings.Join(texts, "\n\n"))
		if lastRole == RoleAssistant {
			out = append(out, anthropic.NewAssistantMessage(block))
		} else {
			out = append(out, anthropic.NewUserMessage(block))
		}
		texts = nil
	}
	for _, m := range msgs {
		if len(texts) > 0 && m.Role != lastRole {
			flush()
		}
		lastRole = m.Role
		texts = append(texts, m.Content)
	}
	flush()
	return out
}

func (r *Router) streamAnthropic(ctx context.Context, req Request, onDelta func(string)) error {
	client := r.anthropicClient(req.Endpoint)
	stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: anthropicMaxTokens,
		System:    []anthropic.TextBlockParam{{Text: req.System}},
		Messages:  anthropicMessages(req.Messages),
	})
	defer stream.Close()
	for stream.Next() {
		event := stream.Current()
		if event.Type != "content_block_delta" {
			continue
		}
		if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			onDelta(event.Delta.Text)
		}
	}
	if err := stream.Err(); err != nil {
		return err
	}
	return ctx.Err()
}

func (r *Router) modelsAnthropic(ctx context.Context, ep Endpoint) ([]string, error) {
	client := r.anthropicClient(ep)
	pager := client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	var ids []string
	for pager.Next() {
		ids = append(ids, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
