package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetQuotes(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		instruments    []QuoteInstrument
		statusCode     int
		responseBody   string
		wantErr        bool
		wantErrContain string
		validate       func(t *testing.T, quotes []Quote)
	}{
		{
			name:      "single quote success",
			accountID: "test-account-123",
			instruments: []QuoteInstrument{
				{Symbol: "AAPL", Type: "EQUITY"},
			},
			statusCode: 200,
			responseBody: `{
				"quotes": [
					{
						"instrument": {"symbol": "AAPL", "type": "EQUITY"},
						"outcome": "SUCCESS",
						"last": "150.00",
						"lastTimestamp": "2024-01-01T12:00:00Z",
						"bid": "149.95",
						"bidSize": 100,
						"bidTimestamp": "2024-01-01T12:00:00Z",
						"ask": "150.05",
						"askSize": 200,
						"askTimestamp": "2024-01-01T12:00:00Z",
						"volume": 1000000
					}
				]
			}`,
			validate: func(t *testing.T, quotes []Quote) {
				require.Len(t, quotes, 1)
				assert.Equal(t, "AAPL", quotes[0].Instrument.Symbol)
				assert.Equal(t, "SUCCESS", quotes[0].Outcome)
				assert.Equal(t, "150.00", quotes[0].Last)
				assert.Equal(t, "149.95", quotes[0].Bid)
				assert.Equal(t, "150.05", quotes[0].Ask)
				assert.Equal(t, int64(1000000), quotes[0].Volume)
			},
		},
		{
			name:      "multiple quotes success",
			accountID: "test-account-123",
			instruments: []QuoteInstrument{
				{Symbol: "AAPL", Type: "EQUITY"},
				{Symbol: "GOOGL", Type: "EQUITY"},
				{Symbol: "MSFT", Type: "EQUITY"},
			},
			statusCode: 200,
			responseBody: `{
				"quotes": [
					{
						"instrument": {"symbol": "AAPL", "type": "EQUITY"},
						"outcome": "SUCCESS",
						"last": "150.00",
						"bid": "149.95",
						"ask": "150.05",
						"volume": 1000000
					},
					{
						"instrument": {"symbol": "GOOGL", "type": "EQUITY"},
						"outcome": "SUCCESS",
						"last": "140.00",
						"bid": "139.95",
						"ask": "140.05",
						"volume": 500000
					},
					{
						"instrument": {"symbol": "MSFT", "type": "EQUITY"},
						"outcome": "SUCCESS",
						"last": "380.00",
						"bid": "379.95",
						"ask": "380.05",
						"volume": 750000
					}
				]
			}`,
			validate: func(t *testing.T, quotes []Quote) {
				require.Len(t, quotes, 3)
				assert.Equal(t, "AAPL", quotes[0].Instrument.Symbol)
				assert.Equal(t, "GOOGL", quotes[1].Instrument.Symbol)
				assert.Equal(t, "MSFT", quotes[2].Instrument.Symbol)
			},
		},
		{
			name:      "quote not found",
			accountID: "test-account-123",
			instruments: []QuoteInstrument{
				{Symbol: "INVALID", Type: "EQUITY"},
			},
			statusCode: 200,
			responseBody: `{
				"quotes": [
					{
						"instrument": {"symbol": "INVALID", "type": "EQUITY"},
						"outcome": "NOT_FOUND"
					}
				]
			}`,
			validate: func(t *testing.T, quotes []Quote) {
				require.Len(t, quotes, 1)
				assert.Equal(t, "INVALID", quotes[0].Instrument.Symbol)
				assert.Equal(t, "NOT_FOUND", quotes[0].Outcome)
			},
		},
		{
			name:        "empty instruments",
			accountID:   "test-account-123",
			instruments: []QuoteInstrument{},
			statusCode:  200,
			responseBody: `{
				"quotes": []
			}`,
			validate: func(t *testing.T, quotes []Quote) {
				assert.Empty(t, quotes)
			},
		},
		{
			name:      "API error 401",
			accountID: "test-account-123",
			instruments: []QuoteInstrument{
				{Symbol: "AAPL", Type: "EQUITY"},
			},
			statusCode:     401,
			responseBody:   `{"error": "unauthorized"}`,
			wantErr:        true,
			wantErrContain: "API error: 401",
		},
		{
			name:      "API error 500",
			accountID: "test-account-123",
			instruments: []QuoteInstrument{
				{Symbol: "AAPL", Type: "EQUITY"},
			},
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
				expectedPath := "/userapigateway/marketdata/" + tt.accountID + "/quotes"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var reqBody QuoteRequest
				err = json.Unmarshal(body, &reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.instruments, reqBody.Instruments)

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			quotes, err := client.GetQuotes(context.Background(), tt.accountID, tt.instruments)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContain)
				assert.Nil(t, quotes)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, quotes)
			if tt.validate != nil {
				tt.validate(t, quotes)
			}
		})
	}
}

func TestClient_GetQuotes_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:1", "test-token")
	quotes, err := client.GetQuotes(context.Background(), "test-account", []QuoteInstrument{
		{Symbol: "AAPL", Type: "EQUITY"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch quotes")
	assert.Nil(t, quotes)
}

func TestClient_GetQuotes_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	quotes, err := client.GetQuotes(context.Background(), "test-account", []QuoteInstrument{
		{Symbol: "AAPL", Type: "EQUITY"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
	assert.Nil(t, quotes)
}
