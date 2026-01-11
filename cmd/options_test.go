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
