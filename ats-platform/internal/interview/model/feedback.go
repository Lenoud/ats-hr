package model

import (
	"time"

	"github.com/google/uuid"
)

// Recommendation 推荐意见
type Recommendation string

const (
	RecommendationStrongYes Recommendation = "strong_yes"
	RecommendationYes       Recommendation = "yes"
	RecommendationNo        Recommendation = "no"
	RecommendationStrongNo  Recommendation = "strong_no"
)

// Feedback 面评
type Feedback struct {
	ID             uuid.UUID     `json:"id" gorm:"type:uuid;primary_key"`
	InterviewID    uuid.UUID     `json:"interview_id" gorm:"type:uuid;not null;uniqueIndex"`
	Rating         int           `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Content        string        `json:"content" gorm:"type:text"`
	Recommendation Recommendation `json:"recommendation" gorm:"type:varchar(20)"`
	CreatedAt      time.Time     `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (Feedback) TableName() string {
	return "feedbacks"
}

// BeforeCreate 创建前生成 UUID
func (f *Feedback) BeforeCreate() error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// SubmitFeedbackRequest 提交面评请求
type SubmitFeedbackRequest struct {
	Rating         int    `json:"rating" binding:"required,min=1,max=5"`
	Content        string `json:"content"`
	Recommendation string `json:"recommendation" binding:"required,oneof=strong_yes yes no strong_no"`
}
