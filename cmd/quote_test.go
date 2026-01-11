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

func TestQuoteCmd_SingleSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/marketdata/test-account/quotes", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify request body
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		instruments := req["instruments"].([]any)
		assert.Len(t, instruments, 1)
		inst := instruments[0].(map[string]any)
		assert.Equal(t, "AAPL", inst["symbol"])
		assert.Equal(t, "EQUITY", inst["type"])

		resp := map[string]any{
			"quotes": []map[string]any{
				{
					"instrument": map[string]any{"symbol": "AAPL", "type": "EQUITY"},
					"outcome":    "SUCCESS",
					"last":       "175.50",
					"bid":        "175.45",
					"bidSize":    100,
					"ask":        "175.55",
					"askSize":    200,
					"volume":     50000000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newQuoteCmd(quoteOptions{
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
	assert.Contains(t, output, "175.50")
	assert.Contains(t, output, "175.45")
	assert.Contains(t, output, "175.55")
}

func TestQuoteCmd_MultipleSymbols(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		instruments := req["instruments"].([]any)
		assert.Len(t, instruments, 3)

		resp := map[string]any{
			"quotes": []map[string]any{
				{
					"instrument": map[string]any{"symbol": "AAPL", "type": "EQUITY"},
					"outcome":    "SUCCESS",
					"last":       "175.50",
					"bid":        "175.45",
					"ask":        "175.55",
					"volume":     50000000,
				},
				{
					"instrument": map[string]any{"symbol": "GOOGL", "type": "EQUITY"},
					"outcome":    "SUCCESS",
					"last":       "140.25",
					"bid":        "140.20",
					"ask":        "140.30",
					"volume":     25000000,
				},
				{
					"instrument": map[string]any{"symbol": "MSFT", "type": "EQUITY"},
					"outcome":    "SUCCESS",
					"last":       "380.00",
					"bid":        "379.95",
					"ask":        "380.05",
					"volume":     30000000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newQuoteCmd(quoteOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"AAPL", "GOOGL", "MSFT"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "GOOGL")
	assert.Contains(t, output, "MSFT")
	assert.Contains(t, output, "175.50")
	assert.Contains(t, output, "140.25")
	assert.Contains(t, output, "380.00")
}

func TestQuoteCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"quotes": []map[string]any{
				{
					"instrument": map[string]any{"symbol": "AAPL", "type": "EQUITY"},
					"outcome":    "SUCCESS",
					"last":       "175.50",
					"bid":        "175.45",
					"ask":        "175.55",
					"volume":     50000000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newQuoteCmd(quoteOptions{
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
	var result []map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "AAPL", result[0]["Symbol"])
	assert.Equal(t, "175.50", result[0]["Last"])
}

func TestQuoteCmd_NoSymbols(t *testing.T) {
	cmd := newQuoteCmd(quoteOptions{
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
	assert.Contains(t, err.Error(), "requires at least 1 arg")
}

func TestQuoteCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	cmd := newQuoteCmd(quoteOptions{
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

func TestQuoteCmd_RequiresAccount(t *testing.T) {
	cmd := newQuoteCmd(quoteOptions{
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

func TestQuoteCmd_FailedQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"quotes": []map[string]any{
				{
					"instrument": map[string]any{"symbol": "INVALID", "type": "EQUITY"},
					"outcome":    "FAILURE",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newQuoteCmd(quoteOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		accountID: "test-account",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"INVALID"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "INVALID")
	assert.Contains(t, output, "FAILURE")
}
