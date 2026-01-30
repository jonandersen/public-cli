package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jonandersen/public-cli/pkg/publicapi"
)

// =============================================================================
// Options Types (aliased from pkg/publicapi)
// =============================================================================

type (
	OptionInstrument          = publicapi.OptionInstrument
	OptionExpirationsRequest  = publicapi.OptionExpirationsRequest
	OptionExpirationsResponse = publicapi.OptionExpirationsResponse
	OptionChainRequest        = publicapi.OptionChainRequest
	OptionChainResponse       = publicapi.OptionChainResponse
	OptionQuote               = publicapi.OptionQuote
	GreeksResponse            = publicapi.GreeksResponse
	OptionGreeks              = publicapi.OptionGreeks
	GreeksData                = publicapi.GreeksData
	InstrumentIdentifier      = publicapi.InstrumentIdentifier
	InstrumentResponse        = publicapi.InstrumentResponse
)

// GetOptionExpirations retrieves available option expiration dates for a symbol.
func (c *Client) GetOptionExpirations(ctx context.Context, accountID, symbol string) (*OptionExpirationsResponse, error) {
	reqBody := OptionExpirationsRequest{
		Instrument: OptionInstrument{
			Symbol: strings.ToUpper(symbol),
			Type:   "EQUITY",
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	path := fmt.Sprintf("/userapigateway/marketdata/%s/option-expirations", accountID)
	resp, err := c.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch expirations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var expResp OptionExpirationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&expResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &expResp, nil
}

// GetOptionChain retrieves the option chain for a symbol and expiration date.
func (c *Client) GetOptionChain(ctx context.Context, accountID, symbol, expiration string) (*OptionChainResponse, error) {
	reqBody := OptionChainRequest{
		Instrument: OptionInstrument{
			Symbol: strings.ToUpper(symbol),
			Type:   "EQUITY",
		},
		ExpirationDate: expiration,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	path := fmt.Sprintf("/userapigateway/marketdata/%s/option-chain", accountID)
	resp, err := c.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch option chain: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var chainResp OptionChainResponse
	if err := json.NewDecoder(resp.Body).Decode(&chainResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chainResp, nil
}

// GetOptionGreeks retrieves greeks for the given OSI option symbols.
func (c *Client) GetOptionGreeks(ctx context.Context, accountID string, osiSymbols []string) (*GreeksResponse, error) {
	path := fmt.Sprintf("/userapigateway/option-details/%s/greeks", accountID)

	// Build query parameters
	query := "?"
	for i, sym := range osiSymbols {
		if i > 0 {
			query += "&"
		}
		query += "osiSymbols=" + strings.ToUpper(sym)
	}

	resp, err := c.Get(ctx, path+query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch greeks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var greeksResp GreeksResponse
	if err := json.NewDecoder(resp.Body).Decode(&greeksResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &greeksResp, nil
}

// GetInstrument retrieves trading details for a single instrument.
func (c *Client) GetInstrument(ctx context.Context, symbol, instType string) (*InstrumentResponse, error) {
	symbol = strings.ToUpper(symbol)
	instType = strings.ToUpper(instType)

	path := fmt.Sprintf("/userapigateway/trading/instruments/%s/%s", symbol, instType)
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instrument: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var instResp InstrumentResponse
	if err := json.NewDecoder(resp.Body).Decode(&instResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instResp, nil
}
