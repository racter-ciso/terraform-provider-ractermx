// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// paginatedResponse represents the JSON structure of a paginated API response.
type paginatedResponse struct {
	Data json.RawMessage `json:"data"`
	Meta *paginationMeta `json:"meta,omitempty"`
}

// paginationMeta represents the pagination metadata from the API.
type paginationMeta struct {
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
}

// ListAll fetches all items from a paginated API endpoint.
// It iterates through pages using ?page=N&per_page=100 until all pages are fetched.
// For non-paginated endpoints (no meta object), it returns the data array directly.
func (c *Client) ListAll(ctx context.Context, path string) ([]json.RawMessage, error) {
	var allItems []json.RawMessage
	page := 1
	perPage := 100

	for {
		// Build the URL with pagination params
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		paginatedPath := fmt.Sprintf("%s%spage=%d&per_page=%d", path, separator, page, perPage)

		body, err := c.Get(ctx, paginatedPath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		var resp paginatedResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse paginated response: %w", err)
		}

		// Parse the data array
		var items []json.RawMessage
		if err := json.Unmarshal(resp.Data, &items); err != nil {
			return nil, fmt.Errorf("failed to parse data array: %w", err)
		}

		allItems = append(allItems, items...)

		// If no meta object, this is a non-paginated endpoint
		if resp.Meta == nil {
			return allItems, nil
		}

		// Check if we've fetched all pages
		if resp.Meta.CurrentPage >= resp.Meta.LastPage {
			return allItems, nil
		}

		page++
	}
}
