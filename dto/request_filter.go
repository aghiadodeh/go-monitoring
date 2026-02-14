package dto

// RequestFilter extends BaseFilter with request-specific query params.
type RequestFilter struct {
	BaseFilter
	URL        string   `query:"url"`
	Method     string   `query:"method"`     // comma-separated: "GET,POST"
	Exception  *bool    `query:"exception"`  // true → only status 500
	Success    *bool    `query:"success"`
	User       string   `query:"user"`
	DurationGt *float64 `query:"durationGt"` // duration >= value (ms)
	DurationLt *float64 `query:"durationLt"` // duration <= value (ms)
	StatusCode *int     `query:"statusCode"`
}
