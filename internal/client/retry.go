// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// doWithRetry executes an HTTP request with retry logic for transient errors.
// - 429 (Too Many Requests): retry up to 3 times with 1s/2s/4s backoff
// - 5xx (Server Error): retry up to 2 times with 1s/2s backoff
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	// Save the request body for retries
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var lastResp *http.Response
	maxAttempts := 4 // 1 initial + 3 retries (max for 429)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Reset body for retry
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		statusCode := resp.StatusCode

		if statusCode == 429 {
			maxRetries := 3
			if attempt < maxRetries {
				resp.Body.Close()
				backoff := time.Duration(1<<uint(attempt)) * c.BackoffMultiplier
				time.Sleep(backoff)
				lastResp = resp
				continue
			}
			// Exhausted retries for 429
			return resp, nil
		}

		if statusCode >= 500 {
			maxRetries := 2
			if attempt < maxRetries {
				resp.Body.Close()
				backoff := time.Duration(1<<uint(attempt)) * c.BackoffMultiplier
				time.Sleep(backoff)
				lastResp = resp
				continue
			}
			// Exhausted retries for 5xx
			return resp, nil
		}

		// Non-retryable status code
		return resp, nil
	}

	// Should not reach here, but return last response if we do
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, &APIError{StatusCode: 0, Message: "max retries exceeded"}
}
