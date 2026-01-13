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

	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
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
	// Create mock server for token exchange and account fetching
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/userapiauthservice/personal/access-tokens":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken": "test-access-token"}`))
		case "/userapigateway/trading/account":
			// Return empty accounts so selection is skipped
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accounts": []}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create temp directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create mock keyring store and password reader
	store := keyring.NewMockStore()
	pwReader := newMockPasswordReader("test-secret-key", true)
	prompt := newMockPrompt() // Not used since no accounts

	// Create command with test dependencies
	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
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
	prompt := newMockPrompt() // Not used when account provided via flag

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
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
	prompt := newMockPrompt()

	cmd := newConfigureCmd(configureOptions{
		configPath:     filepath.Join(tmpDir, "config.yaml"),
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
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
	prompt := newMockPrompt()

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
		prompt:         prompt,
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
	prompt := newMockPrompt()

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
		prompt:         prompt,
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
	prompt := newMockPrompt()

	cmd := newConfigureCmd(configureOptions{
		configPath:     filepath.Join(tmpDir, "config.yaml"),
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store secret")
}

func TestConfigureCmd_NotATerminal(t *testing.T) {
	pwReader := newMockPasswordReader("test-secret", false) // Not a terminal
	prompt := newMockPrompt()

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
		prompt:         prompt,
	})

	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires an interactive terminal")
}

func TestConfigureCmd_ReadPasswordError(t *testing.T) {
	pwReader := newMockPasswordReader("", true).WithError(errors.New("terminal read failed"))
	prompt := newMockPrompt()

	cmd := newConfigureCmd(configureOptions{
		configPath:     "/tmp/config.yaml",
		baseURL:        "https://api.example.com",
		store:          keyring.NewMockStore(),
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read secret key")
}

// mockPrompt is a test double for interactive menu prompts.
type mockPrompt struct {
	selections []int    // Which option to select for each call
	callIndex  int      // Current call index
	lines      []string // Lines to return for ReadLine calls
	lineIndex  int      // Current line index
}

func newMockPrompt(selections ...int) *mockPrompt {
	return &mockPrompt{selections: selections}
}

func (m *mockPrompt) WithLines(lines ...string) *mockPrompt {
	m.lines = lines
	return m
}

func (m *mockPrompt) SelectOption(options []string) (int, error) {
	if m.callIndex >= len(m.selections) {
		return 0, errors.New("no more mock selections")
	}
	idx := m.selections[m.callIndex]
	m.callIndex++
	return idx, nil
}

func (m *mockPrompt) ReadLine(prompt string) (string, error) {
	if m.lineIndex >= len(m.lines) {
		return "", nil // Default to empty/skip
	}
	line := m.lines[m.lineIndex]
	m.lineIndex++
	return line, nil
}

func TestConfigureCmd_AccountSelectionAfterSetup(t *testing.T) {
	// Mock server that returns accounts after successful token exchange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/userapiauthservice/personal/access-tokens":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken": "test-access-token"}`))
		case "/userapigateway/trading/account":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"accounts": [
					{"accountId": "acc-111", "accountType": "INDIVIDUAL", "optionsLevel": "LEVEL_2"},
					{"accountId": "acc-222", "accountType": "IRA", "optionsLevel": "LEVEL_1"}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := keyring.NewMockStore()
	pwReader := newMockPasswordReader("test-secret-key", true)
	prompt := newMockPrompt(0) // Select first account

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Configuration saved")

	// Verify first account was selected and saved
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "acc-111", cfg.AccountUUID)
}

func TestConfigureCmd_SkipAccountSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/userapiauthservice/personal/access-tokens":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken": "test-access-token"}`))
		case "/userapigateway/trading/account":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"accounts": [
					{"accountId": "acc-111", "accountType": "INDIVIDUAL"}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := keyring.NewMockStore()
	pwReader := newMockPasswordReader("test-secret-key", true)
	prompt := newMockPrompt(1) // Select "Skip" option (index 1)

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)

	// Verify no account was saved
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Empty(t, cfg.AccountUUID)
}

func TestConfigureCmd_ReconfigureMenu(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accessToken": "test-access-token"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Pre-configure: store existing secret
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "existing-secret")

	pwReader := newMockPasswordReader("new-secret", true)
	prompt := newMockPrompt(1) // Select "Configure new secret key"

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "already configured")

	// Verify new secret was stored
	secret, err := store.Get(keyring.ServiceName, keyring.KeySecretKey)
	require.NoError(t, err)
	assert.Equal(t, "new-secret", secret)
}

func TestConfigureCmd_ClearSecret(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Pre-configure
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "existing-secret")

	pwReader := newMockPasswordReader("", true)
	prompt := newMockPrompt(4) // Select "Clear secret key" (index 4 after Toggle trading)

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        "https://api.example.com",
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Secret key cleared")

	// Verify secret was removed
	_, err = store.Get(keyring.ServiceName, keyring.KeySecretKey)
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

func TestConfigureCmd_ToggleTrading(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file with trading disabled (default)
	cfg := &config.Config{
		APIBaseURL:           "https://api.public.com",
		TokenValidityMinutes: 60,
		TradingEnabled:       false,
	}
	require.NoError(t, config.Save(configPath, cfg))

	// Pre-configure
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "existing-secret")

	pwReader := newMockPasswordReader("", true)
	prompt := newMockPrompt(3) // Select "Toggle trading"

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        "https://api.example.com",
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Trading is now ENABLED")

	// Verify config was updated
	loaded, err := config.Load(configPath)
	require.NoError(t, err)
	assert.True(t, loaded.TradingEnabled)
}

func TestConfigureCmd_ToggleTradingOff(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file with trading enabled
	cfg := &config.Config{
		APIBaseURL:           "https://api.public.com",
		TokenValidityMinutes: 60,
		TradingEnabled:       true,
	}
	require.NoError(t, config.Save(configPath, cfg))

	// Pre-configure
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "existing-secret")

	pwReader := newMockPasswordReader("", true)
	prompt := newMockPrompt(3) // Select "Toggle trading"

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        "https://api.example.com",
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Trading is now DISABLED")

	// Verify config was updated
	loaded, err := config.Load(configPath)
	require.NoError(t, err)
	assert.False(t, loaded.TradingEnabled)
}

func TestConfigureCmd_ViewConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file with existing settings
	cfg := &config.Config{
		AccountUUID:          "test-account-uuid",
		APIBaseURL:           "https://api.public.com",
		TokenValidityMinutes: 60,
	}
	require.NoError(t, config.Save(configPath, cfg))

	// Pre-configure
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "existing-secret")

	pwReader := newMockPasswordReader("", true)
	prompt := newMockPrompt(2) // Select "View current configuration"

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        "https://api.public.com",
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)
	output := out.String()
	assert.Contains(t, output, "test-account-uuid")
	assert.Contains(t, output, "Secret key: Configured")
}

func TestConfigureCmd_SelectDifferentAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/userapiauthservice/personal/access-tokens":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken": "test-access-token"}`))
		case "/userapigateway/trading/account":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"accounts": [
					{"accountId": "acc-111", "accountType": "INDIVIDUAL"},
					{"accountId": "acc-222", "accountType": "IRA"}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Pre-configure with existing account
	cfg := &config.Config{
		AccountUUID:          "acc-111",
		APIBaseURL:           server.URL,
		TokenValidityMinutes: 60,
	}
	require.NoError(t, config.Save(configPath, cfg))

	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "existing-secret")

	pwReader := newMockPasswordReader("", true)
	prompt := newMockPrompt(0, 1) // Select "Select different account", then second account

	cmd := newConfigureCmd(configureOptions{
		configPath:     configPath,
		baseURL:        server.URL,
		store:          store,
		passwordReader: pwReader,
		prompt:         prompt,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.NoError(t, err)

	// Verify second account was saved
	updatedCfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "acc-222", updatedCfg.AccountUUID)
}
