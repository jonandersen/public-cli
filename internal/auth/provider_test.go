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
		// Verify request body contains secret
		var reqBody TokenRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "secret-key", reqBody.Secret)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "fresh-token",
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
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "token"})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	token, err := GetToken(ctx, cachePath, server.URL, "secret-key")

	assert.Nil(t, token)
	require.Error(t, err)
}

func TestClearToken_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Create a token file
	token := &Token{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Unix() + 3600,
	}
	err := SaveToken(cachePath, token)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(cachePath)
	require.NoError(t, err)

	// Clear the token
	err = ClearToken(cachePath)
	require.NoError(t, err)

	// Verify file is removed
	_, err = os.Stat(cachePath)
	assert.True(t, os.IsNotExist(err))
}

func TestClearToken_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".nonexistent_token")

	// Clear a non-existent file should succeed
	err := ClearToken(cachePath)
	require.NoError(t, err)
}

func TestClearToken_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	// Create a file inside to make the directory non-empty
	// os.Remove on non-empty directory returns an error
	filePath := filepath.Join(dirPath, "file.txt")
	err = os.WriteFile(filePath, []byte("test"), 0600)
	require.NoError(t, err)

	// Trying to clear a non-empty directory should return an error
	err = ClearToken(dirPath)
	require.Error(t, err)
}
