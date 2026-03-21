package repository

import (
	"context"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PortfolioRepository 作品集仓库接口
type PortfolioRepository interface {
	Create(ctx context.Context, portfolio *model.Portfolio) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error)
	ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]*model.Portfolio, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// portfolioRepo 作品集仓库实现
type portfolioRepo struct {
	db *gorm.DB
}

// NewPortfolioRepository 创建作品集仓库
func NewPortfolioRepository(db *gorm.DB) PortfolioRepository {
	return &portfolioRepo{db: db}
}

func (r *portfolioRepo) Create(ctx context.Context, portfolio *model.Portfolio) error {
	return r.db.WithContext(ctx).Create(portfolio).Error
}

func (r *portfolioRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
	var portfolio model.Portfolio
	err := r.db.WithContext(ctx).First(&portfolio, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &portfolio, nil
}

func (r *portfolioRepo) ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]*model.Portfolio, error) {
	var portfolios []*model.Portfolio
	err := r.db.WithContext(ctx).
		Where("resume_id = ?", resumeID).
		Order("created_at DESC").
		Find(&portfolios).Error
	return portfolios, err
}

func (r *portfolioRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Portfolio{}, "id = ?", id).Error
}
