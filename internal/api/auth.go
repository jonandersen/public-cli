package api

import (
	"context"
	"fmt"
	"time"

	"github.com/jonandersen/public-cli/internal/auth"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// GetAuthToken retrieves a valid auth token using the keyring store.
// It handles secret retrieval, token exchange, and caching.
// If forceRefresh is true, it ignores any cached token.
func GetAuthToken(store keyring.Store, baseURL string, forceRefresh bool) (string, error) {
	secret, err := store.Get(keyring.ServiceName, keyring.KeySecretKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("CLI not configured. Run: pub configure\nOr set PUB_SECRET_KEY environment variable")
		}
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := auth.GetTokenWithRefresh(ctx, auth.TokenCachePath(), baseURL, secret, forceRefresh)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return token.AccessToken, nil
}

// NewClientWithAuth creates a new API client with automatic token retrieval.
// It fetches the auth token using the provided keyring store and base URL.
func NewClientWithAuth(store keyring.Store, baseURL string) (*Client, error) {
	token, err := GetAuthToken(store, baseURL, false)
	if err != nil {
		return nil, err
	}

	client := NewClient(baseURL, token)

	// Set up token refresher for auto-refresh on 401
	client.WithTokenRefresher(func() (string, error) {
		return GetAuthToken(store, baseURL, true)
	})

	return client, nil
}
