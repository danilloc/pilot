package models

// Pagination describes a page of a larger result set, matching the shape
// documented in the API spec for every list endpoint.
type Pagination struct {
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	TotalPages int   `json:"total_pages"`
}

// NewPagination computes Page/TotalPages from limit, offset and the total
// row count matching the query.
func NewPagination(limit, offset int, total int64) Pagination {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	page := 1
	if limit > 0 {
		page = offset/limit + 1
	}
	return Pagination{
		Limit:      limit,
		Offset:     offset,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
	}
}
