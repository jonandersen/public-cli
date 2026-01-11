package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// tokenCache is the JSON structure for the cached token file.
type tokenCache struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

// IsValid returns true if the token has not expired.
func (t *Token) IsValid() bool {
	return t.ExpiresAt > time.Now().Unix()
}

// SaveToken writes a token to the cache file.
// Creates parent directories if needed with 0700 permissions.
// The file is written with 0600 permissions.
func SaveToken(path string, token *Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	cache := tokenCache{
		AccessToken: token.AccessToken,
		ExpiresAt:   token.ExpiresAt,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// LoadToken reads a token from the cache file.
// Returns an error if the file doesn't exist or contains invalid JSON.
func LoadToken(path string) (*Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cache tokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &Token{
		AccessToken: cache.AccessToken,
		ExpiresAt:   cache.ExpiresAt,
	}, nil
}

// DeleteToken removes the token cache file.
// Returns nil if the file doesn't exist.
func DeleteToken(path string) error {
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// TokenCachePath returns the path to the token cache file.
// Uses XDG_CONFIG_HOME if set, otherwise ~/.config/pub.
func TokenCachePath() string {
	var configDir string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = filepath.Join(xdg, "pub")
	} else {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config", "pub")
	}
	return filepath.Join(configDir, ".token_cache")
}
