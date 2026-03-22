package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status constants for resume processing states
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusParsed     = "parsed"
	StatusFailed     = "failed"
	StatusArchived   = "archived"
)

// validStatusTransitions defines the allowed status transitions
var validStatusTransitions = map[string][]string{
	StatusPending:    {StatusProcessing, StatusArchived},
	StatusProcessing: {StatusParsed, StatusFailed},
	StatusParsed:     {StatusArchived},
	StatusFailed:     {StatusPending, StatusArchived},
	StatusArchived:   {}, // No transitions allowed from archived
}

// Resume represents a job applicant's resume document
type Resume struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name       string         `json:"name" gorm:"type:varchar(255);not null"`
	Email      string         `json:"email" gorm:"type:varchar(255)"`
	Phone      string         `json:"phone" gorm:"type:varchar(50)"`
	Source     string         `json:"source" gorm:"type:varchar(100)"` // e.g., "LinkedIn", "Indeed", "Direct Application"
	FileURL    string         `json:"file_url" gorm:"type:text"`
	ParsedData map[string]any `json:"parsed_data" gorm:"type:jsonb"`
	Status     string         `json:"status" gorm:"type:varchar(50);default:pending"`
	CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName specifies the table name for GORM
func (Resume) TableName() string {
	return "resumes"
}

// IsParsed returns true if the resume has been successfully parsed
func (r *Resume) IsParsed() bool {
	return r.Status == StatusParsed
}

// CanTransitionTo checks if a status transition is valid
func (r *Resume) CanTransitionTo(newStatus string) bool {
	allowedTransitions, exists := validStatusTransitions[r.Status]
	if !exists {
		return false
	}
	return slices.Contains(allowedTransitions, newStatus)
}
