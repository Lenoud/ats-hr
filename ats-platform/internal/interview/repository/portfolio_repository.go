package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/ats-platform/internal/interview/model"
)

// PortfolioRepository defines the interface for portfolio data operations
type PortfolioRepository interface {
	Create(ctx context.Context, portfolio *model.Portfolio) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error)
	ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Portfolio, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// portfolioRepo implements PortfolioRepository using GORM with raw SQL
type portfolioRepo struct {
	db *gorm.DB
}

// NewPortfolioRepository creates a new portfolio repository
func NewPortfolioRepository(db *gorm.DB) PortfolioRepository {
	return &portfolioRepo{db: db}
}

// Create inserts a new portfolio into the database
func (r *portfolioRepo) Create(ctx context.Context, portfolio *model.Portfolio) error {
	if portfolio.CreatedAt.IsZero() {
		portfolio.CreatedAt = time.Now()
	}

	query := `INSERT INTO portfolios (id, resume_id, title, file_url, file_type, created_at)
			  VALUES (?, ?, ?, ?, ?, ?)`

	return r.db.WithContext(ctx).Exec(query,
		portfolio.ID.String(),
		portfolio.ResumeID.String(),
		portfolio.Title,
		portfolio.FileURL,
		portfolio.FileType,
		portfolio.CreatedAt,
	).Error
}

// GetByID retrieves a portfolio by its ID
func (r *portfolioRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
	var portfolio model.Portfolio
	query := `SELECT id, resume_id, title, file_url, file_type, created_at
			  FROM portfolios WHERE id = ?`

	result := r.db.WithContext(ctx).Raw(query, id).Scan(&portfolio)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &portfolio, nil
}

// ListByResumeID retrieves all portfolios for a resume
func (r *portfolioRepo) ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Portfolio, error) {
	query := `SELECT id, resume_id, title, file_url, file_type, created_at
			  FROM portfolios WHERE resume_id = ? ORDER BY created_at DESC`

	var portfolios []model.Portfolio
	result := r.db.WithContext(ctx).Raw(query, resumeID).Scan(&portfolios)
	if result.Error != nil {
		return nil, result.Error
	}

	return portfolios, nil
}

// Delete removes a portfolio from the database
func (r *portfolioRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM portfolios WHERE id = ?`
	result := r.db.WithContext(ctx).Exec(query, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
