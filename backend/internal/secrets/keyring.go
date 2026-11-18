// Package secrets keeps GitHub credentials in the operating system's
// keychain (Secret Service on Linux, Keychain on macOS, Credential Manager on
// Windows), never in the database.
package secrets

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const service = "yap-github"

type Keyring struct{}

func NewKeyring() *Keyring {
	return &Keyring{}
}

func wrap(err error) error {
	if errors.Is(err, keyring.ErrNotFound) {
		return model.ErrNoCredential
	}
	return fmt.Errorf("%w: %v", model.ErrSecretStoreUnavailable, err)
}

func (*Keyring) Get(userID string) (model.GitHubCredential, error) {
	raw, err := keyring.Get(service, userID)
	if err != nil {
		return model.GitHubCredential{}, wrap(err)
	}
	var c model.GitHubCredential
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Token == "" {
		// Unreadable entries count as no credential; connecting again
		// overwrites them.
		return model.GitHubCredential{}, model.ErrNoCredential
	}
	return c, nil
}

func (*Keyring) Set(userID string, c model.GitHubCredential) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode credential: %w", err)
	}
	if err := keyring.Set(service, userID, string(raw)); err != nil {
		return wrap(err)
	}
	return nil
}

func (*Keyring) Delete(userID string) error {
	if err := keyring.Delete(service, userID); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return wrap(err)
	}
	return nil
}
