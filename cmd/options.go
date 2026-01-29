package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// chainFilter holds filtering options for the options chain command.
type chainFilter struct {
	minStrike float64
	maxStrike float64
	minOI     int
	minVolume int
	callsOnly bool
	putsOnly  bool
	strikes   int // N strikes around ATM (requires underlying price)
}

// filterOptions filters a slice of OptionQuote based on the given criteria.
func filterOptions(options []api.OptionQuote, filter chainFilter) []api.OptionQuote {
	if len(options) == 0 {
		return options
	}

	var filtered []api.OptionQuote
	for _, opt := range options {
		strike := parseStrikeFloat(opt.Instrument.Symbol)

		// Strike range filter
		if filter.minStrike > 0 && strike < filter.minStrike {
			continue
		}
		if filter.maxStrike > 0 && strike > filter.maxStrike {
			continue
		}

		// Open interest filter
		if filter.minOI > 0 && opt.OpenInterest < filter.minOI {
			continue
		}

		// Volume filter
		if filter.minVolume > 0 && opt.Volume < filter.minVolume {
			continue
		}

		filtered = append(filtered, opt)
	}

	return filtered
}

// filterStrikesAroundATM filters options to N strikes centered around the ATM strike.
// underlyingPrice is the current price of the underlying stock.
func filterStrikesAroundATM(options []api.OptionQuote, n int, underlyingPrice float64) []api.OptionQuote {
	if len(options) == 0 || n <= 0 {
		return options
	}

	// Find the ATM strike (closest to underlying price)
	var closestIdx int
	closestDiff := float64(1e12)
	for i, opt := range options {
		strike := parseStrikeFloat(opt.Instrument.Symbol)
		diff := abs(strike - underlyingPrice)
		if diff < closestDiff {
			closestDiff = diff
			closestIdx = i
		}
	}

	// Take n/2 below and n/2 above ATM
	half := n / 2
	startIdx := closestIdx - half
	endIdx := closestIdx + half + (n % 2) // Add 1 more if n is odd

	if startIdx < 0 {
		endIdx += -startIdx
		startIdx = 0
	}
	if endIdx > len(options) {
		startIdx -= endIdx - len(options)
		endIdx = len(options)
	}
	if startIdx < 0 {
		startIdx = 0
	}

	return options[startIdx:endIdx]
}

// parseStrikeFloat extracts the strike price as a float from an OSI option symbol.
func parseStrikeFloat(symbol string) float64 {
	if len(symbol) < 8 {
		return 0
	}
	strikeStr := symbol[len(symbol)-8:]
	var strike int64
	if _, err := fmt.Sscanf(strikeStr, "%d", &strike); err != nil {
		return 0
	}
	return float64(strike) / 1000.0
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// optionsOptions holds dependencies for options commands.
type optionsOptions struct {
	baseURL   string
	authToken string
	accountID string
	jsonMode  bool
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

	client := api.NewClient(opts.baseURL, opts.authToken)
	expResp, err := client.GetOptionExpirations(ctx, opts.accountID, symbol)
	if err != nil {
		return err
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
// Note: This function is unused; the actual chain command is created inline in init().
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
			return runOptionsChain(cmd, opts, args[0], expiration, chainFilter{})
		},
	}

	cmd.Flags().StringVarP(&expiration, "expiration", "e", "", "Expiration date (YYYY-MM-DD)")
	cmd.SilenceUsage = true

	return cmd
}

func runOptionsChain(cmd *cobra.Command, opts optionsOptions, symbol, expiration string, filter chainFilter) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient(opts.baseURL, opts.authToken)
	chainResp, err := client.GetOptionChain(ctx, opts.accountID, symbol, expiration)
	if err != nil {
		return err
	}

	// Get underlying price if we need to filter by strikes around ATM
	var underlyingPrice float64
	if filter.strikes > 0 {
		instruments := []api.QuoteInstrument{{Symbol: strings.ToUpper(symbol), Type: "EQUITY"}}
		quotes, err := client.GetQuotes(ctx, opts.accountID, instruments)
		if err != nil {
			return fmt.Errorf("failed to get underlying price for ATM filtering: %w", err)
		}
		if len(quotes) > 0 {
			underlyingPrice, _ = strconv.ParseFloat(quotes[0].Last, 64)
		}
	}

	// Apply filters
	calls := chainResp.Calls
	puts := chainResp.Puts

	// First apply strike/OI/volume filters
	if filter.minStrike > 0 || filter.maxStrike > 0 || filter.minOI > 0 || filter.minVolume > 0 {
		if !filter.putsOnly {
			calls = filterOptions(calls, filter)
		}
		if !filter.callsOnly {
			puts = filterOptions(puts, filter)
		}
	}

	// Then apply ATM strikes filter (if specified)
	if filter.strikes > 0 && underlyingPrice > 0 {
		// Sort by strike to ensure proper ordering
		sort.Slice(calls, func(i, j int) bool {
			return parseStrikeFloat(calls[i].Instrument.Symbol) < parseStrikeFloat(calls[j].Instrument.Symbol)
		})
		sort.Slice(puts, func(i, j int) bool {
			return parseStrikeFloat(puts[i].Instrument.Symbol) < parseStrikeFloat(puts[j].Instrument.Symbol)
		})

		if !filter.putsOnly {
			calls = filterStrikesAroundATM(calls, filter.strikes, underlyingPrice)
		}
		if !filter.callsOnly {
			puts = filterStrikesAroundATM(puts, filter.strikes, underlyingPrice)
		}
	}

	// Apply calls-only / puts-only filter
	if filter.callsOnly {
		puts = nil
	}
	if filter.putsOnly {
		calls = nil
	}

	if len(calls) == 0 && len(puts) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No options available for %s expiring %s (after filtering)\n", chainResp.BaseSymbol, expiration)
		return nil
	}

	// Format output
	if opts.jsonMode {
		// Return filtered results in JSON
		filteredResp := api.OptionChainResponse{
			BaseSymbol: chainResp.BaseSymbol,
			Calls:      calls,
			Puts:       puts,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(filteredResp)
	}

	// Table output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Option Chain for %s - Expiration: %s\n\n", chainResp.BaseSymbol, expiration)

	if len(calls) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CALLS\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "Strike", "Bid", "Ask", "Volume", "OI")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "------", "------", "------", "------", "------")
		for _, call := range calls {
			strike := parseStrikeFromSymbol(call.Instrument.Symbol)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10d  %10d\n",
				strike, call.Bid, call.Ask, call.Volume, call.OpenInterest)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n")
	}

	if len(puts) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PUTS\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "Strike", "Bid", "Ask", "Volume", "OI")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s  %8s  %8s  %10s  %10s\n", "------", "------", "------", "------", "------")
		for _, put := range puts {
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

	client := api.NewClient(opts.baseURL, opts.authToken)
	greeksResp, err := client.GetOptionGreeks(ctx, opts.accountID, symbols)
	if err != nil {
		return err
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

// singleLegParams holds parameters for single-leg options orders.
type singleLegParams struct {
	quantity   string
	limitPrice string
	expiration string
	openClose  string // "OPEN" or "CLOSE"
}

func runSingleLegPreflight(opts optionsOptions, symbol, side string, params singleLegParams) (*api.OptionsPreflightResponse, error) {
	// Validate expiration
	expiration := strings.ToUpper(params.expiration)
	if expiration == "" {
		expiration = "DAY"
	}

	preflightReq := api.OptionsPreflightRequest{
		Instrument: api.OrderInstrument{
			Symbol: strings.ToUpper(symbol),
			Type:   "OPTION",
		},
		OrderSide: side,
		OrderType: "LIMIT",
		Expiration: api.OrderExpiration{
			TimeInForce: expiration,
		},
		Quantity:           params.quantity,
		LimitPrice:         params.limitPrice,
		OpenCloseIndicator: strings.ToUpper(params.openClose),
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

	var preflightResp api.OptionsPreflightResponse
	if err := json.NewDecoder(resp.Body).Decode(&preflightResp); err != nil {
		return nil, fmt.Errorf("failed to decode preflight response: %w", err)
	}

	return &preflightResp, nil
}

func runSingleLegOrder(cmd *cobra.Command, opts optionsOptions, symbol, side string, params singleLegParams, skipConfirm, tradingEnabled bool) error {
	// Check trading is enabled
	if !tradingEnabled {
		return config.ErrTradingDisabled
	}

	// Validate inputs
	if opts.accountID == "" {
		return fmt.Errorf("account ID is required (use --account flag or configure default account)")
	}

	if params.quantity == "" {
		return fmt.Errorf("quantity is required (use --quantity flag)")
	}

	if params.limitPrice == "" {
		return fmt.Errorf("limit price is required for options orders (use --limit flag)")
	}

	openClose := strings.ToUpper(params.openClose)
	if openClose != "OPEN" && openClose != "CLOSE" {
		return fmt.Errorf("open/close indicator is required (use --open or --close flag)")
	}

	symbol = strings.ToUpper(symbol)
	orderID := uuid.New().String()

	// Validate expiration
	expiration := strings.ToUpper(params.expiration)
	if expiration != "DAY" && expiration != "GTC" {
		return fmt.Errorf("invalid expiration: %s (use DAY or GTC)", params.expiration)
	}

	// Call preflight to get estimated costs
	preflight, preflightErr := runSingleLegPreflight(opts, symbol, side, params)

	// Show order preview (not in JSON mode)
	if !opts.jsonMode {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nOptions Order Preview:\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Action:     %s to %s\n", side, openClose)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Symbol:     %s\n", symbol)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Quantity:   %s contract(s)\n", params.quantity)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Limit:      $%s\n", params.limitPrice)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Expires:    %s\n", expiration)

		// Show preflight cost estimates if available
		if preflightErr == nil && preflight != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Estimated Cost:\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Order Value:  $%s\n", preflight.OrderValue)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Commission:   $%s\n", preflight.EstimatedCommission)
			totalFees := sumOptionsFees(preflight.RegulatoryFees)
			if totalFees != "0.00" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Reg Fees:     $%s\n", totalFees)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Total:        $%s\n", preflight.EstimatedCost)
			if preflight.EstimatedProceeds != "" && preflight.EstimatedProceeds != "0" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Est Proceeds: $%s\n", preflight.EstimatedProceeds)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Buying Power Required: $%s\n", preflight.BuyingPowerRequirement)
		} else if preflightErr != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  Cost Estimate: unavailable (%s)\n", extractOptionsErrorMessage(preflightErr))
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
	orderReq := api.OptionsOrderRequest{
		OrderID: orderID,
		Instrument: api.OrderInstrument{
			Symbol: symbol,
			Type:   "OPTION",
		},
		OrderSide: side,
		OrderType: "LIMIT",
		Expiration: api.OrderExpiration{
			TimeInForce: expiration,
		},
		Quantity:           params.quantity,
		LimitPrice:         params.limitPrice,
		OpenCloseIndicator: openClose,
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
			"orderId":    orderResp.OrderID,
			"status":     "placed",
			"symbol":     symbol,
			"side":       side,
			"quantity":   params.quantity,
			"limitPrice": params.limitPrice,
			"openClose":  openClose,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Order placed successfully!\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Order ID: %s\n", orderResp.OrderID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s to %s %s contract(s) of %s at $%s\n", side, openClose, params.quantity, symbol, params.limitPrice)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNote: Order placement is asynchronous. Use 'pub order status %s' to check execution status.\n", orderResp.OrderID)

	return nil
}

// sumOptionsFees calculates the total regulatory fees for single-leg options orders.
func sumOptionsFees(fees api.OptionsRegulatoryFees) string {
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

// extractOptionsErrorMessage extracts a human-readable message from an API error.
func extractOptionsErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()

	// Try to extract JSON error message from API response
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
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
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
	var chainMinStrike string
	var chainMaxStrike string
	var chainMinOI int
	var chainMinVolume int
	var chainCallsOnly bool
	var chainPutsOnly bool
	var chainStrikes int

	chainCmd := &cobra.Command{
		Use:   "chain SYMBOL",
		Short: "Display option chain",
		Long: `Display the option chain for an underlying symbol and expiration date.

Filtering options:
  --strikes N          Show N strikes centered around ATM (e.g., --strikes 10 shows 5 above, 5 below)
  --min-strike/--max-strike  Filter by strike price range
  --calls-only/--puts-only   Show only one side of the chain
  --min-oi N           Minimum open interest
  --min-volume N       Minimum daily volume

Examples:
  pub options chain AAPL --expiration 2025-01-17                    # Full chain
  pub options chain AAPL -e 2025-01-17 --strikes 10                 # 10 strikes around ATM
  pub options chain AAPL -e 2025-01-17 --calls-only --min-oi 100    # Liquid calls only
  pub options chain AAPL -e 2025-01-17 --min-strike 170 --max-strike 190  # Strike range`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
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
			if chainCallsOnly && chainPutsOnly {
				return fmt.Errorf("cannot use both --calls-only and --puts-only")
			}

			// Build filter
			filter := chainFilter{
				minOI:     chainMinOI,
				minVolume: chainMinVolume,
				callsOnly: chainCallsOnly,
				putsOnly:  chainPutsOnly,
				strikes:   chainStrikes,
			}
			if chainMinStrike != "" {
				if v, err := strconv.ParseFloat(chainMinStrike, 64); err == nil {
					filter.minStrike = v
				} else {
					return fmt.Errorf("invalid --min-strike value: %s", chainMinStrike)
				}
			}
			if chainMaxStrike != "" {
				if v, err := strconv.ParseFloat(chainMaxStrike, 64); err == nil {
					filter.maxStrike = v
				} else {
					return fmt.Errorf("invalid --max-strike value: %s", chainMaxStrike)
				}
			}

			return runOptionsChain(cmd, opts, args[0], chainExpiration, filter)
		},
	}

	chainCmd.Flags().StringVarP(&chainAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	chainCmd.Flags().StringVarP(&chainExpiration, "expiration", "e", "", "Expiration date (YYYY-MM-DD)")
	chainCmd.Flags().IntVar(&chainStrikes, "strikes", 0, "Limit to N strikes around ATM (e.g., 10 shows 5 above, 5 below)")
	chainCmd.Flags().StringVar(&chainMinStrike, "min-strike", "", "Minimum strike price")
	chainCmd.Flags().StringVar(&chainMaxStrike, "max-strike", "", "Maximum strike price")
	chainCmd.Flags().IntVar(&chainMinOI, "min-oi", 0, "Minimum open interest")
	chainCmd.Flags().IntVar(&chainMinVolume, "min-volume", 0, "Minimum daily volume")
	chainCmd.Flags().BoolVar(&chainCallsOnly, "calls-only", false, "Show only calls")
	chainCmd.Flags().BoolVar(&chainPutsOnly, "puts-only", false, "Show only puts")
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
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
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
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
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
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
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

	// Single-leg options buy command
	var buyAccountID string
	var buyParams singleLegParams
	var buySkipConfirm bool
	var buyOpen bool
	var buyClose bool

	buyCmd := &cobra.Command{
		Use:   "buy SYMBOL",
		Short: "Buy options contracts",
		Long: `Place a buy order for options contracts.

The symbol should be in OCC format (e.g., AAPL250117C00175000).
You must specify whether you are opening or closing a position with --open or --close.

Examples:
  pub options buy AAPL250117C00175000 --quantity 1 --limit 2.50 --open --yes    # Buy to open
  pub options buy AAPL250117P00170000 --quantity 1 --limit 1.25 --close --yes   # Buy to close (cover short)
  pub options buy SBUX260220C00100000 -q 8 -l 1.50 --open --yes                 # Buy 8 contracts`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
			if err != nil {
				return err
			}

			if buyAccountID == "" {
				buyAccountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = buyAccountID
			opts.jsonMode = GetJSONMode()

			// Set openClose from flags
			if buyOpen && buyClose {
				return fmt.Errorf("cannot use both --open and --close")
			}
			if buyOpen {
				buyParams.openClose = "OPEN"
			} else if buyClose {
				buyParams.openClose = "CLOSE"
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(config.ConfigPath())
			return runSingleLegOrder(cmd, opts, args[0], "BUY", buyParams, buySkipConfirm, cfg.TradingEnabled)
		},
	}

	buyCmd.Flags().StringVarP(&buyAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	buyCmd.Flags().StringVarP(&buyParams.quantity, "quantity", "q", "", "Number of contracts (required)")
	buyCmd.Flags().StringVarP(&buyParams.limitPrice, "limit", "l", "", "Limit price (required)")
	buyCmd.Flags().StringVarP(&buyParams.expiration, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
	buyCmd.Flags().BoolVar(&buyOpen, "open", false, "Buy to open a new position")
	buyCmd.Flags().BoolVar(&buyClose, "close", false, "Buy to close an existing short position")
	buyCmd.Flags().BoolVarP(&buySkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	buyCmd.SilenceUsage = true

	// Single-leg options sell command
	var sellAccountID string
	var sellParams singleLegParams
	var sellSkipConfirm bool
	var sellOpen bool
	var sellClose bool

	sellCmd := &cobra.Command{
		Use:   "sell SYMBOL",
		Short: "Sell options contracts",
		Long: `Place a sell order for options contracts.

The symbol should be in OCC format (e.g., AAPL250117C00175000).
You must specify whether you are opening or closing a position with --open or --close.

Examples:
  pub options sell AAPL250117C00175000 --quantity 1 --limit 2.50 --close --yes  # Sell to close (exit long)
  pub options sell AAPL250117P00170000 --quantity 1 --limit 1.25 --open --yes   # Sell to open (write option)
  pub options sell SBUX260220C00100000 -q 8 -l 1.50 --close --yes               # Sell 8 contracts`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := keyring.NewEnvStore(keyring.NewSystemStore())
			token, err := api.GetAuthToken(store, cfg.APIBaseURL, false)
			if err != nil {
				return err
			}

			if sellAccountID == "" {
				sellAccountID = cfg.AccountUUID
			}

			opts.baseURL = cfg.APIBaseURL
			opts.authToken = token
			opts.accountID = sellAccountID
			opts.jsonMode = GetJSONMode()

			// Set openClose from flags
			if sellOpen && sellClose {
				return fmt.Errorf("cannot use both --open and --close")
			}
			if sellOpen {
				sellParams.openClose = "OPEN"
			} else if sellClose {
				sellParams.openClose = "CLOSE"
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(config.ConfigPath())
			return runSingleLegOrder(cmd, opts, args[0], "SELL", sellParams, sellSkipConfirm, cfg.TradingEnabled)
		},
	}

	sellCmd.Flags().StringVarP(&sellAccountID, "account", "a", "", "Account ID (uses default if not specified)")
	sellCmd.Flags().StringVarP(&sellParams.quantity, "quantity", "q", "", "Number of contracts (required)")
	sellCmd.Flags().StringVarP(&sellParams.limitPrice, "limit", "l", "", "Limit price (required)")
	sellCmd.Flags().StringVarP(&sellParams.expiration, "expiration", "e", "DAY", "Order expiration: DAY (default) or GTC")
	sellCmd.Flags().BoolVar(&sellOpen, "open", false, "Sell to open a new short position")
	sellCmd.Flags().BoolVar(&sellClose, "close", false, "Sell to close an existing long position")
	sellCmd.Flags().BoolVarP(&sellSkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	sellCmd.SilenceUsage = true

	optionsCmd.AddCommand(expirationsCmd)
	optionsCmd.AddCommand(chainCmd)
	optionsCmd.AddCommand(greeksCmd)
	optionsCmd.AddCommand(multilegCmd)
	optionsCmd.AddCommand(buyCmd)
	optionsCmd.AddCommand(sellCmd)
	rootCmd.AddCommand(optionsCmd)
}
