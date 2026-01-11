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

func TestOrderCancelCmd_Success(t *testing.T) {
	orderID := "912710f1-1a45-4ef0-88a7-cd513781933d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/test-account/order/"+orderID, r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cmd := newOrderCancelCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{orderID, "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Cancel request submitted")
	assert.Contains(t, output, orderID)
}

func TestOrderCancelCmd_RequiresOrderID(t *testing.T) {
	cmd := newOrderCancelCmd(orderOptions{
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

func TestOrderCancelCmd_RequiresConfirmation(t *testing.T) {
	cmd := newOrderCancelCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation")
}

func TestOrderCancelCmd_TradingDisabled(t *testing.T) {
	cmd := newOrderCancelCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: false,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trading is disabled")
}

func TestOrderCancelCmd_RequiresAccount(t *testing.T) {
	cmd := newOrderCancelCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestOrderCancelCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "order_not_found"}`))
	}))
	defer server.Close()

	cmd := newOrderCancelCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d", "--yes"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestOrderCancelCmd_JSON(t *testing.T) {
	orderID := "912710f1-1a45-4ef0-88a7-cd513781933d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cmd := newOrderCancelCmd(orderOptions{
		baseURL:        server.URL,
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
		jsonMode:       true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{orderID, "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, orderID, result["orderId"])
	assert.Equal(t, "cancel_requested", result["status"])
}

func TestOrderStatusCmd_Success(t *testing.T) {
	orderID := "912710f1-1a45-4ef0-88a7-cd513781933d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/test-account/order/"+orderID, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		resp := map[string]any{
			"orderId": orderID,
			"instrument": map[string]any{
				"symbol": "AAPL",
				"type":   "EQUITY",
			},
			"createdAt":      "2025-01-10T10:30:00Z",
			"type":           "LIMIT",
			"side":           "BUY",
			"status":         "FILLED",
			"quantity":       "10",
			"limitPrice":     "175.00",
			"filledQuantity": "10",
			"averagePrice":   "174.95",
			"closedAt":       "2025-01-10T10:30:05Z",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{orderID})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, orderID)
	assert.Contains(t, output, "FILLED")
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "BUY")
}

func TestOrderStatusCmd_PartiallyFilled(t *testing.T) {
	orderID := "912710f1-1a45-4ef0-88a7-cd513781933d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId": orderID,
			"instrument": map[string]any{
				"symbol": "AAPL",
				"type":   "EQUITY",
			},
			"createdAt":      "2025-01-10T10:30:00Z",
			"type":           "LIMIT",
			"side":           "BUY",
			"status":         "PARTIALLY_FILLED",
			"quantity":       "10",
			"limitPrice":     "175.00",
			"filledQuantity": "5",
			"averagePrice":   "174.95",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{orderID})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "PARTIALLY_FILLED")
	assert.Contains(t, output, "5")  // filledQuantity
	assert.Contains(t, output, "10") // total quantity
}

func TestOrderStatusCmd_JSON(t *testing.T) {
	orderID := "912710f1-1a45-4ef0-88a7-cd513781933d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId": orderID,
			"instrument": map[string]any{
				"symbol": "AAPL",
				"type":   "EQUITY",
			},
			"createdAt":      "2025-01-10T10:30:00Z",
			"type":           "LIMIT",
			"side":           "BUY",
			"status":         "FILLED",
			"quantity":       "10",
			"limitPrice":     "175.00",
			"filledQuantity": "10",
			"averagePrice":   "174.95",
			"closedAt":       "2025-01-10T10:30:05Z",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{orderID})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, orderID, result["orderId"])
	assert.Equal(t, "FILLED", result["status"])
	assert.Equal(t, "10", result["filledQuantity"])
	assert.Equal(t, "174.95", result["averagePrice"])
}

func TestOrderStatusCmd_RequiresOrderID(t *testing.T) {
	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "arg")
}

func TestOrderStatusCmd_RequiresAccount(t *testing.T) {
	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestOrderStatusCmd_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "order_not_found"}`))
	}))
	defer server.Close()

	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestOrderStatusCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal_error"}`))
	}))
	defer server.Close()

	cmd := newOrderStatusCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"912710f1-1a45-4ef0-88a7-cd513781933d"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOrderListCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/test-account/portfolio/v2", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		resp := map[string]any{
			"accountId": "test-account",
			"orders": []map[string]any{
				{
					"orderId": "order-1",
					"instrument": map[string]any{
						"symbol": "AAPL",
						"type":   "EQUITY",
					},
					"side":           "BUY",
					"type":           "LIMIT",
					"status":         "NEW",
					"quantity":       "10",
					"filledQuantity": "0",
					"limitPrice":     "175.00",
					"createdAt":      "2025-01-10T10:30:00Z",
				},
				{
					"orderId": "order-2",
					"instrument": map[string]any{
						"symbol": "TSLA",
						"type":   "EQUITY",
					},
					"side":           "SELL",
					"type":           "MARKET",
					"status":         "PARTIALLY_FILLED",
					"quantity":       "5",
					"filledQuantity": "3",
					"createdAt":      "2025-01-10T11:00:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderListCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "order-1")
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "BUY")
	assert.Contains(t, output, "order-2")
	assert.Contains(t, output, "TSLA")
	assert.Contains(t, output, "SELL")
}

func TestOrderListCmd_NoOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accountId": "test-account",
			"orders":    []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderListCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "No open orders")
}

func TestOrderListCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accountId": "test-account",
			"orders": []map[string]any{
				{
					"orderId": "order-1",
					"instrument": map[string]any{
						"symbol": "AAPL",
						"type":   "EQUITY",
					},
					"side":           "BUY",
					"type":           "LIMIT",
					"status":         "NEW",
					"quantity":       "10",
					"filledQuantity": "0",
					"limitPrice":     "175.00",
					"createdAt":      "2025-01-10T10:30:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOrderListCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "order-1", result[0]["orderId"])
	assert.Equal(t, "BUY", result[0]["side"])
}

func TestOrderListCmd_RequiresAccount(t *testing.T) {
	cmd := newOrderListCmd(orderOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestOrderListCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal_error"}`))
	}))
	defer server.Close()

	cmd := newOrderListCmd(orderOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOrderBuyCmd_LimitOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify limit order fields
		assert.Equal(t, "LIMIT", req["orderType"])
		assert.Equal(t, "175.50", req["limitPrice"])
		assert.Equal(t, "BUY", req["orderSide"])
		assert.Equal(t, "10", req["quantity"])

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
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--limit", "175.50", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Order placed")
	assert.Contains(t, output, "LIMIT")
}

func TestOrderSellCmd_StopOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify stop order fields
		assert.Equal(t, "STOP", req["orderType"])
		assert.Equal(t, "145.00", req["stopPrice"])
		assert.Equal(t, "SELL", req["orderSide"])

		resp := map[string]any{"orderId": req["orderId"]}
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
	cmd.SetArgs([]string{"AAPL", "--quantity", "5", "--stop", "145.00", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Order placed")
	assert.Contains(t, output, "STOP")
}

func TestOrderBuyCmd_StopLimitOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify stop-limit order fields
		assert.Equal(t, "STOP_LIMIT", req["orderType"])
		assert.Equal(t, "175.00", req["limitPrice"])
		assert.Equal(t, "174.00", req["stopPrice"])

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
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--limit", "175.00", "--stop", "174.00", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Order placed")
	assert.Contains(t, output, "STOP_LIMIT")
}

func TestOrderBuyCmd_GTC_Expiration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		expiration := req["expiration"].(map[string]any)
		assert.Equal(t, "GTC", expiration["timeInForce"])

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
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--limit", "175.00", "--expiration", "GTC", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOrderCmd_LimitOrderPreview(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--limit", "175.50"})

	// Without --yes, should show preview
	err := cmd.Execute()
	require.Error(t, err)

	output := out.String()
	assert.Contains(t, output, "Order Preview")
	assert.Contains(t, output, "LIMIT")
	assert.Contains(t, output, "175.50")
}

func TestOrderCmd_StopLimitOrderPreview(t *testing.T) {
	cmd := newOrderBuyCmd(orderOptions{
		baseURL:        "http://localhost",
		authToken:      "test-token",
		accountID:      "test-account",
		tradingEnabled: true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--limit", "175.00", "--stop", "174.00"})

	err := cmd.Execute()
	require.Error(t, err)

	output := out.String()
	assert.Contains(t, output, "Order Preview")
	assert.Contains(t, output, "STOP_LIMIT")
	assert.Contains(t, output, "175.00") // limit price
	assert.Contains(t, output, "174.00") // stop price
}

func TestOrderBuyCmd_LimitOrderJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
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
		jsonMode:       true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "--quantity", "10", "--limit", "175.50", "--yes"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "LIMIT", result["orderType"])
	assert.Equal(t, "175.50", result["limitPrice"])
}
