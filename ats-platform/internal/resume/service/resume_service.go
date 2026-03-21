package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/example/ats-platform/internal/resume/repository"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrResumeNotFound          = errors.New("resume not found")
)

// CreateResumeInput defines the input for creating a resume
type CreateResumeInput struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required,email"`
	Phone   string `json:"phone"`
	Source  string `json:"source"`
	FileURL string `json:"file_url"`
}

// UpdateResumeInput defines the input for updating a resume
type UpdateResumeInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// ResumeService defines the interface for resume business logic
type ResumeService interface {
	Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error)
	List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateResumeInput) (*model.Resume, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error)
}

// resumeService implements ResumeService
type resumeService struct {
	repo repository.ResumeRepository
}

// NewResumeService creates a new ResumeService instance
func NewResumeService(repo repository.ResumeRepository) ResumeService {
	return &resumeService{
		repo: repo,
	}
}

// Create creates a new resume with default status
func (s *resumeService) Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error) {
	resume := &model.Resume{
		ID:      uuid.New(),
		Name:    input.Name,
		Email:   input.Email,
		Phone:   input.Phone,
		Source:  input.Source,
		FileURL: input.FileURL,
		Status:  model.StatusPending, // Default status
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		return nil, err
	}

	return resume, nil
}

// GetByID retrieves a resume by its ID
func (s *resumeService) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrResumeNotFound
		}
		return nil, err
	}
	return resume, nil
}

// List retrieves resumes with filtering and pagination
func (s *resumeService) List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error) {
	filter := repository.ListFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Source:   source,
	}

	resumes, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return resumes, total, nil
}

// Update updates an existing resume
func (s *resumeService) Update(ctx context.Context, id uuid.UUID, input UpdateResumeInput) (*model.Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrResumeNotFound
		}
		return nil, err
	}

	// Update fields if provided
	if input.Name != "" {
		resume.Name = input.Name
	}
	if input.Email != "" {
		resume.Email = input.Email
	}
	if input.Phone != "" {
		resume.Phone = input.Phone
	}

	if err := s.repo.Update(ctx, resume); err != nil {
		return nil, err
	}

	return resume, nil
}

// Delete deletes a resume
func (s *resumeService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrResumeNotFound
		}
		return err
	}
	return nil
}

// UpdateStatus updates the status of a resume with validation
func (s *resumeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrResumeNotFound
		}
		return nil, err
	}

	// Validate status transition
	if !resume.CanTransitionTo(status) {
		return nil, ErrInvalidStatusTransition
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}

	// Update local resume status to return
	resume.Status = status
	return resume, nil
}
