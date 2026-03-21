package service

import (
	"context"
	"errors"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/example/ats-platform/internal/interview/repository"
	"github.com/google/uuid"
)

// FeedbackService 面评服务接口
type FeedbackService interface {
	SubmitFeedback(ctx context.Context, interviewID uuid.UUID, req *model.SubmitFeedbackRequest) (*model.Feedback, error)
	GetFeedback(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error)
}

// feedbackService 面评服务实现
type feedbackService struct {
	feedbackRepo   repository.FeedbackRepository
	interviewRepo  repository.InterviewRepository
}

// NewFeedbackService 创建面评服务
func NewFeedbackService(feedbackRepo repository.FeedbackRepository, interviewRepo repository.InterviewRepository) FeedbackService {
	return &feedbackService{
		feedbackRepo:  feedbackRepo,
		interviewRepo: interviewRepo,
	}
}

func (s *feedbackService) SubmitFeedback(ctx context.Context, interviewID uuid.UUID, req *model.SubmitFeedbackRequest) (*model.Feedback, error) {
	// 检查面试是否存在
	interview, err := s.interviewRepo.GetByID(ctx, interviewID)
	if err != nil {
		return nil, errors.New("interview not found")
	}

	// 检查是否已有面评
	existing, _ := s.feedbackRepo.GetByInterviewID(ctx, interviewID)
	if existing != nil {
		return nil, errors.New("feedback already exists for this interview")
	}

	feedback := &model.Feedback{
		InterviewID:    interview.ID,
		Rating:         req.Rating,
		Content:        req.Content,
		Recommendation: model.Recommendation(req.Recommendation),
	}

	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		return nil, err
	}

	return feedback, nil
}

func (s *feedbackService) GetFeedback(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error) {
	return s.feedbackRepo.GetByInterviewID(ctx, interviewID)
}
