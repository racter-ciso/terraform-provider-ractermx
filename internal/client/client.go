// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the HTTP client for the RacterMX API v2.
type Client struct {
	BaseURL    string       // e.g., "https://ractermx.com/api/v2"
	APIKey     string       // Bearer token
	HTTPClient *http.Client // With configured timeout
	UserAgent  string       // "terraform-provider-ractermx/<version>"

	// BackoffMultiplier controls the base duration for exponential backoff.
	// Defaults to 1*time.Second. Set to a small value in tests.
	BackoffMultiplier time.Duration
}

// NewClient creates a new RacterMX API client.
func NewClient(apiKey, baseURL, version string) *Client {
	// Trim trailing slash and append /api/v2
	baseURL = strings.TrimRight(baseURL, "/") + "/api/v2"

	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent:         fmt.Sprintf("terraform-provider-ractermx/%s", version),
		BackoffMultiplier: 1 * time.Second,
	}
}

// Get performs a GET request. If isRead is true, a 404 returns (nil, nil)
// instead of an error (resource removed from state).
func (c *Client) Get(ctx context.Context, path string, isRead bool) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, isRead)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequestWithBody(ctx, http.MethodPost, path, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequestWithBody(ctx, http.MethodPatch, path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequestWithBody(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = c.handleResponse(resp, false)
	return err
}

// DeleteWithBody performs a DELETE request with a JSON body.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodDelete, path, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = c.handleResponse(resp, false)
	return err
}

// doRequestWithBody is a helper for POST/PATCH/PUT requests with JSON bodies.
func (c *Client) doRequestWithBody(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequest(ctx, method, path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, false)
}

// newRequest creates a new HTTP request with auth and user-agent headers.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}

// handleResponse processes the HTTP response and returns the body or an error.
func (c *Client) handleResponse(resp *http.Response, isRead bool) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}

	return nil, parseAPIError(resp.StatusCode, body, isRead)
}
