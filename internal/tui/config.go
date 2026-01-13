package tui

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/jonandersen/public-cli/internal/config"
)

// UIConfig holds TUI-specific configuration separate from CLI config.
type UIConfig struct {
	Watchlist []string `yaml:"watchlist,omitempty"`
}

// ConfigPath returns the path to the TUI config file.
func ConfigPath() string {
	return filepath.Join(config.ConfigDir(), "ui.yaml")
}

// LoadConfig loads the TUI config from disk.
func LoadConfig() (*UIConfig, error) {
	cfg := &UIConfig{}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveConfig saves the TUI config to disk.
func SaveConfig(cfg *UIConfig) error {
	path := ConfigPath()
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
