package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// AdapterHost This struct holds references to all Adapters in a process
// and provides utility functions to work with them. Methods of this
// struct are safe to call concurrently.
type AdapterHost struct {
	// Map of types to all adapters for that type
	adapters []Adapter
	mutex    sync.RWMutex
}

func NewAdapterHost() *AdapterHost {
	sh := &AdapterHost{
		adapters: make([]Adapter, 0),
	}

	return sh
}

var ErrAdapterAlreadyExists = errors.New("adapter already exists")

// AddAdapters Adds an adapter to this engine
func (sh *AdapterHost) AddAdapters(adapters ...Adapter) error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	for _, newAdapter := range adapters {
		for _, existingAdapter := range sh.adapters {
			if existingAdapter.Type() == newAdapter.Type() && scopesOverlap(existingAdapter.Scopes(), newAdapter.Scopes()) {
				log.Errorf("Error: Adapter with type %s and overlapping scopes already exists. Existing adapter scopes: %v, New adapter scopes: %v",
					existingAdapter.Type(), existingAdapter.Scopes(), newAdapter.Scopes())
				return fmt.Errorf("adapter with type %s and overlapping scopes already exists", newAdapter.Type())
			}
		}
		sh.adapters = append(sh.adapters, newAdapter)
	}

	return nil
}

// scopesOverlap checks if there is any overlap between two slices of scopes
func scopesOverlap(scopes1, scopes2 []string) bool {
	scopeSet := make(map[string]struct{}, len(scopes1))
	for _, scope := range scopes1 {
		scopeSet[scope] = struct{}{}
	}
	for _, scope := range scopes2 {
		if _, exists := scopeSet[scope]; exists {
			return true
		}
	}
	return false
}

// Adapters Returns a slice of all known adapters
func (sh *AdapterHost) Adapters() []Adapter {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	adapters := make([]Adapter, 0)

	adapters = append(adapters, sh.adapters...)

	return adapters
}

// VisibleAdapters Returns a slice of all known adapters excluding hidden ones
func (sh *AdapterHost) VisibleAdapters() []Adapter {
	allAdapters := sh.Adapters()
	result := make([]Adapter, 0)

	// Add all adapters unless they are hidden
	for _, adapter := range allAdapters {
		if hs, ok := adapter.(HiddenAdapter); ok {
			if hs.Hidden() {
				// If the adapter is hidden, continue without adding it
				continue
			}
		}

		result = append(result, adapter)
	}

	return result
}

// AdapterByType Returns the adapters for a given type
func (sh *AdapterHost) AdaptersByType(typ string) []Adapter {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	adapters := make([]Adapter, 0)

	for _, adapter := range sh.adapters {
		if adapter.Type() == typ {
			adapters = append(adapters, adapter)
		}
	}

	return adapters
}

// ExpandQuery Expands queries with wildcards to no longer contain wildcards.
// Meaning that if we support 5 types, and a query comes in with a wildcard
// type, this function will expand that query into 5 queries, one for each
// type.
//
// The same goes for scopes, if we have a query with a wildcard scope, and
// a single adapter that supports 5 scopes, we will end up with 5 queries. The
// exception to this is if we have a adapter that supports all scopes, but is
// unable to list them. In this case there will still be some queries with
// wildcard scopes as they can't be expanded
//
// This functions returns a map of queries with the adapters that they should be
// run against
func (sh *AdapterHost) ExpandQuery(q *sdp.Query) map[*sdp.Query]Adapter {
	var checkAdapters []Adapter

	if IsWildcard(q.GetType()) {
		// If the query has a wildcard type, all non-hidden adapters might try
		// to respond
		checkAdapters = sh.VisibleAdapters()
	} else {
		// If the type is specific, pull just adapters for that type
		checkAdapters = append(checkAdapters, sh.AdaptersByType(q.GetType())...)
	}

	expandedQueries := make(map[*sdp.Query]Adapter)

	for _, adapter := range checkAdapters {
		// is the adapter is hidden
		isHidden := false
		if hs, ok := adapter.(HiddenAdapter); ok {
			isHidden = hs.Hidden()
		}

		for _, adapterScope := range adapter.Scopes() {
			// Create a new query if:
			//
			// * The adapter supports all scopes, or
			// * The query scope is a wildcard (and the adapter is not hidden), or
			// * The query scope substring matches adapter scope
			if IsWildcard(adapterScope) || (IsWildcard(q.GetScope()) && !isHidden) || strings.Contains(adapterScope, q.GetScope()) {
				dest := proto.Clone(q).(*sdp.Query)

				dest.Type = adapter.Type()

				// Choose the more specific scope
				if IsWildcard(adapterScope) {
					dest.Scope = q.GetScope()
				} else {
					dest.Scope = adapterScope
				}

				expandedQueries[dest] = adapter
			}
		}
	}

	return expandedQueries
}

// ClearAllAdapters Removes all adapters from the engine
func (sh *AdapterHost) ClearAllAdapters() {
	sh.mutex.Lock()
	sh.adapters = make([]Adapter, 0)
	sh.mutex.Unlock()
}

// StartPurger Starts the purger for all caching adapters
func (sh *AdapterHost) StartPurger(ctx context.Context) {
	for _, s := range sh.Adapters() {
		if c, ok := s.(CachingAdapter); ok {
			cache := c.Cache()
			if cache != nil {
				err := cache.StartPurger(ctx)
				if err != nil {
					sentry.CaptureException(fmt.Errorf("failed to start purger for adapter %s: %w", s.Name(), err))
				}
			}
		}
	}
}

func (sh *AdapterHost) Purge() {
	for _, s := range sh.Adapters() {
		if c, ok := s.(CachingAdapter); ok {
			cache := c.Cache()
			if cache != nil {
				cache.Purge(time.Now())
			}
		}
	}
}

// ClearCaches Clears caches for all caching adapters
func (sh *AdapterHost) ClearCaches() {
	for _, s := range sh.Adapters() {
		if c, ok := s.(CachingAdapter); ok {
			cache := c.Cache()
			if cache != nil {
				cache.Clear()
			}
		}
	}
}
