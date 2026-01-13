package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// OptionInstrument represents an instrument for options requests.
type OptionInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// OptionExpirationsRequest represents a request for option expirations.
type OptionExpirationsRequest struct {
	Instrument OptionInstrument `json:"instrument"`
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
