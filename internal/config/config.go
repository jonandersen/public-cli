package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// uuidRegex matches standard UUID format (8-4-4-4-12 hex digits)
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	DefaultAPIBaseURL           = "https://api.public.com"
	DefaultTokenValidityMinutes = 60
)

// Config holds the CLI configuration.
type Config struct {
	AccountUUID          string `yaml:"account_uuid"`
	APIBaseURL           string `yaml:"api_base_url"`
	TokenValidityMinutes int    `yaml:"token_validity_minutes"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		APIBaseURL:           DefaultAPIBaseURL,
		TokenValidityMinutes: DefaultTokenValidityMinutes,
	}
}

// Validate checks that the configuration values are valid.
// Returns an error describing all validation failures.
func (c *Config) Validate() error {
	var errs []error

	// Validate AccountUUID (optional, but if set must be valid UUID)
	if c.AccountUUID != "" && !uuidRegex.MatchString(c.AccountUUID) {
		errs = append(errs, fmt.Errorf("account_uuid must be a valid UUID"))
	}

	// Validate APIBaseURL (required, must be valid http/https URL)
	if c.APIBaseURL == "" {
		errs = append(errs, fmt.Errorf("api_base_url cannot be empty"))
	} else {
		parsed, err := url.Parse(c.APIBaseURL)
		if err != nil || parsed.Host == "" {
			errs = append(errs, fmt.Errorf("api_base_url must be a valid URL"))
		} else if parsed.Scheme != "http" && parsed.Scheme != "https" {
			errs = append(errs, fmt.Errorf("api_base_url must use http or https"))
		}
	}

	// Validate TokenValidityMinutes (must be positive)
	if c.TokenValidityMinutes <= 0 {
		errs = append(errs, fmt.Errorf("token_validity_minutes must be positive"))
	}

	return errors.Join(errs...)
}

// Load reads configuration from the given path.
// Returns default config if file doesn't exist.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for zero values
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = DefaultAPIBaseURL
	}
	if cfg.TokenValidityMinutes == 0 {
		cfg.TokenValidityMinutes = DefaultTokenValidityMinutes
	}

	return cfg, nil
}

// Save writes configuration to the given path.
// Creates parent directories if needed.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// ConfigDir returns the configuration directory path.
// Uses XDG_CONFIG_HOME if set, otherwise ~/.config/pub.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "pub")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pub")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}
