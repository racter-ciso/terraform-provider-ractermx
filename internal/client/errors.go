// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// APIError represents a structured error from the RacterMX API.
type APIError struct {
	StatusCode int
	Message    string
	// FieldErrors contains per-field validation errors from 422 responses.
	FieldErrors map[string][]string
}

func (e *APIError) Error() string {
	if len(e.FieldErrors) > 0 {
		var parts []string
		// Sort field names for deterministic output
		fields := make([]string, 0, len(e.FieldErrors))
		for field := range e.FieldErrors {
			fields = append(fields, field)
		}
		sort.Strings(fields)

		for _, field := range fields {
			msgs := e.FieldErrors[field]
			parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(msgs, "; ")))
		}
		return fmt.Sprintf("validation error: %s", strings.Join(parts, ", "))
	}
	return e.Message
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

// apiErrorResponse represents the JSON structure of an API error response.
type apiErrorResponse struct {
	Errors  map[string][]string `json:"errors,omitempty"`
	Message string              `json:"message,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// parseAPIError parses an HTTP error response into an APIError.
// If isRead is true and the status is 404, it returns nil (resource removed from state).
func parseAPIError(statusCode int, body []byte, isRead bool) error {
	// 404 during Read: resource was deleted out-of-band
	if statusCode == 404 && isRead {
		return nil
	}

	switch statusCode {
	case 401:
		return &APIError{
			StatusCode: 401,
			Message:    "Invalid or expired API credentials. Check your api_key configuration.",
		}

	case 404:
		return &APIError{
			StatusCode: 404,
			Message:    "Resource not found",
		}

	case 409:
		msg := parseErrorMessage(body)
		if msg == "" {
			msg = "Resource conflict"
		}
		return &APIError{
			StatusCode: 409,
			Message:    msg,
		}

	case 422:
		return parse422Error(body)

	default:
		msg := parseErrorMessage(body)
		if msg == "" {
			msg = fmt.Sprintf("API returned unexpected status %d", statusCode)
		}
		return &APIError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("API error (status %d): %s", statusCode, msg),
		}
	}
}

// parse422Error parses a 422 validation error response.
func parse422Error(body []byte) error {
	var errResp apiErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			StatusCode: 422,
			Message:    "Validation error (unable to parse response)",
		}
	}

	if len(errResp.Errors) > 0 {
		return &APIError{
			StatusCode:  422,
			FieldErrors: errResp.Errors,
		}
	}

	msg := errResp.Message
	if msg == "" {
		msg = errResp.Error
	}
	if msg == "" {
		msg = "Validation error"
	}

	return &APIError{
		StatusCode: 422,
		Message:    msg,
	}
}

// parseErrorMessage extracts a message from a JSON error response body.
func parseErrorMessage(body []byte) string {
	var errResp apiErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return string(body)
	}
	if errResp.Message != "" {
		return errResp.Message
	}
	if errResp.Error != "" {
		return errResp.Error
	}
	return ""
}
