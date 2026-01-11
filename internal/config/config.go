package config

// Config holds the CLI configuration.
type Config struct {
	AccountUUID          string `yaml:"account_uuid"`
	APIBaseURL           string `yaml:"api_base_url"`
	TokenValidityMinutes int    `yaml:"token_validity_minutes"`
}
