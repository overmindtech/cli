package shared

import (
	"testing"
)

func TestCompositeLookupKey(t *testing.T) {
	tests := []struct {
		name       string
		queryParts []string
		expected   string
	}{
		{
			name:       "Single query part",
			queryParts: []string{"part1"},
			expected:   "part1",
		},
		{
			name:       "Multiple query parts",
			queryParts: []string{"part1", "part2", "part3"},
			expected:   "part1|part2|part3",
		},
		{
			name:       "Empty query parts",
			queryParts: []string{},
			expected:   "",
		},
		{
			name:       "Query parts with empty strings",
			queryParts: []string{"part1", "", "part3"},
			expected:   "part1||part3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompositeLookupKey(tt.queryParts...)
			if result != tt.expected {
				t.Errorf("CompositeLookupKey(%v) = %q; want %q", tt.queryParts, result, tt.expected)
			}
		})
	}
}
