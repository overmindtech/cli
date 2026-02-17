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

// MemoryStats represents memory statistics at a point in time, converted to int64 for safe use
type MemoryStats struct {
	Alloc      int64 // bytes allocated and not yet freed
	HeapAlloc  int64 // bytes allocated and not yet freed (same as Alloc above but specifically for heap objects)
	Sys        int64 // total bytes of memory obtained from the OS
	NumGC      int64 // number of completed GC cycles
	PauseTotal int64 // cumulative nanoseconds in GC stop-the-world pauses
}

// ReadMemoryStats captures current memory statistics and converts them to int64
func ReadMemoryStats() MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return MemoryStats{
		Alloc:      safeUint64ToInt64(memStats.Alloc),
		HeapAlloc:  safeUint64ToInt64(memStats.HeapAlloc),
		Sys:        safeUint64ToInt64(memStats.Sys),
		NumGC:      int64(memStats.NumGC),
		PauseTotal: safeUint64ToInt64(memStats.PauseTotalNs),
	}
}

// SetMemoryAttributes sets memory-related attributes on a span with the given prefix
func SetMemoryAttributes(span trace.Span, prefix string, memStats MemoryStats) {
	span.SetAttributes(
		attribute.Int64(prefix+".memoryBytes", memStats.Alloc),
		attribute.Int64(prefix+".memoryHeapBytes", memStats.HeapAlloc),
		attribute.Int64(prefix+".memorySysBytes", memStats.Sys),
		attribute.Int64(prefix+".memoryNumGC", memStats.NumGC),
		attribute.Int64(prefix+".memoryPauseTotalNs", memStats.PauseTotal),
	)
}

// SetMemoryDeltaAttributes sets memory delta attributes on a span with the given prefix
// It calculates the difference between before and after memory stats
func SetMemoryDeltaAttributes(span trace.Span, prefix string, before, after MemoryStats) {
	deltaAlloc := after.Alloc - before.Alloc
	deltaHeapAlloc := after.HeapAlloc - before.HeapAlloc
	deltaSys := after.Sys - before.Sys
	deltaNumGC := after.NumGC - before.NumGC
	deltaPauseTotal := after.PauseTotal - before.PauseTotal

	span.SetAttributes(
		attribute.Int64(prefix+".memoryDeltaBytes", deltaAlloc),
		attribute.Int64(prefix+".memoryDeltaHeapBytes", deltaHeapAlloc),
		attribute.Int64(prefix+".memoryDeltaSysBytes", deltaSys),
		attribute.Int64(prefix+".memoryDeltaNumGC", deltaNumGC),
		attribute.Int64(prefix+".memoryDeltaPauseTotalNs", deltaPauseTotal),
	)
}
