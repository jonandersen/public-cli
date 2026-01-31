package publicapi

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/userapigateway/marketdata/acc-123/quotes", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req QuoteRequest
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Len(t, req.Instruments, 2)
		assert.Equal(t, "AAPL", req.Instruments[0].Symbol)
		assert.Equal(t, "STOCK", req.Instruments[0].Type)
		assert.Equal(t, "GOOGL", req.Instruments[1].Symbol)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"quotes": [
				{
					"instrument": {"symbol": "AAPL", "type": "STOCK"},
					"outcome": "SUCCESS",
					"last": "150.25",
					"lastTimestamp": "2024-01-15T10:30:00Z",
					"bid": "150.20",
					"bidSize": 100,
					"bidTimestamp": "2024-01-15T10:30:00Z",
					"ask": "150.30",
					"askSize": 200,
					"askTimestamp": "2024-01-15T10:30:00Z",
					"volume": 1500000
				},
				{
					"instrument": {"symbol": "GOOGL", "type": "STOCK"},
					"outcome": "SUCCESS",
					"last": "140.50",
					"lastTimestamp": "2024-01-15T10:30:00Z",
					"bid": "140.45",
					"bidSize": 50,
					"bidTimestamp": "2024-01-15T10:30:00Z",
					"ask": "140.55",
					"askSize": 75,
					"askTimestamp": "2024-01-15T10:30:00Z",
					"volume": 2000000
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	instruments := []QuoteInstrument{
		{Symbol: "AAPL", Type: "STOCK"},
		{Symbol: "GOOGL", Type: "STOCK"},
	}

	quotes, err := client.GetQuotes(context.Background(), "acc-123", instruments)
	require.NoError(t, err)
	require.Len(t, quotes, 2)

	// Check first quote
	assert.Equal(t, "AAPL", quotes[0].Instrument.Symbol)
	assert.Equal(t, "SUCCESS", quotes[0].Outcome)
	assert.Equal(t, "150.25", quotes[0].Last)
	assert.Equal(t, "150.20", quotes[0].Bid)
	assert.Equal(t, 100, quotes[0].BidSize)
	assert.Equal(t, "150.30", quotes[0].Ask)
	assert.Equal(t, 200, quotes[0].AskSize)
	assert.Equal(t, int64(1500000), quotes[0].Volume)

	// Check second quote
	assert.Equal(t, "GOOGL", quotes[1].Instrument.Symbol)
	assert.Equal(t, "140.50", quotes[1].Last)
	assert.Equal(t, int64(2000000), quotes[1].Volume)
}

func TestClient_GetQuotes_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"quotes": []}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	quotes, err := client.GetQuotes(context.Background(), "acc-123", []QuoteInstrument{})
	require.NoError(t, err)
	assert.Empty(t, quotes)
}

func TestClient_GetQuotes_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid symbol"}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	instruments := []QuoteInstrument{
		{Symbol: "INVALID", Type: "STOCK"},
	}

	quotes, err := client.GetQuotes(context.Background(), "acc-123", instruments)
	require.Error(t, err)
	assert.Nil(t, quotes)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestClient_GetQuotes_EmptyAccountID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	quotes, err := client.GetQuotes(context.Background(), "", []QuoteInstrument{})
	require.Error(t, err)
	assert.Nil(t, quotes)
	assert.Contains(t, err.Error(), "accountID is required")
}

func TestClient_GetQuotes_WithOpenInterest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"quotes": [
				{
					"instrument": {"symbol": "AAPL250117C00200000", "type": "OPTION"},
					"outcome": "SUCCESS",
					"last": "5.25",
					"lastTimestamp": "2024-01-15T10:30:00Z",
					"bid": "5.20",
					"bidSize": 10,
					"bidTimestamp": "2024-01-15T10:30:00Z",
					"ask": "5.30",
					"askSize": 20,
					"askTimestamp": "2024-01-15T10:30:00Z",
					"volume": 5000,
					"openInterest": 12500
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	instruments := []QuoteInstrument{
		{Symbol: "AAPL250117C00200000", Type: "OPTION"},
	}

	quotes, err := client.GetQuotes(context.Background(), "acc-123", instruments)
	require.NoError(t, err)
	require.Len(t, quotes, 1)

	assert.Equal(t, "OPTION", quotes[0].Instrument.Type)
	require.NotNil(t, quotes[0].OpenInterest)
	assert.Equal(t, int64(12500), *quotes[0].OpenInterest)
}
