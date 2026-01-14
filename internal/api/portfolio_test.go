package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetPortfolio(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		statusCode     int
		responseBody   string
		wantErr        bool
		wantErrContain string
		validate       func(t *testing.T, portfolio *Portfolio)
	}{
		{
			name:       "successful response",
			accountID:  "test-account-123",
			statusCode: 200,
			responseBody: `{
				"accountId": "test-account-123",
				"accountType": "INDIVIDUAL",
				"buyingPower": {
					"cashOnlyBuyingPower": "1000.00",
					"buyingPower": "2000.00",
					"optionsBuyingPower": "500.00"
				},
				"equity": [
					{"type": "CASH", "value": "1000.00", "percentageOfPortfolio": "50"},
					{"type": "STOCKS", "value": "1000.00", "percentageOfPortfolio": "50"}
				],
				"positions": [
					{
						"instrument": {"symbol": "AAPL", "name": "Apple Inc.", "type": "EQUITY"},
						"quantity": "10",
						"currentValue": "1500.00",
						"percentOfPortfolio": "75",
						"lastPrice": {"lastPrice": "150.00", "timestamp": "2024-01-01T12:00:00Z"},
						"instrumentGain": {"gainValue": "100.00", "gainPercentage": "7.14", "timestamp": "2024-01-01"},
						"positionDailyGain": {"gainValue": "25.00", "gainPercentage": "1.69", "timestamp": "2024-01-01"},
						"costBasis": {"totalCost": "1400.00", "unitCost": "140.00", "gainValue": "100.00", "gainPercentage": "7.14", "lastUpdate": "2024-01-01"}
					}
				]
			}`,
			validate: func(t *testing.T, portfolio *Portfolio) {
				assert.Equal(t, "test-account-123", portfolio.AccountID)
				assert.Equal(t, "INDIVIDUAL", portfolio.AccountType)
				assert.Equal(t, "2000.00", portfolio.BuyingPower.BuyingPower)
				assert.Equal(t, "500.00", portfolio.BuyingPower.OptionsBuyingPower)
				assert.Len(t, portfolio.Equity, 2)
				assert.Len(t, portfolio.Positions, 1)
				assert.Equal(t, "AAPL", portfolio.Positions[0].Instrument.Symbol)
				assert.Equal(t, "10", portfolio.Positions[0].Quantity)
				assert.Equal(t, "150.00", portfolio.Positions[0].LastPrice.LastPrice)
			},
		},
		{
			name:       "empty portfolio",
			accountID:  "test-account-empty",
			statusCode: 200,
			responseBody: `{
				"accountId": "test-account-empty",
				"accountType": "INDIVIDUAL",
				"buyingPower": {
					"cashOnlyBuyingPower": "0",
					"buyingPower": "0",
					"optionsBuyingPower": "0"
				},
				"equity": [],
				"positions": []
			}`,
			validate: func(t *testing.T, portfolio *Portfolio) {
				assert.Equal(t, "test-account-empty", portfolio.AccountID)
				assert.Empty(t, portfolio.Positions)
				assert.Empty(t, portfolio.Equity)
			},
		},
		{
			name:           "API error 401",
			accountID:      "test-account-123",
			statusCode:     401,
			responseBody:   `{"error": "unauthorized"}`,
			wantErr:        true,
			wantErrContain: "API error: 401",
		},
		{
			name:           "API error 404",
			accountID:      "invalid-account",
			statusCode:     404,
			responseBody:   `{"error": "account not found"}`,
			wantErr:        true,
			wantErrContain: "API error: 404",
		},
		{
			name:           "API error 500",
			accountID:      "test-account-123",
			statusCode:     500,
			responseBody:   `{"error": "internal server error"}`,
			wantErr:        true,
			wantErrContain: "API error: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path
				expectedPath := "/userapigateway/trading/" + tt.accountID + "/portfolio/v2"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			portfolio, err := client.GetPortfolio(context.Background(), tt.accountID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContain)
				assert.Nil(t, portfolio)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, portfolio)
			if tt.validate != nil {
				tt.validate(t, portfolio)
			}
		})
	}
}

func TestClient_GetPortfolio_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:1", "test-token")
	portfolio, err := client.GetPortfolio(context.Background(), "test-account")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch portfolio")
	assert.Nil(t, portfolio)
}

func TestClient_GetPortfolio_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	portfolio, err := client.GetPortfolio(context.Background(), "test-account")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
	assert.Nil(t, portfolio)
}
