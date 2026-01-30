// Package publicapi provides a Go client for the Public.com API.
//
// This package can be imported by external projects to interact with
// Public.com's trading API programmatically.
package publicapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenProvider is an interface for obtaining authentication tokens.
// Implementations should handle token caching and refresh logic internally.
type TokenProvider interface {
	// Token returns a valid authentication token.
	// It may return a cached token or fetch a new one.
	Token() (string, error)
}

// Client handles HTTP requests to the Public.com API.
type Client struct {
	BaseURL       string
	TokenProvider TokenProvider
	HTTPClient    *http.Client

	// staticToken is used when a fixed token is provided (no refresh capability)
	staticToken string
}

// NewClient creates a new API client with the given base URL and token provider.
// The token provider will be called to obtain tokens, allowing for automatic
// token refresh when needed.
func NewClient(baseURL string, tokenProvider TokenProvider) *Client {
	return &Client{
		BaseURL:       strings.TrimSuffix(baseURL, "/"),
		TokenProvider: tokenProvider,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithToken creates a new API client with a static token.
// This is useful for short-lived operations where token refresh is not needed.
// The client will not attempt to refresh the token on 401 responses.
func NewClientWithToken(baseURL, token string) *Client {
	return &Client{
		BaseURL:     strings.TrimSuffix(baseURL, "/"),
		staticToken: token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request to the specified path.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// GetWithParams performs a GET request to the specified path with query parameters.
func (c *Client) GetWithParams(ctx context.Context, path string, params map[string]string) (*http.Response, error) {
	if len(params) > 0 {
		query := url.Values{}
		for k, v := range params {
			query.Set(k, v)
		}
		path = path + "?" + query.Encode()
	}
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request to the specified path with the given body.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// Delete performs a DELETE request to the specified path.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil)
}

// getToken returns the current authentication token.
func (c *Client) getToken() (string, error) {
	if c.TokenProvider != nil {
		return c.TokenProvider.Token()
	}
	return c.staticToken, nil
}

// do performs an HTTP request with auth header injection.
// On 401, if a TokenProvider is configured, it will refresh the token and retry once.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Buffer body if present so we can retry
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	// Get initial token
	token, err := c.getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	resp, err := c.doOnce(ctx, method, path, bodyBytes, token)
	if err != nil {
		return nil, err
	}

	// Retry on 401 if we have a token provider (can refresh)
	if resp.StatusCode == http.StatusUnauthorized && c.TokenProvider != nil {
		_ = resp.Body.Close()

		// Get fresh token
		newToken, refreshErr := c.TokenProvider.Token()
		if refreshErr != nil {
			// Refresh failed, re-do request to get a fresh response
			return c.doOnce(ctx, method, path, bodyBytes, token)
		}

		return c.doOnce(ctx, method, path, bodyBytes, newToken)
	}

	return resp, nil
}

// doOnce performs a single HTTP request.
func (c *Client) doOnce(ctx context.Context, method, path string, bodyBytes []byte, token string) (*http.Response, error) {
	url := c.BaseURL + path

	var body io.Reader
	if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}
