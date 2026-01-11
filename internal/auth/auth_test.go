package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeToken_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/userapiauthservice/personal/access-tokens", r.URL.Path)

		// Verify content type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body contains secret and validity
		var reqBody TokenRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test-secret-key", reqBody.Secret)
		assert.Equal(t, DefaultTokenValidityMinutes, reqBody.ValidityInMinutes)

		// Return successful response
		resp := TokenResponse{
			AccessToken: "access-token-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Execute
	token, err := ExchangeToken(context.Background(), server.URL, "test-secret-key")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "access-token-123", token.AccessToken)
	// ExpiresAt should be roughly now + DefaultTokenValidityMinutes*60 seconds
	expectedExpiry := time.Now().Unix() + int64(DefaultTokenValidityMinutes)*60
	assert.InDelta(t, expectedExpiry, token.ExpiresAt, 5) // Allow 5 second delta
}

func TestExchangeTokenWithValidity_CustomValidity(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body contains custom validity
		var reqBody TokenRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test-secret-key", reqBody.Secret)
		assert.Equal(t, 120, reqBody.ValidityInMinutes)

		// Return successful response
		resp := TokenResponse{
			AccessToken: "access-token-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Execute with custom validity
	token, err := ExchangeTokenWithValidity(context.Background(), server.URL, "test-secret-key", 120)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "access-token-123", token.AccessToken)
	// ExpiresAt should be roughly now + 120*60 seconds
	expectedExpiry := time.Now().Unix() + 120*60
	assert.InDelta(t, expectedExpiry, token.ExpiresAt, 5)
}

func TestExchangeToken_HTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error": "invalid secret key"}`,
			wantErr:    "token exchange failed: 401",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"error": "access denied"}`,
			wantErr:    "token exchange failed: 403",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error": "internal error"}`,
			wantErr:    "token exchange failed: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			_, err := ExchangeToken(context.Background(), server.URL, "bad-key")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestExchangeToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	_, err := ExchangeToken(context.Background(), server.URL, "test-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestExchangeToken_NetworkError(t *testing.T) {
	// Use a server that's immediately closed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	_, err := ExchangeToken(context.Background(), server.URL, "test-key")

	require.Error(t, err)
}

func TestExchangeToken_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "token"})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := ExchangeToken(ctx, server.URL, "test-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestExchangeToken_EmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: ""})
	}))
	defer server.Close()

	_, err := ExchangeToken(context.Background(), server.URL, "test-key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty access token")
}
