package grpc

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/example/ats-platform/internal/resume/service"
	"github.com/example/ats-platform/internal/shared/pb/resume"
)

type ResumeServiceServer struct {
	resume.UnimplementedResumeServiceServer
	svc service.ResumeService
}

func NewResumeServiceServer(svc service.ResumeService) *ResumeServiceServer {
	return &ResumeServiceServer{svc: svc}
}

func (s *ResumeServiceServer) GetResume(ctx context.Context, req *resume.GetResumeRequest) (*resume.Resume, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	r, err := s.svc.GetByID(ctx, id)
	if err != nil {
		if err == service.ErrResumeNotFound {
			return nil, status.Errorf(codes.NotFound, "resume not found")
		}
		return nil, status.Errorf(codes.Internal, "get resume failed: %v", err)
	}

	return toProto(r), nil
}

func (s *ResumeServiceServer) CreateResume(ctx context.Context, req *resume.CreateResumeRequest) (*resume.Resume, error) {
	if req.GetName() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}
	if req.GetEmail() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}

	input := service.CreateResumeInput{
		Name:    req.GetName(),
		Email:   req.GetEmail(),
		Phone:   req.GetPhone(),
		Source:  req.GetSource(),
		FileURL: req.GetFileUrl(),
	}

	r, err := s.svc.Create(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create failed: %v", err)
	}

	return toProto(r), nil
}

func (s *ResumeServiceServer) UpdateResume(ctx context.Context, req *resume.UpdateResumeRequest) (*resume.Resume, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	input := service.UpdateResumeInput{
		Name:  req.GetName(),
		Email: req.GetEmail(),
		Phone: req.GetPhone(),
	}

	r, err := s.svc.Update(ctx, id, input)
	if err != nil {
		if err == service.ErrResumeNotFound {
			return nil, status.Errorf(codes.NotFound, "resume not found")
		}
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}

	return toProto(r), nil
}

func (s *ResumeServiceServer) UpdateStatus(ctx context.Context, req *resume.UpdateStatusRequest) (*resume.Resume, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	newStatus := req.GetStatus()
	if !isValidStatus(newStatus) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status value: %s", newStatus)
	}

	r, err := s.svc.UpdateStatus(ctx, id, newStatus)
	if err != nil {
		switch err {
		case service.ErrResumeNotFound:
			return nil, status.Errorf(codes.NotFound, "resume not found")
		case service.ErrInvalidStatusTransition:
			return nil, status.Errorf(codes.FailedPrecondition, "invalid status transition")
		default:
			return nil, status.Errorf(codes.Internal, "update status failed: %v", err)
		}
	}

	return toProto(r), nil
}

func (s *ResumeServiceServer) ListResumes(ctx context.Context, req *resume.ListResumesRequest) (*resume.ListResumesResponse, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	resumes, total, err := s.svc.List(ctx, page, pageSize, req.GetStatus(), req.GetSource())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}

	pbResumes := make([]*resume.Resume, len(resumes))
	for i, r := range resumes {
		pbResumes[i] = toProto(&r)
	}

	return &resume.ListResumesResponse{
		Resumes: pbResumes,
		Total:   total,
	}, nil
}

func toProto(r *model.Resume) *resume.Resume {
	if r == nil {
		return &resume.Resume{}
	}

	var parsedData []byte
	if r.ParsedData != nil {
		var err error
		parsedData, err = json.Marshal(r.ParsedData)
		if err != nil {
			parsedData = []byte("{}") // fallback to empty object
		}
	}

	return &resume.Resume{
		Id:         r.ID.String(),
		Name:       r.Name,
		Email:      r.Email,
		Phone:      r.Phone,
		Source:     r.Source,
		FileUrl:    r.FileURL,
		ParsedData: parsedData,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt.Unix(),
		UpdatedAt:  r.UpdatedAt.Unix(),
	}
}

func isValidStatus(status string) bool {
	switch status {
	case model.StatusPending, model.StatusProcessing, model.StatusParsed, model.StatusFailed, model.StatusArchived:
		return true
	default:
		return false
	}
}
