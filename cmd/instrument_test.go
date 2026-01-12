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

func TestInstrumentCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/instruments/AAPL/EQUITY", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		resp := InstrumentResponse{
			Instrument: InstrumentIdentifier{
				Symbol: "AAPL",
				Type:   "EQUITY",
			},
			Trading:             "BUY_AND_SELL",
			FractionalTrading:   "BUY_AND_SELL",
			OptionTrading:       "BUY_AND_SELL",
			OptionSpreadTrading: "BUY_AND_SELL",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentCmd(instrumentOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "EQUITY")
	assert.Contains(t, output, "BUY_AND_SELL")
}

func TestInstrumentCmd_WithType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/instruments/BTC/CRYPTO", r.URL.Path)

		resp := InstrumentResponse{
			Instrument: InstrumentIdentifier{
				Symbol: "BTC",
				Type:   "CRYPTO",
			},
			Trading:             "BUY_AND_SELL",
			FractionalTrading:   "BUY_AND_SELL",
			OptionTrading:       "DISABLED",
			OptionSpreadTrading: "DISABLED",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentCmd(instrumentOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"BTC", "--type", "CRYPTO"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "BTC")
	assert.Contains(t, output, "CRYPTO")
}

func TestInstrumentCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := InstrumentResponse{
			Instrument: InstrumentIdentifier{
				Symbol: "AAPL",
				Type:   "EQUITY",
			},
			Trading:             "BUY_AND_SELL",
			FractionalTrading:   "BUY_AND_SELL",
			OptionTrading:       "BUY_AND_SELL",
			OptionSpreadTrading: "BUY_AND_SELL",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentCmd(instrumentOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result InstrumentResponse
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", result.Instrument.Symbol)
	assert.Equal(t, "EQUITY", result.Instrument.Type)
	assert.Equal(t, "BUY_AND_SELL", result.Trading)
}

func TestInstrumentCmd_NoSymbol(t *testing.T) {
	cmd := newInstrumentCmd(instrumentOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

func TestInstrumentCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "instrument not found"}`))
	}))
	defer server.Close()

	cmd := newInstrumentCmd(instrumentOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"INVALID"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestInstrumentCmd_LiquidationOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := InstrumentResponse{
			Instrument: InstrumentIdentifier{
				Symbol: "DELISTED",
				Type:   "EQUITY",
			},
			Trading:             "LIQUIDATION_ONLY",
			FractionalTrading:   "DISABLED",
			OptionTrading:       "DISABLED",
			OptionSpreadTrading: "DISABLED",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newInstrumentCmd(instrumentOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"DELISTED"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "LIQUIDATION_ONLY")
	assert.Contains(t, output, "DISABLED")
}
