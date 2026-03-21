package repository

import (
	"context"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InterviewRepository 面试仓库接口
type InterviewRepository interface {
	Create(ctx context.Context, interview *model.Interview) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error)
	ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]*model.Interview, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// interviewRepo 面试仓库实现
type interviewRepo struct {
	db *gorm.DB
}

// NewInterviewRepository 创建面试仓库
func NewInterviewRepository(db *gorm.DB) InterviewRepository {
	return &interviewRepo{db: db}
}

func (r *interviewRepo) Create(ctx context.Context, interview *model.Interview) error {
	return r.db.WithContext(ctx).Create(interview).Error
}

func (r *interviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error) {
	var interview model.Interview
	err := r.db.WithContext(ctx).First(&interview, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &interview, nil
}

func (r *interviewRepo) ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]*model.Interview, error) {
	var interviews []*model.Interview
	err := r.db.WithContext(ctx).
		Where("resume_id = ?", resumeID).
		Order("round ASC, created_at DESC").
		Find(&interviews).Error
	return interviews, err
}

func (r *interviewRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.Interview{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *interviewRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Interview{}, "id = ?", id).Error
}
