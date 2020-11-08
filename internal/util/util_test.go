package util

import (
	"testing"
)

func TestGetPageCount(t *testing.T) {
	testTable := []struct {
		totalCount        int
		itemCountPerPage  int
		pageLimit         int
		expectedPageCount int
		description       string
	}{
		{1000, 100, 1000, 10, "exact division"},
		{1000, 600, 1000, 2, "non exact division"},
		{100, 100, 1000, 1, "exactly 1 page"},
		{50, 100, 1000, 1, "\"less\" than 1 page"},
		{1000, 100, 5, 5, "reach pageLimit"},
	}

	for _, test := range testTable {
		pageCount := GetPageCount(test.totalCount, test.itemCountPerPage, test.pageLimit)
		if pageCount != test.expectedPageCount {
			t.Errorf("Incorrect result for %s, got: %d, want: %d.", test.description, pageCount, test.expectedPageCount)
		}
	}
}
