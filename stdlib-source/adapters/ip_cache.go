package adapters

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/google/btree"
)

type entry[EntryType any] struct {
	Network *net.IPNet // The CIDR this entry is for
	Expiry  time.Time  // When this entry expires
	Object  EntryType  // The actual stored object
}

type IPCache[EntryType any] struct {
	storage *btree.BTreeG[entry[EntryType]]
	mu      sync.RWMutex
}

func NewIPCache[EntryType any]() *IPCache[EntryType] {
	return &IPCache[EntryType]{
		storage: btree.NewG[entry[EntryType]](2, func(a, b entry[EntryType]) bool {
			// Sort by the network mask number i.e. /8, /16, /24, etc in numeric
			// order. This means if we want to find the most specific CIDR that
			// contains an IP, we can just iterate through the tree in descending
			// order
			aSize, _ := a.Network.Mask.Size()
			bSize, _ := b.Network.Mask.Size()

			return aSize < bSize
		}),
	}
}

// Stores an object in the cache for the given duration. The "Key" is the CIDR
func (c *IPCache[EntryType]) Store(cidr *net.IPNet, object EntryType, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.storage.ReplaceOrInsert(entry[EntryType]{
		Network: cidr,
		Expiry:  time.Now().Add(duration),
		Object:  object,
	})
}

// Searched for the most specific CIDR that contains the specified IP
func (c *IPCache[EntryType]) SearchIP(ip net.IP) (EntryType, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var found *entry[EntryType]
	var object EntryType

	// Iterate through the tree in descending order
	c.storage.Descend(func(current entry[EntryType]) bool {
		if current.Network.Contains(ip) {
			found = &current
			return false
		}
		return true
	})

	if found != nil {
		object = found.Object

		return object, true
	}

	return object, false
}

// Search the cache for the specified CIDR
func (c *IPCache[EntryType]) SearchCIDR(cidr *net.IPNet) (EntryType, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var found *entry[EntryType]
	var object EntryType

	// Iterate through the tree in descending order
	c.storage.Descend(func(current entry[EntryType]) bool {
		if current.Network.String() == cidr.String() {
			found = &current
			return false
		}
		return true
	})

	if found != nil {
		object = found.Object

		return object, true
	}

	return object, false
}

// Finds items that have expired and removes them from the cache, returns the
// number of expired items. You need to pass in the current time, this will
// usually be time.Now() but it can be useful to pass in a fixed time for
// testing
func (c *IPCache[EntryType]) Expire(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expired int

	c.storage.Ascend(func(current entry[EntryType]) bool {
		if current.Expiry.Before(now) {
			c.storage.Delete(current)
			expired++
		}

		return true
	})

	return expired
}

// Starts a goroutine that will periodically check for expired items and removes
// them from the cache. You can pass in a context to cancel the goroutine and
// stop the purging
func (c *IPCache[EntryType]) StartPurger(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.Expire(time.Now())
			case <-ctx.Done():
				return
			}
		}
	}()
}
