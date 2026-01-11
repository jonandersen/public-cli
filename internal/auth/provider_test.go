package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetToken_FromCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Pre-populate cache with valid token
	cachedToken := &Token{
		AccessToken: "cached-token",
		ExpiresAt:   time.Now().Unix() + 3600,
	}
	err := SaveToken(cachePath, cachedToken)
	require.NoError(t, err)

	// Server should NOT be called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when cache is valid")
	}))
	defer server.Close()

	token, err := GetToken(context.Background(), cachePath, server.URL, "secret-key")

	require.NoError(t, err)
	assert.Equal(t, "cached-token", token.AccessToken)
}

func TestGetToken_RefreshExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Pre-populate cache with expired token
	expiredToken := &Token{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Unix() - 60, // expired 1 minute ago
	}
	err := SaveToken(cachePath, expiredToken)
	require.NoError(t, err)

	// Server returns new token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer secret-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "fresh-token",
			ExpiresIn:   3600,
		})
	}))
	defer server.Close()

	token, err := GetToken(context.Background(), cachePath, server.URL, "secret-key")

	require.NoError(t, err)
	assert.Equal(t, "fresh-token", token.AccessToken)

	// Verify cache was updated
	cached, err := LoadToken(cachePath)
	require.NoError(t, err)
	assert.Equal(t, "fresh-token", cached.AccessToken)
}

func TestGetToken_NoCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")
	// Don't create cache file

	// Server returns new token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "new-token",
			ExpiresIn:   3600,
		})
	}))
	defer server.Close()

	token, err := GetToken(context.Background(), cachePath, server.URL, "secret-key")

	require.NoError(t, err)
	assert.Equal(t, "new-token", token.AccessToken)

	// Verify cache was created
	cached, err := LoadToken(cachePath)
	require.NoError(t, err)
	assert.Equal(t, "new-token", cached.AccessToken)
}

func TestGetToken_ExchangeError(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Server returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	token, err := GetToken(context.Background(), cachePath, server.URL, "bad-key")

	assert.Nil(t, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestGetToken_CorruptedCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Write corrupted cache file
	err := os.WriteFile(cachePath, []byte("not json"), 0600)
	require.NoError(t, err)

	// Server returns new token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "recovered-token",
			ExpiresIn:   3600,
		})
	}))
	defer server.Close()

	token, err := GetToken(context.Background(), cachePath, server.URL, "secret-key")

	require.NoError(t, err)
	assert.Equal(t, "recovered-token", token.AccessToken)
}

func TestGetToken_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "token", ExpiresIn: 3600})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	token, err := GetToken(ctx, cachePath, server.URL, "secret-key")

	assert.Nil(t, token)
	require.Error(t, err)
}
