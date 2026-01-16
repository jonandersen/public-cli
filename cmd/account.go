package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
	"github.com/jonandersen/public-cli/internal/output"
	"github.com/jonandersen/public-cli/pkg/publicapi"
)

// accountOptions holds dependencies for the account command.
type accountOptions struct {
	baseURL          string
	authToken        string
	jsonMode         bool
	defaultAccountID string
	tokenRefresher   api.TokenRefresher
}

// newAccountCmd creates the account command with the given options.
func newAccountCmd(opts accountOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "View account information",
		Long: `View your Public.com accounts, portfolio, and positions.

Examples:
  pub account              # List all accounts
  pub account portfolio    # View portfolio (requires --account or default account)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAccountList(cmd, opts)
		},
	}

	cmd.SilenceUsage = true

	// Add portfolio subcommand
	portfolioCmd := newPortfolioCmd(opts)
	cmd.AddCommand(portfolioCmd)

	return cmd
}

func runAccountList(cmd *cobra.Command, opts accountOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken).WithTokenRefresher(opts.tokenRefresher)
	resp, err := client.Get(ctx, "/userapigateway/trading/account")
	if err != nil {
		return fmt.Errorf("failed to fetch accounts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var accountsResp api.AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(accountsResp.Accounts) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No accounts found")
		return nil
	}

	// Format output
	formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)
	headers := []string{"Account ID", "Type", "Options Level", "Margin", "Permissions"}
	rows := make([][]string, 0, len(accountsResp.Accounts))
	for _, acc := range accountsResp.Accounts {
		rows = append(rows, []string{
			acc.AccountID,
			acc.AccountType,
			acc.OptionsLevel,
			acc.BrokerageAccountType,
			acc.TradePermissions,
		})
	}

	return formatter.Table(headers, rows)
}

func newPortfolioCmd(opts accountOptions) *cobra.Command {
	var flagAccountID string

	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "View portfolio positions and balances",
		Long: `View your portfolio including buying power, positions, and daily gains.

Uses the default account from config if --account is not specified.

Examples:
  pub account portfolio                          # Use default account
  pub account portfolio --account YOUR_ACCOUNT_ID`,
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := flagAccountID
			if accountID == "" {
				accountID = opts.defaultAccountID
			}
			if accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or set default with 'pub configure')")
			}
			return runPortfolio(cmd, opts, accountID)
		},
	}

	cmd.Flags().StringVarP(&flagAccountID, "account", "a", "", "Account ID (uses default if configured)")
	cmd.SilenceUsage = true

	return cmd
}

func runPortfolio(cmd *cobra.Command, opts accountOptions, accountID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken).WithTokenRefresher(opts.tokenRefresher)
	path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", accountID)
	resp, err := client.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to fetch portfolio: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var portfolio api.Portfolio
	if err := json.NewDecoder(resp.Body).Decode(&portfolio); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)

	// Print buying power summary
	if !opts.jsonMode {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Buying Power: $%s\n", portfolio.BuyingPower.BuyingPower)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Options Buying Power: $%s\n\n", portfolio.BuyingPower.OptionsBuyingPower)

		// Print equity summary if available
		if len(portfolio.Equity) > 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Account Summary:")
			for _, eq := range portfolio.Equity {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: $%s (%s%%)\n", eq.Type, eq.Value, eq.PercentageOfPortfolio)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	if len(portfolio.Positions) == 0 {
		if opts.jsonMode {
			return formatter.Print(map[string]any{
				"buyingPower": portfolio.BuyingPower,
				"equity":      portfolio.Equity,
				"positions":   []any{},
			})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No positions")
		return nil
	}

	if opts.jsonMode {
		return formatter.Print(map[string]any{
			"buyingPower": portfolio.BuyingPower,
			"equity":      portfolio.Equity,
			"positions":   portfolio.Positions,
		})
	}

	// Format positions as table
	headers := []string{"Symbol", "Qty", "Value", "Daily G/L", "Daily %", "Total G/L", "Total %"}
	rows := make([][]string, 0, len(portfolio.Positions))
	for _, pos := range portfolio.Positions {
		// Use costBasis for total gain (more accurate than instrumentGain)
		totalGainValue := pos.CostBasis.GainValue
		totalGainPct := pos.CostBasis.GainPercentage
		if totalGainValue == "" {
			totalGainValue = "0"
			totalGainPct = "0"
		}
		rows = append(rows, []string{
			pos.Instrument.Symbol,
			pos.Quantity,
			"$" + pos.CurrentValue,
			publicapi.FormatGainLoss(pos.PositionDailyGain.GainValue),
			pos.PositionDailyGain.GainPercentage + "%",
			publicapi.FormatGainLoss(totalGainValue),
			totalGainPct + "%",
		})
	}

	return formatter.Table(headers, rows)
}

func init() {
	// Create a wrapper command that handles auth lazily
	var opts accountOptions

	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "View account information",
		Long: `View your Public.com accounts, portfolio, and positions.

Examples:
  pub account              # List all accounts
  pub account portfolio    # View portfolio (requires --account or default account)`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Get auth token
			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
			if err != nil {
				return err
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.jsonMode = GetJSONMode()
			opts.defaultAccountID = cfg.AccountUUID
			// Create token refresher for 401 retry
			opts.tokenRefresher = func() (string, error) {
				return api.GetAuthToken(store, cfg.APIBaseURL, true)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAccountList(cmd, opts)
		},
	}

	accountCmd.SilenceUsage = true

	// Add portfolio subcommand
	var portfolioAccountID string
	portfolioCmd := &cobra.Command{
		Use:   "portfolio",
		Short: "View portfolio positions and balances",
		Long: `View your portfolio including buying power, positions, and daily gains.

Uses the default account from config if --account is not specified.

Examples:
  pub account portfolio                          # Use default account
  pub account portfolio --account YOUR_ACCOUNT_ID`,
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := portfolioAccountID
			if accountID == "" {
				accountID = opts.defaultAccountID
			}
			if accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or set default with 'pub configure')")
			}
			return runPortfolio(cmd, opts, accountID)
		},
	}
	portfolioCmd.Flags().StringVarP(&portfolioAccountID, "account", "a", "", "Account ID (uses default if configured)")
	portfolioCmd.SilenceUsage = true

	accountCmd.AddCommand(portfolioCmd)
	rootCmd.AddCommand(accountCmd)
}
