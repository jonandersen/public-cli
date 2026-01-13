package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
	"github.com/jonandersen/public-cli/internal/output"
)

// instrumentsOptions holds dependencies for the instruments command.
type instrumentsOptions struct {
	baseURL   string
	authToken string
	jsonMode  bool
}

// InstrumentsResponse represents the API response for listing instruments.
type InstrumentsResponse struct {
	Instruments []api.InstrumentResponse `json:"instruments"`
}

// newInstrumentsCmd creates the instruments command with the given options.
func newInstrumentsCmd(opts instrumentsOptions) *cobra.Command {
	var typeFilter string
	var tradingFilter string

	cmd := &cobra.Command{
		Use:   "instruments",
		Short: "List all available instruments",
		Long: `List all available trading instruments.

Supports filtering by instrument type and trading status.

Examples:
  pub instruments                           # List all instruments
  pub instruments --type EQUITY             # List only stocks
  pub instruments --type CRYPTO             # List only crypto
  pub instruments --trading BUY_AND_SELL    # List tradeable instruments
  pub instruments --json                    # Output in JSON format`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstruments(cmd, opts, typeFilter, tradingFilter)
		},
	}

	cmd.Flags().StringVarP(&typeFilter, "type", "t", "", "Filter by instrument type (EQUITY, OPTION, CRYPTO, ALT, TREASURY, BOND, INDEX)")
	cmd.Flags().StringVar(&tradingFilter, "trading", "", "Filter by trading status (BUY_AND_SELL, LIQUIDATION_ONLY, DISABLED)")
	cmd.SilenceUsage = true

	return cmd
}

func runInstruments(cmd *cobra.Command, opts instrumentsOptions, typeFilter, tradingFilter string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)

	// Build query parameters
	params := url.Values{}
	if typeFilter != "" {
		params.Set("typeFilter", strings.ToUpper(typeFilter))
	}
	if tradingFilter != "" {
		params.Set("tradingFilter", strings.ToUpper(tradingFilter))
	}

	path := "/userapigateway/trading/instruments"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := client.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to fetch instruments: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var instResp InstrumentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&instResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Format output
	formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)

	if opts.jsonMode {
		return formatter.Print(instResp)
	}

	if len(instResp.Instruments) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No instruments found")
		return nil
	}

	headers := []string{"Symbol", "Type", "Trading", "Fractional", "Options", "Spreads"}
	rows := make([][]string, 0, len(instResp.Instruments))
	for _, inst := range instResp.Instruments {
		rows = append(rows, []string{
			inst.Instrument.Symbol,
			inst.Instrument.Type,
			inst.Trading,
			inst.FractionalTrading,
			inst.OptionTrading,
			inst.OptionSpreadTrading,
		})
	}

	return formatter.Table(headers, rows)
}

func init() {
	var opts instrumentsOptions
	var typeFilter string
	var tradingFilter string

	instrumentsCmd := &cobra.Command{
		Use:   "instruments",
		Short: "List all available instruments",
		Long: `List all available trading instruments.

Supports filtering by instrument type and trading status.

Examples:
  pub instruments                           # List all instruments
  pub instruments --type EQUITY             # List only stocks
  pub instruments --type CRYPTO             # List only crypto
  pub instruments --trading BUY_AND_SELL    # List tradeable instruments
  pub instruments --json                    # Output in JSON format`,
		Args: cobra.NoArgs,
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

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstruments(cmd, opts, typeFilter, tradingFilter)
		},
	}

	instrumentsCmd.Flags().StringVarP(&typeFilter, "type", "t", "", "Filter by instrument type (EQUITY, OPTION, CRYPTO, ALT, TREASURY, BOND, INDEX)")
	instrumentsCmd.Flags().StringVar(&tradingFilter, "trading", "", "Filter by trading status (BUY_AND_SELL, LIQUIDATION_ONLY, DISABLED)")
	instrumentsCmd.SilenceUsage = true

	rootCmd.AddCommand(instrumentsCmd)
}
