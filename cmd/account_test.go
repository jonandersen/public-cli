package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonandersen/pub/internal/keyring"
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
				{"type": "CASH", "value": "5000.00", "percentageOfPortfolio": "50.00"},
				{"type": "STOCK", "value": "5000.00", "percentageOfPortfolio": "50.00"},
			},
			"positions": []map[string]any{
				{
					"instrument":   map[string]any{"symbol": "AAPL", "name": "Apple Inc.", "type": "EQUITY"},
					"quantity":     "10",
					"currentValue": "1750.00",
					"lastPrice": map[string]any{
						"lastPrice": "175.00",
						"timestamp": "2024-01-15T10:30:00Z",
					},
					"instrumentGain": map[string]any{
						"gainValue":      "250.00",
						"gainPercentage": "16.67",
						"timestamp":      "2024-01-15T10:30:00Z",
					},
					"positionDailyGain": map[string]any{
						"gainValue":      "50.00",
						"gainPercentage": "2.94",
						"timestamp":      "2024-01-15T10:30:00Z",
					},
					"costBasis": map[string]any{
						"totalCost":      "1500.00",
						"unitCost":       "150.00",
						"gainValue":      "250.00",
						"gainPercentage": "16.67",
						"lastUpdate":     "2024-01-15T10:30:00Z",
					},
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
	// Check equity summary is displayed
	assert.Contains(t, output, "CASH")
	assert.Contains(t, output, "5000.00")
	assert.Contains(t, output, "50.00%")
	// Check positions are displayed
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
		// No defaultAccountID set
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"portfolio"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account ID is required")
	assert.Contains(t, err.Error(), "pub configure")
}

func TestAccountPortfolioCmd_UsesDefaultAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it uses the default account ID
		assert.Equal(t, "/userapigateway/trading/default-account-123/portfolio/v2", r.URL.Path)

		resp := map[string]any{
			"accountId":   "default-account-123",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"buyingPower":        "10000.00",
				"optionsBuyingPower": "5000.00",
			},
			"equity":    []map[string]any{},
			"positions": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:          server.URL,
		authToken:        "test-token",
		defaultAccountID: "default-account-123",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"portfolio"}) // No --account flag

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Buying Power")
}

func TestAccountPortfolioCmd_FlagOverridesDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it uses the flag value, not the default
		assert.Equal(t, "/userapigateway/trading/flag-account-456/portfolio/v2", r.URL.Path)

		resp := map[string]any{
			"accountId":   "flag-account-456",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"buyingPower":        "10000.00",
				"optionsBuyingPower": "5000.00",
			},
			"equity":    []map[string]any{},
			"positions": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:          server.URL,
		authToken:        "test-token",
		defaultAccountID: "default-account-123", // Has a default
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "flag-account-456"}) // Override with flag

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestGetAuthToken_NotConfigured(t *testing.T) {
	store := keyring.NewMockStore() // Empty store - no secret key

	_, err := getAuthToken(store, "https://api.public.com", false)
	require.Error(t, err)

	// Verify error message matches expected format
	assert.Contains(t, err.Error(), "CLI not configured")
	assert.Contains(t, err.Error(), "pub configure")
	assert.Contains(t, err.Error(), "PUB_SECRET_KEY")
}

func TestGetAuthToken_KeyringError(t *testing.T) {
	store := keyring.NewMockStore().WithGetError(assert.AnError)

	_, err := getAuthToken(store, "https://api.public.com", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve secret")
}

func TestGetAuthToken_Success(t *testing.T) {
	// Use temp directory for token cache isolation
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Mock server returns a valid token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/userapiauthservice/personal/access-tokens", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"accessToken": "test-access-token-123",
		})
	}))
	defer server.Close()

	store := keyring.NewMockStore().WithData(keyring.ServiceName, keyring.KeySecretKey, "test-secret-key")

	token, err := getAuthToken(store, server.URL, false)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token-123", token)
}

func TestGetAuthToken_ExchangeError(t *testing.T) {
	// Use temp directory for token cache isolation
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Mock server returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid secret"}`))
	}))
	defer server.Close()

	store := keyring.NewMockStore().WithData(keyring.ServiceName, keyring.KeySecretKey, "bad-secret")

	_, err := getAuthToken(store, server.URL, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authenticate")
}

func TestFormatGainLoss(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "positive value",
			input:    "250.00",
			expected: "+$250.00",
		},
		{
			name:     "negative value",
			input:    "-50.00",
			expected: "-$50.00",
		},
		{
			name:     "zero string",
			input:    "0",
			expected: "$0.00",
		},
		{
			name:     "zero decimal",
			input:    "0.00",
			expected: "$0.00",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "$0.00",
		},
		{
			name:     "large positive",
			input:    "12345.67",
			expected: "+$12345.67",
		},
		{
			name:     "large negative",
			input:    "-98765.43",
			expected: "-$98765.43",
		},
		{
			name:     "small positive",
			input:    "0.01",
			expected: "+$0.01",
		},
		{
			name:     "small negative",
			input:    "-0.01",
			expected: "-$0.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGainLoss(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccountPortfolioCmd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accountId":   "abc123",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"cashOnlyBuyingPower": "10000.00",
				"buyingPower":         "10000.00",
				"optionsBuyingPower":  "5000.00",
			},
			"equity": []map[string]any{
				{"type": "CASH", "value": "5000.00", "percentageOfPortfolio": "50.00"},
			},
			"positions": []map[string]any{
				{
					"instrument":   map[string]any{"symbol": "AAPL", "name": "Apple Inc.", "type": "EQUITY"},
					"quantity":     "10",
					"currentValue": "1750.00",
					"lastPrice": map[string]any{
						"lastPrice": "175.00",
						"timestamp": "2024-01-15T10:30:00Z",
					},
					"positionDailyGain": map[string]any{
						"gainValue":      "50.00",
						"gainPercentage": "2.94",
					},
					"costBasis": map[string]any{
						"totalCost":      "1500.00",
						"gainValue":      "250.00",
						"gainPercentage": "16.67",
					},
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
	cmd.SetArgs([]string{"portfolio", "--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify JSON output is parseable
	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, result, "buyingPower")
	assert.Contains(t, result, "equity")
	assert.Contains(t, result, "positions")

	positions := result["positions"].([]any)
	assert.Len(t, positions, 1)
}

func TestAccountPortfolioCmd_NoPositionsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accountId":   "abc123",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"buyingPower":        "10000.00",
				"optionsBuyingPower": "5000.00",
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
		jsonMode:  true,
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "abc123"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify JSON output with empty positions
	var result map[string]any
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)

	positions := result["positions"].([]any)
	assert.Empty(t, positions)
}

func TestAccountPortfolioCmd_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "account not found"}`))
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "nonexistent"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestAccountPortfolioCmd_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   server.URL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "abc123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestAccountPortfolioCmd_EmptyCostBasis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"accountId":   "abc123",
			"accountType": "BROKERAGE",
			"buyingPower": map[string]any{
				"buyingPower":        "10000.00",
				"optionsBuyingPower": "5000.00",
			},
			"equity": []map[string]any{},
			"positions": []map[string]any{
				{
					"instrument":   map[string]any{"symbol": "AAPL", "name": "Apple Inc.", "type": "EQUITY"},
					"quantity":     "10",
					"currentValue": "1750.00",
					"lastPrice": map[string]any{
						"lastPrice": "175.00",
					},
					"positionDailyGain": map[string]any{
						"gainValue":      "50.00",
						"gainPercentage": "2.94",
					},
					// costBasis with empty values - should default to 0
					"costBasis": map[string]any{
						"gainValue":      "",
						"gainPercentage": "",
					},
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
	// Empty costBasis should show $0.00 for total G/L
	assert.Contains(t, output, "$0.00")
}

func TestAccountListCmd_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
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
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestAccountListCmd_NetworkError(t *testing.T) {
	// Use a closed server to simulate connection refused
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   serverURL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch accounts")
}

func TestAccountPortfolioCmd_NetworkError(t *testing.T) {
	// Use a closed server to simulate connection refused
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	cmd := newAccountCmd(accountOptions{
		baseURL:   serverURL,
		authToken: "test-token",
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"portfolio", "--account", "abc123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch portfolio")
}
