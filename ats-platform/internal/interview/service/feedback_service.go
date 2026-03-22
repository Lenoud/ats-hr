package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/example/ats-platform/internal/interview/repository"
)

// SubmitFeedbackInput defines the input for submitting feedback
type SubmitFeedbackInput struct {
	Rating         int    `json:"rating" binding:"required,min=1,max=5"`
	Content        string `json:"content"`
	Recommendation string `json:"recommendation" binding:"required"`
}

// FeedbackService defines the interface for feedback business logic
type FeedbackService interface {
	Submit(ctx context.Context, interviewID uuid.UUID, input SubmitFeedbackInput) (*model.Feedback, error)
	GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error)
}

// feedbackService implements FeedbackService
type feedbackService struct {
	feedbackRepo  repository.FeedbackRepository
	interviewRepo repository.InterviewRepository
}

// NewFeedbackService creates a new FeedbackService instance
func NewFeedbackService(feedbackRepo repository.FeedbackRepository, interviewRepo repository.InterviewRepository) FeedbackService {
	return &feedbackService{
		feedbackRepo:  feedbackRepo,
		interviewRepo: interviewRepo,
	}
}

// Submit creates feedback for an interview
func (s *feedbackService) Submit(ctx context.Context, interviewID uuid.UUID, input SubmitFeedbackInput) (*model.Feedback, error) {
	// Check if interview exists
	_, err := s.interviewRepo.GetByID(ctx, interviewID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInterviewNotFound
		}
		return nil, err
	}

	// Check if feedback already exists
	existing, _ := s.feedbackRepo.GetByInterviewID(ctx, interviewID)
	if existing != nil {
		return nil, ErrFeedbackAlreadyExists
	}

	// Validate recommendation
	recommendation := input.Recommendation
	if !isValidRecommendation(recommendation) {
		return nil, errors.New("invalid recommendation value")
	}

	feedback := &model.Feedback{
		ID:             uuid.New(),
		InterviewID:    interviewID,
		Rating:         input.Rating,
		Content:        input.Content,
		Recommendation: recommendation,
	}

	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		return nil, err
	}

	return feedback, nil
}

// GetByInterviewID retrieves feedback by interview ID
func (s *feedbackService) GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error) {
	feedback, err := s.feedbackRepo.GetByInterviewID(ctx, interviewID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrFeedbackNotFound
		}
		return nil, err
	}
	return feedback, nil
}

func isValidRecommendation(r string) bool {
	switch r {
	case model.RecommendationStrongYes, model.RecommendationYes, model.RecommendationNo, model.RecommendationStrongNo:
		return true
	default:
		return false
	}
}
