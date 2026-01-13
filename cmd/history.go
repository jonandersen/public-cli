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
)

// historyOptions holds dependencies for the history command.
type historyOptions struct {
	baseURL          string
	authToken        string
	jsonMode         bool
	defaultAccountID string
}

// Transaction represents a single transaction in account history.
type Transaction struct {
	ID              string `json:"id"`
	Timestamp       string `json:"timestamp"`
	Type            string `json:"type"`
	SubType         string `json:"subType"`
	AccountNumber   string `json:"accountNumber"`
	Symbol          string `json:"symbol"`
	SecurityType    string `json:"securityType"`
	Side            string `json:"side"`
	Description     string `json:"description"`
	NetAmount       string `json:"netAmount"`
	PrincipalAmount string `json:"principalAmount"`
	Quantity        string `json:"quantity"`
	Direction       string `json:"direction"`
	Fees            string `json:"fees"`
}

// HistoryResponse represents the API response for account history.
type HistoryResponse struct {
	Transactions []Transaction `json:"transactions"`
	NextToken    string        `json:"nextToken"`
	Start        string        `json:"start"`
	End          string        `json:"end"`
	PageSize     int           `json:"pageSize"`
}

// newHistoryCmd creates the history command with the given options.
func newHistoryCmd(opts historyOptions) *cobra.Command {
	var (
		flagAccountID string
		flagStart     string
		flagEnd       string
		flagLimit     int
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "View account transaction history",
		Long: `View your account transaction history including trades, deposits, withdrawals, and dividends.

Uses the default account from config if --account is not specified.

Examples:
  pub history                                    # Use default account
  pub history --account YOUR_ACCOUNT_ID          # Specific account
  pub history --start 2025-01-01T00:00:00Z       # Filter by start date
  pub history --limit 10                         # Limit results`,
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := flagAccountID
			if accountID == "" {
				accountID = opts.defaultAccountID
			}
			if accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or set default with 'pub configure')")
			}
			return runHistory(cmd, opts, accountID, flagStart, flagEnd, flagLimit)
		},
	}

	cmd.Flags().StringVarP(&flagAccountID, "account", "a", "", "Account ID (uses default if configured)")
	cmd.Flags().StringVar(&flagStart, "start", "", "Start timestamp (ISO 8601 format, e.g., 2025-01-01T00:00:00Z)")
	cmd.Flags().StringVar(&flagEnd, "end", "", "End timestamp (ISO 8601 format, e.g., 2025-01-31T23:59:59Z)")
	cmd.Flags().IntVarP(&flagLimit, "limit", "l", 0, "Maximum number of transactions to return")
	cmd.SilenceUsage = true

	return cmd
}

func runHistory(cmd *cobra.Command, opts historyOptions, accountID, start, end string, limit int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/history", accountID)

	// Build query parameters
	queryParams := make(map[string]string)
	if start != "" {
		queryParams["start"] = start
	}
	if end != "" {
		queryParams["end"] = end
	}
	if limit > 0 {
		queryParams["pageSize"] = fmt.Sprintf("%d", limit)
	}

	resp, err := client.GetWithParams(ctx, path, queryParams)
	if err != nil {
		return fmt.Errorf("failed to fetch history: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var historyResp HistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(historyResp.Transactions) == 0 {
		if opts.jsonMode {
			formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)
			return formatter.Print(map[string]any{
				"transactions": []any{},
			})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No transactions found")
		return nil
	}

	formatter := output.New(cmd.OutOrStdout(), opts.jsonMode)

	if opts.jsonMode {
		return formatter.Print(map[string]any{
			"transactions": historyResp.Transactions,
			"nextToken":    historyResp.NextToken,
			"pageSize":     historyResp.PageSize,
		})
	}

	// Format as table
	headers := []string{"ID", "Date", "Type", "Symbol", "Description", "Amount"}
	rows := make([][]string, 0, len(historyResp.Transactions))
	for _, txn := range historyResp.Transactions {
		// Format timestamp to just the date portion for readability
		date := formatTransactionDate(txn.Timestamp)
		txnType := txn.Type
		if txn.SubType != "" {
			txnType = txn.SubType
		}
		rows = append(rows, []string{
			txn.ID,
			date,
			txnType,
			txn.Symbol,
			truncateString(txn.Description, 30),
			formatAmount(txn.NetAmount),
		})
	}

	return formatter.Table(headers, rows)
}

// formatTransactionDate formats an ISO timestamp to a readable date.
func formatTransactionDate(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return t.Format("2006-01-02")
}

// formatAmount formats an amount string with currency symbol.
func formatAmount(amount string) string {
	if amount == "" {
		return "$0.00"
	}
	if amount[0] == '-' {
		return "-$" + amount[1:]
	}
	return "$" + amount
}

// truncateString truncates a string to maxLen characters with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	var opts historyOptions

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "View account transaction history",
		Long: `View your account transaction history including trades, deposits, withdrawals, and dividends.

Uses the default account from config if --account is not specified.

Examples:
  pub history                                    # Use default account
  pub history --account YOUR_ACCOUNT_ID          # Specific account
  pub history --start 2025-01-01T00:00:00Z       # Filter by start date
  pub history --limit 10                         # Limit results`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
			if err != nil {
				return err
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.jsonMode = GetJSONMode()
			opts.defaultAccountID = cfg.AccountUUID
			return nil
		},
	}

	historyCmd.SilenceUsage = true

	var (
		flagAccountID string
		flagStart     string
		flagEnd       string
		flagLimit     int
	)

	historyCmd.Flags().StringVarP(&flagAccountID, "account", "a", "", "Account ID (uses default if configured)")
	historyCmd.Flags().StringVar(&flagStart, "start", "", "Start timestamp (ISO 8601 format, e.g., 2025-01-01T00:00:00Z)")
	historyCmd.Flags().StringVar(&flagEnd, "end", "", "End timestamp (ISO 8601 format, e.g., 2025-01-31T23:59:59Z)")
	historyCmd.Flags().IntVarP(&flagLimit, "limit", "l", 0, "Maximum number of transactions to return")

	historyCmd.RunE = func(cmd *cobra.Command, args []string) error {
		accountID := flagAccountID
		if accountID == "" {
			accountID = opts.defaultAccountID
		}
		if accountID == "" {
			return fmt.Errorf("account ID is required (use --account flag or set default with 'pub configure')")
		}
		return runHistory(cmd, opts, accountID, flagStart, flagEnd, flagLimit)
	}

	rootCmd.AddCommand(historyCmd)
}
