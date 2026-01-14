package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonandersen/public-cli/internal/keyring"
)

func TestGetAuthToken_NotConfigured(t *testing.T) {
	store := keyring.NewMockStore() // Empty store - no secret key

	_, err := GetAuthToken(store, "https://api.public.com", false)
	require.Error(t, err)

	// Verify error message matches expected format
	assert.Contains(t, err.Error(), "CLI not configured")
	assert.Contains(t, err.Error(), "pub configure")
	assert.Contains(t, err.Error(), "PUB_SECRET_KEY")
}

func TestGetAuthToken_KeyringError(t *testing.T) {
	store := keyring.NewMockStore().WithGetError(assert.AnError)

	_, err := GetAuthToken(store, "https://api.public.com", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve secret")
}

func TestGetAuthToken_Success(t *testing.T) {
	// Use temp directory for token cache isolation
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Mock server returns a valid token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapiauthservice/personal/access-tokens", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"accessToken": "test-access-token-123",
		})
	}))
	defer server.Close()

	store := keyring.NewMockStore().WithData(keyring.ServiceName, keyring.KeySecretKey, "test-secret-key")

	token, err := GetAuthToken(store, server.URL, false)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token-123", token)
}

func TestGetAuthToken_ExchangeError(t *testing.T) {
	// Use temp directory for token cache isolation
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Mock server returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid secret"}`))
	}))
	defer server.Close()

	store := keyring.NewMockStore().WithData(keyring.ServiceName, keyring.KeySecretKey, "bad-secret")

	_, err := GetAuthToken(store, server.URL, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authenticate")
}

func TestNewClientWithAuth_Success(t *testing.T) {
	// Use temp directory for token cache isolation
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Mock server returns a valid token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"accessToken": "test-access-token-123",
		})
	}))
	defer server.Close()

	store := keyring.NewMockStore().WithData(keyring.ServiceName, keyring.KeySecretKey, "test-secret-key")

	client, err := NewClientWithAuth(store, server.URL)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "test-access-token-123", client.AuthToken)
	assert.NotNil(t, client.TokenRefresher)
}

func TestNewClientWithAuth_AuthError(t *testing.T) {
	store := keyring.NewMockStore() // Empty store - no secret key

	client, err := NewClientWithAuth(store, "https://api.public.com")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "CLI not configured")
}
