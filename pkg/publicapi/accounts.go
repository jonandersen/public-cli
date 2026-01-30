package publicapi

import (
	"context"
	"fmt"
)

// GetAccounts retrieves all accounts for the authenticated user.
func (c *Client) GetAccounts(ctx context.Context) ([]Account, error) {
	resp, err := c.Get(ctx, "/userapigateway/trading/account")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var result AccountsResponse
	if err := DecodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return result.Accounts, nil
}

// GetPortfolio retrieves the portfolio for a specific account.
func (c *Client) GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID is required")
	}

	path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", accountID)
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var portfolio Portfolio
	if err := DecodeJSON(resp, &portfolio); err != nil {
		return nil, err
	}

	return &portfolio, nil
}
