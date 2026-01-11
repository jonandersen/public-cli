package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// orderOptions holds dependencies for the order command.
type orderOptions struct {
	baseURL        string
	authToken      string
	accountID      string
	tradingEnabled bool
	jsonMode       bool
}

// OrderRequest represents an order placement request.
type OrderRequest struct {
	OrderID    string          `json:"orderId"`
	Instrument OrderInstrument `json:"instrument"`
	OrderSide  string          `json:"orderSide"`
	OrderType  string          `json:"orderType"`
	Expiration OrderExpiration `json:"expiration"`
	Quantity   string          `json:"quantity,omitempty"`
	Amount     string          `json:"amount,omitempty"`
	LimitPrice string          `json:"limitPrice,omitempty"`
	StopPrice  string          `json:"stopPrice,omitempty"`
}

// OrderInstrument represents the instrument being traded.
type OrderInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// OrderExpiration represents order time-in-force.
type OrderExpiration struct {
	TimeInForce string `json:"timeInForce"`
}

// OrderResponse represents the API response for order placement.
type OrderResponse struct {
	OrderID string `json:"orderId"`
}

// OrderStatusResponse represents the API response for order status.
type OrderStatusResponse struct {
	OrderID        string          `json:"orderId"`
	Instrument     OrderInstrument `json:"instrument"`
	CreatedAt      string          `json:"createdAt"`
	Type           string          `json:"type"`
	Side           string          `json:"side"`
	Status         string          `json:"status"`
	Quantity       string          `json:"quantity"`
	LimitPrice     string          `json:"limitPrice,omitempty"`
	StopPrice      string          `json:"stopPrice,omitempty"`
	FilledQuantity string          `json:"filledQuantity"`
	AveragePrice   string          `json:"averagePrice,omitempty"`
	ClosedAt       string          `json:"closedAt,omitempty"`
}

// newOrderCmd creates the parent order command.
func newOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Place and manage orders",
		Long: `Place buy and sell orders for stocks and ETFs, check status, and cancel open orders.

Examples:
  pub order buy AAPL --quantity 10                              # Buy 10 shares of Apple
  pub order sell AAPL --quantity 5                              # Sell 5 shares of Apple
  pub order status 912710f1-1a45-4ef0-88a7-cd513781933d         # Check order status
  pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d --yes   # Cancel an order`,
	}

	return cmd
}

// newOrderBuyCmd creates the buy subcommand with the given options.
func newOrderBuyCmd(opts orderOptions) *cobra.Command {
	var quantity string
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "buy SYMBOL",
		Short: "Buy shares of a stock",
		Long: `Place a market buy order for shares of a stock.

Examples:
  pub order buy AAPL --quantity 10        # Buy 10 shares of Apple
  pub order buy AAPL --quantity 10 --yes  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrder(cmd, opts, args[0], "BUY", quantity, skipConfirm)
		},
	}

	cmd.Flags().StringVarP(&quantity, "quantity", "q", "", "Number of shares to buy (required)")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.SilenceUsage = true

	return cmd
}

// newOrderSellCmd creates the sell subcommand with the given options.
func newOrderSellCmd(opts orderOptions) *cobra.Command {
	var quantity string
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "sell SYMBOL",
		Short: "Sell shares of a stock",
		Long: `Place a market sell order for shares of a stock.

Examples:
  pub order sell AAPL --quantity 5        # Sell 5 shares of Apple
  pub order sell AAPL --quantity 5 --yes  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrder(cmd, opts, args[0], "SELL", quantity, skipConfirm)
		},
	}

	cmd.Flags().StringVarP(&quantity, "quantity", "q", "", "Number of shares to sell (required)")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.SilenceUsage = true

	return cmd
}

// newOrderCancelCmd creates the cancel subcommand with the given options.
func newOrderCancelCmd(opts orderOptions) *cobra.Command {
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "cancel ORDER_ID",
		Short: "Cancel an open order",
		Long: `Cancel an open order by its order ID.

Examples:
  pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d        # Cancel order (requires confirmation)
  pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d --yes  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCancelOrder(cmd, opts, args[0], skipConfirm)
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.SilenceUsage = true

	return cmd
}

// newOrderStatusCmd creates the status subcommand with the given options.
func newOrderStatusCmd(opts orderOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status ORDER_ID",
		Short: "Check the status of an order",
		Long: `Check the status of an order by its order ID.

Status values: NEW, PARTIALLY_FILLED, FILLED, CANCELLED, REJECTED, EXPIRED

Examples:
  pub order status 912710f1-1a45-4ef0-88a7-cd513781933d`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrderStatus(cmd, opts, args[0])
		},
	}

	cmd.SilenceUsage = true

	return cmd
}

func runOrderStatus(cmd *cobra.Command, opts orderOptions, orderID string) error {
	// Validate inputs
	if opts.accountID == "" {
		return fmt.Errorf("account ID is required (use --account flag or configure default account)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/order/%s", opts.accountID, orderID)
	resp, err := client.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to get order status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var orderStatus OrderStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderStatus); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output result
	if opts.jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(orderStatus)
	}

	// Human-readable output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nOrder Status:\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID:   %s\n", orderStatus.OrderID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Symbol:     %s\n", orderStatus.Instrument.Symbol)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Side:       %s\n", orderStatus.Side)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Type:       %s\n", orderStatus.Type)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Status:     %s\n", orderStatus.Status)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Quantity:   %s\n", orderStatus.Quantity)
	if orderStatus.LimitPrice != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Limit:      $%s\n", orderStatus.LimitPrice)
	}
	if orderStatus.StopPrice != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Stop:       $%s\n", orderStatus.StopPrice)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Filled:     %s\n", orderStatus.FilledQuantity)
	if orderStatus.AveragePrice != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Avg Price:  $%s\n", orderStatus.AveragePrice)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Created:    %s\n", orderStatus.CreatedAt)
	if orderStatus.ClosedAt != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Closed:     %s\n", orderStatus.ClosedAt)
	}

	return nil
}

func runCancelOrder(cmd *cobra.Command, opts orderOptions, orderID string, skipConfirm bool) error {
	// Check trading is enabled
	if !opts.tradingEnabled {
		return config.ErrTradingDisabled
	}

	// Validate inputs
	if opts.accountID == "" {
		return fmt.Errorf("account ID is required (use --account flag or configure default account)")
	}

	// Show cancel preview (not in JSON mode)
	if !opts.jsonMode {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nCancel Order:\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n\n", orderID)
	}

	// Require confirmation unless --yes flag is set
	if !skipConfirm {
		return fmt.Errorf("cancel requires confirmation (use --yes to confirm)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/order/%s", opts.accountID, orderID)
	resp, err := client.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	// Output result
	if opts.jsonMode {
		result := map[string]any{
			"orderId": orderID,
			"status":  "cancel_requested",
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancel request submitted!\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n", orderID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNote: Cancellation is asynchronous. Use 'pub order status %s' to verify.\n", orderID)

	return nil
}

func runOrder(cmd *cobra.Command, opts orderOptions, symbol, side, quantity string, skipConfirm bool) error {
	// Check trading is enabled
	if !opts.tradingEnabled {
		return config.ErrTradingDisabled
	}

	// Validate inputs
	if opts.accountID == "" {
		return fmt.Errorf("account ID is required (use --account flag or configure default account)")
	}

	if quantity == "" {
		return fmt.Errorf("quantity is required (use --quantity flag)")
	}

	symbol = strings.ToUpper(symbol)
	orderID := uuid.New().String()

	// Show order preview (not in JSON mode)
	if !opts.jsonMode {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nOrder Preview:\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Action:   %s\n", side)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Symbol:   %s\n", symbol)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Quantity: %s shares\n", quantity)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Type:     MARKET\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n\n", orderID)
	}

	// Require confirmation unless --yes flag is set
	if !skipConfirm {
		return fmt.Errorf("order requires confirmation (use --yes to confirm)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build order request
	orderReq := OrderRequest{
		OrderID: orderID,
		Instrument: OrderInstrument{
			Symbol: symbol,
			Type:   "EQUITY",
		},
		OrderSide: side,
		OrderType: "MARKET",
		Expiration: OrderExpiration{
			TimeInForce: "DAY",
		},
		Quantity: quantity,
	}

	body, err := json.Marshal(orderReq)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/order", opts.accountID)
	resp, err := client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var orderResp OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output result
	if opts.jsonMode {
		result := map[string]any{
			"orderId":  orderResp.OrderID,
			"status":   "placed",
			"symbol":   symbol,
			"side":     side,
			"quantity": quantity,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Order placed successfully!\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n", orderResp.OrderID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s %s shares of %s\n", side, quantity, symbol)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNote: Order placement is asynchronous. Use 'pub order status %s' to check execution status.\n", orderResp.OrderID)

	return nil
}

func init() {
	var accountID string

	orderCmd := newOrderCmd()

	// Buy subcommand
	var buyQuantity string
	var buySkipConfirm bool
	buyCmd := &cobra.Command{
		Use:   "buy SYMBOL",
		Short: "Buy shares of a stock",
		Long: `Place a market buy order for shares of a stock.

Examples:
  pub order buy AAPL --quantity 10        # Buy 10 shares of Apple
  pub order buy AAPL --quantity 10 --yes  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil // Validation happens in RunE
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			if accountID == "" {
				accountID = cfg.AccountUUID
			}

			opts := orderOptions{
				baseURL:        cfg.APIBaseURL,
				authToken:      token,
				accountID:      accountID,
				tradingEnabled: cfg.TradingEnabled,
				jsonMode:       GetJSONMode(),
			}

			return runOrder(cmd, opts, args[0], "BUY", buyQuantity, buySkipConfirm)
		},
	}
	buyCmd.Flags().StringVarP(&buyQuantity, "quantity", "q", "", "Number of shares to buy (required)")
	buyCmd.Flags().BoolVarP(&buySkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	buyCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	buyCmd.SilenceUsage = true

	// Sell subcommand
	var sellQuantity string
	var sellSkipConfirm bool
	sellCmd := &cobra.Command{
		Use:   "sell SYMBOL",
		Short: "Sell shares of a stock",
		Long: `Place a market sell order for shares of a stock.

Examples:
  pub order sell AAPL --quantity 5        # Sell 5 shares of Apple
  pub order sell AAPL --quantity 5 --yes  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			if accountID == "" {
				accountID = cfg.AccountUUID
			}

			opts := orderOptions{
				baseURL:        cfg.APIBaseURL,
				authToken:      token,
				accountID:      accountID,
				tradingEnabled: cfg.TradingEnabled,
				jsonMode:       GetJSONMode(),
			}

			return runOrder(cmd, opts, args[0], "SELL", sellQuantity, sellSkipConfirm)
		},
	}
	sellCmd.Flags().StringVarP(&sellQuantity, "quantity", "q", "", "Number of shares to sell (required)")
	sellCmd.Flags().BoolVarP(&sellSkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	sellCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	sellCmd.SilenceUsage = true

	// Cancel subcommand
	var cancelSkipConfirm bool
	cancelCmd := &cobra.Command{
		Use:   "cancel ORDER_ID",
		Short: "Cancel an open order",
		Long: `Cancel an open order by its order ID.

Examples:
  pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d        # Cancel order (requires confirmation)
  pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d --yes  # Skip confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			if accountID == "" {
				accountID = cfg.AccountUUID
			}

			opts := orderOptions{
				baseURL:        cfg.APIBaseURL,
				authToken:      token,
				accountID:      accountID,
				tradingEnabled: cfg.TradingEnabled,
				jsonMode:       GetJSONMode(),
			}

			return runCancelOrder(cmd, opts, args[0], cancelSkipConfirm)
		},
	}
	cancelCmd.Flags().BoolVarP(&cancelSkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	cancelCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	cancelCmd.SilenceUsage = true

	// Status subcommand
	statusCmd := &cobra.Command{
		Use:   "status ORDER_ID",
		Short: "Check the status of an order",
		Long: `Check the status of an order by its order ID.

Status values: NEW, PARTIALLY_FILLED, FILLED, CANCELLED, REJECTED, EXPIRED

Examples:
  pub order status 912710f1-1a45-4ef0-88a7-cd513781933d`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			if accountID == "" {
				accountID = cfg.AccountUUID
			}

			opts := orderOptions{
				baseURL:   cfg.APIBaseURL,
				authToken: token,
				accountID: accountID,
				jsonMode:  GetJSONMode(),
			}

			return runOrderStatus(cmd, opts, args[0])
		},
	}
	statusCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	statusCmd.SilenceUsage = true

	orderCmd.AddCommand(buyCmd)
	orderCmd.AddCommand(sellCmd)
	orderCmd.AddCommand(cancelCmd)
	orderCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(orderCmd)
}
