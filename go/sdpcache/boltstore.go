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

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// cacheOpenOptions are the bbolt options used for every Open call in this
// package. Since this is a cache layer, crash durability is unnecessary:
//   - NoSync skips fdatasync per commit, removing the single-writer bottleneck.
//   - NoFreelistSync skips persisting the freelist, reducing write amplification.
var cacheOpenOptions = &bbolt.Options{
	Timeout:        5 * time.Second,
	NoSync:         true,
	NoFreelistSync: true,
}

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
	expiryNano := int64(expiryNanoUint) //nolint:gosec // G115 (overflow): guarded by underflow check that clamps to zero
	// Check for overflow when converting uint64 to int64
	if expiryNano < 0 && expiryNanoUint > 0 {
		expiryNano = 0
	}
	expiry := time.Unix(0, expiryNano)

	// Find the separators
	rest := key[9:] // skip the first separator
	before, after, ok := bytes.Cut(rest, []byte{'|'})
	if !ok {
		return time.Time{}, "", nil, errors.New("invalid expiry key format")
	}

	sstHash := SSTHash(before)
	entryKey := after

	return expiry, sstHash, entryKey, nil
}

// boltStore holds the bbolt-backed storage implementation reused by both
// BoltCache and ShardedCache. It handles storage and purge execution only;
// purge scheduling (timer, goroutine) is owned by the Cache-level wrapper.
type boltStore struct {
	db   *bbolt.DB
	path string

	// CompactThreshold is the number of deleted bytes before triggering compaction
	CompactThreshold int64

	// Track deleted bytes for compaction
	deletedBytes int64
	deletedMu    sync.Mutex

	// Ensures that compaction operations aren't running concurrently
	// Read operations use RLock, write operations and compaction use Lock
	compactMutex sync.RWMutex
}

// BoltCacheOption is a functional option for configuring bolt-backed storage.
type BoltCacheOption func(*boltStore)

// WithCompactThreshold sets the threshold for triggering compaction
func WithCompactThreshold(bytes int64) BoltCacheOption {
	return func(c *boltStore) {
		c.CompactThreshold = bytes
	}
}

func newBoltCacheStore(path string, opts ...BoltCacheOption) (*boltStore, error) {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// bbolt.Open will open an existing file if present, or create a new one
	db, err := bbolt.Open(path, 0o600, cacheOpenOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt database: %w", err)
	}

	c := &boltStore{
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
func (c *boltStore) initBuckets() error {
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
func (c *boltStore) loadDeletedBytes() error {
	return c.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket(metaBucketName)
		if meta == nil {
			return nil
		}

		data := meta.Get(deletedBytesKey)
		if len(data) == 8 {
			deletedBytesUint := binary.BigEndian.Uint64(data)
			deletedBytes := int64(deletedBytesUint) //nolint:gosec // G115 (overflow): guarded by underflow check that clamps to zero
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
func (c *boltStore) saveDeletedBytes(tx *bbolt.Tx) error {
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
func (c *boltStore) addDeletedBytes(n int64) {
	c.deletedMu.Lock()
	c.deletedBytes += n
	c.deletedMu.Unlock()
}

// getDeletedBytes returns the current deleted bytes count (thread-safe)
func (c *boltStore) getDeletedBytes() int64 {
	c.deletedMu.Lock()
	defer c.deletedMu.Unlock()
	return c.deletedBytes
}

// resetDeletedBytes resets the deleted bytes counter (thread-safe)
func (c *boltStore) resetDeletedBytes() {
	c.deletedMu.Lock()
	c.deletedBytes = 0
	c.deletedMu.Unlock()
}

// getFileSize returns the size of the BoltDB file, logging any errors
func (c *boltStore) getFileSize() int64 {
	if c == nil || c.path == "" {
		return 0
	}

	stat, err := os.Stat(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("BoltDB cache file does not exist: %s", c.path)
		} else {
			log.WithError(err).Warnf("Failed to stat BoltDB cache file: %s", c.path)
		}
		return 0
	}

	return stat.Size()
}

// setDiskUsageAttributes sets disk usage attributes on a span
func (c *boltStore) setDiskUsageAttributes(span trace.Span) {
	if c == nil {
		return
	}

	fileSize := c.getFileSize()
	deletedBytes := c.getDeletedBytes()
	span.SetAttributes(
		attribute.Int64("ovm.boltdb.fileSizeBytes", fileSize),
		attribute.Int64("ovm.boltdb.deletedBytes", deletedBytes),
		attribute.Int64("ovm.boltdb.compactThresholdBytes", c.CompactThreshold),
	)
}

// CloseAndDestroy closes the database and deletes the cache file.
// This method makes the destructive behavior explicit.
func (c *boltStore) CloseAndDestroy() error {
	if c == nil {
		return nil
	}
	// Acquire write lock to prevent compaction from interfering
	c.compactMutex.Lock()
	defer c.compactMutex.Unlock()

	// Get the file path before closing
	path := c.db.Path()

	// Close the database
	if err := c.db.Close(); err != nil {
		return err
	}

	// Delete the cache file
	return os.Remove(path)
}

// deleteCacheFile removes the cache file entirely. This is used as a last resort
// when the disk is full and cleanup doesn't help. It closes the database,
// removes the file, and resets internal state.
func (c *boltStore) deleteCacheFile(ctx context.Context) error {
	if c == nil {
		return nil
	}

	// Create a span for this operation
	ctx, span := tracing.Tracer().Start(ctx, "BoltCache.deleteCacheFile", trace.WithAttributes(
		attribute.String("ovm.cache.path", c.path),
	))
	defer span.End()

	// Acquire write lock to prevent compaction from interfering
	c.compactMutex.Lock()
	defer c.compactMutex.Unlock()

	return c.deleteCacheFileLocked(ctx, span)
}

// deleteCacheFileLocked is the internal version that assumes the caller already holds compactMutex.Lock()
func (c *boltStore) deleteCacheFileLocked(ctx context.Context, span trace.Span) error {
	// Close the database if it's open
	if err := c.db.Close(); err != nil {
		span.RecordError(err)
		sentry.CaptureException(err)
		log.WithContext(ctx).WithError(err).Error("Failed to close database during cache file deletion")
	}

	// Remove the cache file
	if c.path != "" {
		if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
			span.RecordError(err)
			sentry.CaptureException(err)
			log.WithContext(ctx).WithError(err).Error("Failed to remove cache file")
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
		span.SetAttributes(attribute.Bool("ovm.cache.file_deleted", true))
	}

	// Reset internal state
	c.resetDeletedBytes()

	// Reopen the database
	db, err := bbolt.Open(c.path, 0o600, cacheOpenOptions)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to reopen database")
		return fmt.Errorf("failed to reopen database: %w", err)
	}

	c.db = db

	// Initialize buckets
	if err := c.initBuckets(); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to initialize buckets after cache file deletion: %w", err)
	}

	return nil
}

// Search performs a lower-level search using a CacheKey, bypassing pendingWork
// deduplication. This is used by ShardedCache and lookupCoordinator.
// If ctx contains a span, detailed timing metrics will be added as span attributes.
func (c *boltStore) Search(ctx context.Context, ck CacheKey) ([]*sdp.Item, error) {
	if c == nil {
		return nil, nil
	}

	// Get span from context if available
	span := trace.SpanFromContext(ctx)

	// Acquire read lock to prevent compaction from closing the database, but do not lock out other bbolt operations
	lockAcquireStart := time.Now()
	c.compactMutex.RLock()
	lockAcquireDuration := time.Since(lockAcquireStart)
	defer c.compactMutex.RUnlock()

	results := make([]*CachedResult, 0)
	var itemsScanned int

	txStart := time.Now()
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
			itemsScanned++
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
	txDuration := time.Since(txStart)

	// Add detailed search metrics to span if available
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64("ovm.cache.lockAcquireDuration_ms", lockAcquireDuration.Milliseconds()),
			attribute.Int64("ovm.cache.txDuration_ms", txDuration.Milliseconds()),
			attribute.Int("ovm.cache.itemsScanned", itemsScanned),
			attribute.Int("ovm.cache.itemsReturned", len(results)),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cache search failed")
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
			items = append(items, res.Item)
		}
	}

	return items, nil
}

// StoreItem stores an item in the cache with the specified duration.
func (c *boltStore) StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey) {
	if item == nil || c == nil {
		return
	}

	// Acquire read lock to prevent compaction from closing the database, but do not lock out other bbolt operations
	c.compactMutex.RLock()
	defer c.compactMutex.RUnlock()

	methodStr := ""
	if ck.Method != nil {
		methodStr = ck.Method.String()
	}

	ctx, span := tracing.Tracer().Start(ctx, "BoltCache.StoreItem",
		trace.WithAttributes(
			attribute.String("ovm.cache.method", methodStr),
			attribute.String("ovm.cache.scope", ck.SST.Scope),
			attribute.String("ovm.cache.type", ck.SST.Type),
			attribute.String("ovm.cache.sourceName", ck.SST.SourceName),
			attribute.String("ovm.cache.itemType", item.GetType()),
			attribute.String("ovm.cache.itemScope", item.GetScope()),
			attribute.String("ovm.cache.duration", duration.String()),
		),
	)
	defer span.End()

	// Set disk usage metrics
	c.setDiskUsageAttributes(span)

	res := CachedResult{
		Item:   item,
		Error:  nil,
		Expiry: time.Now().Add(duration),
		IndexValues: IndexValues{
			UniqueAttributeValue: item.UniqueAttributeValue(),
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

// StoreUnavailableItem stores an error in the cache with the specified duration.
func (c *boltStore) StoreUnavailableItem(ctx context.Context, err error, duration time.Duration, ck CacheKey) {
	if c == nil || err == nil {
		return
	}

	// Acquire read lock to prevent compaction from closing the database, but do not lock out other bbolt operations
	c.compactMutex.RLock()
	defer c.compactMutex.RUnlock()

	methodStr := ""
	if ck.Method != nil {
		methodStr = ck.Method.String()
	}

	ctx, span := tracing.Tracer().Start(ctx, "BoltCache.StoreUnavailableItem",
		trace.WithAttributes(
			attribute.String("ovm.cache.method", methodStr),
			attribute.String("ovm.cache.scope", ck.SST.Scope),
			attribute.String("ovm.cache.type", ck.SST.Type),
			attribute.String("ovm.cache.sourceName", ck.SST.SourceName),
			attribute.String("ovm.cache.error", err.Error()),
			attribute.String("ovm.cache.duration", duration.String()),
		),
	)
	defer span.End()

	// Set disk usage metrics
	c.setDiskUsageAttributes(span)

	res := CachedResult{
		Item:        nil,
		Error:       err,
		Expiry:      time.Now().Add(duration),
		IndexValues: ck.ToIndexValues(),
	}

	c.storeResult(ctx, res)
}

// storeResult stores a CachedResult in the database
func (c *boltStore) storeResult(ctx context.Context, res CachedResult) {
	span := trace.SpanFromContext(ctx)

	entry, err := fromCachedResult(&res)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize cache result")
		return
	}

	entryBytes, err := encodeCachedEntry(entry)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to encode cache entry")
		return
	}

	entryKey := makeEntryKey(res.IndexValues, res.Item)
	expiryKey := makeExpiryKey(res.Expiry, res.IndexValues.SSTHash, entryKey)

	overwritten := false
	entrySize := int64(len(entryBytes))

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
							attribute.String("ovm.cache.sst_hash", string(res.IndexValues.SSTHash)),
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
	// Note: storeResult is called from StoreItem/StoreUnavailableItem which already holds compactMutex.RLock()
	// so we use the locked versions to avoid deadlock
	if err != nil && isDiskFullError(err) {
		// Attempt cleanup by purging expired items - needs to happen in a
		// goroutine to avoid deadlocks and get a fresh write lock
		go func() {
			// we need a fresh write lock to block concurrent compaction and
			// deleteCacheFileLocked operations. Retrying performUpdate under
			// the write lock will ensure that only one instance of this
			// goroutine will actually perform the deleteCacheFileLocked.
			c.compactMutex.Lock()
			defer c.compactMutex.Unlock()

			ctx, purgeSpan := tracing.Tracer().Start(ctx, "BoltCache.purgeLocked")
			defer purgeSpan.End()
			c.purgeLocked(ctx, time.Now())

			// Retry the write operation once
			err = performUpdate()

			// If still failing with disk full, delete the cache entirely - use locked version
			if err != nil && isDiskFullError(err) {
				deleteCtx, deleteSpan := tracing.Tracer().Start(ctx, "BoltCache.deleteCacheFileLocked", trace.WithAttributes(
					attribute.String("ovm.cache.path", c.path),
				))
				defer deleteSpan.End()
				_ = c.deleteCacheFileLocked(deleteCtx, deleteSpan)
				// After deleting the cache, we can't store the result, so just return
				return
			}
		}()
		// now return to release the read lock and allow the goroutine above to run
		return
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to store result")
		// Update disk usage metrics even on error
		c.setDiskUsageAttributes(span)
		return
	}

	if !overwritten {
		span.SetAttributes(attribute.Bool("ovm.cache.unexpired_overwrite", false))
	}

	// Add entry size and update disk usage metrics
	span.SetAttributes(
		attribute.Int64("ovm.boltdb.entrySizeBytes", entrySize),
	)
	c.setDiskUsageAttributes(span)
}

// Delete removes all entries matching the given cache key.
func (c *boltStore) Delete(ck CacheKey) {
	if c == nil {
		return
	}

	// Acquire read lock to prevent compaction from closing the database, but do not lock out other bbolt operations
	c.compactMutex.RLock()
	defer c.compactMutex.RUnlock()

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
func (c *boltStore) Clear() {
	if c == nil {
		return
	}

	// Acquire read lock to prevent compaction from closing the database, but do not lock out other bbolt operations
	c.compactMutex.RLock()
	defer c.compactMutex.RUnlock()

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
func (c *boltStore) Purge(ctx context.Context, before time.Time) PurgeStats {
	if c == nil {
		return PurgeStats{}
	}

	ctx, span := tracing.Tracer().Start(ctx, "BoltCache.Purge",
		trace.WithAttributes(
			attribute.String("ovm.boltdb.purgeBefore", before.Format(time.RFC3339)),
		),
	)
	defer span.End()

	stats := func() PurgeStats {
		// Acquire read lock to prevent compaction from closing the database, but do not lock out other bbolt operations
		c.compactMutex.RLock()
		defer c.compactMutex.RUnlock()

		return c.purgeLocked(ctx, before)
	}()

	// Check if compaction is needed
	deletedBytesBeforeCompact := c.getDeletedBytes()
	compactionTriggered := deletedBytesBeforeCompact >= c.CompactThreshold

	if compactionTriggered {
		span.SetAttributes(
			attribute.Bool("ovm.boltdb.compactionTriggered", true),
			attribute.Int64("ovm.boltdb.deletedBytesBeforeCompact", deletedBytesBeforeCompact),
		)
		if err := c.compact(ctx); err == nil {
			span.SetAttributes(attribute.Bool("ovm.boltdb.compactionSuccess", true))
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, "compaction failed")
			span.SetAttributes(attribute.Bool("ovm.boltdb.compactionSuccess", false))
		}
	} else {
		span.SetAttributes(attribute.Bool("ovm.boltdb.compactionTriggered", false))
	}

	return stats
}

// purgeLocked is the internal version that assumes the caller already holds compactMutex.Lock()
// It performs the actual purging work and returns the stats, but does not handle compaction.
func (c *boltStore) purgeLocked(ctx context.Context, before time.Time) PurgeStats {
	span := trace.SpanFromContext(ctx)

	// Set initial disk usage metrics
	c.setDiskUsageAttributes(span)

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

	if err := c.db.View(func(tx *bbolt.Tx) error {
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
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "purge scan failed")
	}

	// Delete expired entries
	if len(expired) > 0 {
		if err := c.db.Update(func(tx *bbolt.Tx) error {
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
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "purge delete failed")
		}
	}

	// Update final disk usage metrics
	c.setDiskUsageAttributes(span)

	span.SetAttributes(
		attribute.Int("ovm.boltdb.numPurged", numPurged),
		attribute.Int64("ovm.boltdb.totalDeletedBytes", totalDeleted),
	)
	if nextExpiry != nil {
		span.SetAttributes(attribute.String("ovm.boltdb.nextExpiry", nextExpiry.Format(time.RFC3339)))
	}

	return PurgeStats{
		NumPurged:  numPurged,
		TimeTaken:  time.Since(start),
		NextExpiry: nextExpiry,
	}
}

// compact performs database compaction to reclaim disk space
func (c *boltStore) compact(ctx context.Context) error {
	// Acquire global lock to prevent any concurrent bbolt operations
	c.compactMutex.Lock()
	defer c.compactMutex.Unlock()

	ctx, span := tracing.Tracer().Start(ctx, "BoltCache.compact")
	defer span.End()

	// Set initial disk usage metrics
	c.setDiskUsageAttributes(span)

	fileSizeBefore := c.getFileSize()
	if fileSizeBefore > 0 {
		span.SetAttributes(attribute.Int64("ovm.boltdb.fileSizeBeforeBytes", fileSizeBefore))
	}

	// Create a temporary file for the compacted database
	tempPath := c.path + ".compact"

	// Helper to handle disk full errors during file operations
	// Note: We already hold compactMutex.Lock(), so we use the locked versions
	handleDiskFull := func(err error, operation string) error {
		if isDiskFullError(err) {
			// Attempt cleanup first - use locked version since we already hold the lock
			c.purgeLocked(ctx, time.Now())
			// If cleanup didn't help, delete the cache - use locked version
			deleteCtx, deleteSpan := tracing.Tracer().Start(ctx, "BoltCache.deleteCacheFileLocked", trace.WithAttributes(
				attribute.String("ovm.cache.path", c.path),
			))
			defer deleteSpan.End()
			_ = c.deleteCacheFileLocked(deleteCtx, deleteSpan)
			return fmt.Errorf("disk full during %s, cache deleted: %w", operation, err)
		}
		return err
	}

	// Open the destination database
	dstDB, err := bbolt.Open(tempPath, 0o600, cacheOpenOptions)
	if err != nil {
		if isDiskFullError(err) {
			// Attempt cleanup first - use locked version since we already hold the lock
			c.purgeLocked(ctx, time.Now())
			// Try again
			dstDB, err = bbolt.Open(tempPath, 0o600, cacheOpenOptions)
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
			// Attempt cleanup first - use locked version since we already hold the lock
			c.purgeLocked(ctx, time.Now())
			// Try compaction again
			dstDB2, retryErr := bbolt.Open(tempPath, 0o600, cacheOpenOptions)
			if retryErr != nil {
				return handleDiskFull(retryErr, "temp database creation after cleanup")
			}
			if compactErr := bbolt.Compact(dstDB2, c.db, 0); compactErr != nil {
				dstDB2.Close()
				os.Remove(tempPath)
				return handleDiskFull(compactErr, "compaction after cleanup")
			}
			// Success on retry, continue with dstDB2
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
		c.db, _ = bbolt.Open(c.path, 0o600, cacheOpenOptions)
		return handleDiskFull(err, "rename")
	}

	// Reopen the database
	db, err := bbolt.Open(c.path, 0o600, cacheOpenOptions)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to reopen database")
		return fmt.Errorf("failed to reopen database: %w", err)
	}

	c.db = db

	// Set final disk usage metrics and compaction results
	fileSizeAfter := c.getFileSize()
	spaceReclaimed := fileSizeBefore - fileSizeAfter

	span.SetAttributes(
		attribute.Int64("ovm.boltdb.fileSizeAfterBytes", fileSizeAfter),
		attribute.Int64("ovm.boltdb.spaceReclaimedBytes", spaceReclaimed),
	)
	c.setDiskUsageAttributes(span)

	// update deleted bytes after compaction
	c.resetDeletedBytes()
	_ = c.db.Update(func(tx *bbolt.Tx) error {
		return c.saveDeletedBytes(tx)
	})

	return nil
}
