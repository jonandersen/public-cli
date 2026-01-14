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

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
	"github.com/jonandersen/public-cli/internal/output"
)

// quoteOptions holds dependencies for the quote command.
type quoteOptions struct {
	baseURL   string
	authToken string
	accountID string
	jsonMode  bool
}

// newQuoteCmd creates the quote command with the given options.
func newQuoteCmd(opts quoteOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote SYMBOL [SYMBOL...]",
		Short: "Get stock quotes",
		Long: `Get real-time quotes for one or more stock symbols.

Examples:
  pub quote AAPL              # Get quote for Apple
  pub quote AAPL GOOGL MSFT   # Get quotes for multiple symbols
  pub quote AAPL --json       # Output in JSON format`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one symbol is required")
			}
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			return runQuote(cmd, opts, args)
		},
	}

	cmd.SilenceUsage = true

	return cmd
}

func runQuote(cmd *cobra.Command, opts quoteOptions, symbols []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build request
	instruments := make([]api.QuoteInstrument, 0, len(symbols))
	for _, sym := range symbols {
		instruments = append(instruments, api.QuoteInstrument{
			Symbol: strings.ToUpper(sym),
			Type:   "EQUITY",
		})
	}

	reqBody := api.QuoteRequest{Instruments: instruments}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/marketdata/%s/quotes", opts.accountID)
	resp, err := client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to fetch quotes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var quotesResp api.QuotesResponse
	if err := json.NewDecoder(resp.Body).Decode(&quotesResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(quotesResp.Quotes) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No quotes returned")
		return nil
	}

	// Format output
	formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)
	headers := []string{"Symbol", "Last", "Bid", "Ask", "Volume"}
	rows := make([][]string, 0, len(quotesResp.Quotes))

	for _, q := range quotesResp.Quotes {
		if q.Outcome != "SUCCESS" {
			rows = append(rows, []string{
				q.Instrument.Symbol,
				q.Outcome,
				"-",
				"-",
				"-",
			})
			continue
		}
		rows = append(rows, []string{
			q.Instrument.Symbol,
			q.Last,
			q.Bid,
			q.Ask,
			formatVolume(q.Volume),
		})
	}

	return formatter.Table(headers, rows)
}

// formatVolume formats volume with thousand separators.
func formatVolume(vol int64) string {
	if vol == 0 {
		return "-"
	}
	s := fmt.Sprintf("%d", vol)
	// Add thousand separators
	n := len(s)
	if n <= 3 {
		return s
	}
	// Calculate how many commas we need
	numCommas := (n - 1) / 3
	result := make([]byte, n+numCommas)
	for i, j := n-1, len(result)-1; i >= 0; i-- {
		result[j] = s[i]
		j--
		if (n-i)%3 == 0 && i > 0 {
			result[j] = ','
			j--
		}
	}
	return string(result)
}

func init() {
	var opts quoteOptions
	var accountID string

	quoteCmd := &cobra.Command{
		Use:   "quote SYMBOL [SYMBOL...]",
		Short: "Get stock quotes",
		Long: `Get real-time quotes for one or more stock symbols.

Examples:
  pub quote AAPL              # Get quote for Apple
  pub quote AAPL GOOGL MSFT   # Get quotes for multiple symbols
  pub quote AAPL --json       # Output in JSON format`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Get auth token
			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
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
			if len(args) == 0 {
				return fmt.Errorf("at least one symbol is required")
			}
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			return runQuote(cmd, opts, args)
		},
	}

	quoteCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	quoteCmd.SilenceUsage = true

	rootCmd.AddCommand(quoteCmd)
}
