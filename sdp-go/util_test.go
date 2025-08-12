package sdp

import (
	"fmt"
	"regexp"
	"testing"
)

func TestCalculatePaginationOffsetLimit(t *testing.T) {
	testCases := []struct {
		page               int32
		pageSize           int32
		totalItems         int32
		expectedOffset     int32
		expectedLimit      int32
		expectedPage       int32
		expectedTotalPages int32
	}{
		{page: 2, pageSize: 10, totalItems: 20, expectedOffset: 10, expectedPage: 2, expectedLimit: 10, expectedTotalPages: 2},
		{page: 3, pageSize: 10, totalItems: 25, expectedOffset: 20, expectedPage: 3, expectedLimit: 10, expectedTotalPages: 3},
		{page: 0, pageSize: 5, totalItems: 15, expectedOffset: 0, expectedPage: 1, expectedLimit: 10, expectedTotalPages: 2},
		{page: 5, pageSize: 7, totalItems: 23, expectedOffset: 20, expectedPage: 3, expectedLimit: 10, expectedTotalPages: 3},
		{page: 1, pageSize: 10, totalItems: 3, expectedOffset: 0, expectedPage: 1, expectedLimit: 10, expectedTotalPages: 1},
		{page: -1, pageSize: 10, totalItems: 1, expectedOffset: 0, expectedPage: 1, expectedLimit: 10, expectedTotalPages: 1},
		{page: 1, pageSize: 101, totalItems: 1, expectedOffset: 0, expectedPage: 1, expectedLimit: 100, expectedTotalPages: 1},
		{page: 1, pageSize: 10, totalItems: 0, expectedOffset: 0, expectedPage: 0, expectedLimit: 0, expectedTotalPages: 0},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("page%d pagesize%d totalItems%d", tc.page, tc.pageSize, tc.totalItems), func(t *testing.T) {
			req := PaginationRequest{
				Page:     tc.page,
				PageSize: tc.pageSize,
			}
			offset, limit, correctedPage, pages := CalculatePaginationOffsetLimit(&req, tc.totalItems)
			if offset != tc.expectedOffset {
				t.Errorf("Expected offset %d, but got %d", tc.expectedOffset, offset)
			}
			if correctedPage != tc.expectedPage {
				t.Errorf("Expected correctedPage %d, but got %d", tc.expectedPage, correctedPage)
			}
			if limit != tc.expectedLimit {
				t.Errorf("Expected limit %d, but got %d", tc.expectedLimit, limit)
			}
			if pages != tc.expectedTotalPages {
				t.Errorf("Expected pages %d, but got %d", tc.expectedTotalPages, pages)
			}
		})
	}

	t.Run("Default values", func(t *testing.T) {
		offset, limit, correctedPage, pages := CalculatePaginationOffsetLimit(nil, 100)
		if offset != 0 {
			t.Errorf("Expected offset 0, but got %d", offset)
		}
		if correctedPage != 1 {
			t.Errorf("Expected correctedPage 1, but got %d", correctedPage)
		}
		if limit != 10 {
			t.Errorf("Expected limit 10, but got %d", limit)
		}
		if pages != 10 {
			t.Errorf("Expected pages 10, but got %d", pages)
		}
	})
}

func TestItemDiffParagraphRendering(t *testing.T) {
	t.Parallel()

	// table driven tests for rendering item diffs
	tests := []struct {
		Name                  string
		Before                map[string]any
		After                 map[string]any
		ExpectedDiffParagraph string
	}{
		{
			Name: "no changes",
			Before: map[string]any{
				"name": "test",
				"age":  30,
			},
			After: map[string]any{
				"name": "test",
				"age":  30,
			},
			ExpectedDiffParagraph: "",
		},
		{
			Name: "update changes",
			Before: map[string]any{
				"name": "test",
				"age":  30,
			},
			After: map[string]any{
				"name": "updated",
				"age":  30,
			},
			ExpectedDiffParagraph: "- name: test\n+ name: updated",
		},
		{
			Name: "nested map",
			Before: map[string]any{
				"name": map[string]any{
					"first": "test",
					"last":  "user",
				},
			},
			After: map[string]any{
				"name": map[string]any{
					"first": "test",
					"last":  "updated",
				},
			},
			ExpectedDiffParagraph: "- name.last: user\n+ name.last: updated",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			diff := RenderItemDiff(test.Before, test.After)

			if diff != test.ExpectedDiffParagraph {
				t.Errorf("expected diff paragraph to be '%s', got '%s'", test.ExpectedDiffParagraph, diff)
			}
		})
	}

}

func TestGcpSANameFromAccountName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		accountName string
		expected    string
	}{
		// BEWARE!! If this test needs changing, all currently existing service
		// accounts in GCP will need to be updated, which sounds like an unholy
		// mess.
		{"test-account", "C-testaccount"},
		{"", ""},
		{"6351cbb7-cb45-481a-99cd-909d04a58512", "C-6351cbb7cb45481a99cd909d04a5"},
		{"d408ea46-f4c9-487f-9bf4-b0bcb6843815", "C-d408ea46f4c9487f9bf4b0bcb684"},
		{"63d185c7141237978cfdbaa2", "C-63d185c7141237978cfdbaa2"},
		{"b6c1119a-b80b-4a7b-b8df-acb5348525ac", "C-b6c1119ab80b4a7bb8dfacb53485"},
	}

	pattern := `^[a-zA-Z][a-zA-Z\d\-]*[a-zA-Z\d]$`

	for _, test := range tests {
		t.Run(test.accountName, func(t *testing.T) {
			result := GcpSANameFromAccountName(test.accountName)
			if result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}

			if test.expected != "" {
				matched, err := regexp.MatchString(pattern, result)
				if err != nil {
					t.Fatalf("failed to compile regex: %v", err)
				}
				if !matched {
					t.Errorf("result %q does not match regex %q", result, pattern)
				}

				if len(result) > 30 {
					t.Errorf("result %q exceeds 30 characters", result)
				}

				if len(result) < 6 {
					t.Errorf("result %q is less than 6 characters", result)
				}
			}
		})
	}
}
