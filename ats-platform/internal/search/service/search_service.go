package service

import (
	"context"
	"errors"

	"github.com/example/ats-platform/internal/search/model"
	"github.com/example/ats-platform/internal/search/repository"
)

// SearchService defines the interface for search operations
type SearchService interface {
	// Search searches resumes based on filter
	Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error)

	// IndexResume indexes a resume document
	IndexResume(ctx context.Context, doc *model.ResumeDocument) error

	// DeleteResume removes a resume from index
	DeleteResume(ctx context.Context, id string) error

	// UpdateResumeStatus updates status in index
	UpdateResumeStatus(ctx context.Context, id string, status string) error
}

// ErrDocumentNotFound is returned when a document is not found
var ErrDocumentNotFound = errors.New("document not found")

// searchServiceImpl implements SearchService
type searchServiceImpl struct {
	repo repository.ESRepository
}

// NewSearchService creates a new SearchService
func NewSearchService(repo repository.ESRepository) SearchService {
	return &searchServiceImpl{repo: repo}
}

// Search searches resumes based on filter
func (s *searchServiceImpl) Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 10
	}
	return s.repo.Search(ctx, filter)
}

// IndexResume indexes a resume document
func (s *searchServiceImpl) IndexResume(ctx context.Context, doc *model.ResumeDocument) error {
	return s.repo.Index(ctx, doc)
}

// DeleteResume removes a resume from index
func (s *searchServiceImpl) DeleteResume(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// UpdateResumeStatus updates status in index
func (s *searchServiceImpl) UpdateResumeStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}
