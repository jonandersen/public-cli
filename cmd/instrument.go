package cmd

import (
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
	"github.com/jonandersen/pub/internal/output"
)

// instrumentOptions holds dependencies for the instrument command.
type instrumentOptions struct {
	baseURL   string
	authToken string
	jsonMode  bool
}

// InstrumentIdentifier represents an instrument identifier in API responses.
type InstrumentIdentifier struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// InstrumentResponse represents the API response for instrument details.
type InstrumentResponse struct {
	Instrument          InstrumentIdentifier `json:"instrument"`
	Trading             string               `json:"trading"`
	FractionalTrading   string               `json:"fractionalTrading"`
	OptionTrading       string               `json:"optionTrading"`
	OptionSpreadTrading string               `json:"optionSpreadTrading"`
	InstrumentDetails   any                  `json:"instrumentDetails,omitempty"`
}

// newInstrumentCmd creates the instrument command with the given options.
func newInstrumentCmd(opts instrumentOptions) *cobra.Command {
	var instType string

	cmd := &cobra.Command{
		Use:   "instrument SYMBOL",
		Short: "Get instrument details",
		Long: `Get trading details for a single instrument.

Shows trading capabilities including whether the instrument supports:
- Regular trading (buy/sell)
- Fractional trading
- Options trading
- Multi-leg options trading

Examples:
  pub instrument AAPL                    # Get details for Apple stock
  pub instrument BTC --type CRYPTO       # Get details for Bitcoin
  pub instrument AAPL --json             # Output in JSON format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstrument(cmd, opts, args[0], instType)
		},
	}

	cmd.Flags().StringVarP(&instType, "type", "t", "EQUITY", "Instrument type (EQUITY, OPTION, CRYPTO, ALT, TREASURY, BOND, INDEX)")
	cmd.SilenceUsage = true

	return cmd
}

func runInstrument(cmd *cobra.Command, opts instrumentOptions, symbol, instType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	symbol = strings.ToUpper(symbol)
	instType = strings.ToUpper(instType)

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/instruments/%s/%s", symbol, instType)
	resp, err := client.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to fetch instrument: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var instResp InstrumentResponse
	if err := json.NewDecoder(resp.Body).Decode(&instResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Format output
	formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)

	if opts.jsonMode {
		return formatter.Print(instResp)
	}

	headers := []string{"Property", "Value"}
	rows := [][]string{
		{"Symbol", instResp.Instrument.Symbol},
		{"Type", instResp.Instrument.Type},
		{"Trading", instResp.Trading},
		{"Fractional Trading", instResp.FractionalTrading},
		{"Option Trading", instResp.OptionTrading},
		{"Option Spread Trading", instResp.OptionSpreadTrading},
	}

	return formatter.Table(headers, rows)
}

func init() {
	var opts instrumentOptions
	var instType string

	instrumentCmd := &cobra.Command{
		Use:   "instrument SYMBOL",
		Short: "Get instrument details",
		Long: `Get trading details for a single instrument.

Shows trading capabilities including whether the instrument supports:
- Regular trading (buy/sell)
- Fractional trading
- Options trading
- Multi-leg options trading

Examples:
  pub instrument AAPL                    # Get details for Apple stock
  pub instrument BTC --type CRYPTO       # Get details for Bitcoin
  pub instrument AAPL --json             # Output in JSON format`,
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

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstrument(cmd, opts, args[0], instType)
		},
	}

	instrumentCmd.Flags().StringVarP(&instType, "type", "t", "EQUITY", "Instrument type (EQUITY, OPTION, CRYPTO, ALT, TREASURY, BOND, INDEX)")
	instrumentCmd.SilenceUsage = true

	rootCmd.AddCommand(instrumentCmd)
}
