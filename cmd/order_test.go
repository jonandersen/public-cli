package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderBuyCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/test-account/order", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify order request fields
		assert.NotEmpty(t, req["orderId"])
		assert.Equal(t, "BUY", req["orderSide"])
		assert.Equal(t, "MARKET", req["orderType"])
		assert.Equal(t, "10", req["quantity"])

		instrument := req["instrument"].(map[string]any)
		assert.Equal(t, "AAPL", instrument["symbol"])
		assert.Equal(t, "EQUITY", instrument["type"])

		expiration := req["expiration"].(map[string]any)
		assert.Equal(t, "DAY", expiration["timeInForce"])

		resp := map[string]any{
			"orderId": req["orderId"],
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Order placed")
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "BUY")
}

func TestOrderSellCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "SELL", req["orderSide"])
		assert.Equal(t, "5", req["quantity"])

		instrument := req["instrument"].(map[string]any)
		assert.Equal(t, "AAPL", instrument["symbol"])

		resp := map[string]any{
			"orderId": req["orderId"],
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderSellCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "5", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Order placed")
	assert.Contains(t, output, "SELL")
}

func TestOrderCmd_TradingDisabled(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: false,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trading is disabled")
}

func TestOrderCmd_RequiresAccount(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestOrderCmd_RequiresSymbol(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "arg")
}

func TestOrderCmd_RequiresQuantity(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quantity")
}

func TestOrderCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "insufficient_funds"}`))
	}))
	defer server.Close()

	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestOrderCmd_ShowsPreview(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	// Without --yes flag, should show preview and require confirmation
	cmd.SetArgs([]string{"AAPL", "--quantity", "10"})

	// We expect this to fail since there's no interactive confirmation in test
	// The important thing is it doesn't execute the order without confirmation
	err := cmd.Execute()
	require.Error(t, err)
	// Should show order preview
	output := out.String()
	assert.True(t, strings.Contains(output, "Order Preview") || strings.Contains(err.Error(), "confirmation"))
}

func TestOrderCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		resp := map[string]any{
			"orderId": req["orderId"],
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
		jsonMode:       true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify JSON output
	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result["orderId"])
	assert.Equal(t, "placed", result["status"])
}

func TestOrderCmd_SymbolUppercased(t *testing.T) {
	var receivedSymbol string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		instrument := req["instrument"].(map[string]any)
		receivedSymbol = instrument["symbol"].(string)

		resp := map[string]any{"orderId": req["orderId"]}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"aapl", "--quantity", "10", "--yes"}) // lowercase input

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "AAPL", receivedSymbol) // Should be uppercased
}
