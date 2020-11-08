package util

import (
	"math"
)

func GetPageCount(totalCount int, itemPerPageCount int, limit int) int {
	pageCount := int(math.Ceil(float64(totalCount) / float64(itemPerPageCount)))

	if pageCount < 1 {
		return 1
	}

	if pageCount > limit {
		return limit
	}

	return pageCount
}
