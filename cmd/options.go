package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// optionsOptions holds dependencies for options commands.
type optionsOptions struct {
	baseURL   string
	authToken string
	accountID string
	jsonMode  bool
}

// OptionExpirationsRequest represents a request for option expirations.
type OptionExpirationsRequest struct {
	Instrument OptionInstrument `json:"instrument"`
}

// OptionInstrument represents an instrument for options requests.
type OptionInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// OptionExpirationsResponse represents the API response for option expirations.
type OptionExpirationsResponse struct {
	BaseSymbol  string   `json:"baseSymbol"`
	Expirations []string `json:"expirations"`
}

// newOptionsExpirationsCmd creates the options expirations command with the given options.
func newOptionsExpirationsCmd(opts optionsOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "expirations SYMBOL",
		Short: "List option expiration dates",
		Long: `List available option expiration dates for an underlying symbol.

Examples:
  pub options expirations AAPL           # List expirations for Apple
  pub options expirations AAPL --json    # Output in JSON format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			return runOptionsExpirations(cmd, opts, args[0])
		},
	}

	cmd.SilenceUsage = true

	return cmd
}

func runOptionsExpirations(cmd *cobra.Command, opts optionsOptions, symbol string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build request
	reqBody := OptionExpirationsRequest{
		Instrument: OptionInstrument{
			Symbol: strings.ToUpper(symbol),
			Type:   "EQUITY",
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/marketdata/%s/option-expirations", opts.accountID)
	resp, err := client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to fetch expirations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var expResp OptionExpirationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&expResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(expResp.Expirations) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No expirations available for %s\n", expResp.BaseSymbol)
		return nil
	}

	// Format output
	if opts.jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(expResp)
	}

	// Table output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Option Expirations for %s\n\n", expResp.BaseSymbol)
	for _, exp := range expResp.Expirations {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", exp)
	}

	return nil
}

func init() {
	var opts optionsOptions
	var accountID string

	optionsCmd := &cobra.Command{
		Use:   "options",
		Short: "Options trading commands",
		Long:  `Commands for options trading including expirations and chains.`,
	}

	expirationsCmd := &cobra.Command{
		Use:   "expirations SYMBOL",
		Short: "List option expiration dates",
		Long: `List available option expiration dates for an underlying symbol.

Examples:
  pub options expirations AAPL           # List expirations for Apple
  pub options expirations AAPL --json    # Output in JSON format`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Get auth token
			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			// Use flag value or default from config
			if accountID == "" {
				accountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = accountID
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			return runOptionsExpirations(cmd, opts, args[0])
		},
	}

	expirationsCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	expirationsCmd.SilenceUsage = true

	optionsCmd.AddCommand(expirationsCmd)
	rootCmd.AddCommand(optionsCmd)
}
