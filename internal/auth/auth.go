package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Token represents an access token with its expiry.
type Token struct {
	AccessToken string
	ExpiresAt   int64
}

// TokenResponse represents the API response from token exchange.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExchangeToken exchanges a secret key for an access token.
// It calls the Public.com API to perform the token exchange.
func ExchangeToken(ctx context.Context, baseURL, secretKey string) (*Token, error) {
	url := baseURL + "/userapiauthservice/personal/access-tokens"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+secretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return nil, fmt.Errorf("token exchange failed: %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("token exchange failed: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token in response")
	}

	return &Token{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   time.Now().Unix() + tokenResp.ExpiresIn,
	}, nil
}
