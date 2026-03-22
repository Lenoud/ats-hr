package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/example/ats-platform/internal/interview/repository"
)

var (
	ErrInterviewNotFound      = errors.New("interview not found")
	ErrInvalidStatus          = errors.New("invalid interview status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrFeedbackNotFound       = errors.New("feedback not found")
	ErrFeedbackAlreadyExists  = errors.New("feedback already exists for this interview")
	ErrInvalidFileType        = errors.New("invalid file type")
)

// CreateInterviewInput defines the input for creating an interview
type CreateInterviewInput struct {
	ResumeID    string    `json:"resume_id" binding:"required"`
	Round       int       `json:"round" binding:"required,min=1"`
	Interviewer string    `json:"interviewer" binding:"required"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
}

// UpdateInterviewStatusInput defines the input for updating interview status
type UpdateInterviewStatusInput struct {
	Status string `json:"status" binding:"required"`
}

// InterviewService defines the interface for interview business logic
type InterviewService interface {
	Create(ctx context.Context, input CreateInterviewInput) (*model.Interview, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error)
	ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Interview, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, input UpdateInterviewStatusInput) (*model.Interview, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// interviewService implements InterviewService
type interviewService struct {
	repo repository.InterviewRepository
}

// NewInterviewService creates a new InterviewService instance
func NewInterviewService(repo repository.InterviewRepository) InterviewService {
	return &interviewService{repo: repo}
}

// Create creates a new interview with default status
func (s *interviewService) Create(ctx context.Context, input CreateInterviewInput) (*model.Interview, error) {
	resumeID, err := uuid.Parse(input.ResumeID)
	if err != nil {
		return nil, errors.New("invalid resume_id format")
	}

	interview := &model.Interview{
		ID:          uuid.New(),
		ResumeID:    resumeID,
		Round:       input.Round,
		Interviewer: input.Interviewer,
		ScheduledAt: input.ScheduledAt,
		Status:      model.InterviewStatusScheduled,
	}

	if err := s.repo.Create(ctx, interview); err != nil {
		return nil, err
	}

	return interview, nil
}

// GetByID retrieves an interview by its ID
func (s *interviewService) GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error) {
	interview, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInterviewNotFound
		}
		return nil, err
	}
	return interview, nil
}

// ListByResumeID retrieves all interviews for a resume
func (s *interviewService) ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Interview, error) {
	return s.repo.ListByResumeID(ctx, resumeID)
}

// UpdateStatus updates the status of an interview with validation
func (s *interviewService) UpdateStatus(ctx context.Context, id uuid.UUID, input UpdateInterviewStatusInput) (*model.Interview, error) {
	interview, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInterviewNotFound
		}
		return nil, err
	}

	// Validate status transition
	newStatus := input.Status
	if !interview.CanTransitionTo(newStatus) {
		return nil, ErrInvalidStatusTransition
	}

	if err := s.repo.UpdateStatus(ctx, id, input.Status); err != nil {
		return nil, err
	}

	// Re-fetch to get updated timestamp
	updatedInterview, err := s.repo.GetByID(ctx, id)
	if err != nil {
		interview.Status = newStatus
		return interview, nil
	}

	return updatedInterview, nil
}

// Delete deletes an interview
func (s *interviewService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInterviewNotFound
		}
		return err
	}
	return nil
}
