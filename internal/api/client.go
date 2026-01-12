package api

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

// TokenRefresher is a function that returns a fresh auth token.
// It's called when the API returns 401 Unauthorized.
type TokenRefresher func() (string, error)

// Client handles HTTP requests to the Public.com API.
type Client struct {
	BaseURL        string
	AuthToken      string
	HTTPClient     *http.Client
	TokenRefresher TokenRefresher // Optional: called on 401 to get fresh token
}

// NewClient creates a new API client with the given base URL and auth token.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		AuthToken: authToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithTokenRefresher sets a token refresher function that will be called on 401.
func (c *Client) WithTokenRefresher(refresher TokenRefresher) *Client {
	c.TokenRefresher = refresher
	return c
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

// do performs an HTTP request with auth header injection.
// On 401, if a TokenRefresher is configured, it will refresh the token and retry once.
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

	resp, err := c.doOnce(ctx, method, path, bodyBytes)
	if err != nil {
		return nil, err
	}

	// Retry on 401 if we have a token refresher
	if resp.StatusCode == 401 && c.TokenRefresher != nil {
		_ = resp.Body.Close()

		newToken, refreshErr := c.TokenRefresher()
		if refreshErr != nil {
			// Refresh failed, return original 401 response
			// Re-do the request to get a fresh response body
			return c.doOnce(ctx, method, path, bodyBytes)
		}

		c.AuthToken = newToken
		return c.doOnce(ctx, method, path, bodyBytes)
	}

	return resp, nil
}

// doOnce performs a single HTTP request.
func (c *Client) doOnce(ctx context.Context, method, path string, bodyBytes []byte) (*http.Response, error) {
	url := c.BaseURL + path

	var body io.Reader
	if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}
