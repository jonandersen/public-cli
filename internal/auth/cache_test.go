package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{
			name:      "valid token - expires in future",
			expiresAt: time.Now().Unix() + 3600,
			want:      true,
		},
		{
			name:      "expired token - expires in past",
			expiresAt: time.Now().Unix() - 60,
			want:      false,
		},
		{
			name:      "expired token - expires now",
			expiresAt: time.Now().Unix(),
			want:      false,
		},
		{
			name:      "valid token - expires in 1 second",
			expiresAt: time.Now().Unix() + 1,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				AccessToken: "test-token",
				ExpiresAt:   tt.expiresAt,
			}
			assert.Equal(t, tt.want, token.IsValid())
		})
	}
}

func TestSaveToken(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	token := &Token{
		AccessToken: "my-access-token",
		ExpiresAt:   time.Now().Unix() + 3600,
	}

	err := SaveToken(cachePath, token)
	require.NoError(t, err)

	// Verify file exists
	info, err := os.Stat(cachePath)
	require.NoError(t, err)

	// Verify permissions (0600)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Verify content is valid JSON
	data, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "my-access-token")
}

func TestSaveToken_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", ".token_cache")

	token := &Token{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Unix() + 3600,
	}

	err := SaveToken(nestedPath, token)
	require.NoError(t, err)

	// Verify directory was created with correct permissions
	dirInfo, err := os.Stat(filepath.Dir(nestedPath))
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
	assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())
}

func TestLoadToken_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Save a token first
	originalToken := &Token{
		AccessToken: "saved-token",
		ExpiresAt:   time.Now().Unix() + 3600,
	}
	err := SaveToken(cachePath, originalToken)
	require.NoError(t, err)

	// Load it back
	loadedToken, err := LoadToken(cachePath)
	require.NoError(t, err)

	assert.Equal(t, originalToken.AccessToken, loadedToken.AccessToken)
	assert.Equal(t, originalToken.ExpiresAt, loadedToken.ExpiresAt)
}

func TestLoadToken_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent")

	token, err := LoadToken(cachePath)

	assert.Nil(t, token)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestLoadToken_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Write invalid JSON
	err := os.WriteFile(cachePath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	token, err := LoadToken(cachePath)

	assert.Nil(t, token)
	require.Error(t, err)
}

func TestLoadToken_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Write empty file
	err := os.WriteFile(cachePath, []byte(""), 0600)
	require.NoError(t, err)

	token, err := LoadToken(cachePath)

	assert.Nil(t, token)
	require.Error(t, err)
}

func TestTokenCachePath(t *testing.T) {
	// Save and restore env
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		_ = os.Setenv("XDG_CONFIG_HOME", "/custom/config")
		path := TokenCachePath()
		assert.Equal(t, "/custom/config/pub/.token_cache", path)
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		_ = os.Unsetenv("XDG_CONFIG_HOME")
		path := TokenCachePath()
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".config", "pub", ".token_cache"), path)
	})
}

func TestDeleteToken(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".token_cache")

	// Create a token file
	err := os.WriteFile(cachePath, []byte(`{"access_token":"test"}`), 0600)
	require.NoError(t, err)

	// Delete it
	err = DeleteToken(cachePath)
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(cachePath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteToken_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent")

	// Should not error if file doesn't exist
	err := DeleteToken(cachePath)
	require.NoError(t, err)
}
