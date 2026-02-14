package services

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aghiad-odeh/go-monitoring/dto"
	"github.com/aghiad-odeh/go-monitoring/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequestService handles all request-log queries and analytics.
type RequestService struct {
	DB *gorm.DB
}

// FindAll returns a paginated, filtered list of request logs.
func (s *RequestService) FindAll(f dto.RequestFilter) (*dto.ListResponse[models.RequestLog], error) {
	from, to := parseDateRange(f.BaseFilter)
	q := s.DB.Model(&models.RequestLog{}).Where("created_at BETWEEN ? AND ?", from, to)

	if f.Exception != nil && *f.Exception {
		q = q.Where("response->>'statusCode' = '500'")
	} else if f.StatusCode != nil {
		q = q.Where("response->>'statusCode' = ?", strconv.Itoa(*f.StatusCode))
	}
	if f.URL != "" {
		q = q.Where("url LIKE ?", "%"+f.URL+"%")
	}
	if f.Method != "" {
		methods := strings.Split(f.Method, ",")
		q = q.Where("method IN ?", methods)
	}
	if f.Success != nil {
		q = q.Where("success = ?", *f.Success)
	}
	if f.DurationGt != nil {
		q = q.Where("duration >= ?", *f.DurationGt)
	}
	if f.DurationLt != nil {
		q = q.Where("duration <= ?", *f.DurationLt)
	}

	var total int64
	q.Count(&total)

	perPage, skip := pagination(f.BaseFilter)
	sortKey := f.SortKey
	if sortKey == "" {
		sortKey = "created_at"
	}

	var rows []models.RequestLog
	err := q.Order(sortKey + " DESC").Offset(skip).Limit(perPage).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	return &dto.ListResponse[models.RequestLog]{Total: total, Data: rows}, nil
}

// FindByID returns a single request log.
func (s *RequestService) FindByID(id string) (*models.RequestLog, error) {
	var r models.RequestLog
	err := s.DB.First(&r, "id = ?", id).Error
	return &r, err
}

// AnalyzeResult is the shape returned by Analyze.
type AnalyzeResult struct {
	FromDate           time.Time           `json:"fromDate"`
	ToDate             time.Time           `json:"toDate"`
	Total              int64               `json:"total"`
	Success            int64               `json:"success"`
	Exceptions         int64               `json:"exceptions"`
	Duration           []DurationBucket    `json:"duration"`
	DurationURLs       []DurationURL       `json:"durationURLs"`
	CreatedAt          []TimeBucket        `json:"createdAt"`
	DurationBoundaries []float64           `json:"durationBoundaries"`
}

// DurationBucket groups requests by response-time range.
type DurationBucket struct {
	ID    float64              `json:"id"`
	Count int                  `json:"count"`
	Data  []DurationBucketItem `json:"data"`
}

// DurationBucketItem is a single request inside a duration bucket.
type DurationBucketItem struct {
	Duration float64 `json:"duration"`
	URL      string  `json:"url"`
	Method   string  `json:"method"`
	Success  bool    `json:"success"`
}

// DurationURL aggregates duration stats per endpoint + method.
type DurationURL struct {
	Method  string  `json:"method"`
	URL     string  `json:"url"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// TimeBucket groups requests into time-series intervals.
type TimeBucket struct {
	ID    time.Time        `json:"id"`
	Count int              `json:"count"`
	Data  []TimeBucketItem `json:"data"`
}

// TimeBucketItem is a single request inside a time bucket.
type TimeBucketItem struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Method    string    `json:"method"`
	Success   bool      `json:"success"`
	CreatedAt time.Time `json:"createdAt"`
}

// Analyze returns aggregate analytics for the given date range.
func (s *RequestService) Analyze(f dto.BaseFilter) (*AnalyzeResult, error) {
	from, to := parseDateRange(f)

	baseWhere := "created_at BETWEEN ? AND ?"

	var total int64
	s.DB.Model(&models.RequestLog{}).Where(baseWhere, from, to).Count(&total)

	var success int64
	s.DB.Model(&models.RequestLog{}).Where(baseWhere+" AND success = ?", from, to, true).Count(&success)

	var exceptions int64
	s.DB.Model(&models.RequestLog{}).Where(baseWhere+" AND response->>'statusCode' = '500'", from, to).Count(&exceptions)

	// Load all matching requests for in-memory bucketing.
	var requests []models.RequestLog
	s.DB.Where(baseWhere, from, to).Find(&requests)

	// ---- duration buckets ----
	boundaries := []float64{0, 20, 40, 80, 130, 150, 180, 200, 500, 1000, 2000}
	var durationBuckets []DurationBucket
	for i := 0; i < len(boundaries)-1; i++ {
		lo, hi := boundaries[i], boundaries[i+1]
		var items []DurationBucketItem
		for _, r := range requests {
			if r.Duration >= lo && r.Duration < hi {
				url := r.Path
				if url == "" {
					url = r.URL
				}
				items = append(items, DurationBucketItem{
					Duration: r.Duration,
					URL:      url,
					Method:   r.Method,
					Success:  r.Success,
				})
			}
		}
		if len(items) > 0 {
			durationBuckets = append(durationBuckets, DurationBucket{
				ID:    lo,
				Count: len(items),
				Data:  items,
			})
		}
	}

	// ---- per-endpoint duration stats ----
	type endpointKey struct{ url, method string }
	epMap := make(map[endpointKey][]float64)
	for _, b := range durationBuckets {
		for _, item := range b.Data {
			cleanURL := strings.SplitN(item.URL, "?", 2)[0]
			k := endpointKey{url: cleanURL, method: item.Method}
			if item.Success {
				epMap[k] = append(epMap[k], item.Duration)
			}
		}
	}
	var durationURLs []DurationURL
	for k, durations := range epMap {
		if len(durations) == 0 {
			continue
		}
		mn, mx, sum := durations[0], durations[0], 0.0
		for _, d := range durations {
			sum += d
			mn = math.Min(mn, d)
			mx = math.Max(mx, d)
		}
		durationURLs = append(durationURLs, DurationURL{
			Method:  k.method,
			URL:     k.url,
			Min:     mn,
			Max:     mx,
			Average: sum / float64(len(durations)),
			Count:   len(durations),
		})
	}

	// ---- time-series buckets ----
	ranges := buildTimeRange(from, to)
	if len(ranges) > 0 {
		ranges = append(ranges, to)
	}
	var timeBuckets []TimeBucket
	for i := 0; i < len(ranges)-1; i++ {
		start, end := ranges[i], ranges[i+1]
		var items []TimeBucketItem
		for _, r := range requests {
			if r.CreatedAt.After(start) && r.CreatedAt.Before(end) {
				items = append(items, TimeBucketItem{
					ID:        r.ID,
					URL:       r.URL,
					Method:    r.Method,
					Success:   r.Success,
					CreatedAt: r.CreatedAt,
				})
			}
		}
		if len(items) > 0 {
			timeBuckets = append(timeBuckets, TimeBucket{
				ID:    start,
				Count: len(items),
				Data:  items,
			})
		}
	}

	return &AnalyzeResult{
		FromDate:           from,
		ToDate:             to,
		Total:              total,
		Success:            success,
		Exceptions:         exceptions,
		Duration:           durationBuckets,
		DurationURLs:       durationURLs,
		CreatedAt:          timeBuckets,
		DurationBoundaries: boundaries,
	}, nil
}

// --- shared helpers ---

func parseDateRange(f dto.BaseFilter) (time.Time, time.Time) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	if f.FromDate != "" {
		if t, err := time.Parse(time.RFC3339, f.FromDate); err == nil {
			from = t
		}
	}
	if f.ToDate != "" {
		if t, err := time.Parse(time.RFC3339, f.ToDate); err == nil {
			to = t
		}
	}
	return from, to
}

func pagination(f dto.BaseFilter) (perPage int, skip int) {
	perPage = 20
	page := 1
	if f.PerPage != "" {
		if v, err := strconv.Atoi(f.PerPage); err == nil && v > 0 {
			perPage = v
		}
	}
	if perPage > 50 {
		perPage = 50
	}
	if f.Page != "" {
		if v, err := strconv.Atoi(f.Page); err == nil && v > 0 {
			page = v
		}
	}
	skip = perPage*(page) - perPage
	return
}

// buildTimeRange creates evenly spaced time boundaries between from and to.
func buildTimeRange(from, to time.Time) []time.Time {
	diff := to.Sub(from)
	var step time.Duration
	switch {
	case diff <= time.Hour:
		step = time.Minute
	case diff <= 24*time.Hour:
		step = time.Hour
	case diff <= 31*24*time.Hour:
		step = 24 * time.Hour
	default:
		step = 30 * 24 * time.Hour // ~month
	}

	var r []time.Time
	for t := from; t.Before(to); t = t.Add(step) {
		r = append(r, t)
	}
	if len(r) == 0 {
		r = append(r, from, to)
	} else if len(r) == 1 {
		r = append(r, to)
	}
	return r
}
