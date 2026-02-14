package dto

// BaseFilter contains common pagination and date-range fields.
// All fields are strings because they come from query parameters.
type BaseFilter struct {
	Page    string `query:"page"`
	PerPage string `query:"per_page"`
	SortKey string `query:"sortKey"`
	SortDir string `query:"sortDir"` // "ASC" or "DESC"
	FromDate string `query:"fromDate"`
	ToDate   string `query:"toDate"`
}
