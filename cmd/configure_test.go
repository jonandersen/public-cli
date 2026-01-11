package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonandersen/pub/internal/keyring"
)

func TestConfigureCmd_Success(t *testing.T) {
	// Create mock server for token exchange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapiauthservice/personal/access-tokens", r.URL.Path)
		assert.Equal(t, "Bearer test-secret-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "test-access-token", "expires_in": 3600}`))
	}))
	defer server.Close()

	// Create temp directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create mock keyring store
	store := keyring.NewMockStore()

	// Create command with test dependencies
	cmd := newConfigureCmd(configureOptions{
		configPath: configPath,
		baseURL:    server.URL,
		store:      store,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--secret", "test-secret-key"})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Configuration saved")

	// Verify secret was stored
	secret, err := store.Get("pub", "secret_key")
	require.NoError(t, err)
	assert.Equal(t, "test-secret-key", secret)

	// Verify config file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestConfigureCmd_WithAccountUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "test-token", "expires_in": 3600}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := keyring.NewMockStore()

	cmd := newConfigureCmd(configureOptions{
		configPath: configPath,
		baseURL:    server.URL,
		store:      store,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--secret", "test-secret",
		"--account", "12345678-1234-1234-1234-123456789012",
	})

	err := cmd.Execute()

	require.NoError(t, err)

	// Verify config contains account UUID
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "12345678-1234-1234-1234-123456789012")
}

func TestConfigureCmd_InvalidSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Invalid secret key"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	store := keyring.NewMockStore()

	cmd := newConfigureCmd(configureOptions{
		configPath: filepath.Join(tmpDir, "config.yaml"),
		baseURL:    server.URL,
		store:      store,
	})

	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"--secret", "invalid-secret"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate")
}

func TestConfigureCmd_MissingSecret(t *testing.T) {
	cmd := newConfigureCmd(configureOptions{
		configPath: "/tmp/config.yaml",
		baseURL:    "https://api.example.com",
		store:      keyring.NewMockStore(),
	})

	var errOut bytes.Buffer
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret key is required")
}

func TestConfigureCmd_InvalidAccountUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "test-token", "expires_in": 3600}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	store := keyring.NewMockStore()

	cmd := newConfigureCmd(configureOptions{
		configPath: filepath.Join(tmpDir, "config.yaml"),
		baseURL:    server.URL,
		store:      store,
	})

	cmd.SetArgs([]string{
		"--secret", "test-secret",
		"--account", "not-a-valid-uuid",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account UUID")
}

func TestConfigureCmd_KeyringError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "test-token", "expires_in": 3600}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	store := keyring.NewMockStore().WithSetError(assert.AnError)

	cmd := newConfigureCmd(configureOptions{
		configPath: filepath.Join(tmpDir, "config.yaml"),
		baseURL:    server.URL,
		store:      store,
	})

	cmd.SetArgs([]string{"--secret", "test-secret"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store secret")
}
