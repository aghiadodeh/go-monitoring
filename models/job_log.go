package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// JobLog stores a background / cron job execution record.
type JobLog struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Success   bool           `gorm:"default:true" json:"success"`
	Metadata  datatypes.JSON `gorm:"type:json;not null" json:"metadata"`
	CreatedAt time.Time      `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// TableName overrides the default table name.
func (JobLog) TableName() string {
	return "monitoring_job_logs"
}
