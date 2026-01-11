package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NonExistent(t *testing.T) {
	// When config file doesn't exist, should return defaults
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Errorf("APIBaseURL = %q, want %q", cfg.APIBaseURL, DefaultAPIBaseURL)
	}
	if cfg.TokenValidityMinutes != DefaultTokenValidityMinutes {
		t.Errorf("TokenValidityMinutes = %d, want %d", cfg.TokenValidityMinutes, DefaultTokenValidityMinutes)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	// Create temp dir and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `account_uuid: "test-uuid-123"
api_base_url: "https://custom.api.com"
token_validity_minutes: 30
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.AccountUUID != "test-uuid-123" {
		t.Errorf("AccountUUID = %q, want %q", cfg.AccountUUID, "test-uuid-123")
	}
	if cfg.APIBaseURL != "https://custom.api.com" {
		t.Errorf("APIBaseURL = %q, want %q", cfg.APIBaseURL, "https://custom.api.com")
	}
	if cfg.TokenValidityMinutes != 30 {
		t.Errorf("TokenValidityMinutes = %d, want %d", cfg.TokenValidityMinutes, 30)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	// Config with only some fields should use defaults for missing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `account_uuid: "partial-uuid"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.AccountUUID != "partial-uuid" {
		t.Errorf("AccountUUID = %q, want %q", cfg.AccountUUID, "partial-uuid")
	}
	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Errorf("APIBaseURL = %q, want default %q", cfg.APIBaseURL, DefaultAPIBaseURL)
	}
	if cfg.TokenValidityMinutes != DefaultTokenValidityMinutes {
		t.Errorf("TokenValidityMinutes = %d, want default %d", cfg.TokenValidityMinutes, DefaultTokenValidityMinutes)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `invalid: yaml: content: [broken`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() error = nil, want error for invalid YAML")
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		AccountUUID:          "save-test-uuid",
		APIBaseURL:           "https://save.api.com",
		TokenValidityMinutes: 45,
	}

	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("Config file permissions = %o, want %o", perm, 0600)
	}

	// Load it back and verify
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}

	if loaded.AccountUUID != cfg.AccountUUID {
		t.Errorf("AccountUUID = %q, want %q", loaded.AccountUUID, cfg.AccountUUID)
	}
	if loaded.APIBaseURL != cfg.APIBaseURL {
		t.Errorf("APIBaseURL = %q, want %q", loaded.APIBaseURL, cfg.APIBaseURL)
	}
	if loaded.TokenValidityMinutes != cfg.TokenValidityMinutes {
		t.Errorf("TokenValidityMinutes = %d, want %d", loaded.TokenValidityMinutes, cfg.TokenValidityMinutes)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "deep", "config.yaml")

	cfg := &Config{
		AccountUUID: "nested-uuid",
	}

	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Config file not created: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Errorf("APIBaseURL = %q, want %q", cfg.APIBaseURL, DefaultAPIBaseURL)
	}
	if cfg.TokenValidityMinutes != DefaultTokenValidityMinutes {
		t.Errorf("TokenValidityMinutes = %d, want %d", cfg.TokenValidityMinutes, DefaultTokenValidityMinutes)
	}
	if cfg.AccountUUID != "" {
		t.Errorf("AccountUUID = %q, want empty", cfg.AccountUUID)
	}
}

func TestConfigDir_WithXDG(t *testing.T) {
	// Save and restore original env
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	dir := ConfigDir()

	want := "/custom/config/pub"
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestConfigDir_WithoutXDG(t *testing.T) {
	// Save and restore original env
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	os.Unsetenv("XDG_CONFIG_HOME")
	dir := ConfigDir()

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "pub")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestConfigPath_WithXDG(t *testing.T) {
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	path := ConfigPath()

	want := "/custom/config/pub/config.yaml"
	if path != want {
		t.Errorf("ConfigPath() = %q, want %q", path, want)
	}
}

func TestConfigPath_WithoutXDG(t *testing.T) {
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	os.Unsetenv("XDG_CONFIG_HOME")
	path := ConfigPath()

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "pub", "config.yaml")
	if path != want {
		t.Errorf("ConfigPath() = %q, want %q", path, want)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		AccountUUID:          "550e8400-e29b-41d4-a716-446655440000",
		APIBaseURL:           "https://api.public.com",
		TokenValidityMinutes: 60,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() on default config error = %v, want nil", err)
	}
}

func TestValidate_EmptyAccountUUID(t *testing.T) {
	// Empty account UUID is valid (not yet configured)
	cfg := &Config{
		AccountUUID:          "",
		APIBaseURL:           "https://api.public.com",
		TokenValidityMinutes: 60,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil for empty AccountUUID", err)
	}
}

func TestValidate_InvalidAccountUUID(t *testing.T) {
	cfg := &Config{
		AccountUUID:          "not-a-valid-uuid",
		APIBaseURL:           "https://api.public.com",
		TokenValidityMinutes: 60,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for invalid AccountUUID")
	}
}

func TestValidate_InvalidAPIBaseURL(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		errMsg string
	}{
		{"empty", "", "api_base_url cannot be empty"},
		{"no scheme", "api.public.com", "api_base_url must be a valid URL"},
		{"invalid URL", "://invalid", "api_base_url must be a valid URL"},
		{"non-http scheme", "ftp://api.public.com", "api_base_url must use http or https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				APIBaseURL:           tt.url,
				TokenValidityMinutes: 60,
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Validate() error = nil, want error containing %q", tt.errMsg)
				return
			}
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidate_InvalidTokenValidityMinutes(t *testing.T) {
	tests := []struct {
		name    string
		minutes int
	}{
		{"zero", 0},
		{"negative", -1},
		{"very negative", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				APIBaseURL:           "https://api.public.com",
				TokenValidityMinutes: tt.minutes,
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Validate() error = nil, want error for TokenValidityMinutes=%d", tt.minutes)
			}
		})
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		AccountUUID:          "invalid-uuid",
		APIBaseURL:           "",
		TokenValidityMinutes: -1,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for multiple invalid fields")
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
