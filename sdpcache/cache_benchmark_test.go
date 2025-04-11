package sdpcache

import (
	"math/rand"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
)

const CacheDuration = 10 * time.Second

// NewPopulatedCache Returns a newly populated cache and the CacheQuery that
// matches a randomly selected item in that cache
func NewPopulatedCache(numberItems int) (*Cache, CacheKey) {
	// Populate the cache
	c := NewCache()

	var item *sdp.Item
	var exampleCk CacheKey
	exampleIndex := rand.Intn(numberItems)

	for i := range numberItems {
		item = GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		if i == exampleIndex {
			exampleCk = ck
		}

		c.StoreItem(item, CacheDuration, ck)
	}

	return c, exampleCk
}

func BenchmarkCache1SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(1)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache10SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(10)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache100SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(100)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache1000SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(1000)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache10_000SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(10_000)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(query)

		if err != nil {
			b.Fatal(err)
		}
	}
}
