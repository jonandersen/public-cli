package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")

	assert.Equal(t, "https://api.example.com", client.BaseURL)
	assert.Equal(t, "test-token", client.AuthToken)
	assert.NotNil(t, client.HTTPClient)
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")

	assert.Equal(t, 30*time.Second, client.HTTPClient.Timeout)
}

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/accounts", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accounts": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Get(context.Background(), "/accounts")

	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"accounts": []}`, string(body))
}

func TestClient_Get_AuthHeaderInjected(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "my-secret-token")
	resp, err := client.Get(context.Background(), "/test")

	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "Bearer my-secret-token", receivedAuth)
}

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/orders", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"symbol":"AAPL","qty":10}`, string(body))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"order_id": "123"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Post(context.Background(), "/orders", strings.NewReader(`{"symbol":"AAPL","qty":10}`))

	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestClient_Get_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Get(ctx, "/test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestClient_Get_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:99999", "test-token")

	_, err := client.Get(context.Background(), "/test")

	assert.Error(t, err)
}

func TestClient_Delete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/orders/123", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Delete(context.Background(), "/orders/123")

	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestClient_BaseURLTrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// BaseURL with trailing slash
	client := NewClient(server.URL+"/", "test-token")
	resp, err := client.Get(context.Background(), "/accounts")

	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
