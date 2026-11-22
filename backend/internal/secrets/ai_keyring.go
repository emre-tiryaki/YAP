package secrets

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const aiService = "yap-ai"

// AIKeyring keeps the API keys of AI connections, keyed by connection id.
type AIKeyring struct{}

func NewAIKeyring() *AIKeyring {
	return &AIKeyring{}
}

func (*AIKeyring) Get(connectionID string) (string, error) {
	key, err := keyring.Get(aiService, connectionID)
	if err != nil {
		return "", wrap(err)
	}
	return key, nil
}

func (*AIKeyring) Set(connectionID, key string) error {
	if err := keyring.Set(aiService, connectionID, key); err != nil {
		return wrap(err)
	}
	return nil
}

func (*AIKeyring) Delete(connectionID string) error {
	if err := keyring.Delete(aiService, connectionID); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return wrap(err)
	}
	return nil
}
