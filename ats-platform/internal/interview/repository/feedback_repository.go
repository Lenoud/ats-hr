package repository

import (
	"context"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FeedbackRepository 面评仓库接口
type FeedbackRepository interface {
	Create(ctx context.Context, feedback *model.Feedback) error
	GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error)
}

// feedbackRepo 面评仓库实现
type feedbackRepo struct {
	db *gorm.DB
}

// NewFeedbackRepository 创建面评仓库
func NewFeedbackRepository(db *gorm.DB) FeedbackRepository {
	return &feedbackRepo{db: db}
}

func (r *feedbackRepo) Create(ctx context.Context, feedback *model.Feedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}

func (r *feedbackRepo) GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error) {
	var feedback model.Feedback
	err := r.db.WithContext(ctx).
		Where("interview_id = ?", interviewID).
		First(&feedback).Error
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}
