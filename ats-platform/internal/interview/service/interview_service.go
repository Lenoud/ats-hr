package service

import (
	"context"
	"errors"

	"github.com/example/ats-platform/internal/interview/model"
	"github.com/example/ats-platform/internal/interview/repository"
	"github.com/google/uuid"
)

// InterviewService 面试服务接口
type InterviewService interface {
	CreateInterview(ctx context.Context, req *model.CreateInterviewRequest) (*model.Interview, error)
	GetInterview(ctx context.Context, id uuid.UUID) (*model.Interview, error)
	ListInterviewsByResume(ctx context.Context, resumeID uuid.UUID) ([]*model.Interview, error)
	UpdateInterviewStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteInterview(ctx context.Context, id uuid.UUID) error
}

// interviewService 面试服务实现
type interviewService struct {
	repo repository.InterviewRepository
}

// NewInterviewService 创建面试服务
func NewInterviewService(repo repository.InterviewRepository) InterviewService {
	return &interviewService{repo: repo}
}

func (s *interviewService) CreateInterview(ctx context.Context, req *model.CreateInterviewRequest) (*model.Interview, error) {
	resumeID, err := uuid.Parse(req.ResumeID)
	if err != nil {
		return nil, errors.New("invalid resume_id format")
	}

	interview := &model.Interview{
		ResumeID:    resumeID,
		Round:       req.Round,
		Interviewer: req.Interviewer,
		ScheduledAt: req.ScheduledAt,
			Status:      model.InterviewStatus(model.InterviewStatusScheduled),
	}

	if err := s.repo.Create(ctx, interview); err != nil {
		return nil, err
	}

	return interview, nil
}

func (s *interviewService) GetInterview(ctx context.Context, id uuid.UUID) (*model.Interview, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *interviewService) ListInterviewsByResume(ctx context.Context, resumeID uuid.UUID) ([]*model.Interview, error) {
	return s.repo.ListByResumeID(ctx, resumeID)
}

func (s *interviewService) UpdateInterviewStatus(ctx context.Context, id uuid.UUID, status string) error {
	// 验证状态值
	validStatuses := map[string]bool{
		string(model.InterviewStatusScheduled): true,
		string(model.InterviewStatusCompleted): true,
		string(model.InterviewStatusCancelled): true,
	}
	if !validStatuses[status] {
		return errors.New("invalid status value")
	}

	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *interviewService) DeleteInterview(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
