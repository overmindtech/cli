package sdpcache

import (
	"context"
	"sync"
)

// pendingWork tracks in-flight cache lookups to prevent duplicate work.
// When multiple goroutines request the same cache key simultaneously,
// only the first one does the actual work while others wait for the result.
type pendingWork struct {
	mu      sync.Mutex
	pending map[string]*workEntry
}

// workEntry represents a pending piece of work that one or more goroutines
// are waiting on.
type workEntry struct {
	done      chan struct{} // closed when work is complete
	cancelled bool          // true if work was cancelled, not completed normally
}

// newPendingWork creates a new pendingWork tracker.
func newPendingWork() *pendingWork {
	return &pendingWork{
		pending: make(map[string]*workEntry),
	}
}

// StartWork checks if work is already pending for the given key.
// If no work is pending, it creates a new entry and returns (true, entry) -
// the caller should do the work and call Complete when done.
// If work is already pending, it returns (false, entry) - the caller should
// call Wait on the entry to get the result.
func (p *pendingWork) StartWork(key string) (shouldWork bool, entry *workEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, ok := p.pending[key]; ok {
		return false, existing
	}

	entry = &workEntry{
		done: make(chan struct{}),
	}
	p.pending[key] = entry
	return true, entry
}

// Wait blocks until the work entry is ready or the context is cancelled.
// Returns ok=true if the work completed successfully (caller should re-check cache).
// Returns ok=false if the context was cancelled or work was cancelled.
func (p *pendingWork) Wait(ctx context.Context, entry *workEntry) (ok bool) {
	select {
	case <-entry.done:
		return !entry.cancelled
	case <-ctx.Done():
		return false
	}
}

// Complete marks the work as done and wakes all waiters.
// Waiters will receive ok=true and should re-lookup the cache.
func (p *pendingWork) Complete(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.pending[key]
	if !ok {
		return
	}
	delete(p.pending, key)
	close(entry.done)
}

// Cancel removes a pending work entry without storing a result.
// Waiters will receive ok=false and should retry or return error.
func (p *pendingWork) Cancel(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.pending[key]
	if !ok {
		return
	}
	delete(p.pending, key)
	entry.cancelled = true
	close(entry.done)
}
