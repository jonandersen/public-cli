package keyring

import (
	"errors"

	gokeyring "github.com/zalando/go-keyring"
)

// ErrNotFound is returned when a secret is not found in the keyring.
var ErrNotFound = errors.New("secret not found")

// Store provides an interface for secure secret storage.
type Store interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

// SystemStore implements Store using the system keyring.
type SystemStore struct{}

// NewSystemStore creates a new system keyring store.
func NewSystemStore() *SystemStore {
	return &SystemStore{}
}

// Get retrieves a secret from the system keyring.
func (s *SystemStore) Get(service, key string) (string, error) {
	secret, err := gokeyring.Get(service, key)
	if err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return secret, nil
}

// Set stores a secret in the system keyring.
func (s *SystemStore) Set(service, key, value string) error {
	return gokeyring.Set(service, key, value)
}

// Delete removes a secret from the system keyring.
func (s *SystemStore) Delete(service, key string) error {
	err := gokeyring.Delete(service, key)
	if err != nil && errors.Is(err, gokeyring.ErrNotFound) {
		return nil // Deleting non-existent key is not an error
	}
	return err
}
