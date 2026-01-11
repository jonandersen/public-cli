package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jonandersen/pub/internal/api"
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

// prompter abstracts interactive menu selection for testing.
type prompter interface {
	SelectOption(options []string) (int, error)
	ReadLine(prompt string) (string, error)
}

// terminalPrompter implements prompter using stdin.
type terminalPrompter struct {
	reader io.Reader
	writer io.Writer
}

func newTerminalPrompter(r io.Reader, w io.Writer) *terminalPrompter {
	return &terminalPrompter{reader: r, writer: w}
}

func (p *terminalPrompter) SelectOption(options []string) (int, error) {
	scanner := bufio.NewScanner(p.reader)
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return 0, err
			}
			return 0, fmt.Errorf("no input")
		}
		input := strings.TrimSpace(scanner.Text())
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(options) {
			_, _ = fmt.Fprintf(p.writer, "Please enter a number between 1 and %d: ", len(options))
			continue
		}
		return idx - 1, nil // Convert to 0-indexed
	}
}

func (p *terminalPrompter) ReadLine(prompt string) (string, error) {
	_, _ = fmt.Fprint(p.writer, prompt)
	scanner := bufio.NewScanner(p.reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return strings.TrimSpace(scanner.Text()), nil
}

// configureOptions holds dependencies for the configure command.
// This allows for dependency injection in tests.
type configureOptions struct {
	configPath     string
	baseURL        string
	store          keyring.Store
	passwordReader passwordReader
	prompt         prompter
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

	// Don't show usage info on validation errors - just show the error
	cmd.SilenceUsage = true

	return cmd
}

// reconfigureMenuOptions defines the menu options when already configured.
var reconfigureMenuOptions = []string{
	"Select different default account",
	"Configure new secret key",
	"View current configuration",
	"Clear secret key",
}

func runConfigure(cmd *cobra.Command, opts configureOptions, accountUUID string) error {
	// Verify we're running in an interactive terminal
	if !opts.passwordReader.IsTerminal() {
		return fmt.Errorf("configure requires an interactive terminal\nRun this command directly in your terminal (not piped or in a script)")
	}

	// Validate account UUID format if provided via flag
	if accountUUID != "" && !uuidRegex.MatchString(accountUUID) {
		return fmt.Errorf("invalid account UUID format")
	}

	// Check if already configured
	_, err := opts.store.Get(keyring.ServiceName, keyring.KeySecretKey)
	alreadyConfigured := err == nil

	if alreadyConfigured {
		return runReconfigureMenu(cmd, opts)
	}

	return runInitialSetup(cmd, opts, accountUUID)
}

// runReconfigureMenu shows the reconfigure menu when already configured.
func runReconfigureMenu(cmd *cobra.Command, opts configureOptions) error {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "CLI is already configured. What would you like to do?")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())

	for i, opt := range reconfigureMenuOptions {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s\n", i+1, opt)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "Select option: ")

	choice, err := opts.prompt.SelectOption(reconfigureMenuOptions)
	if err != nil {
		return fmt.Errorf("failed to read selection: %w", err)
	}

	switch choice {
	case 0: // Select different default account
		return runSelectAccount(cmd, opts)
	case 1: // Configure new secret key
		return runInitialSetup(cmd, opts, "")
	case 2: // View current configuration
		return runViewConfiguration(cmd, opts)
	case 3: // Clear secret key
		return runClearSecret(cmd, opts)
	default:
		return fmt.Errorf("invalid selection")
	}
}

// runInitialSetup handles the initial secret key configuration.
func runInitialSetup(cmd *cobra.Command, opts configureOptions, accountUUID string) error {
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

	token, err := auth.ExchangeToken(ctx, opts.baseURL, secretKey)
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

	// Update config with provided values from flag
	if accountUUID != "" {
		cfg.AccountUUID = accountUUID
	} else {
		// Offer account selection if no account was provided via flag
		selectedAccount, err := promptAccountSelection(cmd, opts, token.AccessToken)
		if err != nil {
			// Non-fatal: just skip account selection
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Note: Could not fetch accounts: %v\n", err)
		} else if selectedAccount != "" {
			cfg.AccountUUID = selectedAccount
		}
	}

	// Save config
	if err := config.Save(opts.configPath, cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Configuration saved successfully!")
	return nil
}

// promptAccountSelection fetches accounts and prompts user to select one.
func promptAccountSelection(cmd *cobra.Command, opts configureOptions, accessToken string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, accessToken)
	resp, err := client.Get(ctx, "/userapigateway/trading/account")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var accountsResp AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResp); err != nil {
		return "", err
	}

	if len(accountsResp.Accounts) == 0 {
		return "", nil
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Select a default account:")

	options := make([]string, 0, len(accountsResp.Accounts)+1)
	for i, acc := range accountsResp.Accounts {
		optionText := fmt.Sprintf("%s (%s)", acc.AccountID, acc.AccountType)
		options = append(options, optionText)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s\n", i+1, optionText)
	}
	options = append(options, "Skip")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d. Skip\n", len(accountsResp.Accounts)+1)
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "Select account: ")

	choice, err := opts.prompt.SelectOption(options)
	if err != nil {
		return "", err
	}

	// If "Skip" was selected
	if choice >= len(accountsResp.Accounts) {
		return "", nil
	}

	return accountsResp.Accounts[choice].AccountID, nil
}

// runSelectAccount handles selecting a different default account.
func runSelectAccount(cmd *cobra.Command, opts configureOptions) error {
	// Get existing secret to authenticate
	secret, err := opts.store.Get(keyring.ServiceName, keyring.KeySecretKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := auth.ExchangeToken(ctx, opts.baseURL, secret)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	selectedAccount, err := promptAccountSelection(cmd, opts, token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to select account: %w", err)
	}

	if selectedAccount == "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No account selected.")
		return nil
	}

	// Load existing config
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	cfg.AccountUUID = selectedAccount

	if err := config.Save(opts.configPath, cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default account set to: %s\n", selectedAccount)
	return nil
}

// runViewConfiguration displays the current configuration.
func runViewConfiguration(cmd *cobra.Command, opts configureOptions) error {
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Current Configuration:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "----------------------")

	// Check if secret is configured
	_, err = opts.store.Get(keyring.ServiceName, keyring.KeySecretKey)
	if err == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Secret key: Configured")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Secret key: Not configured")
	}

	if cfg.AccountUUID != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default account: %s\n", cfg.AccountUUID)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Default account: Not set")
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "API base URL: %s\n", cfg.APIBaseURL)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Token validity: %d minutes\n", cfg.TokenValidityMinutes)

	return nil
}

// runClearSecret removes the stored secret key.
func runClearSecret(cmd *cobra.Command, opts configureOptions) error {
	if err := opts.store.Delete(keyring.ServiceName, keyring.KeySecretKey); err != nil {
		return fmt.Errorf("failed to clear secret: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Secret key cleared successfully.")
	return nil
}

func init() {
	// Create configure command with production dependencies
	configureCmd := newConfigureCmd(configureOptions{
		configPath:     config.ConfigPath(),
		baseURL:        config.DefaultAPIBaseURL,
		store:          keyring.NewEnvStore(keyring.NewSystemStore()),
		passwordReader: newTerminalReader(int(os.Stdin.Fd())),
		prompt:         newTerminalPrompter(os.Stdin, os.Stdout),
	})
	rootCmd.AddCommand(configureCmd)
}
