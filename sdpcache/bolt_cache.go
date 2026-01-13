package sdpcache

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"go.etcd.io/bbolt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

// Bucket names for bbolt
var (
	itemsBucketName  = []byte("items")
	expiryBucketName = []byte("expiry")
	metaBucketName   = []byte("meta")
	deletedBytesKey  = []byte("deletedBytes")
)

// DefaultCompactThreshold is the default threshold for triggering compaction (100MB)
const DefaultCompactThreshold = 100 * 1024 * 1024

// isDiskFullError checks if an error is due to disk being full (ENOSPC)
func isDiskFullError(err error) bool {
	if err == nil {
		return false
	}
	// Check if it wraps ENOSPC
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.ENOSPC {
		return true
	}
	// Check using errors.Is for wrapped errors
	return errors.Is(err, syscall.ENOSPC)
}

// encodeCachedEntry serializes a CachedEntry to bytes using protobuf
func encodeCachedEntry(e *sdp.CachedEntry) ([]byte, error) {
	return proto.Marshal(e)
}

// decodeCachedEntry deserializes bytes to a CachedEntry using protobuf
func decodeCachedEntry(data []byte) (*sdp.CachedEntry, error) {
	e := &sdp.CachedEntry{}
	if err := proto.Unmarshal(data, e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached entry: %w", err)
	}
	return e, nil
}

// toCachedResult converts a CachedEntry to a CachedResult
func cachedEntryToCachedResult(e *sdp.CachedEntry) *CachedResult {
	result := &CachedResult{
		Item:   e.GetItem(),
		Expiry: time.Unix(0, e.GetExpiryUnixNano()),
		IndexValues: IndexValues{
			SSTHash:              SSTHash(e.GetSstHash()),
			UniqueAttributeValue: e.GetUniqueAttributeValue(),
			Method:               e.GetMethod(),
			Query:                e.GetQuery(),
		},
	}
	// Only set Error if it's actually meaningful (not nil and not zero-value)
	err := e.GetError()
	if err != nil && (err.GetErrorType() != 0 || err.GetErrorString() != "" || err.GetScope() != "" || err.GetSourceName() != "" || err.GetItemType() != "") {
		result.Error = err
	}
	return result
}

// fromCachedResult creates a CachedEntry from a CachedResult
func fromCachedResult(cr *CachedResult) (*sdp.CachedEntry, error) {
	e := &sdp.CachedEntry{
		Item:                 cr.Item,
		ExpiryUnixNano:       cr.Expiry.UnixNano(),
		UniqueAttributeValue: cr.IndexValues.UniqueAttributeValue,
		Method:               cr.IndexValues.Method,
		Query:                cr.IndexValues.Query,
		SstHash:              string(cr.IndexValues.SSTHash),
	}

	if cr.Error != nil {
		// Try to cast to QueryError for protobuf serialization
		var qErr *sdp.QueryError
		if errors.As(cr.Error, &qErr) {
			e.Error = qErr
		} else {
			// For non-QueryError errors, wrap in a QueryError
			e.Error = &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: cr.Error.Error(),
			}
		}
	}

	return e, nil
}

// makeEntryKey creates a key for storing an entry in the items bucket
// Format: {method}|{query}|{uniqueAttributeValue}|{globallyUniqueName}
func makeEntryKey(iv IndexValues, item *sdp.Item) []byte {
	var gun string
	if item != nil {
		gun = item.GloballyUniqueName()
	}
	key := fmt.Sprintf("%d|%s|%s|%s", iv.Method, iv.Query, iv.UniqueAttributeValue, gun)
	return []byte(key)
}

// makeExpiryKey creates a key for the expiry index
// Format: {expiryNano}|{sstHash}|{entryKey}
func makeExpiryKey(expiry time.Time, sstHash SSTHash, entryKey []byte) []byte {
	// Use big-endian encoding for expiry so keys sort chronologically
	buf := make([]byte, 8+1+len(sstHash)+1+len(entryKey))
	expiryNano := expiry.UnixNano()
	var expiryNanoUint uint64
	if expiryNano < 0 {
		expiryNanoUint = 0
	} else {
		expiryNanoUint = uint64(expiryNano)
	}
	binary.BigEndian.PutUint64(buf[0:8], expiryNanoUint)
	buf[8] = '|'
	copy(buf[9:], []byte(sstHash))
	buf[9+len(sstHash)] = '|'
	copy(buf[10+len(sstHash):], entryKey)
	return buf
}

// parseExpiryKey extracts the expiry time, sst hash, and entry key from an expiry key
func parseExpiryKey(key []byte) (time.Time, SSTHash, []byte, error) {
	if len(key) < 10 {
		return time.Time{}, "", nil, errors.New("expiry key too short")
	}

	expiryNanoUint := binary.BigEndian.Uint64(key[0:8])
	expiryNano := int64(expiryNanoUint)
	// Check for overflow when converting uint64 to int64
	if expiryNano < 0 && expiryNanoUint > 0 {
		expiryNano = 0
	}
	expiry := time.Unix(0, expiryNano)

	// Find the separators
	rest := key[9:] // skip the first separator
	sepIdx := bytes.IndexByte(rest, '|')
	if sepIdx < 0 {
		return time.Time{}, "", nil, errors.New("invalid expiry key format")
	}

	sstHash := SSTHash(rest[:sepIdx])
	entryKey := rest[sepIdx+1:]

	return expiry, sstHash, entryKey, nil
}

// BoltCache implements the Cache interface using bbolt for persistent storage
type BoltCache struct {
	db   *bbolt.DB
	path string

	// Minimum amount of time to wait between cache purges
	MinWaitTime time.Duration

	// CompactThreshold is the number of deleted bytes before triggering compaction
	CompactThreshold int64

	// The timer that is used to trigger the next purge
	purgeTimer *time.Timer

	// The time that the purger will run next
	nextPurge time.Time

	// Ensures that purge stats like `purgeTimer` and `nextPurge` aren't being
	// modified concurrently
	purgeMutex sync.Mutex

	// Track deleted bytes for compaction
	deletedBytes int64
	deletedMu    sync.Mutex
}

// BoltCacheOption is a functional option for configuring BoltCache
type BoltCacheOption func(*BoltCache)

// WithMinWaitTime sets the minimum wait time between purges
func WithMinWaitTime(d time.Duration) BoltCacheOption {
	return func(c *BoltCache) {
		c.MinWaitTime = d
	}
}

// WithCompactThreshold sets the threshold for triggering compaction
func WithCompactThreshold(bytes int64) BoltCacheOption {
	return func(c *BoltCache) {
		c.CompactThreshold = bytes
	}
}

// NewBoltCache creates a new BoltCache at the specified path.
// If a cache file already exists at the path, it will be opened and used.
// The existing file will be automatically handled by the purge process,
// which removes expired items. No explicit cleanup is needed on startup.
func NewBoltCache(path string, opts ...BoltCacheOption) (*BoltCache, error) {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// bbolt.Open will open an existing file if present, or create a new one
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt database: %w", err)
	}

	c := &BoltCache{
		db:               db,
		path:             path,
		CompactThreshold: DefaultCompactThreshold,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize buckets
	if err := c.initBuckets(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize buckets: %w", err)
	}

	// Load deleted bytes from meta
	if err := c.loadDeletedBytes(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load deleted bytes: %w", err)
	}

	return c, nil
}

// initBuckets creates the required buckets if they don't exist
func (c *BoltCache) initBuckets() error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(itemsBucketName); err != nil {
			return fmt.Errorf("failed to create items bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(expiryBucketName); err != nil {
			return fmt.Errorf("failed to create expiry bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(metaBucketName); err != nil {
			return fmt.Errorf("failed to create meta bucket: %w", err)
		}
		return nil
	})
}

// loadDeletedBytes loads the deleted bytes counter from the meta bucket
func (c *BoltCache) loadDeletedBytes() error {
	return c.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket(metaBucketName)
		if meta == nil {
			return nil
		}

		data := meta.Get(deletedBytesKey)
		if len(data) == 8 {
			deletedBytesUint := binary.BigEndian.Uint64(data)
			deletedBytes := int64(deletedBytesUint)
			// Check for overflow when converting uint64 to int64
			if deletedBytes < 0 && deletedBytesUint > 0 {
				deletedBytes = 0
			}
			c.deletedBytes = deletedBytes
		}
		return nil
	})
}

// saveDeletedBytes saves the deleted bytes counter to the meta bucket
func (c *BoltCache) saveDeletedBytes(tx *bbolt.Tx) error {
	meta := tx.Bucket(metaBucketName)
	if meta == nil {
		return errors.New("meta bucket not found")
	}

	buf := make([]byte, 8)
	deletedBytes := c.deletedBytes
	var deletedBytesUint uint64
	if deletedBytes < 0 {
		deletedBytesUint = 0
	} else {
		deletedBytesUint = uint64(deletedBytes)
	}
	binary.BigEndian.PutUint64(buf, deletedBytesUint)
	return meta.Put(deletedBytesKey, buf)
}

// addDeletedBytes adds to the deleted bytes counter (thread-safe)
func (c *BoltCache) addDeletedBytes(n int64) {
	c.deletedMu.Lock()
	c.deletedBytes += n
	c.deletedMu.Unlock()
}

// getDeletedBytes returns the current deleted bytes count (thread-safe)
func (c *BoltCache) getDeletedBytes() int64 {
	c.deletedMu.Lock()
	defer c.deletedMu.Unlock()
	return c.deletedBytes
}

// resetDeletedBytes resets the deleted bytes counter (thread-safe)
func (c *BoltCache) resetDeletedBytes() {
	c.deletedMu.Lock()
	c.deletedBytes = 0
	c.deletedMu.Unlock()
}

// Close closes the database
func (c *BoltCache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

// deleteCacheFile removes the cache file entirely. This is used as a last resort
// when the disk is full and cleanup doesn't help. It closes the database,
// removes the file, and resets internal state.
func (c *BoltCache) deleteCacheFile() error {
	if c == nil {
		return nil
	}

	// Close the database if it's open
	if c.db != nil {
		_ = c.db.Close()
		c.db = nil
	}

	// Remove the cache file
	if c.path != "" {
		if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
	}

	// Reset internal state
	c.resetDeletedBytes()

	return nil
}

// Lookup performs a cache lookup for the given query parameters.
func (c *BoltCache) Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError) {
	span := trace.SpanFromContext(ctx)
	ck := CacheKeyFromParts(srcName, method, scope, typ, query)

	if c == nil || c.db == nil {
		span.SetAttributes(
			attribute.String("ovm.cache.result", "cache not initialised"),
			attribute.Bool("ovm.cache.hit", false),
		)
		return false, ck, nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "cache has not been initialised",
			Scope:       scope,
			SourceName:  srcName,
			ItemType:    typ,
		}
	}

	if ignoreCache {
		span.SetAttributes(
			attribute.String("ovm.cache.result", "ignore cache"),
			attribute.Bool("ovm.cache.hit", false),
		)
		return false, ck, nil, nil
	}

	items, err := c.Search(ck)

	if err != nil {
		var qErr *sdp.QueryError
		if errors.Is(err, ErrCacheNotFound) {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache miss"),
				attribute.Bool("ovm.cache.hit", false),
			)
			return false, ck, nil, nil
		} else if errors.As(err, &qErr) {
			if qErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				span.SetAttributes(attribute.String("ovm.cache.result", "cache hit: item not found"))
			} else {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "cache hit: QueryError"),
					attribute.String("ovm.cache.error", err.Error()),
				)
			}

			span.SetAttributes(attribute.Bool("ovm.cache.hit", true))
			return true, ck, nil, qErr
		} else {
			qErr = &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
				SourceName:  srcName,
				ItemType:    typ,
			}

			span.SetAttributes(
				attribute.String("ovm.cache.error", err.Error()),
				attribute.String("ovm.cache.result", "cache hit: unknown QueryError"),
				attribute.Bool("ovm.cache.hit", true),
			)

			return true, ck, nil, qErr
		}
	}

	if method == sdp.QueryMethod_GET {
		if len(items) < 2 {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache hit: 1 item"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, ck, items, nil
		} else {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache returned >1 value, purging and continuing"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", false),
			)
			c.Delete(ck)
			return false, ck, nil, nil
		}
	}

	span.SetAttributes(
		attribute.String("ovm.cache.result", "cache hit: multiple items"),
		attribute.Int("ovm.cache.numItems", len(items)),
		attribute.Bool("ovm.cache.hit", true),
	)

	return true, ck, items, nil
}

// Search performs a lower-level search using a CacheKey.
func (c *BoltCache) Search(ck CacheKey) ([]*sdp.Item, error) {
	if c == nil || c.db == nil {
		return nil, nil
	}

	results := make([]*CachedResult, 0)

	err := c.db.View(func(tx *bbolt.Tx) error {
		items := tx.Bucket(itemsBucketName)
		if items == nil {
			return nil
		}

		sstHash := ck.SST.Hash()
		sstBucket := items.Bucket([]byte(sstHash))
		if sstBucket == nil {
			return nil
		}

		now := time.Now()

		// Scan through all entries in this SST bucket
		cursor := sstBucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			entry, err := decodeCachedEntry(v)
			if err != nil {
				continue // Skip corrupted entries
			}

			// Check if expired
			expiry := time.Unix(0, entry.GetExpiryUnixNano())
			if expiry.Before(now) {
				continue
			}

			// Check if matches the cache key
			entryIV := IndexValues{
				SSTHash:              SSTHash(entry.GetSstHash()),
				UniqueAttributeValue: entry.GetUniqueAttributeValue(),
				Method:               entry.GetMethod(),
				Query:                entry.GetQuery(),
			}
			if !ck.Matches(entryIV) {
				continue
			}

			result := cachedEntryToCachedResult(entry)
			results = append(results, result)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return nil, ErrCacheNotFound
	}

	// Check for errors first
	items := make([]*sdp.Item, 0, len(results))
	for _, res := range results {
		if res.Error != nil {
			return nil, res.Error
		}

		if res.Item != nil {
			// Return a copy of the item
			itemCopy := proto.Clone(res.Item).(*sdp.Item)
			items = append(items, itemCopy)
		}
	}

	return items, nil
}

// StoreItem stores an item in the cache with the specified duration.
func (c *BoltCache) StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey) {
	if item == nil || c == nil || c.db == nil {
		return
	}

	itemCopy := proto.Clone(item).(*sdp.Item)

	// Ensure minimum duration to avoid items expiring immediately
	// This handles cases where time.Until() returns 0 or negative due to timing
	// Use 100ms to account for race detector overhead and slow CI environments
	if duration <= 100*time.Millisecond {
		duration = 100 * time.Millisecond
	}

	res := CachedResult{
		Item:   itemCopy,
		Error:  nil,
		Expiry: time.Now().Add(duration),
		IndexValues: IndexValues{
			UniqueAttributeValue: itemCopy.UniqueAttributeValue(),
		},
	}

	if ck.Method != nil {
		res.IndexValues.Method = *ck.Method
	}
	if ck.Query != nil {
		res.IndexValues.Query = *ck.Query
	}

	res.IndexValues.SSTHash = ck.SST.Hash()

	c.storeResult(ctx, res)
}

// StoreError stores an error in the cache with the specified duration.
func (c *BoltCache) StoreError(ctx context.Context, err error, duration time.Duration, ck CacheKey) {
	if c == nil || c.db == nil || err == nil {
		return
	}

	// Ensure minimum duration to avoid items expiring immediately
	// Use 100ms to account for race detector overhead and slow CI environments
	if duration <= 100*time.Millisecond {
		duration = 100 * time.Millisecond
	}

	res := CachedResult{
		Item:        nil,
		Error:       err,
		Expiry:      time.Now().Add(duration),
		IndexValues: ck.ToIndexValues(),
	}

	c.storeResult(ctx, res)
}

// storeResult stores a CachedResult in the database
func (c *BoltCache) storeResult(ctx context.Context, res CachedResult) {
	entry, err := fromCachedResult(&res)
	if err != nil {
		return // Silently fail on serialization errors
	}

	entryBytes, err := encodeCachedEntry(entry)
	if err != nil {
		return
	}

	entryKey := makeEntryKey(res.IndexValues, res.Item)
	expiryKey := makeExpiryKey(res.Expiry, res.IndexValues.SSTHash, entryKey)

	span := trace.SpanFromContext(ctx)
	overwritten := false

	// Helper function to perform the actual database update
	performUpdate := func() error {
		return c.db.Update(func(tx *bbolt.Tx) error {
			items := tx.Bucket(itemsBucketName)
			if items == nil {
				return errors.New("items bucket not found")
			}

			// Get or create the SST sub-bucket
			sstBucket, err := items.CreateBucketIfNotExists([]byte(res.IndexValues.SSTHash))
			if err != nil {
				return fmt.Errorf("failed to create sst bucket: %w", err)
			}

			// Check if we're overwriting an unexpired entry
			existingData := sstBucket.Get(entryKey)
			if existingData != nil {
				existingEntry, err := decodeCachedEntry(existingData)
				if err == nil {
					existingExpiry := time.Unix(0, existingEntry.GetExpiryUnixNano())
					now := time.Now()
					if existingExpiry.After(now) {
						overwritten = true
						timeUntilExpiry := existingExpiry.Sub(now)

						attrs := []attribute.KeyValue{
							attribute.Bool("ovm.cache.unexpired_overwrite", true),
							attribute.String("ovm.cache.time_until_expiry", timeUntilExpiry.String()),
							attribute.String("ovm.cache.source_name", string(res.IndexValues.SSTHash)),
							attribute.String("ovm.cache.query_method", res.IndexValues.Method.String()),
						}

						if res.Item != nil {
							attrs = append(attrs,
								attribute.String("ovm.cache.item_type", res.Item.GetType()),
								attribute.String("ovm.cache.item_scope", res.Item.GetScope()),
							)
						}

						if res.IndexValues.Query != "" {
							attrs = append(attrs, attribute.String("ovm.cache.query", res.IndexValues.Query))
						}

						if res.IndexValues.UniqueAttributeValue != "" {
							attrs = append(attrs, attribute.String("ovm.cache.unique_attribute", res.IndexValues.UniqueAttributeValue))
						}

						span.SetAttributes(attrs...)

						// Delete old expiry key
						expiry := tx.Bucket(expiryBucketName)
						if expiry != nil {
							oldExpiryKey := makeExpiryKey(existingExpiry, res.IndexValues.SSTHash, entryKey)
							_ = expiry.Delete(oldExpiryKey)
						}
					}
				}
			}

			// Store the entry
			if err := sstBucket.Put(entryKey, entryBytes); err != nil {
				return fmt.Errorf("failed to store entry: %w", err)
			}

			// Store in expiry index
			expiry := tx.Bucket(expiryBucketName)
			if expiry == nil {
				return errors.New("expiry bucket not found")
			}
			if err := expiry.Put(expiryKey, nil); err != nil {
				return fmt.Errorf("failed to store expiry: %w", err)
			}

			return nil
		})
	}

	err = performUpdate()

	// Handle disk full errors
	if err != nil && isDiskFullError(err) {
		// Attempt cleanup by purging expired items
		c.Purge(time.Now())

		// Retry the write operation once
		err = performUpdate()

		// If still failing with disk full, delete the cache entirely
		if err != nil && isDiskFullError(err) {
			_ = c.deleteCacheFile()
			// After deleting the cache, we can't store the result, so just return
			return
		}
	}

	if err != nil {
		return
	}

	// Check if db is still valid (might have been deleted in error handling)
	if c.db == nil {
		return
	}

	if !overwritten {
		span.SetAttributes(attribute.Bool("ovm.cache.unexpired_overwrite", false))
	}

	// Update the purge time if required
	c.setNextPurgeIfEarlier(res.Expiry)
}

// Delete removes all entries matching the given cache key.
func (c *BoltCache) Delete(ck CacheKey) {
	if c == nil || c.db == nil {
		return
	}

	var totalDeleted int64

	_ = c.db.Update(func(tx *bbolt.Tx) error {
		items := tx.Bucket(itemsBucketName)
		if items == nil {
			return nil
		}

		sstHash := ck.SST.Hash()
		sstBucket := items.Bucket([]byte(sstHash))
		if sstBucket == nil {
			return nil
		}

		expiry := tx.Bucket(expiryBucketName)

		// Collect keys to delete
		keysToDelete := make([][]byte, 0)
		cursor := sstBucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			entry, err := decodeCachedEntry(v)
			if err != nil {
				continue
			}

			entryIV := IndexValues{
				SSTHash:              SSTHash(entry.GetSstHash()),
				UniqueAttributeValue: entry.GetUniqueAttributeValue(),
				Method:               entry.GetMethod(),
				Query:                entry.GetQuery(),
			}
			if ck.Matches(entryIV) {
				keysToDelete = append(keysToDelete, append([]byte(nil), k...))
				totalDeleted += int64(len(k) + len(v))

				// Delete from expiry index
				if expiry != nil {
					expiryTime := time.Unix(0, entry.GetExpiryUnixNano())
					expiryKey := makeExpiryKey(expiryTime, SSTHash(entry.GetSstHash()), k)
					_ = expiry.Delete(expiryKey)
				}
			}
		}

		// Delete the entries
		for _, k := range keysToDelete {
			_ = sstBucket.Delete(k)
		}

		return nil
	})

	if totalDeleted > 0 {
		c.addDeletedBytes(totalDeleted)
	}
}

// Clear removes all entries from the cache.
func (c *BoltCache) Clear() {
	if c == nil || c.db == nil {
		return
	}

	_ = c.db.Update(func(tx *bbolt.Tx) error {
		// Delete and recreate buckets
		_ = tx.DeleteBucket(itemsBucketName)
		_ = tx.DeleteBucket(expiryBucketName)

		_, _ = tx.CreateBucketIfNotExists(itemsBucketName)
		_, _ = tx.CreateBucketIfNotExists(expiryBucketName)

		// Reset deleted bytes in meta
		meta := tx.Bucket(metaBucketName)
		if meta != nil {
			buf := make([]byte, 8)
			_ = meta.Put(deletedBytesKey, buf)
		}

		return nil
	})

	c.resetDeletedBytes()
}

// Purge removes all expired items from the cache.
func (c *BoltCache) Purge(before time.Time) PurgeStats {
	if c == nil || c.db == nil {
		return PurgeStats{}
	}

	start := time.Now()
	var nextExpiry *time.Time
	numPurged := 0
	var totalDeleted int64

	// Collect expired entries
	type expiredEntry struct {
		sstHash  SSTHash
		entryKey []byte
		size     int64
	}
	expired := make([]expiredEntry, 0)

	_ = c.db.View(func(tx *bbolt.Tx) error {
		expiry := tx.Bucket(expiryBucketName)
		if expiry == nil {
			return nil
		}

		items := tx.Bucket(itemsBucketName)

		cursor := expiry.Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			expiryTime, sstHash, entryKey, err := parseExpiryKey(k)
			if err != nil {
				continue
			}

			if expiryTime.Before(before) {
				// Calculate size for deleted bytes tracking
				var size int64
				if items != nil {
					if sstBucket := items.Bucket([]byte(sstHash)); sstBucket != nil {
						if v := sstBucket.Get(entryKey); v != nil {
							size = int64(len(k) + len(entryKey) + len(v))
						}
					}
				}
				expired = append(expired, expiredEntry{
					sstHash:  sstHash,
					entryKey: append([]byte(nil), entryKey...),
					size:     size,
				})
			} else {
				// Found first non-expired entry
				nextExpiry = &expiryTime
				break
			}
		}

		return nil
	})

	// Delete expired entries
	if len(expired) > 0 {
		_ = c.db.Update(func(tx *bbolt.Tx) error {
			items := tx.Bucket(itemsBucketName)
			expiry := tx.Bucket(expiryBucketName)

			for _, e := range expired {
				// Delete from items
				if items != nil {
					if sstBucket := items.Bucket([]byte(e.sstHash)); sstBucket != nil {
						_ = sstBucket.Delete(e.entryKey)
					}
				}

				// Delete from expiry index
				if expiry != nil {
					// We need to reconstruct the expiry key
					cursor := expiry.Cursor()
					for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
						_, sstHash, entryKey, err := parseExpiryKey(k)
						if err != nil {
							continue
						}
						if sstHash == e.sstHash && bytes.Equal(entryKey, e.entryKey) {
							_ = expiry.Delete(k)
							break
						}
					}
				}

				totalDeleted += e.size
				numPurged++
			}

			// Save deleted bytes
			c.addDeletedBytes(totalDeleted)
			return c.saveDeletedBytes(tx)
		})
	}

	// Check if compaction is needed
	if c.getDeletedBytes() >= c.CompactThreshold {
		if err := c.compact(); err == nil {
			c.resetDeletedBytes()
			// Save reset state
			_ = c.db.Update(func(tx *bbolt.Tx) error {
				return c.saveDeletedBytes(tx)
			})
		}
	}

	return PurgeStats{
		NumPurged:  numPurged,
		TimeTaken:  time.Since(start),
		NextExpiry: nextExpiry,
	}
}

// compact performs database compaction to reclaim disk space
func (c *BoltCache) compact() error {
	// Create a temporary file for the compacted database
	tempPath := c.path + ".compact"

	// Helper to handle disk full errors during file operations
	handleDiskFull := func(err error, operation string) error {
		if isDiskFullError(err) {
			// Attempt cleanup first
			c.Purge(time.Now())
			// If cleanup didn't help, delete the cache
			_ = c.deleteCacheFile()
			return fmt.Errorf("disk full during %s, cache deleted: %w", operation, err)
		}
		return err
	}

	// Open the destination database
	dstDB, err := bbolt.Open(tempPath, 0600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		if isDiskFullError(err) {
			// Attempt cleanup first
			c.Purge(time.Now())
			// Try again
			dstDB, err = bbolt.Open(tempPath, 0600, &bbolt.Options{Timeout: 5 * time.Second})
			if err != nil {
				return handleDiskFull(err, "temp database creation")
			}
		} else {
			return fmt.Errorf("failed to create temp database: %w", err)
		}
	}

	// Compact from source to destination
	if err := bbolt.Compact(dstDB, c.db, 0); err != nil {
		dstDB.Close()
		os.Remove(tempPath)
		if isDiskFullError(err) {
			// Attempt cleanup first
			c.Purge(time.Now())
			// Try compaction again
			dstDB2, retryErr := bbolt.Open(tempPath, 0600, &bbolt.Options{Timeout: 5 * time.Second})
			if retryErr != nil {
				return handleDiskFull(retryErr, "temp database creation after cleanup")
			}
			if compactErr := bbolt.Compact(dstDB2, c.db, 0); compactErr != nil {
				dstDB2.Close()
				os.Remove(tempPath)
				return handleDiskFull(compactErr, "compaction after cleanup")
			}
			// Success on retry, continue with dstDB2
			dstDB.Close()
			dstDB = dstDB2
		} else {
			return fmt.Errorf("compaction failed: %w", err)
		}
	}

	// Close the destination database
	if err := dstDB.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp database: %w", err)
	}

	// Close the current database
	if err := c.db.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Replace the old file with the compacted one
	if err := os.Rename(tempPath, c.path); err != nil {
		// Try to reopen the original database
		c.db, _ = bbolt.Open(c.path, 0600, &bbolt.Options{Timeout: 5 * time.Second})
		return handleDiskFull(err, "rename")
	}

	// Reopen the database
	db, err := bbolt.Open(c.path, 0600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return fmt.Errorf("failed to reopen database: %w", err)
	}

	c.db = db
	return nil
}

// GetMinWaitTime returns the minimum time between purge operations
func (c *BoltCache) GetMinWaitTime() time.Duration {
	if c == nil {
		return 0
	}

	if c.MinWaitTime == 0 {
		return MinWaitDefault
	}

	return c.MinWaitTime
}

// StartPurger starts a background goroutine that automatically purges expired items.
func (c *BoltCache) StartPurger(ctx context.Context) error {
	if c == nil || c.db == nil {
		return nil
	}

	c.purgeMutex.Lock()
	if c.purgeTimer == nil {
		c.purgeTimer = time.NewTimer(0)
		c.purgeMutex.Unlock()
	} else {
		c.purgeMutex.Unlock()
		return errors.New("purger already running")
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-c.purgeTimer.C:
				stats := c.Purge(time.Now())
				c.setNextPurgeFromStats(stats)
			case <-ctx.Done():
				c.purgeMutex.Lock()
				defer c.purgeMutex.Unlock()

				c.purgeTimer.Stop()
				c.purgeTimer = nil
				return
			}
		}
	}(ctx)

	return nil
}

// setNextPurgeFromStats sets when the next purge should run based on the stats
func (c *BoltCache) setNextPurgeFromStats(stats PurgeStats) {
	c.purgeMutex.Lock()
	defer c.purgeMutex.Unlock()

	if stats.NextExpiry == nil {
		c.purgeTimer.Reset(1000 * time.Hour)
		c.nextPurge = time.Now().Add(1000 * time.Hour)
	} else {
		if time.Until(*stats.NextExpiry) < c.GetMinWaitTime() {
			c.purgeTimer.Reset(c.GetMinWaitTime())
			c.nextPurge = time.Now().Add(c.GetMinWaitTime())
		} else {
			c.purgeTimer.Reset(time.Until(*stats.NextExpiry))
			c.nextPurge = *stats.NextExpiry
		}
	}
}

// setNextPurgeIfEarlier sets the next purge time if the provided time is sooner
func (c *BoltCache) setNextPurgeIfEarlier(t time.Time) {
	c.purgeMutex.Lock()
	defer c.purgeMutex.Unlock()

	if t.Before(c.nextPurge) {
		if c.purgeTimer == nil {
			return
		}

		c.purgeTimer.Stop()
		c.nextPurge = t
		c.purgeTimer.Reset(time.Until(t))
	}
}
