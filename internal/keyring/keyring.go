package keyring

import (
	"errors"
	"os"

	gokeyring "github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keyring service name for storing secrets.
	// Uses reverse domain notation for proper namespacing.
	ServiceName = "com.public.pub"

	// KeySecretKey is the keyring key for the API secret key.
	KeySecretKey = "secret_key"

	// EnvSecretKey is the environment variable name for the secret key.
	// When set, it overrides keyring lookups for CI/headless environments.
	EnvSecretKey = "PUB_SECRET_KEY"
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

// EnvStore wraps another Store and checks environment variables first.
// This enables CI/headless environments to provide credentials via env vars.
type EnvStore struct {
	underlying Store
}

// NewEnvStore creates a new EnvStore wrapping the given store.
func NewEnvStore(underlying Store) *EnvStore {
	return &EnvStore{underlying: underlying}
}

// Get retrieves a secret, checking env var first for secret_key lookups.
func (e *EnvStore) Get(service, key string) (string, error) {
	// Check env var for secret_key lookups
	if key == KeySecretKey {
		if envVal := os.Getenv(EnvSecretKey); envVal != "" {
			return envVal, nil
		}
	}
	return e.underlying.Get(service, key)
}

// Set stores a secret in the underlying store.
func (e *EnvStore) Set(service, key, value string) error {
	return e.underlying.Set(service, key, value)
}

// Delete removes a secret from the underlying store.
func (e *EnvStore) Delete(service, key string) error {
	return e.underlying.Delete(service, key)
}
