package auth

import (
	"context"
	"os"
)

// GetToken returns a valid access token, refreshing if necessary.
// It first tries to load a cached token. If the cached token is valid,
// it returns immediately. If the token is expired, missing, or corrupted,
// it exchanges the secret key for a new token and caches it.
func GetToken(ctx context.Context, cachePath, baseURL, secretKey string) (*Token, error) {
	// Try to load cached token
	token, err := LoadToken(cachePath)
	if err == nil && token.IsValid() {
		return token, nil
	}

	// Token missing, expired, or corrupted - exchange for new one
	token, err = ExchangeToken(ctx, baseURL, secretKey)
	if err != nil {
		return nil, err
	}

	// Cache the new token (ignore save errors - token is still usable)
	_ = SaveToken(cachePath, token)

	return token, nil
}

// ClearToken removes the cached token, forcing a refresh on next GetToken call.
func ClearToken(cachePath string) error {
	err := os.Remove(cachePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
