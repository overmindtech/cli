package tracing

import (
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// safeUint64ToInt64 safely converts uint64 to int64 for OpenTelemetry attributes
// Returns int64 max value if the uint64 exceeds int64 maximum to prevent overflow
func safeUint64ToInt64(val uint64) int64 {
	const maxInt64 = 9223372036854775807 // 2^63 - 1
	if val > maxInt64 {                  // Check if val exceeds int64 max
		return maxInt64 // Return int64 max value
	}
	return int64(val)
}

// MemoryStats represents memory statistics at a point in time, converted to
// int64 for safe use as OpenTelemetry attributes.
//
// To diagnose OOMs we need to be able to tell three different "memory"
// numbers apart, because Go's accounting and the Linux RSS view diverge:
//   - Alloc/HeapAlloc: live heap objects right now (drops on every GC).
//   - HeapInuse: bytes in non-empty spans (live + per-span fragmentation).
//   - HeapIdle: bytes in empty spans the runtime is hanging onto.
//   - HeapReleased: idle bytes the scavenger has handed back to the OS via
//     madvise(MADV_DONTNEED). RSS-equivalent ≈ HeapInuse + HeapIdle - HeapReleased.
//   - HeapSys / Sys: total mappings ever obtained from the OS. Effectively a
//     high-water mark — does NOT decrease when the scavenger releases memory,
//     because madvise keeps the mapping in place. Comparing Sys vs. HeapReleased
//     tells us whether a "8 GB Sys" reading is real RSS pressure or just
//     bookkeeping from a previous peak.
type MemoryStats struct {
	Alloc        int64 // bytes allocated and not yet freed
	HeapAlloc    int64 // bytes allocated and not yet freed (same as Alloc above but specifically for heap objects)
	HeapInuse    int64 // bytes in in-use spans
	HeapIdle     int64 // bytes in idle (unused) spans
	HeapReleased int64 // bytes returned to the OS via madvise(MADV_DONTNEED)
	HeapSys      int64 // bytes of heap memory obtained from the OS (high-water mark)
	Sys          int64 // total bytes of memory obtained from the OS (heap + stacks + GC metadata + ...)
	NumGC        int64 // number of completed GC cycles
	PauseTotal   int64 // cumulative nanoseconds in GC stop-the-world pauses
}

// ReadMemoryStats captures current memory statistics and converts them to int64
func ReadMemoryStats() MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return MemoryStats{
		Alloc:        safeUint64ToInt64(memStats.Alloc),
		HeapAlloc:    safeUint64ToInt64(memStats.HeapAlloc),
		HeapInuse:    safeUint64ToInt64(memStats.HeapInuse),
		HeapIdle:     safeUint64ToInt64(memStats.HeapIdle),
		HeapReleased: safeUint64ToInt64(memStats.HeapReleased),
		HeapSys:      safeUint64ToInt64(memStats.HeapSys),
		Sys:          safeUint64ToInt64(memStats.Sys),
		NumGC:        int64(memStats.NumGC),
		PauseTotal:   safeUint64ToInt64(memStats.PauseTotalNs),
	}
}

// SetMemoryAttributes sets memory-related attributes on a span with the given prefix
func SetMemoryAttributes(span trace.Span, prefix string, memStats MemoryStats) {
	span.SetAttributes(
		attribute.Int64(prefix+".memoryBytes", memStats.Alloc),
		attribute.Int64(prefix+".memoryHeapBytes", memStats.HeapAlloc),
		attribute.Int64(prefix+".memoryHeapInuseBytes", memStats.HeapInuse),
		attribute.Int64(prefix+".memoryHeapIdleBytes", memStats.HeapIdle),
		attribute.Int64(prefix+".memoryHeapReleasedBytes", memStats.HeapReleased),
		attribute.Int64(prefix+".memoryHeapSysBytes", memStats.HeapSys),
		attribute.Int64(prefix+".memorySysBytes", memStats.Sys),
		attribute.Int64(prefix+".memoryNumGC", memStats.NumGC),
		attribute.Int64(prefix+".memoryPauseTotalNs", memStats.PauseTotal),
	)
}

// SetMemoryDeltaAttributes sets memory delta attributes on a span with the given prefix
// It calculates the difference between before and after memory stats
func SetMemoryDeltaAttributes(span trace.Span, prefix string, before, after MemoryStats) {
	span.SetAttributes(
		attribute.Int64(prefix+".memoryDeltaBytes", after.Alloc-before.Alloc),
		attribute.Int64(prefix+".memoryDeltaHeapBytes", after.HeapAlloc-before.HeapAlloc),
		attribute.Int64(prefix+".memoryDeltaHeapInuseBytes", after.HeapInuse-before.HeapInuse),
		attribute.Int64(prefix+".memoryDeltaHeapIdleBytes", after.HeapIdle-before.HeapIdle),
		attribute.Int64(prefix+".memoryDeltaHeapReleasedBytes", after.HeapReleased-before.HeapReleased),
		attribute.Int64(prefix+".memoryDeltaHeapSysBytes", after.HeapSys-before.HeapSys),
		attribute.Int64(prefix+".memoryDeltaSysBytes", after.Sys-before.Sys),
		attribute.Int64(prefix+".memoryDeltaNumGC", after.NumGC-before.NumGC),
		attribute.Int64(prefix+".memoryDeltaPauseTotalNs", after.PauseTotal-before.PauseTotal),
	)
}
