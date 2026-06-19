// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: terraform-provider-ractermx, Property 5: Pagination completeness
// For any total item count T and page size P (where P ≥ 1 and T ≥ 0), the
// client's ListAll method should return exactly T items by fetching ⌈T/P⌉ pages,
// and the returned items should be the concatenation of all pages in order.
//
// **Validates: Requirements 2.7**

func TestProperty_PaginationCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		totalItems := rapid.IntRange(0, 500).Draw(t, "totalItems")
		// The client always requests per_page=100, so we simulate the server
		// responding with pages of up to 100 items.
		perPage := 100

		// Generate all items with sequential IDs for ordering verification
		allItems := make([]map[string]int, totalItems)
		for i := 0; i < totalItems; i++ {
			allItems[i] = map[string]int{"id": i}
		}

		// Calculate expected pages
		lastPage := 1
		if totalItems > 0 {
			lastPage = (totalItems + perPage - 1) / perPage
		}

		var pagesRequested []int

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pageStr := r.URL.Query().Get("page")
			page, _ := strconv.Atoi(pageStr)
			if page == 0 {
				page = 1
			}
			pagesRequested = append(pagesRequested, page)

			// Calculate the slice of items for this page
			start := (page - 1) * perPage
			end := start + perPage
			if end > totalItems {
				end = totalItems
			}
			if start > totalItems {
				start = totalItems
			}

			pageItems := allItems[start:end]

			resp := map[string]interface{}{
				"data": pageItems,
				"meta": map[string]interface{}{
					"total":        totalItems,
					"per_page":     perPage,
					"current_page": page,
					"last_page":    lastPage,
				},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		client := NewClient("test-key", ts.URL, "1.0.0")
		client.BackoffMultiplier = 1 * time.Millisecond

		ctx := context.Background()
		results, err := client.ListAll(ctx, "/api/v2/test-items")
		if err != nil {
			t.Fatalf("ListAll failed: %v", err)
		}

		// Verify total count
		if len(results) != totalItems {
			t.Errorf("got %d items, want %d", len(results), totalItems)
		}

		// Verify ordering by checking sequential IDs
		for i, raw := range results {
			var item map[string]int
			if err := json.Unmarshal(raw, &item); err != nil {
				t.Fatalf("failed to unmarshal item %d: %v", i, err)
			}
			if item["id"] != i {
				t.Errorf("item[%d] has id=%d, want %d", i, item["id"], i)
			}
		}

		// Verify correct number of pages were requested
		expectedPages := lastPage
		if totalItems == 0 {
			expectedPages = 1 // At least one request is made
		}
		if len(pagesRequested) != expectedPages {
			t.Errorf("requested %d pages, want %d", len(pagesRequested), expectedPages)
		}

		// Verify pages were requested in order
		for i, page := range pagesRequested {
			if page != i+1 {
				t.Errorf("page request[%d] = %d, want %d", i, page, i+1)
			}
		}
	})
}

func TestListAll_NonPaginated(t *testing.T) {
	// Test non-paginated endpoints (no meta object)
	items := []map[string]string{
		{"id": "1", "name": "webhook1"},
		{"id": "2", "name": "webhook2"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": items,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient("test-key", ts.URL, "1.0.0")
	client.BackoffMultiplier = 1 * time.Millisecond

	results, err := client.ListAll(context.Background(), fmt.Sprintf("/api/v2/webhooks"))
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("got %d items, want 2", len(results))
	}
}
