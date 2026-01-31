package publicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// GetQuotes retrieves quotes for the given instruments.
func (c *Client) GetQuotes(ctx context.Context, accountID string, instruments []QuoteInstrument) ([]Quote, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID is required")
	}

	reqBody := QuoteRequest{Instruments: instruments}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	path := fmt.Sprintf("/userapigateway/marketdata/%s/quotes", accountID)
	resp, err := c.Post(ctx, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var quotesResp QuotesResponse
	if err := DecodeJSON(resp, &quotesResp); err != nil {
		return nil, err
	}

	return quotesResp.Quotes, nil
}
