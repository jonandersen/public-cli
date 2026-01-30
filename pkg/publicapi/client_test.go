package publicapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenProvider implements TokenProvider for testing.
type mockTokenProvider struct {
	token string
	err   error
	calls int
}

func (m *mockTokenProvider) Token() (string, error) {
	m.calls++
	return m.token, m.err
}

func TestNewClient(t *testing.T) {
	provider := &mockTokenProvider{token: "test-token"}
	client := NewClient("https://api.example.com", provider)

	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.BaseURL)
	assert.NotNil(t, client.HTTPClient)
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	provider := &mockTokenProvider{token: "test-token"}
	client := NewClient("https://api.example.com/", provider)

	assert.Equal(t, "https://api.example.com", client.BaseURL)
}

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/test-path", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	provider := &mockTokenProvider{token: "test-token"}
	client := NewClient(server.URL, provider)

	resp, err := client.Get(context.Background(), "/test-path")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, `{"status":"ok"}`, string(body))
}

func TestClient_GetWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "value1", r.URL.Query().Get("key1"))
		assert.Equal(t, "value2", r.URL.Query().Get("key2"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &mockTokenProvider{token: "test-token"}
	client := NewClient(server.URL, provider)

	params := map[string]string{"key1": "value1", "key2": "value2"}
	resp, err := client.GetWithParams(context.Background(), "/test", params)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"data":"test"}`, string(body))
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	provider := &mockTokenProvider{token: "test-token"}
	client := NewClient(server.URL, provider)

	resp, err := client.Post(context.Background(), "/create", strings.NewReader(`{"data":"test"}`))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/resource/123", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider := &mockTokenProvider{token: "test-token"}
	client := NewClient(server.URL, provider)

	resp, err := client.Delete(context.Background(), "/resource/123")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestClient_TokenRefreshOn401(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call returns 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Second call with new token should succeed
		assert.Equal(t, "Bearer new-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &mockTokenProvider{token: "old-token"}
	client := NewClient(server.URL, provider)

	// After first 401, provider returns new token
	provider.token = "new-token"

	resp, err := client.Get(context.Background(), "/protected")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, callCount, "should have made 2 requests (original + retry)")
	assert.Equal(t, 2, provider.calls, "should have called token provider twice")
}

func TestClient_NoRetryOn401WithoutProvider(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	// Create client with nil token provider (static token mode)
	client := NewClientWithToken(server.URL, "static-token")

	resp, err := client.Get(context.Background(), "/protected")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, 1, callCount, "should only make 1 request without provider")
}

func TestNewClientWithToken(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "my-token")

	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.BaseURL)
	assert.Nil(t, client.TokenProvider)
}
