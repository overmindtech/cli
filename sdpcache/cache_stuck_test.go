package sdpcache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
)

// TestListErrorWithProperCleanup tests the correct behavior where:
// 1. A LIST operation is performed and gets a cache miss
// 2. The caller starts the work
// 3. The query encounters an error
// 4. The caller properly calls StoreError to cache the error
// 5. Subsequent requests get the cached error immediately (don't block)
//
// This test documents the fix for the cache timeout bug.
func TestListErrorWithProperCleanup(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			ctx := t.Context()

			sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
			method := sdp.QueryMethod_LIST
			query := ""

			var wg sync.WaitGroup
			startBarrier := make(chan struct{})

			// Track timing
			var secondCallDuration time.Duration

			// First goroutine: Gets cache miss, simulates work that errors,
			// and properly calls StoreError to cache the error
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()

				if hit {
					t.Error("first goroutine: expected cache miss")
					return
				}

				// Simulate work that takes time and then errors
				time.Sleep(50 * time.Millisecond)

				// CORRECT BEHAVIOR: Worker encounters an error and properly caches it
				err := &sdp.QueryError{
					ErrorType:   sdp.QueryError_OTHER,
					ErrorString: "simulated list error",
				}
				cache.StoreError(ctx, err, 1*time.Hour, ck)
				t.Log("First goroutine: properly called StoreError")
			}()

			// Second goroutine: Should get cached error immediately
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				// Small delay to ensure first goroutine starts first
				time.Sleep(10 * time.Millisecond)

				// Use a short timeout to detect blocking
				timeoutCtx, done := context.WithTimeout(ctx, 500*time.Millisecond)
				defer done()

				start := time.Now()
				hit, _, _, qErr, done := cache.Lookup(timeoutCtx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()
				secondCallDuration = time.Since(start)

				if !hit {
					t.Error("second goroutine: expected cache hit (cached error)")
				}
				if qErr == nil {
					t.Error("second goroutine: expected cached error")
				}
				t.Logf("Second goroutine: got cached error after %v", secondCallDuration)
			}()

			// Release all goroutines
			close(startBarrier)
			wg.Wait()

			// Verify the second call got the result quickly (didn't block)
			if secondCallDuration > 200*time.Millisecond {
				t.Fatalf("Second call took too long (%v), possibly blocked waiting for pending work", secondCallDuration)
			}

			t.Logf("✓ Second call returned quickly (%v) with cached error - proper cleanup is working", secondCallDuration)
		})
	}
}

// TestListErrorWithProperCancellation tests the CORRECT behavior where:
// 1. A LIST operation is performed and gets a cache miss
// 2. The query encounters an error
// 3. The caller properly calls the done function
// 4. Subsequent requests should get a cache miss immediately (not block)
func TestListErrorWithProperDone(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			ctx := t.Context()

			sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
			method := sdp.QueryMethod_LIST
			query := ""

			var wg sync.WaitGroup
			startBarrier := make(chan struct{})

			// Track timing
			var secondCallDuration time.Duration

			// First goroutine: Gets cache miss, simulates work that errors,
			// and PROPERLY calls the done function
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)

				if hit {
					t.Error("first goroutine: expected cache miss")
					done() // Clean up even on error
					return
				}

				// Simulate work that takes time and then errors
				time.Sleep(100 * time.Millisecond)

				// CORRECT BEHAVIOR: Call done to release resources
				done()
				t.Log("First goroutine: properly called done()")
			}()

			// Second goroutine: Should receive cache miss quickly (not block)
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				// Small delay to ensure first goroutine starts first
				time.Sleep(10 * time.Millisecond)

				start := time.Now()
				hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()
				secondCallDuration = time.Since(start)

				if hit {
					t.Error("second goroutine: expected cache miss")
				}

				t.Logf("Second goroutine: got cache miss after %v", secondCallDuration)
			}()

			// Release all goroutines
			close(startBarrier)
			wg.Wait()

			// The second call should NOT block for long
			// It should get a cache miss shortly after the first call done() (~100ms)
			if secondCallDuration > 300*time.Millisecond {
				t.Errorf("Expected second call to return quickly after cancellation, but it took %v", secondCallDuration)
			}

			t.Logf("Test demonstrates correct behavior: second call returned in %v", secondCallDuration)
		})
	}
}

// TestListErrorWithStoreError tests the CORRECT behavior where:
// 1. A LIST operation is performed and gets a cache miss
// 2. The query encounters an error
// 3. The caller properly calls StoreError
// 4. Subsequent requests should get the cached error immediately
func TestListErrorWithStoreError(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			ctx := t.Context()

			sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
			method := sdp.QueryMethod_LIST
			query := ""

			var wg sync.WaitGroup
			startBarrier := make(chan struct{})

			expectedError := &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list returned error",
				Scope:       sst.Scope,
				SourceName:  sst.SourceName,
				ItemType:    sst.Type,
			}

			// Track results
			var secondCallHit bool
			var secondCallError *sdp.QueryError
			var secondCallDuration time.Duration

			// First goroutine: Gets cache miss, simulates work that errors,
			// and PROPERLY calls StoreError
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()

				if hit {
					t.Error("first goroutine: expected cache miss")
					return
				}

				// Simulate work that takes time and then errors
				time.Sleep(100 * time.Millisecond)

				// CORRECT BEHAVIOR: Store the error so other callers can get it
				cache.StoreError(ctx, expectedError, 10*time.Second, ck)
				t.Log("First goroutine: properly called StoreError")
			}()

			// Second goroutine: Should receive the cached error
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				// Small delay to ensure first goroutine starts first
				time.Sleep(10 * time.Millisecond)

				start := time.Now()
				var items []*sdp.Item
				var done func()
				secondCallHit, _, items, secondCallError, done = cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()
				secondCallDuration = time.Since(start)

				if items != nil {
					t.Error("second goroutine: expected nil items with error")
				}

				t.Logf("Second goroutine: got result after %v", secondCallDuration)
			}()

			// Release all goroutines
			close(startBarrier)
			wg.Wait()

			// The second call should get the cached error
			if !secondCallHit {
				t.Error("Expected cache hit with error")
			}

			if secondCallError == nil {
				t.Error("Expected error to be returned")
			}

			if secondCallError != nil && secondCallError.GetErrorType() != expectedError.GetErrorType() {
				t.Errorf("Expected error type %v, got %v", expectedError.GetErrorType(), secondCallError.GetErrorType())
			}

			// Should return relatively quickly (~100ms for first goroutine work)
			if secondCallDuration > 300*time.Millisecond {
				t.Errorf("Expected second call to return quickly with cached error, but it took %v", secondCallDuration)
			}

			t.Logf("Test demonstrates correct behavior: second call got cached error in %v", secondCallDuration)
		})
	}
}

// TestListReturnsEmptyButNoStore tests the scenario where:
// 1. A LIST operation completes successfully but finds no items
// 2. The caller calls Complete() but doesn't store anything
// 3. Subsequent requests should get cache miss (not error)
func TestListReturnsEmptyButNoStore(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			ctx := t.Context()

			sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
			method := sdp.QueryMethod_LIST
			query := ""

			var wg sync.WaitGroup
			startBarrier := make(chan struct{})

			var secondCallHit bool
			var secondCallDuration time.Duration

			// First goroutine: LIST returns 0 items, completes without storing
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()

				if hit {
					t.Error("first goroutine: expected cache miss")
					return
				}

				// Simulate work that completes and finds no items
				time.Sleep(100 * time.Millisecond)

				// Complete without storing anything (LIST found 0 items)
				// This is handled by the underlying pending work mechanism
				switch c := cache.(type) {
				case *MemoryCache:
					c.pending.Complete(ck.String())
				case *BoltCache:
					c.pending.Complete(ck.String())
				}

				t.Log("First goroutine: completed work but stored nothing")
			}()

			// Second goroutine: Should get cache miss
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				// Small delay to ensure first goroutine starts first
				time.Sleep(10 * time.Millisecond)

				start := time.Now()
				secondCallHit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
				defer done()
				secondCallDuration = time.Since(start)

				t.Logf("Second goroutine: hit=%v, duration=%v", secondCallHit, secondCallDuration)
			}()

			// Release all goroutines
			close(startBarrier)
			wg.Wait()

			// Second call should get cache miss (not error)
			if secondCallHit {
				t.Error("Expected cache miss when first caller completed without storing")
			}

			// Should return relatively quickly (~100ms for first goroutine work)
			if secondCallDuration > 300*time.Millisecond {
				t.Errorf("Expected second call to return quickly, but it took %v", secondCallDuration)
			}
		})
	}
}
