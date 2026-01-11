package api

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name: "with message",
			err: &APIError{
				StatusCode: 401,
				Message:    "Invalid credentials",
			},
			expected: "API error (401): Invalid credentials",
		},
		{
			name: "without message",
			err: &APIError{
				StatusCode: 500,
				Message:    "",
			},
			expected: "API error (500): Internal Server Error",
		},
		{
			name: "with code and message",
			err: &APIError{
				StatusCode: 400,
				Code:       "INVALID_SYMBOL",
				Message:    "Symbol not found",
			},
			expected: "API error (400): Symbol not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	assert.True(t, (&APIError{StatusCode: 404}).IsNotFound())
	assert.False(t, (&APIError{StatusCode: 401}).IsNotFound())
}

func TestAPIError_IsUnauthorized(t *testing.T) {
	assert.True(t, (&APIError{StatusCode: 401}).IsUnauthorized())
	assert.False(t, (&APIError{StatusCode: 404}).IsUnauthorized())
}

func TestAPIError_IsForbidden(t *testing.T) {
	assert.True(t, (&APIError{StatusCode: 403}).IsForbidden())
	assert.False(t, (&APIError{StatusCode: 401}).IsForbidden())
}

func TestCheckResponse_Success(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok": true}`))),
	}

	err := CheckResponse(resp)
	assert.NoError(t, err)
}

func TestCheckResponse_SuccessStatuses(t *testing.T) {
	statuses := []int{200, 201, 204, 299}

	for _, status := range statuses {
		resp := &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}
		assert.NoError(t, CheckResponse(resp), "status %d should not error", status)
	}
}

func TestCheckResponse_ErrorWithJSONBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 401,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "Invalid token", "code": "AUTH_FAILED"}`))),
	}

	err := CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 401, apiErr.StatusCode)
	assert.Equal(t, "Invalid token", apiErr.Message)
	assert.Equal(t, "AUTH_FAILED", apiErr.Code)
}

func TestCheckResponse_ErrorWithMessageField(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"message": "Bad request"}`))),
	}

	err := CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, "Bad request", apiErr.Message)
}

func TestCheckResponse_ErrorWithEmptyBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}

	err := CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 500, apiErr.StatusCode)
	assert.Empty(t, apiErr.Message)
}

func TestCheckResponse_ErrorWithNonJSONBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 502,
		Body:       io.NopCloser(bytes.NewReader([]byte("Bad Gateway"))),
	}

	err := CheckResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 502, apiErr.StatusCode)
}

func TestDecodeJSON_Success(t *testing.T) {
	type Account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "123", "name": "Test Account"}`))),
	}

	var account Account
	err := DecodeJSON(resp, &account)

	require.NoError(t, err)
	assert.Equal(t, "123", account.ID)
	assert.Equal(t, "Test Account", account.Name)
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`not json`))),
	}

	var result map[string]interface{}
	err := DecodeJSON(resp, &result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(``))),
	}

	var result map[string]interface{}
	err := DecodeJSON(resp, &result)

	assert.Error(t, err)
}
