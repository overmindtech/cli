package tracing

import (
	"testing"
)

func TestSafeUint64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint64
		expected int64
	}{
		{
			name:     "small value",
			input:    1000,
			expected: 1000,
		},
		{
			name:     "int64 max value",
			input:    9223372036854775807, // 2^63 - 1
			expected: 9223372036854775807,
		},
		{
			name:     "int64 max + 1",
			input:    9223372036854775808, // 2^63
			expected: 9223372036854775807, // Should be clamped to int64 max
		},
		{
			name:     "very large value",
			input:    18446744073709551615, // uint64 max
			expected: 9223372036854775807,  // Should be clamped to int64 max
		},
	}

	for _, tt := range tests {
		test := tt // capture loop variable
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := safeUint64ToInt64(test.input)
			if result != test.expected {
				t.Errorf("safeUint64ToInt64(%d) = %d, expected %d", test.input, result, test.expected)
			}
		})
	}
}

func TestReadMemoryStats(t *testing.T) {
	t.Parallel()

	stats := ReadMemoryStats()

	// Basic sanity checks - these values should be reasonable
	if stats.Alloc <= 0 {
		t.Errorf("Alloc should be greater than 0, got %d", stats.Alloc)
	}
	if stats.HeapAlloc <= 0 {
		t.Errorf("HeapAlloc should be greater than 0, got %d", stats.HeapAlloc)
	}
	if stats.Sys <= 0 {
		t.Errorf("Sys should be greater than 0, got %d", stats.Sys)
	}

	// Verify that values are within int64 range (they should be since we convert them)
	if stats.Alloc < 0 {
		t.Errorf("Alloc should not be negative, got %d", stats.Alloc)
	}
	if stats.HeapAlloc < 0 {
		t.Errorf("HeapAlloc should not be negative, got %d", stats.HeapAlloc)
	}
	if stats.Sys < 0 {
		t.Errorf("Sys should not be negative, got %d", stats.Sys)
	}
}
