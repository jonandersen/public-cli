package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetOptionExpirations_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/userapigateway/marketdata/test-account/option-expirations", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req OptionExpirationsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "AAPL", req.Instrument.Symbol)
		assert.Equal(t, "EQUITY", req.Instrument.Type)

		resp := OptionExpirationsResponse{
			BaseSymbol:  "AAPL",
			Expirations: []string{"2025-01-17", "2025-01-24", "2025-01-31"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetOptionExpirations(context.Background(), "test-account", "AAPL")

	require.NoError(t, err)
	assert.Equal(t, "AAPL", resp.BaseSymbol)
	assert.Len(t, resp.Expirations, 3)
	assert.Equal(t, "2025-01-17", resp.Expirations[0])
}

func TestClient_GetOptionExpirations_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid symbol"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetOptionExpirations(context.Background(), "test-account", "INVALID")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestClient_GetOptionChain_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/userapigateway/marketdata/test-account/option-chain", r.URL.Path)

		var req OptionChainRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "AAPL", req.Instrument.Symbol)
		assert.Equal(t, "2025-01-17", req.ExpirationDate)

		resp := OptionChainResponse{
			BaseSymbol: "AAPL",
			Calls: []OptionQuote{
				{
					Instrument:   OptionInstrument{Symbol: "AAPL250117C00175000", Type: "OPTION"},
					Bid:          "2.50",
					Ask:          "2.55",
					Volume:       1000,
					OpenInterest: 5000,
				},
			},
			Puts: []OptionQuote{
				{
					Instrument:   OptionInstrument{Symbol: "AAPL250117P00175000", Type: "OPTION"},
					Bid:          "1.80",
					Ask:          "1.85",
					Volume:       800,
					OpenInterest: 3000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetOptionChain(context.Background(), "test-account", "AAPL", "2025-01-17")

	require.NoError(t, err)
	assert.Equal(t, "AAPL", resp.BaseSymbol)
	assert.Len(t, resp.Calls, 1)
	assert.Len(t, resp.Puts, 1)
	assert.Equal(t, "AAPL250117C00175000", resp.Calls[0].Instrument.Symbol)
	assert.Equal(t, "2.50", resp.Calls[0].Bid)
}

func TestClient_GetOptionChain_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "no options available"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetOptionChain(context.Background(), "test-account", "AAPL", "2025-01-17")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestClient_GetOptionGreeks_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/userapigateway/option-details/test-account/greeks", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "osiSymbols=AAPL250117C00175000")
		assert.Contains(t, r.URL.RawQuery, "osiSymbols=AAPL250117P00175000")

		resp := GreeksResponse{
			Greeks: []OptionGreeks{
				{
					Symbol: "AAPL250117C00175000",
					Greeks: GreeksData{
						Delta:             "0.55",
						Gamma:             "0.08",
						Theta:             "-0.12",
						Vega:              "0.25",
						Rho:               "0.05",
						ImpliedVolatility: "0.35",
					},
				},
				{
					Symbol: "AAPL250117P00175000",
					Greeks: GreeksData{
						Delta:             "-0.45",
						Gamma:             "0.08",
						Theta:             "-0.10",
						Vega:              "0.24",
						Rho:               "-0.04",
						ImpliedVolatility: "0.36",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetOptionGreeks(context.Background(), "test-account", []string{"AAPL250117C00175000", "AAPL250117P00175000"})

	require.NoError(t, err)
	assert.Len(t, resp.Greeks, 2)
	assert.Equal(t, "AAPL250117C00175000", resp.Greeks[0].Symbol)
	assert.Equal(t, "0.55", resp.Greeks[0].Greeks.Delta)
	assert.Equal(t, "-0.45", resp.Greeks[1].Greeks.Delta)
}

func TestClient_GetOptionGreeks_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetOptionGreeks(context.Background(), "test-account", []string{"INVALID"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_GetInstrument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/userapigateway/trading/instruments/AAPL/EQUITY", r.URL.Path)

		resp := InstrumentResponse{
			Instrument:          InstrumentIdentifier{Symbol: "AAPL", Type: "EQUITY"},
			Trading:             "BUY_AND_SELL",
			FractionalTrading:   "ENABLED",
			OptionTrading:       "ENABLED",
			OptionSpreadTrading: "ENABLED",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetInstrument(context.Background(), "AAPL", "EQUITY")

	require.NoError(t, err)
	assert.Equal(t, "AAPL", resp.Instrument.Symbol)
	assert.Equal(t, "EQUITY", resp.Instrument.Type)
	assert.Equal(t, "BUY_AND_SELL", resp.Trading)
	assert.Equal(t, "ENABLED", resp.OptionTrading)
}

func TestClient_GetInstrument_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "instrument not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetInstrument(context.Background(), "INVALID", "EQUITY")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestClient_GetInstrument_CryptoType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/instruments/BTC/CRYPTO", r.URL.Path)

		resp := InstrumentResponse{
			Instrument:          InstrumentIdentifier{Symbol: "BTC", Type: "CRYPTO"},
			Trading:             "BUY_AND_SELL",
			FractionalTrading:   "ENABLED",
			OptionTrading:       "DISABLED",
			OptionSpreadTrading: "DISABLED",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetInstrument(context.Background(), "BTC", "CRYPTO")

	require.NoError(t, err)
	assert.Equal(t, "BTC", resp.Instrument.Symbol)
	assert.Equal(t, "CRYPTO", resp.Instrument.Type)
	assert.Equal(t, "DISABLED", resp.OptionTrading)
}
