package publicapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/userapigateway/trading/account", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"accounts": [
				{
					"accountId": "acc-123",
					"accountType": "INDIVIDUAL",
					"optionsLevel": "LEVEL_2",
					"brokerageAccountType": "CASH",
					"tradePermissions": "FULL"
				},
				{
					"accountId": "acc-456",
					"accountType": "IRA",
					"optionsLevel": "LEVEL_1",
					"brokerageAccountType": "MARGIN",
					"tradePermissions": "LIMITED"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	accounts, err := client.GetAccounts(context.Background())
	require.NoError(t, err)
	require.Len(t, accounts, 2)

	assert.Equal(t, "acc-123", accounts[0].AccountID)
	assert.Equal(t, "INDIVIDUAL", accounts[0].AccountType)
	assert.Equal(t, "LEVEL_2", accounts[0].OptionsLevel)
	assert.Equal(t, "CASH", accounts[0].BrokerageAccountType)
	assert.Equal(t, "FULL", accounts[0].TradePermissions)

	assert.Equal(t, "acc-456", accounts[1].AccountID)
	assert.Equal(t, "IRA", accounts[1].AccountType)
}

func TestClient_GetAccounts_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accounts": []}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	accounts, err := client.GetAccounts(context.Background())
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestClient_GetAccounts_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid token"}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	accounts, err := client.GetAccounts(context.Background())
	require.Error(t, err)
	assert.Nil(t, accounts)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "invalid token", apiErr.Message)
}

func TestClient_GetPortfolio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/userapigateway/trading/acc-123/portfolio/v2", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"accountId": "acc-123",
			"accountType": "INDIVIDUAL",
			"buyingPower": {
				"buyingPower": "10000.00",
				"optionsBuyingPower": "5000.00",
				"cashOnlyBuyingPower": "10000.00"
			},
			"equity": [
				{
					"type": "STOCKS",
					"value": "25000.00",
					"percentageOfPortfolio": "71.43"
				},
				{
					"type": "CASH",
					"value": "10000.00",
					"percentageOfPortfolio": "28.57"
				}
			],
			"positions": [
				{
					"instrument": {
						"symbol": "AAPL",
						"name": "Apple Inc.",
						"type": "STOCK"
					},
					"quantity": "10",
					"currentValue": "1500.00",
					"percentOfPortfolio": "6.00",
					"lastPrice": {
						"lastPrice": "150.00",
						"timestamp": "2024-01-15T10:30:00Z"
					},
					"instrumentGain": {
						"gainValue": "100.00",
						"gainPercentage": "7.14",
						"timestamp": "2024-01-15T10:30:00Z"
					},
					"positionDailyGain": {
						"gainValue": "25.00",
						"gainPercentage": "1.69",
						"timestamp": "2024-01-15T10:30:00Z"
					},
					"costBasis": {
						"totalCost": "1400.00",
						"unitCost": "140.00",
						"gainValue": "100.00",
						"gainPercentage": "7.14",
						"lastUpdate": "2024-01-15T10:30:00Z"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	portfolio, err := client.GetPortfolio(context.Background(), "acc-123")
	require.NoError(t, err)
	require.NotNil(t, portfolio)

	assert.Equal(t, "acc-123", portfolio.AccountID)
	assert.Equal(t, "INDIVIDUAL", portfolio.AccountType)

	// Check buying power
	assert.Equal(t, "10000.00", portfolio.BuyingPower.BuyingPower)
	assert.Equal(t, "5000.00", portfolio.BuyingPower.OptionsBuyingPower)
	assert.Equal(t, "10000.00", portfolio.BuyingPower.CashOnlyBuyingPower)

	// Check equity
	require.Len(t, portfolio.Equity, 2)
	assert.Equal(t, "STOCKS", portfolio.Equity[0].Type)
	assert.Equal(t, "25000.00", portfolio.Equity[0].Value)

	// Check positions
	require.Len(t, portfolio.Positions, 1)
	pos := portfolio.Positions[0]
	assert.Equal(t, "AAPL", pos.Instrument.Symbol)
	assert.Equal(t, "Apple Inc.", pos.Instrument.Name)
	assert.Equal(t, "10", pos.Quantity)
	assert.Equal(t, "1500.00", pos.CurrentValue)
	assert.Equal(t, "150.00", pos.LastPrice.LastPrice)
	assert.Equal(t, "25.00", pos.PositionDailyGain.GainValue)
	assert.Equal(t, "100.00", pos.CostBasis.GainValue)
}

func TestClient_GetPortfolio_NoPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"accountId": "acc-123",
			"accountType": "INDIVIDUAL",
			"buyingPower": {
				"buyingPower": "10000.00",
				"optionsBuyingPower": "5000.00",
				"cashOnlyBuyingPower": "10000.00"
			},
			"equity": [],
			"positions": []
		}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	portfolio, err := client.GetPortfolio(context.Background(), "acc-123")
	require.NoError(t, err)
	require.NotNil(t, portfolio)

	assert.Empty(t, portfolio.Positions)
	assert.Empty(t, portfolio.Equity)
}

func TestClient_GetPortfolio_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "account not found"}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	portfolio, err := client.GetPortfolio(context.Background(), "invalid-acc")
	require.Error(t, err)
	assert.Nil(t, portfolio)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.True(t, apiErr.IsNotFound())
}

func TestClient_GetPortfolio_EmptyAccountID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	portfolio, err := client.GetPortfolio(context.Background(), "")
	require.Error(t, err)
	assert.Nil(t, portfolio)
	assert.Contains(t, err.Error(), "accountID is required")
}
