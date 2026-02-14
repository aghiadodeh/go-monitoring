package dto

// JobFilter extends BaseFilter with job-specific query params.
type JobFilter struct {
	BaseFilter
	Name    string `query:"name"`
	Success *bool  `query:"success"`
}
