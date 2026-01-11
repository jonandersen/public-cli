package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests that verify the API client works correctly with
// CheckResponse and DecodeJSON for complete request/response flows.

func TestIntegration_GetWithCheckAndDecode(t *testing.T) {
	type Account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/accounts/123", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Account{ID: "123", Name: "Main Account"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Get(context.Background(), "/accounts/123")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.NoError(t, err)

	var account Account
	err = DecodeJSON(resp, &account)
	require.NoError(t, err)

	assert.Equal(t, "123", account.ID)
	assert.Equal(t, "Main Account", account.Name)
}

func TestIntegration_PostWithCheckAndDecode(t *testing.T) {
	type OrderRequest struct {
		Symbol string `json:"symbol"`
		Qty    int    `json:"qty"`
	}
	type OrderResponse struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/orders", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req OrderRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "AAPL", req.Symbol)
		assert.Equal(t, 10, req.Qty)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(OrderResponse{OrderID: "order-456", Status: "pending"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	body := strings.NewReader(`{"symbol":"AAPL","qty":10}`)
	resp, err := client.Post(context.Background(), "/orders", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.NoError(t, err)

	var order OrderResponse
	err = DecodeJSON(resp, &order)
	require.NoError(t, err)

	assert.Equal(t, "order-456", order.OrderID)
	assert.Equal(t, "pending", order.Status)
}

func TestIntegration_ErrorResponseHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Invalid token", "code": "AUTH_FAILED"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-token")
	resp, err := client.Get(context.Background(), "/accounts")
	require.NoError(t, err) // HTTP request succeeded
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.True(t, apiErr.IsUnauthorized())
	assert.Equal(t, "Invalid token", apiErr.Message)
	assert.Equal(t, "AUTH_FAILED", apiErr.Code)
}

func TestIntegration_NotFoundErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Account not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Get(context.Background(), "/accounts/nonexistent")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.True(t, apiErr.IsNotFound())
	assert.Equal(t, "Account not found", apiErr.Message)
}

func TestIntegration_ForbiddenErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "Insufficient permissions"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Delete(context.Background(), "/admin/settings")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.True(t, apiErr.IsForbidden())
}

func TestIntegration_ServerErrorWithPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Bad Gateway"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Get(context.Background(), "/accounts")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 502, apiErr.StatusCode)
	// Plain text response should not crash, message may be empty
}

func TestIntegration_MultipleRequestsWithSameClient(t *testing.T) {
	type Quote struct {
		Symbol string  `json:"symbol"`
		Price  float64 `json:"price"`
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "Bearer reused-token", r.Header.Get("Authorization"))

		symbol := strings.TrimPrefix(r.URL.Path, "/quotes/")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Quote{Symbol: symbol, Price: 150.00})
	}))
	defer server.Close()

	client := NewClient(server.URL, "reused-token")

	// Make multiple requests with same client
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		resp, err := client.Get(context.Background(), "/quotes/"+symbol)
		require.NoError(t, err)

		err = CheckResponse(resp)
		require.NoError(t, err)

		var quote Quote
		err = DecodeJSON(resp, &quote)
		require.NoError(t, err)
		_ = resp.Body.Close()

		assert.Equal(t, symbol, quote.Symbol)
	}

	assert.Equal(t, 3, requestCount)
}

func TestIntegration_DeleteWithNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/orders/order-789", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Delete(context.Background(), "/orders/order-789")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestIntegration_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "Rate limit exceeded", "code": "RATE_LIMIT"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.Get(context.Background(), "/accounts")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 429, apiErr.StatusCode)
	assert.Equal(t, "Rate limit exceeded", apiErr.Message)
	assert.Equal(t, "RATE_LIMIT", apiErr.Code)
}

func TestIntegration_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid quantity: must be positive", "code": "VALIDATION_ERROR"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	body := strings.NewReader(`{"symbol":"AAPL","qty":-5}`)
	resp, err := client.Post(context.Background(), "/orders", body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	err = CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.Equal(t, "Invalid quantity: must be positive", apiErr.Message)
}
