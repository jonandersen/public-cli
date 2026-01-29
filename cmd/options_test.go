package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonandersen/public-cli/internal/api"
)

func TestOptionsExpirationsCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/marketdata/test-account/option-expirations", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify request body
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		inst := req["instrument"].(map[string]any)
		assert.Equal(t, "AAPL", inst["symbol"])
		assert.Equal(t, "EQUITY", inst["type"])

		resp := map[string]any{
			"baseSymbol": "AAPL",
			"expirations": []string{
				"2025-01-17",
				"2025-01-24",
				"2025-01-31",
				"2025-02-21",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsExpirationsCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "2025-01-17")
	assert.Contains(t, output, "2025-01-24")
	assert.Contains(t, output, "2025-01-31")
	assert.Contains(t, output, "2025-02-21")
}

func TestOptionsExpirationsCmd_LowercaseSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		inst := req["instrument"].(map[string]any)
		assert.Equal(t, "AAPL", inst["symbol"]) // Should be uppercased

		resp := map[string]any{
			"baseSymbol":  "AAPL",
			"expirations": []string{"2025-01-17"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsExpirationsCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"aapl"}) // lowercase input

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOptionsExpirationsCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"baseSymbol": "AAPL",
			"expirations": []string{
				"2025-01-17",
				"2025-01-24",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsExpirationsCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify JSON output
	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", result["baseSymbol"])
	expirations := result["expirations"].([]any)
	assert.Len(t, expirations, 2)
	assert.Equal(t, "2025-01-17", expirations[0])
}

func TestOptionsExpirationsCmd_NoSymbol(t *testing.T) {
	cmd := newOptionsExpirationsCmd(optionsOptions{
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
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestOptionsExpirationsCmd_RequiresAccount(t *testing.T) {
	cmd := newOptionsExpirationsCmd(optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "", // No account
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestOptionsExpirationsCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	cmd := newOptionsExpirationsCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestOptionsExpirationsCmd_NoExpirations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"baseSymbol":  "XYZ",
			"expirations": []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsExpirationsCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"XYZ"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "No expirations")
}

func TestOptionsChainCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/marketdata/test-account/option-chain", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify request body
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		inst := req["instrument"].(map[string]any)
		assert.Equal(t, "AAPL", inst["symbol"])
		assert.Equal(t, "EQUITY", inst["type"])
		assert.Equal(t, "2025-01-17", req["expirationDate"])

		resp := map[string]any{
			"baseSymbol": "AAPL",
			"calls": []map[string]any{
				{
					"instrument": map[string]string{
						"symbol": "AAPL250117C00175000",
						"type":   "OPTION",
					},
					"outcome":      "SUCCESS",
					"last":         "5.50",
					"bid":          "5.45",
					"bidSize":      50,
					"ask":          "5.55",
					"askSize":      100,
					"volume":       1000,
					"openInterest": 5000,
				},
				{
					"instrument": map[string]string{
						"symbol": "AAPL250117C00180000",
						"type":   "OPTION",
					},
					"outcome":      "SUCCESS",
					"last":         "3.25",
					"bid":          "3.20",
					"bidSize":      25,
					"ask":          "3.30",
					"askSize":      50,
					"volume":       500,
					"openInterest": 2500,
				},
			},
			"puts": []map[string]any{
				{
					"instrument": map[string]string{
						"symbol": "AAPL250117P00175000",
						"type":   "OPTION",
					},
					"outcome":      "SUCCESS",
					"last":         "4.50",
					"bid":          "4.45",
					"bidSize":      30,
					"ask":          "4.55",
					"askSize":      75,
					"volume":       800,
					"openInterest": 3000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "--expiration", "2025-01-17"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "2025-01-17")
	assert.Contains(t, output, "CALLS")
	assert.Contains(t, output, "PUTS")
	assert.Contains(t, output, "175")
	assert.Contains(t, output, "180")
	assert.Contains(t, output, "5.45")
	assert.Contains(t, output, "5.55")
}

func TestOptionsChainCmd_LowercaseSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		inst := req["instrument"].(map[string]any)
		assert.Equal(t, "AAPL", inst["symbol"]) // Should be uppercased

		resp := map[string]any{
			"baseSymbol": "AAPL",
			"calls":      []map[string]any{},
			"puts":       []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"aapl", "--expiration", "2025-01-17"}) // lowercase input

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOptionsChainCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"baseSymbol": "AAPL",
			"calls": []map[string]any{
				{
					"instrument": map[string]string{
						"symbol": "AAPL250117C00175000",
						"type":   "OPTION",
					},
					"outcome":      "SUCCESS",
					"last":         "5.50",
					"bid":          "5.45",
					"ask":          "5.55",
					"volume":       1000,
					"openInterest": 5000,
				},
			},
			"puts": []map[string]any{
				{
					"instrument": map[string]string{
						"symbol": "AAPL250117P00175000",
						"type":   "OPTION",
					},
					"outcome":      "SUCCESS",
					"last":         "4.50",
					"bid":          "4.45",
					"ask":          "4.55",
					"volume":       800,
					"openInterest": 3000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "--expiration", "2025-01-17"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify JSON output
	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", result["baseSymbol"])
	calls := result["calls"].([]any)
	assert.Len(t, calls, 1)
	puts := result["puts"].([]any)
	assert.Len(t, puts, 1)
}

func TestOptionsChainCmd_NoSymbol(t *testing.T) {
	cmd := newOptionsChainCmd(optionsOptions{
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
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestOptionsChainCmd_RequiresExpiration(t *testing.T) {
	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL"}) // No --expiration flag

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiration")
}

func TestOptionsChainCmd_RequiresAccount(t *testing.T) {
	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "", // No account
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--expiration", "2025-01-17"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}

func TestOptionsChainCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"AAPL", "--expiration", "2025-01-17"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestOptionsChainCmd_EmptyChain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"baseSymbol": "XYZ",
			"calls":      []map[string]any{},
			"puts":       []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newOptionsChainCmd(optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"XYZ", "--expiration", "2025-01-17"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "No options")
}

func TestRunMultilegOrder_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		if r.URL.Path == "/userapigateway/trading/test-account/preflight/multi-leg" {
			// Preflight request
			resp := MultilegPreflightResponse{
				BaseSymbol:             "AAPL",
				StrategyName:           "VERTICAL CALL SPREAD",
				EstimatedCommission:    "0.00",
				EstimatedCost:          "250.00",
				OrderValue:             "250.00",
				BuyingPowerRequirement: "250.00",
				EstimatedQuantity:      "1",
				Legs: []MultilegPreflightLeg{
					{
						Instrument:         MultilegInstrument{Symbol: "AAPL250117C00175000", Type: "OPTION"},
						Side:               "BUY",
						OpenCloseIndicator: "OPEN",
						RatioQuantity:      1,
					},
					{
						Instrument:         MultilegInstrument{Symbol: "AAPL250117C00180000", Type: "OPTION"},
						Side:               "SELL",
						OpenCloseIndicator: "OPEN",
						RatioQuantity:      1,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/userapigateway/trading/test-account/order/multi-leg" {
			// Order request
			var req MultilegOrderRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			// Verify order request fields
			assert.NotEmpty(t, req.OrderID)
			assert.Equal(t, "LIMIT", req.OrderType)
			assert.Equal(t, "2.50", req.LimitPrice)
			assert.Equal(t, "1", req.Quantity)
			assert.Equal(t, "DAY", req.Expiration.TimeInForce)
			assert.Len(t, req.Legs, 2)

			resp := MultilegOrderResponse{
				OrderID: req.OrderID,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  false,
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN",
		"SELL AAPL250117C00180000 OPEN",
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "DAY", true)
	require.NoError(t, err)

	output := cmd.OutOrStdout().(*bytes.Buffer).String()
	assert.Contains(t, output, "Order placed successfully")
	assert.Contains(t, output, "VERTICAL CALL SPREAD")
	assert.Contains(t, output, "AAPL")
}

func TestRunMultilegOrder_RequiresConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return preflight response
		resp := MultilegPreflightResponse{
			BaseSymbol:             "AAPL",
			StrategyName:           "VERTICAL CALL SPREAD",
			EstimatedCost:          "250.00",
			BuyingPowerRequirement: "250.00",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  false,
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN",
		"SELL AAPL250117C00180000 OPEN",
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "DAY", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires confirmation")
}

func TestRunMultilegOrder_MinimumTwoLegs(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN", // Only 1 leg
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "DAY", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 legs")
}

func TestRunMultilegOrder_MaximumSixLegs(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN",
		"SELL AAPL250117C00180000 OPEN",
		"BUY AAPL250117C00185000 OPEN",
		"SELL AAPL250117C00190000 OPEN",
		"BUY AAPL250117P00165000 OPEN",
		"SELL AAPL250117P00160000 OPEN",
		"BUY AAPL250117P00155000 OPEN", // 7th leg
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "DAY", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 6 legs")
}

func TestRunMultilegOrder_InvalidExpiration(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN",
		"SELL AAPL250117C00180000 OPEN",
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "INVALID", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expiration")
}

func TestRunMultilegOrder_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userapigateway/trading/test-account/preflight/multi-leg" {
			resp := MultilegPreflightResponse{
				BaseSymbol:   "AAPL",
				StrategyName: "VERTICAL CALL SPREAD",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/userapigateway/trading/test-account/order/multi-leg" {
			var req MultilegOrderRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp := MultilegOrderResponse{OrderID: req.OrderID}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  true,
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN",
		"SELL AAPL250117C00180000 OPEN",
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "DAY", true)
	require.NoError(t, err)

	output := cmd.OutOrStdout().(*bytes.Buffer).String()
	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "placed", result["status"])
	assert.Equal(t, "VERTICAL CALL SPREAD", result["strategy"])
	assert.Equal(t, float64(2), result["legs"])
}

func TestRunMultilegOrder_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userapigateway/trading/test-account/preflight/multi-leg" {
			resp := MultilegPreflightResponse{
				BaseSymbol:   "AAPL",
				StrategyName: "VERTICAL CALL SPREAD",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Order fails
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "insufficient buying power"}`))
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	}

	legs := []string{
		"BUY AAPL250117C00175000 OPEN",
		"SELL AAPL250117C00180000 OPEN",
	}

	cmd := newTestCmd()
	err := runMultilegOrder(cmd, opts, legs, "2.50", "1", "DAY", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "insufficient buying power")
}

// newTestCmd creates a cobra.Command for testing with a buffer for output.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	return cmd
}

// Tests for filtering logic

func TestParseStrikeFloat(t *testing.T) {
	tests := []struct {
		symbol   string
		expected float64
	}{
		{"AAPL250117C00175000", 175.0},
		{"AAPL250117C00185500", 185.5},
		{"SPY260131P00550000", 550.0},
		{"TSLA260220C00250500", 250.5},
		{"SHORT", 0},                 // Invalid symbol
		{"", 0},                      // Empty
		{"AAPL250117C00000500", 0.5}, // Very low strike
	}

	for _, tc := range tests {
		t.Run(tc.symbol, func(t *testing.T) {
			result := parseStrikeFloat(tc.symbol)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFilterOptions_MinMaxStrike(t *testing.T) {
	options := []api.OptionQuote{
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00170000"}, Volume: 100, OpenInterest: 500},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00175000"}, Volume: 200, OpenInterest: 1000},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00180000"}, Volume: 150, OpenInterest: 750},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00185000"}, Volume: 100, OpenInterest: 500},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00190000"}, Volume: 50, OpenInterest: 250},
	}

	// Test min-strike filter
	filter := chainFilter{minStrike: 175}
	result := filterOptions(options, filter)
	assert.Len(t, result, 4)
	assert.Equal(t, "AAPL250117C00175000", result[0].Instrument.Symbol)

	// Test max-strike filter
	filter = chainFilter{maxStrike: 180}
	result = filterOptions(options, filter)
	assert.Len(t, result, 3)
	assert.Equal(t, "AAPL250117C00180000", result[2].Instrument.Symbol)

	// Test min and max strike together
	filter = chainFilter{minStrike: 175, maxStrike: 185}
	result = filterOptions(options, filter)
	assert.Len(t, result, 3)
}

func TestFilterOptions_MinOI(t *testing.T) {
	options := []api.OptionQuote{
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00170000"}, OpenInterest: 50},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00175000"}, OpenInterest: 100},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00180000"}, OpenInterest: 500},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00185000"}, OpenInterest: 1000},
	}

	filter := chainFilter{minOI: 100}
	result := filterOptions(options, filter)
	assert.Len(t, result, 3)

	filter = chainFilter{minOI: 500}
	result = filterOptions(options, filter)
	assert.Len(t, result, 2)
}

func TestFilterOptions_MinVolume(t *testing.T) {
	options := []api.OptionQuote{
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00170000"}, Volume: 10},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00175000"}, Volume: 50},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00180000"}, Volume: 100},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00185000"}, Volume: 500},
	}

	filter := chainFilter{minVolume: 50}
	result := filterOptions(options, filter)
	assert.Len(t, result, 3)

	filter = chainFilter{minVolume: 100}
	result = filterOptions(options, filter)
	assert.Len(t, result, 2)
}

func TestFilterOptions_Combined(t *testing.T) {
	options := []api.OptionQuote{
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00170000"}, Volume: 10, OpenInterest: 50},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00175000"}, Volume: 100, OpenInterest: 500},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00180000"}, Volume: 200, OpenInterest: 1000},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00185000"}, Volume: 50, OpenInterest: 200},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00190000"}, Volume: 150, OpenInterest: 300},
	}

	// Combine all filters
	filter := chainFilter{
		minStrike: 175,
		maxStrike: 185,
		minOI:     200,
		minVolume: 50,
	}
	result := filterOptions(options, filter)
	// 175 (vol=100, OI=500), 180 (vol=200, OI=1000), 185 (vol=50, OI=200) all pass
	assert.Len(t, result, 3)
}

func TestFilterStrikesAroundATM(t *testing.T) {
	options := []api.OptionQuote{
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00165000"}}, // idx 0
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00170000"}}, // idx 1
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00175000"}}, // idx 2
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00180000"}}, // idx 3 (ATM)
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00185000"}}, // idx 4
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00190000"}}, // idx 5
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00195000"}}, // idx 6
	}

	// Underlying at 180, get 4 strikes (2 below, 2 above ATM at idx 3)
	// With n=4, half=2: indices 1,2,3,4 -> 170,175,180,185
	result := filterStrikesAroundATM(options, 4, 180.0)
	assert.Len(t, result, 4)
	assert.Equal(t, "AAPL250117C00170000", result[0].Instrument.Symbol)
	assert.Equal(t, "AAPL250117C00185000", result[3].Instrument.Symbol)

	// Underlying at 177 (closest to 175, idx 2), get 6 strikes
	// With n=6, half=3: indices -1 to 5 -> adjusted to 0 to 6 -> 165,170,175,180,185,190
	result = filterStrikesAroundATM(options, 6, 177.0)
	assert.Len(t, result, 6)

	// Edge case: underlying at edge (165, idx 0)
	// With n=4, half=2: indices -2 to 2 -> adjusted to 0 to 4 -> 165,170,175,180
	result = filterStrikesAroundATM(options, 4, 165.0)
	assert.Len(t, result, 4)
	assert.Equal(t, "AAPL250117C00165000", result[0].Instrument.Symbol)
}

func TestFilterStrikesAroundATM_SmallList(t *testing.T) {
	options := []api.OptionQuote{
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00175000"}},
		{Instrument: api.OptionInstrument{Symbol: "AAPL250117C00180000"}},
	}

	// Request more strikes than available
	result := filterStrikesAroundATM(options, 10, 177.5)
	assert.Len(t, result, 2) // Returns all available
}

func TestFilterStrikesAroundATM_Empty(t *testing.T) {
	result := filterStrikesAroundATM(nil, 10, 180.0)
	assert.Empty(t, result)

	result = filterStrikesAroundATM([]api.OptionQuote{}, 10, 180.0)
	assert.Empty(t, result)
}

// Tests for single-leg options orders

func TestRunSingleLegOrder_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		if r.URL.Path == "/userapigateway/trading/test-account/preflight/single-leg" {
			// Preflight request
			var req api.OptionsPreflightRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, "SBUX260220C00100000", req.Instrument.Symbol)
			assert.Equal(t, "OPTION", req.Instrument.Type)
			assert.Equal(t, "BUY", req.OrderSide)
			assert.Equal(t, "LIMIT", req.OrderType)
			assert.Equal(t, "8", req.Quantity)
			assert.Equal(t, "1.50", req.LimitPrice)
			assert.Equal(t, "OPEN", req.OpenCloseIndicator)

			resp := api.OptionsPreflightResponse{
				Instrument:             api.OrderInstrument{Symbol: "SBUX260220C00100000", Type: "OPTION"},
				EstimatedCommission:    "0.00",
				EstimatedCost:          "1200.00",
				OrderValue:             "1200.00",
				BuyingPowerRequirement: "1200.00",
				EstimatedQuantity:      "8",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/userapigateway/trading/test-account/order" {
			// Order request
			var req api.OptionsOrderRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			// Verify order request fields
			assert.NotEmpty(t, req.OrderID)
			assert.Equal(t, "SBUX260220C00100000", req.Instrument.Symbol)
			assert.Equal(t, "OPTION", req.Instrument.Type)
			assert.Equal(t, "BUY", req.OrderSide)
			assert.Equal(t, "LIMIT", req.OrderType)
			assert.Equal(t, "8", req.Quantity)
			assert.Equal(t, "1.50", req.LimitPrice)
			assert.Equal(t, "OPEN", req.OpenCloseIndicator)
			assert.Equal(t, "DAY", req.Expiration.TimeInForce)

			resp := api.OrderResponse{OrderID: req.OrderID}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  false,
	}

	params := singleLegParams{
		quantity:   "8",
		limitPrice: "1.50",
		expiration: "DAY",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "SBUX260220C00100000", "BUY", params, true, true)
	require.NoError(t, err)

	output := cmd.OutOrStdout().(*bytes.Buffer).String()
	assert.Contains(t, output, "Order placed successfully")
	assert.Contains(t, output, "SBUX260220C00100000")
	assert.Contains(t, output, "BUY")
	assert.Contains(t, output, "OPEN")
}

func TestRunSingleLegOrder_SellToClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userapigateway/trading/test-account/preflight/single-leg" {
			resp := api.OptionsPreflightResponse{
				EstimatedCost:     "250.00",
				EstimatedProceeds: "250.00",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/userapigateway/trading/test-account/order" {
			var req api.OptionsOrderRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, "SELL", req.OrderSide)
			assert.Equal(t, "CLOSE", req.OpenCloseIndicator)

			resp := api.OrderResponse{OrderID: req.OrderID}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		expiration: "GTC",
		openClose:  "CLOSE",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "SELL", params, true, true)
	require.NoError(t, err)
}

func TestRunSingleLegOrder_RequiresConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := api.OptionsPreflightResponse{EstimatedCost: "250.00"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		expiration: "DAY",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires confirmation")
}

func TestRunSingleLegOrder_RequiresQuantity(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "", // Missing
		limitPrice: "2.50",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quantity is required")
}

func TestRunSingleLegOrder_RequiresLimitPrice(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "", // Missing
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit price is required")
}

func TestRunSingleLegOrder_RequiresOpenClose(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		openClose:  "", // Missing
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open/close indicator is required")
}

func TestRunSingleLegOrder_InvalidExpiration(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		expiration: "INVALID",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expiration")
}

func TestRunSingleLegOrder_TradingDisabled(t *testing.T) {
	opts := optionsOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trading is disabled")
}

func TestRunSingleLegOrder_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userapigateway/trading/test-account/preflight/single-leg" {
			resp := api.OptionsPreflightResponse{EstimatedCost: "250.00"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/userapigateway/trading/test-account/order" {
			var req api.OptionsOrderRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp := api.OrderResponse{OrderID: req.OrderID}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
		jsonMode:  true,
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		expiration: "DAY",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "AAPL250117C00175000", "BUY", params, true, true)
	require.NoError(t, err)

	output := cmd.OutOrStdout().(*bytes.Buffer).String()
	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "placed", result["status"])
	assert.Equal(t, "AAPL250117C00175000", result["symbol"])
	assert.Equal(t, "BUY", result["side"])
	assert.Equal(t, "OPEN", result["openClose"])
}

func TestRunSingleLegOrder_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userapigateway/trading/test-account/preflight/single-leg" {
			resp := api.OptionsPreflightResponse{EstimatedCost: "250.00"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Order fails
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":140,"message":"Provided symbol 'XYZ' is not valid"}`))
	}))
	defer server.Close()

	opts := optionsOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	}

	params := singleLegParams{
		quantity:   "1",
		limitPrice: "2.50",
		expiration: "DAY",
		openClose:  "OPEN",
	}

	cmd := newTestCmd()
	err := runSingleLegOrder(cmd, opts, "XYZ", "BUY", params, true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "not valid")
}

func TestSumOptionsFees(t *testing.T) {
	tests := []struct {
		name     string
		fees     api.OptionsRegulatoryFees
		expected string
	}{
		{
			name:     "All zeros",
			fees:     api.OptionsRegulatoryFees{},
			expected: "0.00",
		},
		{
			name: "All fees",
			fees: api.OptionsRegulatoryFees{
				SECFee:      "0.01",
				TAFFee:      "0.02",
				ORFFee:      "0.03",
				ExchangeFee: "0.04",
				OCCFee:      "0.05",
				CATFee:      "0.06",
			},
			expected: "0.21",
		},
		{
			name: "Partial fees",
			fees: api.OptionsRegulatoryFees{
				SECFee: "0.10",
				OCCFee: "0.15",
			},
			expected: "0.25",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sumOptionsFees(tc.fees)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractOptionsErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected string
	}{
		{
			name:     "Nil error",
			input:    nil,
			expected: "",
		},
		{
			name:     "Simple error",
			input:    assert.AnError,
			expected: "assert.AnError general error for testing",
		},
		{
			name:     "JSON error with message",
			input:    fmt.Errorf(`API error: 400 - {"code":140,"message":"Symbol is not valid"}`),
			expected: "Symbol is not valid",
		},
		{
			name:     "JSON error with header only",
			input:    fmt.Errorf(`API error: 400 - {"code":140,"header":"Bad Request"}`),
			expected: "Bad Request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractOptionsErrorMessage(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
