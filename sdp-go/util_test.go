package sdp

import (
	"fmt"
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
