package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RequestLog stores a single HTTP request/response cycle.
type RequestLog struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key             string         `gorm:"type:varchar(255)" json:"key"`
	Path            string         `gorm:"type:varchar(500)" json:"path"`
	URL             string         `gorm:"type:varchar(2048)" json:"url"`
	Method          string         `gorm:"type:varchar(10)" json:"method"`
	User            datatypes.JSON `gorm:"type:json" json:"user"`
	Request         datatypes.JSON `gorm:"type:json" json:"request"`
	Response        datatypes.JSON `gorm:"type:json" json:"response"`
	ResponseHeaders datatypes.JSON `gorm:"type:json" json:"responseHeaders"`
	Success         bool           `gorm:"default:true" json:"success"`
	Duration        float64        `gorm:"type:double precision" json:"duration"`
	CreatedAt       time.Time      `gorm:"index" json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// TableName overrides the default table name.
func (RequestLog) TableName() string {
	return "monitoring_request_logs"
}
