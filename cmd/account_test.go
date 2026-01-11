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

func TestAccountListCmd_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/account", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		resp := map[string]any{
			"accounts": []map[string]any{
				{
					"accountId":            "abc123-def456-ghi789-jkl012-mno345",
					"accountType":          "BROKERAGE",
					"optionsLevel":         "LEVEL_2",
					"brokerageAccountType": "CASH",
					"tradePermissions":     "BUY_AND_SELL",
				},
				{
					"accountId":            "xyz789-uvw456-rst123-opq890-lmn567",
					"accountType":          "ROTH_IRA",
					"optionsLevel":         "LEVEL_1",
					"brokerageAccountType": "CASH",
					"tradePermissions":     "BUY_AND_SELL",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "abc123-def456-ghi789-jkl012-mno345")
	assert.Contains(t, output, "BROKERAGE")
	assert.Contains(t, output, "ROTH_IRA")
}

func TestAccountListCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accounts": []map[string]any{
				{
					"accountId":            "abc123-def456-ghi789-jkl012-mno345",
					"accountType":          "BROKERAGE",
					"optionsLevel":         "LEVEL_2",
					"brokerageAccountType": "CASH",
					"tradePermissions":     "BUY_AND_SELL",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify JSON output
	var result []map[string]string
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "abc123-def456-ghi789-jkl012-mno345", result[0]["Account ID"])
}

func TestAccountListCmd_NoAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accounts": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No accounts found")
}

func TestAccountListCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestAccountPortfolioCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapigateway/trading/abc123/portfolio/v2", r.URL.Path)

		resp := map[string]any{
			"accountId":   "abc123",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"cashOnlyBuyingPower": "10000.00",
				"buyingPower":         "10000.00",
				"optionsBuyingPower":  "5000.00",
			},
			"equity": []map[string]any{
				{"type": "CASH", "value": "5000.00", "portfolioPercentage": "50.00"},
				{"type": "EQUITY", "value": "5000.00", "portfolioPercentage": "50.00"},
			},
			"positions": []map[string]any{
				{
					"instrument":            map[string]any{"symbol": "AAPL", "type": "EQUITY"},
					"quantity":              "10",
					"currentValue":          "1750.00",
					"lastPrice":             "175.00",
					"unrealizedGain":        "250.00",
					"unrealizedGainPercent": "16.67",
					"dailyGain":             "50.00",
					"dailyGainPercent":      "2.94",
					"costBasis":             "1500.00",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "AAPL")
	assert.Contains(t, output, "10")
	assert.Contains(t, output, "1750.00")
}

func TestAccountPortfolioCmd_NoPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accountId":   "abc123",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"cashOnlyBuyingPower": "10000.00",
				"buyingPower":         "10000.00",
				"optionsBuyingPower":  "5000.00",
			},
			"equity":    []map[string]any{},
			"positions": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Buying Power")
	assert.Contains(t, output, "10000.00")
	assert.Contains(t, output, "No positions")
}

func TestAccountPortfolioCmd_RequiresAccount(t *testing.T) {
	cmd := newAccountCmd(accountOptions{
		baseURL:   "http://localhost",
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"portfolio"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account")
}
