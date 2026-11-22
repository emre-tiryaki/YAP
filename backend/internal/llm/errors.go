package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	openaisdk "github.com/openai/openai-go/v3"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// mapError turns SDK and network errors into the model.ErrAI* sentinels. The
// original text stays in the wrapped message for logs; handlers never send it.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	status := 0
	var oe *openaisdk.Error
	var ae *anthropicsdk.Error
	switch {
	case errors.As(err, &oe):
		status = oe.StatusCode
	case errors.As(err, &ae):
		status = ae.StatusCode
	}
	if status != 0 {
		return fmt.Errorf("%w: %v", sentinelForStatus(status), err)
	}
	// Dial, DNS and broken-stream errors all mean the server cannot be used.
	return fmt.Errorf("%w: %v", model.ErrAIUnreachable, err)
}

func sentinelForStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return model.ErrAIAuthFailed
	case status == http.StatusNotFound:
		return model.ErrAIModelNotFound
	case status == http.StatusTooManyRequests:
		return model.ErrAIRateLimited
	case status >= 500:
		return model.ErrAIUnreachable
	default:
		return model.ErrAIRejected
	}
}
