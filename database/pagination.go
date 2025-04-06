package database

import (
	"strconv"
)

// Pagination represents pagination parameters
type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

// GetPagination calculates pagination values (offset, limit) from query parameters
func GetPagination(pageParam, pageLimitParam string) Pagination {
	page, _ := strconv.Atoi(pageParam)
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(pageLimitParam)
	switch {
	case pageSize > 100:
		pageSize = 100
	case pageSize <= 0:
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
	}
}
