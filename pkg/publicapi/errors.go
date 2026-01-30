package publicapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error response from the Public.com API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, msg)
}

// IsNotFound returns true if the error is a 404 Not Found.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401 Unauthorized.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if the error is a 403 Forbidden.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// errorResponse represents the JSON structure of API error responses.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// CheckResponse checks the API response for errors.
// If the response status code indicates an error (>= 400), it parses
// the error body and returns an APIError. Otherwise, returns nil.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
	}

	// Try to parse error body
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return apiErr
	}

	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// Body is not JSON, ignore parsing error
		return apiErr
	}

	// Use "error" field if present, otherwise "message" field
	if errResp.Error != "" {
		apiErr.Message = errResp.Error
	} else if errResp.Message != "" {
		apiErr.Message = errResp.Message
	}
	apiErr.Code = errResp.Code

	return apiErr
}

// DecodeJSON decodes a JSON response body into the given target.
func DecodeJSON(resp *http.Response, target interface{}) error {
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
