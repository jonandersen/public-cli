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
	Symbol string     `json:"symbol"`
	Greeks GreeksData `json:"greeks"`
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

// MultilegPreflightRequest represents a multi-leg preflight request.
type MultilegPreflightRequest struct {
	OrderType  string             `json:"orderType"`
	Expiration MultilegExpiration `json:"expiration"`
	Quantity   string             `json:"quantity"`
	LimitPrice string             `json:"limitPrice"`
	Legs       []MultilegLeg      `json:"legs"`
}

// MultilegExpiration represents time-in-force for multi-leg orders.
type MultilegExpiration struct {
	TimeInForce string `json:"timeInForce"`
}

// MultilegLeg represents a single leg in a multi-leg order.
type MultilegLeg struct {
	Instrument         MultilegInstrument `json:"instrument"`
	Side               string             `json:"side"`
	OpenCloseIndicator string             `json:"openCloseIndicator"`
	RatioQuantity      int                `json:"ratioQuantity"`
}

// MultilegInstrument represents an instrument in a multi-leg order.
type MultilegInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// MultilegPreflightResponse represents the API response for multi-leg preflight.
type MultilegPreflightResponse struct {
	BaseSymbol              string                 `json:"baseSymbol"`
	StrategyName            string                 `json:"strategyName"`
	Legs                    []MultilegPreflightLeg `json:"legs"`
	EstimatedCommission     string                 `json:"estimatedCommission"`
	RegulatoryFees          MultilegRegulatoryFees `json:"regulatoryFees"`
	EstimatedIndexOptionFee string                 `json:"estimatedIndexOptionFee"`
	OrderValue              string                 `json:"orderValue"`
	EstimatedQuantity       string                 `json:"estimatedQuantity"`
	EstimatedCost           string                 `json:"estimatedCost"`
	BuyingPowerRequirement  string                 `json:"buyingPowerRequirement"`
	EstimatedProceeds       string                 `json:"estimatedProceeds"`
	PriceIncrement          MultilegPriceIncrement `json:"priceIncrement"`
}

// MultilegPreflightLeg represents a leg in the preflight response.
type MultilegPreflightLeg struct {
	Instrument         MultilegInstrument `json:"instrument"`
	Side               string             `json:"side"`
	OpenCloseIndicator string             `json:"openCloseIndicator"`
	RatioQuantity      int                `json:"ratioQuantity"`
}

// MultilegRegulatoryFees represents regulatory fees for multi-leg orders.
type MultilegRegulatoryFees struct {
	SECFee      string `json:"secFee"`
	TAFFee      string `json:"tafFee"`
	ORFFee      string `json:"orfFee"`
	ExchangeFee string `json:"exchangeFee"`
	OCCFee      string `json:"occFee"`
	CATFee      string `json:"catFee"`
}

// MultilegPriceIncrement represents price increment information.
type MultilegPriceIncrement struct {
	IncrementBelow3  string `json:"incrementBelow3"`
	IncrementAbove3  string `json:"incrementAbove3"`
	CurrentIncrement string `json:"currentIncrement"`
}

// MultilegOrderRequest represents a multi-leg order request.
type MultilegOrderRequest struct {
	OrderID    string             `json:"orderId"`
	OrderType  string             `json:"orderType"`
	Expiration MultilegExpiration `json:"expiration"`
	Quantity   string             `json:"quantity"`
	LimitPrice string             `json:"limitPrice"`
	Legs       []MultilegLeg      `json:"legs"`
}

// MultilegOrderResponse represents the API response for a multi-leg order.
type MultilegOrderResponse struct {
	OrderID string `json:"orderId"`
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

// parseLeg parses a leg string in format "SIDE SYMBOL OPEN|CLOSE [RATIO]"
// Example: "BUY AAPL250117C00175000 OPEN" or "SELL AAPL250117C00180000 OPEN 2"
func parseLeg(legStr string) (MultilegLeg, error) {
	parts := strings.Fields(legStr)
	if len(parts) < 3 {
		return MultilegLeg{}, fmt.Errorf("invalid leg format: expected 'SIDE SYMBOL OPEN|CLOSE [RATIO]', got %q", legStr)
	}

	side := strings.ToUpper(parts[0])
	if side != "BUY" && side != "SELL" {
		return MultilegLeg{}, fmt.Errorf("invalid side %q: must be BUY or SELL", parts[0])
	}

	symbol := strings.ToUpper(parts[1])

	openClose := strings.ToUpper(parts[2])
	if openClose != "OPEN" && openClose != "CLOSE" {
		return MultilegLeg{}, fmt.Errorf("invalid open/close %q: must be OPEN or CLOSE", parts[2])
	}

	ratio := 1
	if len(parts) >= 4 {
		if _, err := fmt.Sscanf(parts[3], "%d", &ratio); err != nil {
			return MultilegLeg{}, fmt.Errorf("invalid ratio %q: must be an integer", parts[3])
		}
	}

	// Determine instrument type from symbol (options have 21+ chars in OSI format)
	instType := "OPTION"
	if len(symbol) <= 5 {
		instType = "EQUITY"
	}

	return MultilegLeg{
		Instrument: MultilegInstrument{
			Symbol: symbol,
			Type:   instType,
		},
		Side:               side,
		OpenCloseIndicator: openClose,
		RatioQuantity:      ratio,
	}, nil
}

func runMultilegPreflight(cmd *cobra.Command, opts optionsOptions, legs []string, limitPrice, quantity, expiration string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse legs
	var parsedLegs []MultilegLeg
	for _, legStr := range legs {
		leg, err := parseLeg(legStr)
		if err != nil {
			return err
		}
		parsedLegs = append(parsedLegs, leg)
	}

	if len(parsedLegs) < 2 {
		return fmt.Errorf("multi-leg orders require at least 2 legs")
	}
	if len(parsedLegs) > 6 {
		return fmt.Errorf("multi-leg orders support at most 6 legs")
	}

	// Validate expiration
	exp := strings.ToUpper(expiration)
	if exp != "DAY" && exp != "GTC" {
		return fmt.Errorf("invalid expiration: %s (use DAY or GTC)", expiration)
	}

	// Build request
	req := MultilegPreflightRequest{
		OrderType: "LIMIT",
		Expiration: MultilegExpiration{
			TimeInForce: exp,
		},
		Quantity:   quantity,
		LimitPrice: limitPrice,
		Legs:       parsedLegs,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	client := api.NewClient(opts.baseURL, opts.authToken)
	path := fmt.Sprintf("/userapigateway/trading/%s/preflight/multi-leg", opts.accountID)
	resp, err := client.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to call preflight: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var preflightResp MultilegPreflightResponse
	if err := json.NewDecoder(resp.Body).Decode(&preflightResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Format output
	if opts.jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(preflightResp)
	}

	// Human-readable output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nMulti-Leg Order Preview\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", strings.Repeat("-", 40))

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Strategy:    %s\n", preflightResp.StrategyName)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Underlying:  %s\n", preflightResp.BaseSymbol)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Quantity:    %s\n", preflightResp.EstimatedQuantity)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Limit:       $%s\n\n", limitPrice)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Legs:\n")
	for _, leg := range preflightResp.Legs {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s %dx %s (%s)\n",
			leg.Side, leg.RatioQuantity, leg.Instrument.Symbol, leg.OpenCloseIndicator)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nEstimated Costs:\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order Value:     $%s\n", preflightResp.OrderValue)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Commission:      $%s\n", preflightResp.EstimatedCommission)

	// Sum up regulatory fees
	fees := preflightResp.RegulatoryFees
	totalFees := sumMultilegFees(fees)
	if totalFees != "0.00" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Regulatory Fees: $%s\n", totalFees)
	}
	if preflightResp.EstimatedIndexOptionFee != "" && preflightResp.EstimatedIndexOptionFee != "0" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Index Option Fee: $%s\n", preflightResp.EstimatedIndexOptionFee)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Total Cost:      $%s\n", preflightResp.EstimatedCost)

	if preflightResp.EstimatedProceeds != "" && preflightResp.EstimatedProceeds != "0" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Est. Proceeds:   $%s\n", preflightResp.EstimatedProceeds)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nBuying Power Required: $%s\n", preflightResp.BuyingPowerRequirement)

	if preflightResp.PriceIncrement.CurrentIncrement != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Price Increment:       $%s\n", preflightResp.PriceIncrement.CurrentIncrement)
	}

	return nil
}

// sumMultilegFees calculates the total regulatory fees for multi-leg orders.
func sumMultilegFees(fees MultilegRegulatoryFees) string {
	var total float64
	for _, feeStr := range []string{fees.SECFee, fees.TAFFee, fees.ORFFee, fees.ExchangeFee, fees.OCCFee, fees.CATFee} {
		if feeStr == "" {
			continue
		}
		var v float64
		if _, err := fmt.Sscanf(feeStr, "%f", &v); err == nil {
			total += v
		}
	}
	return fmt.Sprintf("%.2f", total)
}

func runMultilegOrder(cmd *cobra.Command, opts optionsOptions, legs []string, limitPrice, quantity, expiration string, skipConfirm bool) error {
	// Parse legs
	var parsedLegs []MultilegLeg
	for _, legStr := range legs {
		leg, err := parseLeg(legStr)
		if err != nil {
			return err
		}
		parsedLegs = append(parsedLegs, leg)
	}

	if len(parsedLegs) < 2 {
		return fmt.Errorf("multi-leg orders require at least 2 legs")
	}
	if len(parsedLegs) > 6 {
		return fmt.Errorf("multi-leg orders support at most 6 legs")
	}

	// Validate expiration
	exp := strings.ToUpper(expiration)
	if exp != "DAY" && exp != "GTC" {
		return fmt.Errorf("invalid expiration: %s (use DAY or GTC)", expiration)
	}

	// Generate order ID
	orderID := uuid.New().String()

	// Call preflight to get cost estimate
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	preflightReq := MultilegPreflightRequest{
		OrderType: "LIMIT",
		Expiration: MultilegExpiration{
			TimeInForce: exp,
		},
		Quantity:   quantity,
		LimitPrice: limitPrice,
		Legs:       parsedLegs,
	}

	preflightBody, err := json.Marshal(preflightReq)
	if err != nil {
		return fmt.Errorf("failed to encode preflight request: %w", err)
	}

	client := api.NewClient(opts.baseURL, opts.authToken)
	preflightPath := fmt.Sprintf("/userapigateway/trading/%s/preflight/multi-leg", opts.accountID)
	preflightResp, err := client.Post(ctx, preflightPath, bytes.NewReader(preflightBody))
	if err != nil {
		return fmt.Errorf("failed to call preflight: %w", err)
	}
	defer func() { _ = preflightResp.Body.Close() }()

	var preflight MultilegPreflightResponse
	if preflightResp.StatusCode == 200 {
		if err := json.NewDecoder(preflightResp.Body).Decode(&preflight); err != nil {
			return fmt.Errorf("failed to decode preflight response: %w", err)
		}
	}

	// Display order preview
	if !opts.jsonMode {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nMulti-Leg Order Preview\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", strings.Repeat("-", 40))

		if preflightResp.StatusCode == 200 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Strategy:    %s\n", preflight.StrategyName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Underlying:  %s\n", preflight.BaseSymbol)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Quantity:    %s\n", quantity)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Limit:       $%s\n\n", limitPrice)

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Legs:\n")
		for _, leg := range parsedLegs {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s %dx %s (%s)\n",
				leg.Side, leg.RatioQuantity, leg.Instrument.Symbol, leg.OpenCloseIndicator)
		}

		if preflightResp.StatusCode == 200 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nEstimated Costs:\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order Value:     $%s\n", preflight.OrderValue)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Commission:      $%s\n", preflight.EstimatedCommission)
			totalFees := sumMultilegFees(preflight.RegulatoryFees)
			if totalFees != "0.00" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Regulatory Fees: $%s\n", totalFees)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Total Cost:      $%s\n", preflight.EstimatedCost)
			if preflight.EstimatedProceeds != "" && preflight.EstimatedProceeds != "0" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Est. Proceeds:   $%s\n", preflight.EstimatedProceeds)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nBuying Power Required: $%s\n", preflight.BuyingPowerRequirement)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Order ID: %s\n\n", orderID)
	}

	// Require confirmation unless --yes flag is set
	if !skipConfirm {
		return fmt.Errorf("order requires confirmation (use --yes to confirm)")
	}

	// Place the order
	orderCtx, orderCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer orderCancel()

	orderReq := MultilegOrderRequest{
		OrderID:   orderID,
		OrderType: "LIMIT",
		Expiration: MultilegExpiration{
			TimeInForce: exp,
		},
		Quantity:   quantity,
		LimitPrice: limitPrice,
		Legs:       parsedLegs,
	}

	orderBody, err := json.Marshal(orderReq)
	if err != nil {
		return fmt.Errorf("failed to encode order request: %w", err)
	}

	orderPath := fmt.Sprintf("/userapigateway/trading/%s/order/multi-leg", opts.accountID)
	orderResp, err := client.Post(orderCtx, orderPath, bytes.NewReader(orderBody))
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}
	defer func() { _ = orderResp.Body.Close() }()

	if orderResp.StatusCode != 200 {
		respBody, _ := io.ReadAll(orderResp.Body)
		return fmt.Errorf("API error: %d - %s", orderResp.StatusCode, string(respBody))
	}

	var orderResult MultilegOrderResponse
	if err := json.NewDecoder(orderResp.Body).Decode(&orderResult); err != nil {
		return fmt.Errorf("failed to decode order response: %w", err)
	}

	// Output result
	if opts.jsonMode {
		result := map[string]any{
			"orderId":    orderResult.OrderID,
			"status":     "placed",
			"strategy":   preflight.StrategyName,
			"underlying": preflight.BaseSymbol,
			"quantity":   quantity,
			"limitPrice": limitPrice,
			"legs":       len(parsedLegs),
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Order placed successfully!\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n", orderResult.OrderID)
	if preflight.StrategyName != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Strategy: %s on %s\n", preflight.StrategyName, preflight.BaseSymbol)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s spread(s) at $%s limit\n", quantity, limitPrice)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNote: Order placement is asynchronous. Use 'pub order status %s' to check execution status.\n", orderResult.OrderID)

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

	// Multileg commands
	multilegCmd := &cobra.Command{
		Use:   "multileg",
		Short: "Multi-leg options orders",
		Long:  `Commands for multi-leg options strategies (spreads, straddles, etc.).`,
	}

	var multilegPreflightAccountID string
	var multilegPreflightLegs []string
	var multilegPreflightLimit string
	var multilegPreflightQty string
	var multilegPreflightExp string

	multilegPreflightCmd := &cobra.Command{
		Use:   "preflight",
		Short: "Preview a multi-leg order",
		Long: `Preview estimated costs for a multi-leg options order before placing it.

Each leg is specified with --leg in format: "SIDE SYMBOL OPEN|CLOSE [RATIO]"
  - SIDE: BUY or SELL
  - SYMBOL: Option symbol in OSI format (e.g., AAPL250117C00175000)
  - OPEN|CLOSE: Whether opening or closing the position
  - RATIO: Optional ratio quantity (default 1)

Examples:
  # Vertical call spread (buy lower strike, sell higher strike)
  pub options multileg preflight \
    --leg "BUY AAPL250117C00175000 OPEN" \
    --leg "SELL AAPL250117C00180000 OPEN" \
    --limit 2.50 --quantity 1

  # Iron condor (4 legs)
  pub options multileg preflight \
    --leg "SELL AAPL250117P00165000 OPEN" \
    --leg "BUY AAPL250117P00160000 OPEN" \
    --leg "SELL AAPL250117C00185000 OPEN" \
    --leg "BUY AAPL250117C00190000 OPEN" \
    --limit 1.20 --quantity 1`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			if multilegPreflightAccountID == "" {
				multilegPreflightAccountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = multilegPreflightAccountID
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			if len(multilegPreflightLegs) < 2 {
				return fmt.Errorf("at least 2 legs required (use --leg flag)")
			}
			if multilegPreflightLimit == "" {
				return fmt.Errorf("limit price is required (use --limit flag)")
			}
			if multilegPreflightQty == "" {
				multilegPreflightQty = "1"
			}
			return runMultilegPreflight(cmd, opts, multilegPreflightLegs, multilegPreflightLimit, multilegPreflightQty, multilegPreflightExp)
		},
	}

	multilegPreflightCmd.Flags().StringVarP(&multilegPreflightAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	multilegPreflightCmd.Flags().StringArrayVarP(&multilegPreflightLegs, "leg", "L", nil, "Leg in format 'SIDE SYMBOL OPEN|CLOSE [RATIO]' (repeat for each leg)")
	multilegPreflightCmd.Flags().StringVarP(&multilegPreflightLimit, "limit", "l", "", "Limit price (required)")
	multilegPreflightCmd.Flags().StringVarP(&multilegPreflightQty, "quantity", "q", "1", "Number of spreads/strategies")
	multilegPreflightCmd.Flags().StringVarP(&multilegPreflightExp, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
	multilegPreflightCmd.SilenceUsage = true

	// Multileg order command
	var multilegOrderAccountID string
	var multilegOrderLegs []string
	var multilegOrderLimit string
	var multilegOrderQty string
	var multilegOrderExp string
	var multilegOrderConfirm bool

	multilegOrderCmd := &cobra.Command{
		Use:   "order",
		Short: "Place a multi-leg options order",
		Long: `Place a multi-leg options order (spreads, straddles, iron condors, etc.).

Each leg is specified with --leg in format: "SIDE SYMBOL OPEN|CLOSE [RATIO]"
  - SIDE: BUY or SELL
  - SYMBOL: Option symbol in OSI format (e.g., AAPL250117C00175000)
  - OPEN|CLOSE: Whether opening or closing the position
  - RATIO: Optional ratio quantity (default 1)

Examples:
  # Vertical call spread (buy lower strike, sell higher strike)
  pub options multileg order \
    --leg "BUY AAPL250117C00175000 OPEN" \
    --leg "SELL AAPL250117C00180000 OPEN" \
    --limit 2.50 --quantity 1 --yes

  # Iron condor (4 legs)
  pub options multileg order \
    --leg "SELL AAPL250117P00165000 OPEN" \
    --leg "BUY AAPL250117P00160000 OPEN" \
    --leg "SELL AAPL250117C00185000 OPEN" \
    --leg "BUY AAPL250117C00190000 OPEN" \
    --limit 1.20 --quantity 1 --yes`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := getAuthToken(store, cfg.APIBaseURL)
			if err != nil {
				return err
			}

			if multilegOrderAccountID == "" {
				multilegOrderAccountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = multilegOrderAccountID
			opts.jsonMode = GetJSONMode()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.accountID == "" {
				return fmt.Errorf("account ID is required (use --account flag or configure default account)")
			}
			if len(multilegOrderLegs) < 2 {
				return fmt.Errorf("at least 2 legs required (use --leg flag)")
			}
			if multilegOrderLimit == "" {
				return fmt.Errorf("limit price is required (use --limit flag)")
			}
			if multilegOrderQty == "" {
				multilegOrderQty = "1"
			}
			return runMultilegOrder(cmd, opts, multilegOrderLegs, multilegOrderLimit, multilegOrderQty, multilegOrderExp, multilegOrderConfirm)
		},
	}

	multilegOrderCmd.Flags().StringVarP(&multilegOrderAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	multilegOrderCmd.Flags().StringArrayVarP(&multilegOrderLegs, "leg", "L", nil, "Leg in format 'SIDE SYMBOL OPEN|CLOSE [RATIO]' (repeat for each leg)")
	multilegOrderCmd.Flags().StringVarP(&multilegOrderLimit, "limit", "l", "", "Limit price (required)")
	multilegOrderCmd.Flags().StringVarP(&multilegOrderQty, "quantity", "q", "1", "Number of spreads/strategies")
	multilegOrderCmd.Flags().StringVarP(&multilegOrderExp, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
	multilegOrderCmd.Flags().BoolVarP(&multilegOrderConfirm, "yes", "y", false, "Confirm order placement (required)")
	multilegOrderCmd.SilenceUsage = true

	multilegCmd.AddCommand(multilegPreflightCmd)
	multilegCmd.AddCommand(multilegOrderCmd)

	optionsCmd.AddCommand(expirationsCmd)
	optionsCmd.AddCommand(chainCmd)
	optionsCmd.AddCommand(greeksCmd)
	optionsCmd.AddCommand(multilegCmd)
	rootCmd.AddCommand(optionsCmd)
}
