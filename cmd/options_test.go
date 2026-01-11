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
