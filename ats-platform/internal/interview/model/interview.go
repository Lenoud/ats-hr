package model

import (
	"time"

	"github.com/google/uuid"
)

// InterviewStatus 面试状态
type InterviewStatus string

const (
	InterviewStatusScheduled InterviewStatus = "scheduled"
	InterviewStatusCompleted InterviewStatus = "completed"
	InterviewStatusCancelled InterviewStatus = "cancelled"
)

// Interview 面试记录
type Interview struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key"`
	ResumeID    uuid.UUID       `json:"resume_id" gorm:"type:uuid;not null;index"`
	Round       int             `json:"round" gorm:"not null"`
	Interviewer string          `json:"interviewer" gorm:"size:100"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	Status      InterviewStatus `json:"status" gorm:"size:20;default:'scheduled'"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (Interview) TableName() string {
	return "interviews"
}

// BeforeCreate 创建前生成 UUID
func (i *Interview) BeforeCreate() error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// CreateInterviewRequest 创建面试请求
type CreateInterviewRequest struct {
	ResumeID    string    `json:"resume_id" binding:"required"`
	Round       int       `json:"round" binding:"required,min=1"`
	Interviewer string    `json:"interviewer" binding:"required"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
}

// UpdateInterviewStatusRequest 更新面试状态请求
type UpdateInterviewStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=scheduled completed cancelled"`
}
