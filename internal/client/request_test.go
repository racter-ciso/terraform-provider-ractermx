// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: terraform-provider-ractermx, Property 2: HTTP request construction
// For any non-empty API key, valid base URL, and version string, every HTTP
// request constructed by the client should:
// - Include an Authorization header with value "Bearer {api_key}"
// - Target a URL that starts with "{base_url}/api/v2"
// - Include a User-Agent header matching "terraform-provider-ractermx/{version}"
//
// **Validates: Requirements 1.5, 1.6, 2.6**

func TestProperty_HTTPRequestConstruction(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate non-empty API key (printable ASCII, no whitespace)
		apiKey := rapid.StringMatching(`[a-zA-Z0-9_]{1,64}`).Draw(t, "apiKey")
		version := rapid.StringMatching(`[a-z0-9.]{1,20}`).Draw(t, "version")

		// Create a test server that captures the request
		var capturedReq *http.Request
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedReq = r.Clone(r.Context())
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}))
		defer ts.Close()

		client := NewClient(apiKey, ts.URL, version)

		// Pick a random HTTP method to test
		methods := []string{"GET", "POST", "PATCH", "PUT", "DELETE"}
		methodIdx := rapid.IntRange(0, len(methods)-1).Draw(t, "methodIdx")
		method := methods[methodIdx]

		path := "/" + rapid.StringMatching(`[a-z0-9/]{0,30}`).Draw(t, "path")
		ctx := context.Background()

		switch method {
		case "GET":
			client.Get(ctx, path, false)
		case "POST":
			client.Post(ctx, path, map[string]string{"key": "value"})
		case "PATCH":
			client.Patch(ctx, path, map[string]string{"key": "value"})
		case "PUT":
			client.Put(ctx, path, map[string]string{"key": "value"})
		case "DELETE":
			client.Delete(ctx, path)
		}

		if capturedReq == nil {
			t.Fatal("no request was captured by the test server")
		}

		// Verify Authorization header
		authHeader := capturedReq.Header.Get("Authorization")
		expectedAuth := "Bearer " + apiKey
		if authHeader != expectedAuth {
			t.Errorf("Authorization header = %q, want %q", authHeader, expectedAuth)
		}

		// Verify User-Agent header
		uaHeader := capturedReq.Header.Get("User-Agent")
		expectedUA := "terraform-provider-ractermx/" + version
		if uaHeader != expectedUA {
			t.Errorf("User-Agent header = %q, want %q", uaHeader, expectedUA)
		}

		// Verify URL starts with base_url/api/v2
		requestURL := capturedReq.URL.Path
		expectedPrefix := "/api/v2" + path
		if !strings.HasPrefix(requestURL, expectedPrefix) {
			t.Errorf("URL path = %q, want prefix %q", requestURL, expectedPrefix)
		}

		// Verify Content-Type for methods with body
		if method == "POST" || method == "PATCH" || method == "PUT" {
			ct := capturedReq.Header.Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type header = %q, want %q", ct, "application/json")
			}
		}
	})
}
