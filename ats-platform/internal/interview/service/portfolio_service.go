package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/example/ats-platform/internal/interview/repository"
)

// CreatePortfolioInput defines the input for creating a portfolio
type CreatePortfolioInput struct {
	Title    string `json:"title" binding:"required,max=200"`
	FileURL  string `json:"file_url" binding:"required"`
	FileType string `json:"file_type" binding:"required"`
}

// PortfolioService defines the interface for portfolio business logic
type PortfolioService interface {
	Create(ctx context.Context, resumeID uuid.UUID, input CreatePortfolioInput) (*model.Portfolio, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error)
	ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Portfolio, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// portfolioService implements PortfolioService
type portfolioService struct {
	repo repository.PortfolioRepository
}

// NewPortfolioService creates a new PortfolioService instance
func NewPortfolioService(repo repository.PortfolioRepository) PortfolioService {
	return &portfolioService{repo: repo}
}

// Create creates a new portfolio item
func (s *portfolioService) Create(ctx context.Context, resumeID uuid.UUID, input CreatePortfolioInput) (*model.Portfolio, error) {
	// Validate file type
	if !isValidFileType(input.FileType) {
		return nil, ErrInvalidFileType
	}

	portfolio := &model.Portfolio{
		ID:       uuid.New(),
		ResumeID: resumeID,
		Title:    input.Title,
		FileURL:  input.FileURL,
		FileType: input.FileType,
	}

	if err := s.repo.Create(ctx, portfolio); err != nil {
		return nil, err
	}

	return portfolio, nil
}

// GetByID retrieves a portfolio by its ID
func (s *portfolioService) GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
	portfolio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("portfolio not found")
		}
		return nil, err
	}
	return portfolio, nil
}

// ListByResumeID retrieves all portfolios for a resume
func (s *portfolioService) ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Portfolio, error) {
	return s.repo.ListByResumeID(ctx, resumeID)
}

// Delete deletes a portfolio
func (s *portfolioService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("portfolio not found")
		}
		return err
	}
	return nil
}

func isValidFileType(fileType string) bool {
	for _, t := range model.ValidFileTypes {
		if t == fileType {
			return true
		}
	}
	return false
}
