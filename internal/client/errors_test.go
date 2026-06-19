// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: terraform-provider-ractermx, Property 3: API error response parsing
// For any valid JSON object with an "errors" field containing a map of field
// names to string arrays, the client's error parser should produce one Terraform
// diagnostic per field, where each diagnostic's detail contains the field name
// and all associated error messages.
//
// **Validates: Requirements 2.1**

func TestProperty_APIErrorResponseParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random errors map with 1-5 fields, each with 1-3 messages
		numFields := rapid.IntRange(1, 5).Draw(t, "numFields")
		errorsMap := make(map[string][]string)

		for i := 0; i < numFields; i++ {
			fieldName := rapid.StringMatching(`[a-z_]{1,20}`).Draw(t, "fieldName")
			numMessages := rapid.IntRange(1, 3).Draw(t, "numMessages")
			messages := make([]string, numMessages)
			for j := 0; j < numMessages; j++ {
				messages[j] = rapid.StringMatching(`[A-Za-z0-9 ]{1,50}`).Draw(t, "message")
			}
			errorsMap[fieldName] = messages
		}

		// Build the JSON body
		body := map[string]interface{}{
			"errors": errorsMap,
		}
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal test body: %v", err)
		}

		// Parse the error
		result := parseAPIError(422, jsonBody, false)
		if result == nil {
			t.Fatal("parseAPIError returned nil for 422 with errors")
		}

		apiErr, ok := result.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", result)
		}

		// Verify status code
		if apiErr.StatusCode != 422 {
			t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
		}

		// Verify one entry per field in FieldErrors
		if len(apiErr.FieldErrors) != len(errorsMap) {
			t.Errorf("FieldErrors has %d fields, want %d", len(apiErr.FieldErrors), len(errorsMap))
		}

		// Verify each field's messages are preserved
		for field, expectedMsgs := range errorsMap {
			actualMsgs, exists := apiErr.FieldErrors[field]
			if !exists {
				t.Errorf("field %q missing from FieldErrors", field)
				continue
			}
			if len(actualMsgs) != len(expectedMsgs) {
				t.Errorf("field %q has %d messages, want %d", field, len(actualMsgs), len(expectedMsgs))
				continue
			}
			for i, msg := range expectedMsgs {
				if actualMsgs[i] != msg {
					t.Errorf("field %q message[%d] = %q, want %q", field, i, actualMsgs[i], msg)
				}
			}
		}

		// Verify the Error() string contains each field name and its messages
		errStr := apiErr.Error()
		for field, msgs := range errorsMap {
			if !strings.Contains(errStr, field) {
				t.Errorf("Error() string %q does not contain field name %q", errStr, field)
			}
			for _, msg := range msgs {
				if !strings.Contains(errStr, msg) {
					t.Errorf("Error() string %q does not contain message %q", errStr, msg)
				}
			}
		}
	})
}

func TestParseAPIError_401(t *testing.T) {
	err := parseAPIError(401, []byte(`{}`), false)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Message, "Invalid or expired API credentials") {
		t.Errorf("unexpected message: %s", apiErr.Message)
	}
}

func TestParseAPIError_404_Read(t *testing.T) {
	err := parseAPIError(404, []byte(`{}`), true)
	if err != nil {
		t.Errorf("expected nil for 404 during read, got: %v", err)
	}
}

func TestParseAPIError_404_NonRead(t *testing.T) {
	err := parseAPIError(404, []byte(`{}`), false)
	if err == nil {
		t.Fatal("expected error for 404 non-read")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "Resource not found" {
		t.Errorf("unexpected message: %s", apiErr.Message)
	}
}

func TestParseAPIError_409(t *testing.T) {
	body := `{"message": "Alias already exists"}`
	err := parseAPIError(409, []byte(body), false)
	if err == nil {
		t.Fatal("expected error for 409")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 409 {
		t.Errorf("StatusCode = %d, want 409", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Message, "Alias already exists") {
		t.Errorf("unexpected message: %s", apiErr.Message)
	}
}

func TestParseAPIError_5xx(t *testing.T) {
	body := `{"message": "Internal server error"}`
	err := parseAPIError(500, []byte(body), false)
	if err == nil {
		t.Fatal("expected error for 500")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}
