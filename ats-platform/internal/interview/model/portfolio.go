package model

import (
	"time"

	"github.com/google/uuid"
)

// FileType constants for portfolio files
const (
	FileTypePDF   = "pdf"
	FileTypeLink  = "link"
	FileTypeImage = "image"
)

// ValidFileTypes contains all valid file type values
var ValidFileTypes = []string{
	FileTypePDF,
	FileTypeLink,
	FileTypeImage,
}

// Portfolio represents a candidate's portfolio item
type Portfolio struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ResumeID  uuid.UUID `json:"resume_id" gorm:"type:uuid;not null;index"`
	Title     string    `json:"title" gorm:"type:varchar(200);not null"`
	FileURL   string    `json:"file_url" gorm:"type:text"`
	FileType  string    `json:"file_type" gorm:"type:varchar(50);not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for GORM
func (Portfolio) TableName() string {
	return "portfolios"
}

// IsValidFileType checks if the file type is valid
func (p *Portfolio) IsValidFileType() bool {
	for _, t := range ValidFileTypes {
		if p.FileType == t {
			return true
		}
	}
	return false
}
