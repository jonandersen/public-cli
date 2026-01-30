package publicapi

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "with message",
			err:      &APIError{StatusCode: 400, Message: "Invalid request"},
			expected: "API error (400): Invalid request",
		},
		{
			name:     "without message uses status text",
			err:      &APIError{StatusCode: 404},
			expected: "API error (404): Not Found",
		},
		{
			name:     "unauthorized without message",
			err:      &APIError{StatusCode: 401},
			expected: "API error (401): Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAPIError_StatusChecks(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		isNotFound     bool
		isUnauthorized bool
		isForbidden    bool
	}{
		{"404", 404, true, false, false},
		{"401", 401, false, true, false},
		{"403", 403, false, false, true},
		{"500", 500, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			assert.Equal(t, tt.isNotFound, err.IsNotFound())
			assert.Equal(t, tt.isUnauthorized, err.IsUnauthorized())
			assert.Equal(t, tt.isForbidden, err.IsForbidden())
		})
	}
}

func TestCheckResponse_Success(t *testing.T) {
	tests := []int{200, 201, 204, 299}

	for _, code := range tests {
		resp := &http.Response{StatusCode: code}
		assert.NoError(t, CheckResponse(resp))
	}
}

func TestCheckResponse_Error(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		body         string
		expectedMsg  string
		expectedCode string
	}{
		{
			name:        "with error field",
			statusCode:  400,
			body:        `{"error":"Bad request data"}`,
			expectedMsg: "Bad request data",
		},
		{
			name:        "with message field",
			statusCode:  404,
			body:        `{"message":"Resource not found"}`,
			expectedMsg: "Resource not found",
		},
		{
			name:         "with code field",
			statusCode:   422,
			body:         `{"error":"Validation failed","code":"VALIDATION_ERROR"}`,
			expectedMsg:  "Validation failed",
			expectedCode: "VALIDATION_ERROR",
		},
		{
			name:        "empty body",
			statusCode:  500,
			body:        "",
			expectedMsg: "",
		},
		{
			name:        "non-json body",
			statusCode:  502,
			body:        "Bad Gateway",
			expectedMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			err := CheckResponse(resp)
			assert.Error(t, err)

			apiErr, ok := err.(*APIError)
			assert.True(t, ok)
			assert.Equal(t, tt.statusCode, apiErr.StatusCode)
			assert.Equal(t, tt.expectedMsg, apiErr.Message)
			assert.Equal(t, tt.expectedCode, apiErr.Code)
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("valid json", func(t *testing.T) {
		resp := &http.Response{
			Body: io.NopCloser(strings.NewReader(`{"name":"test","value":42}`)),
		}

		var data testData
		err := DecodeJSON(resp, &data)
		assert.NoError(t, err)
		assert.Equal(t, "test", data.Name)
		assert.Equal(t, 42, data.Value)
	})

	t.Run("invalid json", func(t *testing.T) {
		resp := &http.Response{
			Body: io.NopCloser(strings.NewReader(`not json`)),
		}

		var data testData
		err := DecodeJSON(resp, &data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
	})
}
