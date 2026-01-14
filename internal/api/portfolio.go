package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// GetPortfolio retrieves the portfolio for the given account ID.
func (c *Client) GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error) {
	path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", accountID)
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var portfolio Portfolio
	if err := json.NewDecoder(resp.Body).Decode(&portfolio); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &portfolio, nil
}
