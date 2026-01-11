package cmd

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonandersen/pub/internal/keyring"
)

// mockPasswordReader is a test double for password input.
type mockPasswordReader struct {
	password   string
	err        error
	isTerminal bool
	readCalled bool
}

func newMockPasswordReader(password string, isTerminal bool) *mockPasswordReader {
	return &mockPasswordReader{
		password:   password,
		isTerminal: isTerminal,
	}
}

func (m *mockPasswordReader) WithError(err error) *mockPasswordReader {
	m.err = err
	return m
}

func (m *mockPasswordReader) ReadPassword() (string, error) {
	m.readCalled = true
	if m.err != nil {
		return "", m.err
	}
	return m.password, nil
}

func (m *mockPasswordReader) IsTerminal() bool {
	return m.isTerminal
}

func TestConfigureCmd_Success(t *testing.T) {
	// Create mock server for token exchange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapiauthservice/personal/access-tokens", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accessToken": "test-access-token"}`))
	}))
	defer server.Close()

	// Create temp directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create mock keyring store and password reader
	store := keyring.NewMockStore()
	pwReader := newMockPasswordReader("test-secret-key", true)

	// Create command with test dependencies
	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Enter your secret key:")
	assert.Contains(t, out.String(), "Configuration saved")
	assert.True(t, pwReader.readCalled)

	// Verify secret was stored
	secret, err := store.Get(keyring.ServiceName, keyring.KeySecretKey)
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
		_, _ = w.Write([]byte(`{"accessToken": "test-token"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := keyring.NewMockStore()
	pwReader := newMockPasswordReader("test-secret", true)

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
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
	pwReader := newMockPasswordReader("invalid-secret", true)

	cmd := newConfigureCmd(configureOptions{
		configPath:     filepath.Join(tmpDir, "config.yaml"),
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
	})

	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate")
}

func TestConfigureCmd_EmptySecret(t *testing.T) {
	pwReader := newMockPasswordReader("", true) // Empty secret

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret key cannot be empty")
}

func TestConfigureCmd_InvalidAccountUUID(t *testing.T) {
	pwReader := newMockPasswordReader("test-secret", true)

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
	})

	cmd.SetArgs([]string{
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
		_, _ = w.Write([]byte(`{"accessToken": "test-token"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	store := keyring.NewMockStore().WithSetError(assert.AnError)
	pwReader := newMockPasswordReader("test-secret", true)

	cmd := newConfigureCmd(configureOptions{
		configPath:     filepath.Join(tmpDir, "config.yaml"),
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
	})

	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store secret")
}

func TestConfigureCmd_NotATerminal(t *testing.T) {
	pwReader := newMockPasswordReader("test-secret", false) // Not a terminal

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
	})

	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires an interactive terminal")
}

func TestConfigureCmd_ReadPasswordError(t *testing.T) {
	pwReader := newMockPasswordReader("", true).WithError(errors.New("terminal read failed"))

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read secret key")
}
