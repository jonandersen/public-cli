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

// OptionChainRequest represents a request for an option chain.
type OptionChainRequest struct {
	Instrument     OptionInstrument `json:"instrument"`
	ExpirationDate string           `json:"expirationDate"`
}

// OptionChainResponse represents the API response for an option chain.
type OptionChainResponse struct {
	BaseSymbol string        `json:"baseSymbol"`
	Calls      []OptionQuote `json:"calls"`
	Puts       []OptionQuote `json:"puts"`
}

// OptionQuote represents a single option quote in the chain.
type OptionQuote struct {
	Instrument   OptionInstrument `json:"instrument"`
	Outcome      string           `json:"outcome"`
	Last         string           `json:"last"`
	Bid          string           `json:"bid"`
	BidSize      int              `json:"bidSize"`
	Ask          string           `json:"ask"`
	AskSize      int              `json:"askSize"`
	Volume       int              `json:"volume"`
	OpenInterest int              `json:"openInterest"`
}

// GreeksResponse represents the API response for option greeks.
type GreeksResponse struct {
	Greeks []OptionGreeks `json:"greeks"`
}

// OptionGreeks represents greeks for a single option.
type OptionGreeks struct {
	Symbol string      `json:"symbol"`
	Greeks GreeksData  `json:"greeks"`
}

// GreeksData contains the actual greek values.
type GreeksData struct {
	Delta             string `json:"delta"`
	Gamma             string `json:"gamma"`
	Theta             string `json:"theta"`
	Vega              string `json:"vega"`
	Rho               string `json:"rho"`
	ImpliedVolatility string `json:"impliedVolatility"`
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

// newOptionsChainCmd creates the options chain command with the given options.
func newOptionsChainCmd(opts optionsOptions) *cobra.Command {
	var expiration string

	cmd := &cobra.Command{
		Use:   "chain SYMBOL",
		Short: "Display option chain",
		Long: `Display the option chain for an underlying symbol and expiration date.

Examples:
  pub options chain AAPL --expiration 2025-01-17        # Show chain for date
  pub options chain AAPL --expiration 2025-01-17 --json # Output in JSON format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			if expiration == "" {
				return fmt.Errorf("expiration date is required (use --expiration flag)")
			}
			return runOptionsChain(cmd, opts, args[0], expiration)
		},
	}

	cmd.Flags().StringVarP(&expiration, "expiration", "e", "", "Expiration date (YYYY-MM-DD)")
	cmd.SilenceUsage = true

	return cmd
}

func runOptionsChain(cmd *cobra.Command, opts optionsOptions, symbol, expiration string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build request
	reqBody := OptionChainRequest{
		Instrument: OptionInstrument{
			Symbol: strings.ToUpper(symbol),
			Type:   "EQUITY",
		},
		ExpirationDate: expiration,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/marketdata/%s/option-chain", opts.accountID)
	resp, err := client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to fetch option chain: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var chainResp OptionChainResponse
	if err := json.NewDecoder(resp.Body).Decode(&chainResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chainResp.Calls) == 0 && len(chainResp.Puts) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No options available for %s expiring %s\n", chainResp.BaseSymbol, expiration)
		return nil
	}

	// Format output
	if opts.jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(chainResp)
	}

	// Table output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Option Chain for %s - Expiration: %s\n\n", chainResp.BaseSymbol, expiration)

	if len(chainResp.Calls) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CALLS\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "Strike", "Bid", "Ask", "Volume", "OI")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "------", "------", "------", "------", "------")
		for _, call := range chainResp.Calls {
			strike := parseStrikeFromSymbol(call.Instrument.Symbol)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10d  %10d\n",
				strike, call.Bid, call.Ask, call.Volume, call.OpenInterest)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n")
	}

	if len(chainResp.Puts) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PUTS\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "Strike", "Bid", "Ask", "Volume", "OI")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "------", "------", "------", "------", "------")
		for _, put := range chainResp.Puts {
			strike := parseStrikeFromSymbol(put.Instrument.Symbol)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10d  %10d\n",
				strike, put.Bid, put.Ask, put.Volume, put.OpenInterest)
		}
	}

	return nil
}

// parseStrikeFromSymbol extracts the strike price from an OSI option symbol.
// Example: AAPL250117C00175000 -> 175.00
func parseStrikeFromSymbol(symbol string) string {
	if len(symbol) < 8 {
		return symbol
	}
	// Last 8 characters are the strike price in format: SSSSSSSS (8 digits, price * 1000)
	strikeStr := symbol[len(symbol)-8:]
	// Parse as integer and convert to decimal
	var strike int64
	if _, err := fmt.Sscanf(strikeStr, "%d", &strike); err != nil {
		return symbol
	}
	dollars := strike / 1000
	cents := (strike % 1000) / 10
	if cents == 0 {
		return fmt.Sprintf("%d", dollars)
	}
	return fmt.Sprintf("%d.%02d", dollars, cents)
}

func runOptionsGreeks(cmd *cobra.Command, opts optionsOptions, symbols []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build query parameters
	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/option-details/%s/greeks", opts.accountID)

	// Add symbols as query params
	query := "?"
	for i, sym := range symbols {
		if i > 0 {
			query += "&"
		}
		query += "osiSymbols=" + strings.ToUpper(sym)
	}

	resp, err := client.Get(ctx, path+query)
	if err != nil {
		return fmt.Errorf("failed to fetch greeks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var greeksResp GreeksResponse
	if err := json.NewDecoder(resp.Body).Decode(&greeksResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(greeksResp.Greeks) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No greeks data available")
		return nil
	}

	// Format output
	if opts.jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(greeksResp)
	}

	// Table output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%-22s  %8s  %8s  %8s  %8s  %8s  %8s\n",
		"SYMBOL", "DELTA", "GAMMA", "THETA", "VEGA", "RHO", "IV")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Repeat("-", 85))

	for _, og := range greeksResp.Greeks {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-22s  %8s  %8s  %8s  %8s  %8s  %8s\n",
			og.Symbol,
			og.Greeks.Delta,
			og.Greeks.Gamma,
			og.Greeks.Theta,
			og.Greeks.Vega,
			og.Greeks.Rho,
			og.Greeks.ImpliedVolatility)
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

	var chainAccountID string
	var chainExpiration string
	chainCmd := &cobra.Command{
		Use:   "chain SYMBOL",
		Short: "Display option chain",
		Long: `Display the option chain for an underlying symbol and expiration date.

Examples:
  pub options chain AAPL --expiration 2025-01-17        # Show chain for date
  pub options chain AAPL --expiration 2025-01-17 --json # Output in JSON format`,
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
			if chainAccountID == "" {
				chainAccountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = chainAccountID
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			if chainExpiration == "" {
				return fmt.Errorf("expiration date is required (use --expiration flag)")
			}
			return runOptionsChain(cmd, opts, args[0], chainExpiration)
		},
	}

	chainCmd.Flags().StringVarP(&chainAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	chainCmd.Flags().StringVarP(&chainExpiration, "expiration", "e", "", "Expiration date (YYYY-MM-DD)")
	chainCmd.SilenceUsage = true

	var greeksAccountID string
	greeksCmd := &cobra.Command{
		Use:   "greeks SYMBOL [SYMBOL...]",
		Short: "Display option greeks",
		Long: `Display greeks (delta, gamma, theta, vega, rho, IV) for option symbols.

Symbols should be in OSI format (e.g., AAPL250117C00175000).

Examples:
  pub options greeks AAPL250117C00175000                    # Single option
  pub options greeks AAPL250117C00175000 AAPL250117P00175000  # Multiple options
  pub options greeks AAPL250117C00175000 --json             # Output as JSON`,
		Args: cobra.MinimumNArgs(1),
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
			if greeksAccountID == "" {
				greeksAccountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = greeksAccountID
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			return runOptionsGreeks(cmd, opts, args)
		},
	}

	greeksCmd.Flags().StringVarP(&greeksAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	greeksCmd.SilenceUsage = true

	optionsCmd.AddCommand(expirationsCmd)
	optionsCmd.AddCommand(chainCmd)
	optionsCmd.AddCommand(greeksCmd)
	rootCmd.AddCommand(optionsCmd)
}
