package repository

import (
	"context"
	"errors"

	"github.com/example/ats-platform/internal/search/model"
)

// ErrNotFound is returned when a document is not found
var ErrNotFound = errors.New("document not found")

// SearchFilter defines search filter options
type SearchFilter struct {
	Query         string   `json:"query"`
	Skills        []string `json:"skills"`
	Status        string   `json:"status"`
	Source        string   `json:"source"`
	MinExperience int    `json:"min_experience"`
	MaxExperience int    `json:"max_experience"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
}

// SearchResult represents search results
type SearchResult struct {
	Documents []model.ResumeDocument `json:"documents"`
	Total     int64                      `json:"total"`
}

// ESRepository defines the interface for Elasticsearch operations
type ESRepository interface {
	Index(ctx context.Context, doc *model.ResumeDocument) error
	GetByID(ctx context.Context, id string) (*model.ResumeDocument, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filter SearchFilter) (*SearchResult, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}
