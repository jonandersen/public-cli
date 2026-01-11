package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"

	"github.com/jonandersen/pub/internal/auth"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// uuidRegex matches standard UUID format
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// configureOptions holds dependencies for the configure command.
// This allows for dependency injection in tests.
type configureOptions struct {
	configPath string
	baseURL    string
	store      keyring.Store
}

// newConfigureCmd creates the configure command with the given options.
func newConfigureCmd(opts configureOptions) *cobra.Command {
	var secretKey string
	var accountUUID string

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure CLI credentials",
		Long: `Configure the CLI with your Public.com API credentials.

Get your secret key from: https://public.com/settings/security/api

Example:
  pub configure --secret YOUR_SECRET_KEY
  pub configure --secret YOUR_SECRET_KEY --account YOUR_ACCOUNT_UUID`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigure(cmd, opts, secretKey, accountUUID)
		},
	}

	cmd.Flags().StringVar(&secretKey, "secret", "", "Your Public.com API secret key (required)")
	cmd.Flags().StringVar(&accountUUID, "account", "", "Default account UUID (optional)")

	return cmd
}

func runConfigure(cmd *cobra.Command, opts configureOptions, secretKey, accountUUID string) error {
	// Validate required secret key
	if secretKey == "" {
		return fmt.Errorf("secret key is required (use --secret flag)")
	}

	// Validate account UUID format if provided
	if accountUUID != "" && !uuidRegex.MatchString(accountUUID) {
		return fmt.Errorf("invalid account UUID format")
	}

	// Validate secret key by exchanging for token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := auth.ExchangeToken(ctx, opts.baseURL, secretKey)
	if err != nil {
		return fmt.Errorf("failed to validate secret key: %w", err)
	}

	// Store secret in keyring
	if err := opts.store.Set("pub", "secret_key", secretKey); err != nil {
		return fmt.Errorf("failed to store secret in keyring: %w", err)
	}

	// Load existing config or create new one
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Update config with provided values
	if accountUUID != "" {
		cfg.AccountUUID = accountUUID
	}

	// Save config
	if err := config.Save(opts.configPath, cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Configuration saved successfully!")
	return nil
}

func init() {
	// Create configure command with production dependencies
	configureCmd := newConfigureCmd(configureOptions{
		configPath: config.ConfigPath(),
		baseURL:    config.DefaultAPIBaseURL,
		store:      keyring.NewEnvStore(keyring.NewSystemStore()),
	})
	rootCmd.AddCommand(configureCmd)
}
