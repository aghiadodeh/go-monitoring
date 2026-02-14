package services

import (
	"encoding/json"
	"fmt"

	"github.com/aghiad-odeh/go-monitoring/dto"
	"github.com/aghiad-odeh/go-monitoring/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// JobService handles job-log CRUD and queries.
type JobService struct {
	DB *gorm.DB
}

// Create inserts a new job log record.
// metadata must be a value that is serializable to valid JSON (struct, map,
// slice, json.RawMessage, etc.). Channels, funcs and other non-serializable
// types will return an error immediately without touching the database.
func (s *JobService) Create(name string, success bool, metadata any) error {
	metaJSON, err := toJSON(metadata)
	if err != nil {
		return fmt.Errorf("monitoring: metadata is not valid JSON: %w", err)
	}
	return s.DB.Create(&models.JobLog{
		Name:     name,
		Success:  success,
		Metadata: metaJSON,
	}).Error
}

// toJSON converts v to a datatypes.JSON value, validating that the result
// is well-formed JSON. If v is already json.RawMessage or []byte it is
// validated in place without a redundant marshal→unmarshal round-trip.
func toJSON(v any) (datatypes.JSON, error) {
	switch raw := v.(type) {
	case json.RawMessage:
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid json.RawMessage")
		}
		return datatypes.JSON(raw), nil
	case []byte:
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid []byte JSON")
		}
		return datatypes.JSON(raw), nil
	case datatypes.JSON:
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid datatypes.JSON")
		}
		return raw, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return datatypes.JSON(b), nil
	}
}

// FindAll returns a paginated, filtered list of job logs.
func (s *JobService) FindAll(f dto.JobFilter) (*dto.ListResponse[models.JobLog], error) {
	from, to := parseDateRange(f.BaseFilter)
	q := s.DB.Model(&models.JobLog{}).Where("created_at BETWEEN ? AND ?", from, to)

	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.Success != nil {
		q = q.Where("success = ?", *f.Success)
	}

	var total int64
	q.Count(&total)

	perPage, skip := pagination(f.BaseFilter)
	sortKey := f.SortKey
	if sortKey == "" {
		sortKey = "created_at"
	}

	var rows []models.JobLog
	err := q.Order(sortKey + " DESC").Offset(skip).Limit(perPage).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	return &dto.ListResponse[models.JobLog]{Total: total, Data: rows}, nil
}

// FindByID returns a single job log by primary key.
func (s *JobService) FindByID(id string) (*models.JobLog, error) {
	var j models.JobLog
	err := s.DB.First(&j, "id = ?", id).Error
	return &j, err
}

// ClearAll deletes all monitoring data (request logs + job logs).
func (s *JobService) ClearAll() error {
	if err := s.DB.Where("1 = 1").Delete(&models.RequestLog{}).Error; err != nil {
		return err
	}
	return s.DB.Where("1 = 1").Delete(&models.JobLog{}).Error
}
