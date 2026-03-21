package repository

import (
	"context"
	"sync"

	"github.com/example/ats-platform/internal/search/model"
)

// MockRepository implements ESRepository for testing
type MockRepository struct {
	mu   sync.RWMutex
	docs map[string]*model.ResumeDocument
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		docs: make(map[string]*model.ResumeDocument),
	}
}

// Index adds a document to the repository
func (r *MockRepository) Index(ctx context.Context, doc *model.ResumeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ResumeID] = doc
	return nil
}

// GetByID retrieves a document by ID
func (r *MockRepository) GetByID(ctx context.Context, id string) (*model.ResumeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.docs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return doc, nil
}

// Delete removes a document from the repository
func (r *MockRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.docs, id)
	return nil
}

// Search searches documents based on filter
func (r *MockRepository) Search(ctx context.Context, filter SearchFilter) (*SearchResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []model.ResumeDocument
	for _, doc := range r.docs {
		// Filter by status
		if filter.Status != "" && doc.Status != filter.Status {
			continue
		}
		// Filter by source
		if filter.Source != "" && doc.Source != filter.Source {
			continue
		}
		// Filter by min experience
		if filter.MinExperience > 0 && doc.ExperienceYears < filter.MinExperience {
			continue
		}
		// Filter by max experience
		if filter.MaxExperience > 0 && doc.ExperienceYears > filter.MaxExperience {
			continue
		}
		results = append(results, *doc)
	}

	// Apply pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(results) {
		return &SearchResult{Documents: []model.ResumeDocument{}, Total: int64(len(results))}, nil
	}
	if end > len(results) {
		end = len(results)
	}

	return &SearchResult{
		Documents: results[start:end],
		Total:     int64(len(results)),
	}, nil
}

// UpdateStatus updates a document's status
func (r *MockRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, ok := r.docs[id]
	if !ok {
		return ErrNotFound
	}
	doc.Status = status
	return nil
}
