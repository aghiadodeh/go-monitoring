package dto

// ListResponse is a generic paginated list response.
type ListResponse[T any] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}
