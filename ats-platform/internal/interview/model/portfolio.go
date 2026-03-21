package model

import (
	"time"

	"github.com/google/uuid"
)

// FileType 文件类型
type FileType string

const (
	FileTypePDF   FileType = "pdf"
	FileTypeLink  FileType = "link"
	FileTypeImage FileType = "image"
)

// Portfolio 作品集
type Portfolio struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	ResumeID  uuid.UUID `json:"resume_id" gorm:"type:uuid;not null;index"`
	Title     string    `json:"title" gorm:"size:200;not null"`
	FileURL   string    `json:"file_url" gorm:"type:text"`
	FileType  FileType  `json:"file_type" gorm:"size:50;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (Portfolio) TableName() string {
	return "portfolios"
}

// BeforeCreate 创建前生成 UUID
func (p *Portfolio) BeforeCreate() error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// CreatePortfolioRequest 创建作品集请求
type CreatePortfolioRequest struct {
	Title    string `json:"title" binding:"required,max=200"`
	FileURL  string `json:"file_url" binding:"required"`
	FileType string `json:"file_type" binding:"required,oneof=pdf link image"`
}
