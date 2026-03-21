package service

import (
	"context"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/example/ats-platform/internal/interview/repository"
	"github.com/google/uuid"
)

// PortfolioService 作品集服务接口
type PortfolioService interface {
	CreatePortfolio(ctx context.Context, resumeID uuid.UUID, req *model.CreatePortfolioRequest) (*model.Portfolio, error)
	GetPortfolio(ctx context.Context, id uuid.UUID) (*model.Portfolio, error)
	ListPortfoliosByResume(ctx context.Context, resumeID uuid.UUID) ([]*model.Portfolio, error)
	DeletePortfolio(ctx context.Context, id uuid.UUID) error
}

// portfolioService 作品集服务实现
type portfolioService struct {
	repo repository.PortfolioRepository
}

// NewPortfolioService 创建作品集服务
func NewPortfolioService(repo repository.PortfolioRepository) PortfolioService {
	return &portfolioService{repo: repo}
}

func (s *portfolioService) CreatePortfolio(ctx context.Context, resumeID uuid.UUID, req *model.CreatePortfolioRequest) (*model.Portfolio, error) {
	 portfolio := &model.Portfolio{
        ResumeID:  resumeID,
        Title:    req.Title,
        FileURL:  req.FileURL,
        FileType: model.FileType(req.FileType),
    }

    if err := s.repo.Create(ctx, portfolio); err != nil {
        return nil, err
    }

    return portfolio, nil
}

func (s *portfolioService) GetPortfolio(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *portfolioService) ListPortfoliosByResume(ctx context.Context, resumeID uuid.UUID) ([]*model.Portfolio, error) {
    return s.repo.ListByResumeID(ctx, resumeID)
}

func (s *portfolioService) DeletePortfolio(ctx context.Context, id uuid.UUID) error {
    return s.repo.Delete(ctx, id)
}
