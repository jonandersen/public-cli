package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// GetQuotes retrieves quotes for the given instruments.
func (c *Client) GetQuotes(ctx context.Context, accountID string, instruments []QuoteInstrument) ([]Quote, error) {
	reqBody := QuoteRequest{Instruments: instruments}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	path := fmt.Sprintf("/userapigateway/marketdata/%s/quotes", accountID)
	resp, err := c.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var quotesResp QuotesResponse
	if err := json.NewDecoder(resp.Body).Decode(&quotesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return quotesResp.Quotes, nil
}
