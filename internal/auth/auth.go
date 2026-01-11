package auth

import (
	"bytes"
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

// TokenRequest represents the request body for token exchange.
type TokenRequest struct {
	ValidityInMinutes int    `json:"validityInMinutes"`
	Secret            string `json:"secret"`
}

// TokenResponse represents the API response from token exchange.
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
}

// DefaultTokenValidityMinutes is the default token validity period.
const DefaultTokenValidityMinutes = 60

// ExchangeToken exchanges a secret key for an access token.
// It calls the Public.com API to perform the token exchange.
func ExchangeToken(ctx context.Context, baseURL, secretKey string) (*Token, error) {
	return ExchangeTokenWithValidity(ctx, baseURL, secretKey, DefaultTokenValidityMinutes)
}

// ExchangeTokenWithValidity exchanges a secret key for an access token with a custom validity period.
func ExchangeTokenWithValidity(ctx context.Context, baseURL, secretKey string, validityMinutes int) (*Token, error) {
	url := baseURL + "/userapiauthservice/personal/access-tokens"

	reqBody := TokenRequest{
		ValidityInMinutes: validityMinutes,
		Secret:            secretKey,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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
		ExpiresAt:   time.Now().Unix() + int64(validityMinutes)*60,
	}, nil
}
