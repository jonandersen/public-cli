package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jonandersen/pub/internal/auth"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// passwordReader abstracts terminal password input for testing.
type passwordReader interface {
	ReadPassword() (string, error)
	IsTerminal() bool
}

// terminalReader reads passwords from the terminal using golang.org/x/term.
type terminalReader struct {
	fd int
}

// newTerminalReader creates a reader for the given file descriptor.
func newTerminalReader(fd int) *terminalReader {
	return &terminalReader{fd: fd}
}

func (r *terminalReader) ReadPassword() (string, error) {
	password, err := term.ReadPassword(r.fd)
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func (r *terminalReader) IsTerminal() bool {
	return term.IsTerminal(r.fd)
}

// uuidRegex matches standard UUID format
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// configureOptions holds dependencies for the configure command.
// This allows for dependency injection in tests.
type configureOptions struct {
	configPath     string
	baseURL        string
	store          keyring.Store
	passwordReader passwordReader
}

// newConfigureCmd creates the configure command with the given options.
func newConfigureCmd(opts configureOptions) *cobra.Command {
	var accountUUID string

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure CLI credentials",
		Long: `Configure the CLI with your Public.com API credentials.

You will be prompted to enter your secret key securely.
Get your secret key from: https://public.com/settings/security/api

Example:
  pub configure
  pub configure --account YOUR_ACCOUNT_UUID`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigure(cmd, opts, accountUUID)
		},
	}

	cmd.Flags().StringVar(&accountUUID, "account", "", "Default account UUID (optional)")

	return cmd
}

func runConfigure(cmd *cobra.Command, opts configureOptions, accountUUID string) error {
	// Verify we're running in an interactive terminal
	if !opts.passwordReader.IsTerminal() {
		return fmt.Errorf("configure requires an interactive terminal\nRun this command directly in your terminal (not piped or in a script)")
	}

	// Validate account UUID format if provided
	if accountUUID != "" && !uuidRegex.MatchString(accountUUID) {
		return fmt.Errorf("invalid account UUID format")
	}

	// Prompt for secret key
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "Enter your secret key: ")
	secretKey, err := opts.passwordReader.ReadPassword()
	if err != nil {
		return fmt.Errorf("failed to read secret key: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout()) // Print newline after hidden input

	// Validate secret key is not empty
	if secretKey == "" {
		return fmt.Errorf("secret key cannot be empty")
	}

	// Validate secret key by exchanging for token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = auth.ExchangeToken(ctx, opts.baseURL, secretKey)
	if err != nil {
		return fmt.Errorf("failed to validate secret key: %w", err)
	}

	// Store secret in keyring
	if err := opts.store.Set(keyring.ServiceName, keyring.KeySecretKey, secretKey); err != nil {
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
		configPath:     config.ConfigPath(),
		baseURL:        config.DefaultAPIBaseURL,
		store:          keyring.NewEnvStore(keyring.NewSystemStore()),
		passwordReader: newTerminalReader(int(os.Stdin.Fd())),
	})
	rootCmd.AddCommand(configureCmd)
}
