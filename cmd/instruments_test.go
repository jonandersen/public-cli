package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstrumentsCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/instruments", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		resp := InstrumentsResponse{
			Instruments: []InstrumentResponse{
				{
					Instrument:          InstrumentIdentifier{Symbol: "AAPL", Type: "EQUITY"},
					Trading:             "BUY_AND_SELL",
					FractionalTrading:   "BUY_AND_SELL",
					OptionTrading:       "BUY_AND_SELL",
					OptionSpreadTrading: "BUY_AND_SELL",
				},
				{
					Instrument:          InstrumentIdentifier{Symbol: "MSFT", Type: "EQUITY"},
					Trading:             "BUY_AND_SELL",
					FractionalTrading:   "BUY_AND_SELL",
					OptionTrading:       "BUY_AND_SELL",
					OptionSpreadTrading: "BUY_AND_SELL",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "MSFT")
	assert.Contains(t, output, "EQUITY")
}

func TestInstrumentsCmd_WithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/instruments", r.URL.Path)
		assert.Equal(t, "CRYPTO", r.URL.Query().Get("typeFilter"))

		resp := InstrumentsResponse{
			Instruments: []InstrumentResponse{
				{
					Instrument:          InstrumentIdentifier{Symbol: "BTC", Type: "CRYPTO"},
					Trading:             "BUY_AND_SELL",
					FractionalTrading:   "BUY_AND_SELL",
					OptionTrading:       "DISABLED",
					OptionSpreadTrading: "DISABLED",
				},
				{
					Instrument:          InstrumentIdentifier{Symbol: "ETH", Type: "CRYPTO"},
					Trading:             "BUY_AND_SELL",
					FractionalTrading:   "BUY_AND_SELL",
					OptionTrading:       "DISABLED",
					OptionSpreadTrading: "DISABLED",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--type", "CRYPTO"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "BTC")
	assert.Contains(t, output, "ETH")
	assert.Contains(t, output, "CRYPTO")
}

func TestInstrumentsCmd_WithTradingFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "LIQUIDATION_ONLY", r.URL.Query().Get("tradingFilter"))

		resp := InstrumentsResponse{
			Instruments: []InstrumentResponse{
				{
					Instrument:          InstrumentIdentifier{Symbol: "DELISTED", Type: "EQUITY"},
					Trading:             "LIQUIDATION_ONLY",
					FractionalTrading:   "DISABLED",
					OptionTrading:       "DISABLED",
					OptionSpreadTrading: "DISABLED",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--trading", "LIQUIDATION_ONLY"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "DELISTED")
	assert.Contains(t, output, "LIQUIDATION_ONLY")
}

func TestInstrumentsCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := InstrumentsResponse{
			Instruments: []InstrumentResponse{
				{
					Instrument:          InstrumentIdentifier{Symbol: "AAPL", Type: "EQUITY"},
					Trading:             "BUY_AND_SELL",
					FractionalTrading:   "BUY_AND_SELL",
					OptionTrading:       "BUY_AND_SELL",
					OptionSpreadTrading: "BUY_AND_SELL",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	var result InstrumentsResponse
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result.Instruments, 1)
	assert.Equal(t, "AAPL", result.Instruments[0].Instrument.Symbol)
	assert.Equal(t, "EQUITY", result.Instruments[0].Instrument.Type)
}

func TestInstrumentsCmd_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := InstrumentsResponse{
			Instruments: []InstrumentResponse{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "No instruments found")
}

func TestInstrumentsCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestInstrumentsCmd_MultipleFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "EQUITY", r.URL.Query().Get("typeFilter"))
		assert.Equal(t, "BUY_AND_SELL", r.URL.Query().Get("tradingFilter"))

		resp := InstrumentsResponse{
			Instruments: []InstrumentResponse{
				{
					Instrument:          InstrumentIdentifier{Symbol: "AAPL", Type: "EQUITY"},
					Trading:             "BUY_AND_SELL",
					FractionalTrading:   "BUY_AND_SELL",
					OptionTrading:       "BUY_AND_SELL",
					OptionSpreadTrading: "BUY_AND_SELL",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentsCmd(instrumentsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--type", "EQUITY", "--trading", "BUY_AND_SELL"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
}
