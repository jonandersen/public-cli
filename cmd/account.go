package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/auth"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
	"github.com/jonandersen/pub/internal/output"
)

// accountOptions holds dependencies for the account command.
type accountOptions struct {
	baseURL          string
	authToken        string
	jsonMode         bool
	defaultAccountID string
}

// Account represents a Public.com account.
type Account struct {
	AccountID            string `json:"accountId"`
	AccountType          string `json:"accountType"`
	OptionsLevel         string `json:"optionsLevel"`
	BrokerageAccountType string `json:"brokerageAccountType"`
	TradePermissions     string `json:"tradePermissions"`
}

// AccountsResponse represents the API response for listing accounts.
type AccountsResponse struct {
	Accounts []Account `json:"accounts"`
}

// Portfolio represents a portfolio response.
type Portfolio struct {
	AccountID   string      `json:"accountId"`
	AccountType string      `json:"accountType"`
	BuyingPower BuyingPower `json:"buyingPower"`
	Equity      []Equity    `json:"equity"`
	Positions   []Position  `json:"positions"`
}

// BuyingPower represents buying power information.
type BuyingPower struct {
	CashOnlyBuyingPower string `json:"cashOnlyBuyingPower"`
	BuyingPower         string `json:"buyingPower"`
	OptionsBuyingPower  string `json:"optionsBuyingPower"`
}

// Equity represents an equity breakdown item.
type Equity struct {
	Type                  string `json:"type"`
	Value                 string `json:"value"`
	PercentageOfPortfolio string `json:"percentageOfPortfolio"`
}

// Position represents a portfolio position.
type Position struct {
	Instrument         Instrument `json:"instrument"`
	Quantity           string     `json:"quantity"`
	CurrentValue       string     `json:"currentValue"`
	PercentOfPortfolio string     `json:"percentOfPortfolio"`
	LastPrice          Price      `json:"lastPrice"`
	InstrumentGain     Gain       `json:"instrumentGain"`
	PositionDailyGain  Gain       `json:"positionDailyGain"`
	CostBasis          CostBasis  `json:"costBasis"`
}

// Instrument represents a trading instrument.
type Instrument struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// Price represents a price with timestamp.
type Price struct {
	LastPrice string `json:"lastPrice"`
	Timestamp string `json:"timestamp"`
}

// Gain represents a gain/loss value with percentage.
type Gain struct {
	GainValue      string `json:"gainValue"`
	GainPercentage string `json:"gainPercentage"`
	Timestamp      string `json:"timestamp"`
}

// CostBasis represents cost basis information.
type CostBasis struct {
	TotalCost      string `json:"totalCost"`
	UnitCost       string `json:"unitCost"`
	GainValue      string `json:"gainValue"`
	GainPercentage string `json:"gainPercentage"`
	LastUpdate     string `json:"lastUpdate"`
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

	client := api.NewClient(opts.baseURL, opts.authToken)
	resp, err := client.Get(ctx, "/userapigateway/trading/account")
	if err != nil {
		return fmt.Errorf("failed to fetch accounts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var accountsResp AccountsResponse
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

	client := api.NewClient(opts.baseURL, opts.authToken)
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

	var portfolio Portfolio
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
		rows = append(rows, []string{
			pos.Instrument.Symbol,
			pos.Quantity,
			"$" + pos.CurrentValue,
			formatGainLoss(pos.PositionDailyGain.GainValue),
			pos.PositionDailyGain.GainPercentage + "%",
			formatGainLoss(pos.InstrumentGain.GainValue),
			pos.InstrumentGain.GainPercentage + "%",
		})
	}

	return formatter.Table(headers, rows)
}

// formatGainLoss formats a gain/loss value with a + prefix for positive values.
func formatGainLoss(value string) string {
	if value == "" || value == "0" || value == "0.00" {
		return "$0.00"
	}
	if value[0] != '-' {
		return "+$" + value
	}
	return "-$" + value[1:]
}

// getAuthToken retrieves the secret key and exchanges it for an access token.
func getAuthToken(store keyring.Store, baseURL string) (string, error) {
	secret, err := store.Get(keyring.ServiceName, keyring.KeySecretKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("CLI not configured. Run: pub configure\nOr set PUB_SECRET_KEY environment variable")
		}
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := auth.ExchangeToken(ctx, baseURL, secret)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return token.AccessToken, nil
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
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.jsonMode = GetJSONMode()
			opts.defaultAccountID = cfg.AccountUUID
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
