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

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// orderOptions holds dependencies for the order command.
type orderOptions struct {
	baseURL        string
	authToken      string
	accountID      string
	tradingEnabled bool
	jsonMode       bool
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
  pub order list                                                # List open orders
  pub order status 912710f1-1a45-4ef0-88a7-cd513781933d         # Check order status
  pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d --yes   # Cancel an order`,
	}

	return cmd
}

// orderParams holds the parameters for an order.
type orderParams struct {
	quantity   string
	limitPrice string
	stopPrice  string
	expiration string
}

// newOrderBuyCmd creates the buy subcommand with the given options.
func newOrderBuyCmd(opts orderOptions) *cobra.Command {
	var params orderParams
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "buy SYMBOL",
		Short: "Buy shares of a stock",
		Long: `Place a buy order for shares of a stock.

Order types are determined by the flags used:
  - No price flags: MARKET order (executes at current market price)
  - --limit: LIMIT order (executes at limit price or better)
  - --stop: STOP order (triggers when stop price is reached)
  - --limit and --stop: STOP_LIMIT order (triggers at stop, executes at limit)

Examples:
  pub order buy AAPL --quantity 10                           # Market order
  pub order buy AAPL --quantity 10 --limit 175.00            # Limit order
  pub order buy AAPL --quantity 10 --stop 180.00             # Stop order
  pub order buy AAPL --quantity 10 --limit 175.00 --stop 174.00  # Stop-limit order
  pub order buy AAPL --quantity 10 --limit 175.00 --expiration GTC  # Good till cancelled`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrder(cmd, opts, args[0], "BUY", params, skipConfirm)
		},
	}

	cmd.Flags().StringVarP(&params.quantity, "quantity", "q", "", "Number of shares to buy (required)")
	cmd.Flags().StringVarP(&params.limitPrice, "limit", "l", "", "Limit price for LIMIT or STOP_LIMIT orders")
	cmd.Flags().StringVarP(&params.stopPrice, "stop", "s", "", "Stop price for STOP or STOP_LIMIT orders")
	cmd.Flags().StringVarP(&params.expiration, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.SilenceUsage = true

	return cmd
}

// newOrderSellCmd creates the sell subcommand with the given options.
func newOrderSellCmd(opts orderOptions) *cobra.Command {
	var params orderParams
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "sell SYMBOL",
		Short: "Sell shares of a stock",
		Long: `Place a sell order for shares of a stock.

Order types are determined by the flags used:
  - No price flags: MARKET order (executes at current market price)
  - --limit: LIMIT order (executes at limit price or better)
  - --stop: STOP order (triggers when stop price is reached)
  - --limit and --stop: STOP_LIMIT order (triggers at stop, executes at limit)

Examples:
  pub order sell AAPL --quantity 5                           # Market order
  pub order sell AAPL --quantity 5 --limit 180.00            # Limit order
  pub order sell AAPL --quantity 5 --stop 145.00             # Stop loss order
  pub order sell AAPL --quantity 5 --limit 144.00 --stop 145.00  # Stop-limit order
  pub order sell AAPL --quantity 5 --limit 180.00 --expiration GTC  # Good till cancelled`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrder(cmd, opts, args[0], "SELL", params, skipConfirm)
		},
	}

	cmd.Flags().StringVarP(&params.quantity, "quantity", "q", "", "Number of shares to sell (required)")
	cmd.Flags().StringVarP(&params.limitPrice, "limit", "l", "", "Limit price for LIMIT or STOP_LIMIT orders")
	cmd.Flags().StringVarP(&params.stopPrice, "stop", "s", "", "Stop price for STOP or STOP_LIMIT orders")
	cmd.Flags().StringVarP(&params.expiration, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
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

	var orderStatus api.OrderStatusResponse
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

// newOrderListCmd creates the list subcommand with the given options.
func newOrderListCmd(opts orderOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List open orders",
		Long: `List all open orders for your account.

Shows orders that are pending, new, or partially filled.

Examples:
  pub order list                # List open orders
  pub order list --json         # Output as JSON`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrderList(cmd, opts)
		},
	}

	cmd.SilenceUsage = true

	return cmd
}

func runOrderList(cmd *cobra.Command, opts orderOptions) error {
	// Validate inputs
	if opts.accountID == "" {
		return fmt.Errorf("account ID is required (use --account flag or configure default account)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", opts.accountID)
	resp, err := client.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to fetch orders: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var orderList api.OrderListResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderList); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output result
	if opts.jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(orderList.Orders)
	}

	if len(orderList.Orders) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No open orders")
		return nil
	}

	// Human-readable table output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%-38s %-6s %-5s %-8s %-10s %-6s %s\n",
		"ORDER ID", "SYMBOL", "SIDE", "TYPE", "STATUS", "QTY", "FILLED")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Repeat("-", 90))

	for _, order := range orderList.Orders {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-38s %-6s %-5s %-8s %-10s %-6s %s\n",
			order.OrderID,
			order.Instrument.Symbol,
			order.Side,
			order.Type,
			order.Status,
			order.Quantity,
			order.FilledQuantity)
	}

	return nil
}

// sumFees calculates the total regulatory fees.
func sumFees(fees api.RegulatoryFees) string {
	var total float64
	if v, err := parseFloat(fees.SECFee); err == nil {
		total += v
	}
	if v, err := parseFloat(fees.TAFFee); err == nil {
		total += v
	}
	if v, err := parseFloat(fees.ORFFee); err == nil {
		total += v
	}
	return fmt.Sprintf("%.2f", total)
}

// parseFloat parses a string as a float64, returning 0 on error.
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}

// extractErrorMessage extracts a human-readable message from an API error.
func extractErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()

	// Try to extract JSON error message from API response
	// Format: "preflight API error: 400 - {"code":3003,"header":"...","message":"..."}"
	if idx := strings.Index(errStr, "{"); idx != -1 {
		jsonPart := errStr[idx:]
		var apiErr struct {
			Code    any    `json:"code"`
			Header  string `json:"header"`
			Message string `json:"message"`
		}
		if json.Unmarshal([]byte(jsonPart), &apiErr) == nil {
			if apiErr.Message != "" {
				return apiErr.Message
			}
			if apiErr.Header != "" {
				return apiErr.Header
			}
		}
	}

	// Fallback: return a shortened version of the error
	if len(errStr) > 80 {
		return errStr[:80] + "..."
	}
	return errStr
}

// determineOrderType determines the order type based on the provided prices.
func determineOrderType(limitPrice, stopPrice string) string {
	hasLimit := limitPrice != ""
	hasStop := stopPrice != ""

	switch {
	case hasLimit && hasStop:
		return "STOP_LIMIT"
	case hasLimit:
		return "LIMIT"
	case hasStop:
		return "STOP"
	default:
		return "MARKET"
	}
}

// runPreflight calls the preflight API to get estimated costs for an order.
func runPreflight(opts orderOptions, symbol, side string, params orderParams) (*api.PreflightResponse, error) {
	orderType := determineOrderType(params.limitPrice, params.stopPrice)

	// Validate expiration
	expiration := strings.ToUpper(params.expiration)
	if expiration == "" {
		expiration = "DAY"
	}

	preflightReq := api.PreflightRequest{
		Instrument: api.OrderInstrument{
			Symbol: strings.ToUpper(symbol),
			Type:   "EQUITY",
		},
		OrderSide: side,
		OrderType: orderType,
		Expiration: api.OrderExpiration{
			TimeInForce: expiration,
		},
		Quantity:   params.quantity,
		LimitPrice: params.limitPrice,
		StopPrice:  params.stopPrice,
	}

	body, err := json.Marshal(preflightReq)
	if err != nil {
		return nil, fmt.Errorf("failed to encode preflight request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/preflight/single-leg", opts.accountID)
	resp, err := client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call preflight: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("preflight API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var preflightResp api.PreflightResponse
	if err := json.NewDecoder(resp.Body).Decode(&preflightResp); err != nil {
		return nil, fmt.Errorf("failed to decode preflight response: %w", err)
	}

	return &preflightResp, nil
}

func runOrder(cmd *cobra.Command, opts orderOptions, symbol, side string, params orderParams, skipConfirm bool) error {
	// Check trading is enabled
	if !opts.tradingEnabled {
		return config.ErrTradingDisabled
	}

	// Validate inputs
	if opts.accountID == "" {
		return fmt.Errorf("account ID is required (use --account flag or configure default account)")
	}

	if params.quantity == "" {
		return fmt.Errorf("quantity is required (use --quantity flag)")
	}

	symbol = strings.ToUpper(symbol)
	orderID := uuid.New().String()
	orderType := determineOrderType(params.limitPrice, params.stopPrice)

	// Validate expiration
	expiration := strings.ToUpper(params.expiration)
	if expiration != "DAY" && expiration != "GTC" {
		return fmt.Errorf("invalid expiration: %s (use DAY or GTC)", params.expiration)
	}

	// Call preflight to get estimated costs
	preflight, preflightErr := runPreflight(opts, symbol, side, params)

	// Show order preview (not in JSON mode)
	if !opts.jsonMode {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nOrder Preview:\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Action:   %s\n", side)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Symbol:   %s\n", symbol)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Quantity: %s shares\n", params.quantity)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Type:     %s\n", orderType)
		if params.limitPrice != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Limit:    $%s\n", params.limitPrice)
		}
		if params.stopPrice != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Stop:     $%s\n", params.stopPrice)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Expires:  %s\n", expiration)

		// Show preflight cost estimates if available
		if preflightErr == nil && preflight != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Estimated Cost:\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Order Value:  $%s\n", preflight.OrderValue)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Commission:   $%s\n", preflight.EstimatedCommission)
			totalFees := sumFees(preflight.RegulatoryFees)
			if totalFees != "0.00" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Reg Fees:     $%s\n", totalFees)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Total:        $%s\n", preflight.EstimatedCost)
		} else if preflightErr != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Cost Estimate: unavailable (%s)\n", extractErrorMessage(preflightErr))
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Order ID: %s\n\n", orderID)
	}

	// Require confirmation unless --yes flag is set
	if !skipConfirm {
		return fmt.Errorf("order requires confirmation (use --yes to confirm)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build order request
	orderReq := api.OrderRequest{
		OrderID: orderID,
		Instrument: api.OrderInstrument{
			Symbol: symbol,
			Type:   "EQUITY",
		},
		OrderSide: side,
		OrderType: orderType,
		Expiration: api.OrderExpiration{
			TimeInForce: expiration,
		},
		Quantity:   params.quantity,
		LimitPrice: params.limitPrice,
		StopPrice:  params.stopPrice,
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

	var orderResp api.OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output result
	if opts.jsonMode {
		result := map[string]any{
			"orderId":   orderResp.OrderID,
			"status":    "placed",
			"symbol":    symbol,
			"side":      side,
			"quantity":  params.quantity,
			"orderType": orderType,
		}
		if params.limitPrice != "" {
			result["limitPrice"] = params.limitPrice
		}
		if params.stopPrice != "" {
			result["stopPrice"] = params.stopPrice
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Order placed successfully!\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n", orderResp.OrderID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s %s shares of %s (%s)\n", side, params.quantity, symbol, orderType)
	if params.limitPrice != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Limit: $%s\n", params.limitPrice)
	}
	if params.stopPrice != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Stop: $%s\n", params.stopPrice)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNote: Order placement is asynchronous. Use 'pub order status %s' to check execution status.\n", orderResp.OrderID)

	return nil
}

func init() {
	var accountID string

	orderCmd := newOrderCmd()

	// Buy subcommand
	var buyParams orderParams
	var buySkipConfirm bool
	buyCmd := &cobra.Command{
		Use:   "buy SYMBOL",
		Short: "Buy shares of a stock",
		Long: `Place a buy order for shares of a stock.

Order types are determined by the flags used:
  - No price flags: MARKET order (executes at current market price)
  - --limit: LIMIT order (executes at limit price or better)
  - --stop: STOP order (triggers when stop price is reached)
  - --limit and --stop: STOP_LIMIT order (triggers at stop, executes at limit)

Examples:
  pub order buy AAPL --quantity 10                           # Market order
  pub order buy AAPL --quantity 10 --limit 175.00            # Limit order
  pub order buy AAPL --quantity 10 --stop 180.00             # Stop order
  pub order buy AAPL --quantity 10 --limit 175.00 --stop 174.00  # Stop-limit order
  pub order buy AAPL --quantity 10 --limit 175.00 --expiration GTC  # Good till cancelled`,
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
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
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

			return runOrder(cmd, opts, args[0], "BUY", buyParams, buySkipConfirm)
		},
	}
	buyCmd.Flags().StringVarP(&buyParams.quantity, "quantity", "q", "", "Number of shares to buy (required)")
	buyCmd.Flags().StringVarP(&buyParams.limitPrice, "limit", "l", "", "Limit price for LIMIT or STOP_LIMIT orders")
	buyCmd.Flags().StringVarP(&buyParams.stopPrice, "stop", "s", "", "Stop price for STOP or STOP_LIMIT orders")
	buyCmd.Flags().StringVarP(&buyParams.expiration, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
	buyCmd.Flags().BoolVarP(&buySkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	buyCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	buyCmd.SilenceUsage = true

	// Sell subcommand
	var sellParams orderParams
	var sellSkipConfirm bool
	sellCmd := &cobra.Command{
		Use:   "sell SYMBOL",
		Short: "Sell shares of a stock",
		Long: `Place a sell order for shares of a stock.

Order types are determined by the flags used:
  - No price flags: MARKET order (executes at current market price)
  - --limit: LIMIT order (executes at limit price or better)
  - --stop: STOP order (triggers when stop price is reached)
  - --limit and --stop: STOP_LIMIT order (triggers at stop, executes at limit)

Examples:
  pub order sell AAPL --quantity 5                           # Market order
  pub order sell AAPL --quantity 5 --limit 180.00            # Limit order
  pub order sell AAPL --quantity 5 --stop 145.00             # Stop loss order
  pub order sell AAPL --quantity 5 --limit 144.00 --stop 145.00  # Stop-limit order
  pub order sell AAPL --quantity 5 --limit 180.00 --expiration GTC  # Good till cancelled`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
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

			return runOrder(cmd, opts, args[0], "SELL", sellParams, sellSkipConfirm)
		},
	}
	sellCmd.Flags().StringVarP(&sellParams.quantity, "quantity", "q", "", "Number of shares to sell (required)")
	sellCmd.Flags().StringVarP(&sellParams.limitPrice, "limit", "l", "", "Limit price for LIMIT or STOP_LIMIT orders")
	sellCmd.Flags().StringVarP(&sellParams.stopPrice, "stop", "s", "", "Stop price for STOP or STOP_LIMIT orders")
	sellCmd.Flags().StringVarP(&sellParams.expiration, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
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
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
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
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
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

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List open orders",
		Long: `List all open orders for your account.

Shows orders that are pending, new, or partially filled.

Examples:
  pub order list                # List open orders
  pub order list --json         # Output as JSON`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL, false)
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

			return runOrderList(cmd, opts)
		},
	}
	listCmd.Flags().StringVarP(&accountID, "account", "a", "", "Account ID (uses default if not specified)")
	listCmd.SilenceUsage = true

	orderCmd.AddCommand(buyCmd)
	orderCmd.AddCommand(sellCmd)
	orderCmd.AddCommand(cancelCmd)
	orderCmd.AddCommand(statusCmd)
	orderCmd.AddCommand(listCmd)
	rootCmd.AddCommand(orderCmd)
}
