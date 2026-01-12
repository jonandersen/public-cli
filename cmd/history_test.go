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

func TestHistoryCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/abc123/history", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		resp := map[string]any{
			"transactions": []map[string]any{
				{
					"id":          "txn-001",
					"timestamp":   "2025-01-15T10:30:00Z",
					"type":        "TRADE",
					"subType":     "BUY",
					"symbol":      "AAPL",
					"description": "Buy 10 shares of AAPL",
					"quantity":    "10",
					"netAmount":   "-1750.00",
					"fees":        "0.00",
				},
				{
					"id":          "txn-002",
					"timestamp":   "2025-01-14T09:00:00Z",
					"type":        "MONEY_MOVEMENT",
					"subType":     "DEPOSIT",
					"symbol":      "",
					"description": "ACH Deposit",
					"quantity":    "",
					"netAmount":   "5000.00",
					"fees":        "0.00",
				},
			},
			"nextToken": "",
			"pageSize":  50,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "txn-001")
	assert.Contains(t, output, "BUY") // Shows subType when available
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "DEPOSIT")
}

func TestHistoryCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"transactions": []map[string]any{
				{
					"id":          "txn-001",
					"timestamp":   "2025-01-15T10:30:00Z",
					"type":        "TRADE",
					"subType":     "BUY",
					"symbol":      "AAPL",
					"description": "Buy 10 shares of AAPL",
					"quantity":    "10",
					"netAmount":   "-1750.00",
					"fees":        "0.00",
				},
			},
			"nextToken": "",
			"pageSize":  50,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "transactions")

	transactions := result["transactions"].([]any)
	assert.Len(t, transactions, 1)
}

func TestHistoryCmd_NoTransactions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"transactions": []map[string]any{},
			"nextToken":    "",
			"pageSize":     50,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No transactions found")
}

func TestHistoryCmd_RequiresAccount(t *testing.T) {
	cmd := newHistoryCmd(historyOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account ID is required")
}

func TestHistoryCmd_UsesDefaultAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/default-account-123/history", r.URL.Path)

		resp := map[string]any{
			"transactions": []map[string]any{},
			"nextToken":    "",
			"pageSize":     50,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:          server.URL,
		authToken:        "test-token",
		defaultAccountID: "default-account-123",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestHistoryCmd_WithDateFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2025-01-01T00:00:00Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2025-01-31T23:59:59Z", r.URL.Query().Get("end"))

		resp := map[string]any{
			"transactions": []map[string]any{},
			"nextToken":    "",
			"pageSize":     50,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"--account", "abc123",
		"--start", "2025-01-01T00:00:00Z",
		"--end", "2025-01-31T23:59:59Z",
	})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestHistoryCmd_WithLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("pageSize"))

		resp := map[string]any{
			"transactions": []map[string]any{},
			"nextToken":    "",
			"pageSize":     10,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--account", "abc123", "--limit", "10"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestHistoryCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid account ID"}`))
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--account", "invalid"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestHistoryCmd_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--account", "abc123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestHistoryCmd_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:   serverURL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--account", "abc123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch history")
}

func TestHistoryCmd_FlagOverridesDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/flag-account-456/history", r.URL.Path)

		resp := map[string]any{
			"transactions": []map[string]any{},
			"nextToken":    "",
			"pageSize":     50,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newHistoryCmd(historyOptions{
		baseURL:          server.URL,
		authToken:        "test-token",
		defaultAccountID: "default-account-123",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--account", "flag-account-456"})

	err := cmd.Execute()
	require.NoError(t, err)
}
