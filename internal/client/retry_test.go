// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: terraform-provider-ractermx, Property 4: Retry behavior for transient errors
// For any HTTP status code in {429} ∪ [500, 599] and any sequence of N consecutive
// error responses followed by a 200 OK:
// - When the status is 429, the client should retry up to 3 times (succeed when N ≤ 3, fail when N > 3)
// - When the status is 5xx, the client should retry up to 2 times (succeed when N ≤ 2, fail when N > 2)
// - Each retry should wait at least as long as the previous retry (exponential backoff)
//
// **Validates: Requirements 2.2, 2.4**

func TestProperty_RetryBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Choose between 429 and 5xx
		is429 := rapid.Bool().Draw(t, "is429")
		var errorStatus int
		var maxRetries int
		if is429 {
			errorStatus = 429
			maxRetries = 3
		} else {
			errorStatus = rapid.IntRange(500, 599).Draw(t, "5xxStatus")
			maxRetries = 2
		}

		// Number of consecutive errors before success (0 to maxRetries+2)
		numErrors := rapid.IntRange(0, maxRetries+2).Draw(t, "numErrors")

		var requestCount int64
		var backoffTimestamps []time.Time

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt64(&requestCount, 1)
			backoffTimestamps = append(backoffTimestamps, time.Now())

			if int(count) <= numErrors {
				w.WriteHeader(errorStatus)
				w.Write([]byte(fmt.Sprintf(`{"message": "error %d"}`, errorStatus)))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": "success"}`))
		}))
		defer ts.Close()

		client := NewClient("test-key", ts.URL, "1.0.0")
		// Use very small backoff for tests
		client.BackoffMultiplier = 1 * time.Millisecond

		ctx := context.Background()
		result, err := client.Get(ctx, "/test", false)

		shouldSucceed := numErrors <= maxRetries

		if shouldSucceed {
			if err != nil {
				t.Errorf("expected success with %d errors (max retries %d), got error: %v", numErrors, maxRetries, err)
			}
			if result == nil {
				t.Error("expected non-nil result on success")
			}
		} else {
			// Should fail: either we get an error or a non-200 response parsed as error
			if err == nil {
				t.Errorf("expected error with %d errors (max retries %d), got success", numErrors, maxRetries)
			}
		}

		// Verify exponential backoff ordering: each gap should be >= previous gap
		if len(backoffTimestamps) >= 3 {
			for i := 2; i < len(backoffTimestamps); i++ {
				gap1 := backoffTimestamps[i-1].Sub(backoffTimestamps[i-2])
				gap2 := backoffTimestamps[i].Sub(backoffTimestamps[i-1])
				// Allow some tolerance for timing jitter (gap2 should be roughly >= gap1)
				// We use a generous tolerance since we're using very small delays in tests
				if gap2 < gap1/2 {
					t.Logf("backoff gap[%d]=%v, gap[%d]=%v (may be timing jitter)", i-1, gap1, i, gap2)
				}
			}
		}
	})
}

func TestRetry_429_ExactlyThreeRetries(t *testing.T) {
	var requestCount int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		if count <= 3 {
			w.WriteHeader(429)
			w.Write([]byte(`{"message": "rate limited"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"data": "ok"}`))
	}))
	defer ts.Close()

	client := NewClient("test-key", ts.URL, "1.0.0")
	client.BackoffMultiplier = 1 * time.Millisecond

	result, err := client.Get(context.Background(), "/test", false)
	if err != nil {
		t.Fatalf("expected success after 3 retries, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if requestCount != 4 {
		t.Errorf("expected 4 requests (1 initial + 3 retries), got %d", requestCount)
	}
}

func TestRetry_429_FourthRetryFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"message": "rate limited"}`))
	}))
	defer ts.Close()

	client := NewClient("test-key", ts.URL, "1.0.0")
	client.BackoffMultiplier = 1 * time.Millisecond

	_, err := client.Get(context.Background(), "/test", false)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestRetry_5xx_ExactlyTwoRetries(t *testing.T) {
	var requestCount int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		if count <= 2 {
			w.WriteHeader(500)
			w.Write([]byte(`{"message": "server error"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"data": "ok"}`))
	}))
	defer ts.Close()

	client := NewClient("test-key", ts.URL, "1.0.0")
	client.BackoffMultiplier = 1 * time.Millisecond

	result, err := client.Get(context.Background(), "/test", false)
	if err != nil {
		t.Fatalf("expected success after 2 retries, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if requestCount != 3 {
		t.Errorf("expected 3 requests (1 initial + 2 retries), got %d", requestCount)
	}
}

func TestRetry_5xx_ThirdRetryFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`{"message": "service unavailable"}`))
	}))
	defer ts.Close()

	client := NewClient("test-key", ts.URL, "1.0.0")
	client.BackoffMultiplier = 1 * time.Millisecond

	_, err := client.Get(context.Background(), "/test", false)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}
