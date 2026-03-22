package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/example/ats-platform/internal/resume/repository"
	"github.com/example/ats-platform/internal/shared/events"
	"github.com/example/ats-platform/internal/shared/llm"
	"github.com/example/ats-platform/internal/shared/storage"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrResumeNotFound          = errors.New("resume not found")
	ErrInvalidFileType         = errors.New("invalid file type, only PDF, DOC, DOCX are allowed")
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
	UploadFile(ctx context.Context, id uuid.UUID, filename string, reader io.Reader, size int64) (*model.Resume, error)
	ParseResume(ctx context.Context, id uuid.UUID) (*ParsedResume, error)
	UploadAndParse(ctx context.Context, filename string, reader io.Reader, size int64, source string) (*model.Resume, *ParsedResume, error)
}

// resumeService implements ResumeService
type resumeService struct {
	repo      repository.ResumeRepository
	storage   storage.FileStorage
	parser    ResumeParser
	publisher *events.EventPublisher
}

// NewResumeService creates a new ResumeService instance
func NewResumeService(repo repository.ResumeRepository, storage storage.FileStorage, publisher *events.EventPublisher) ResumeService {
	return &resumeService{
		repo:      repo,
		storage:   storage,
		parser:    NewResumeParser(),
		publisher: publisher,
	}
}

// NewResumeServiceWithLLM creates a new ResumeService instance with LLM parsing support
func NewResumeServiceWithLLM(repo repository.ResumeRepository, storage storage.FileStorage, publisher *events.EventPublisher, llmClient *llm.Client) ResumeService {
	return &resumeService{
		repo:      repo,
		storage:   storage,
		parser:    NewResumeParserWithLLM(llmClient),
		publisher: publisher,
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

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishCreated(ctx, resume.ID.String(), resume)
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

	// Re-fetch resume to get updated timestamp from database (consistent with UpdateStatus/UploadFile)
	updatedResume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Fallback: return the local resume if re-fetch fails
		return resume, nil
	}

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishUpdated(ctx, id.String(), updatedResume)
	}

	return updatedResume, nil
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

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishDeleted(ctx, id.String())
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

	oldStatus := resume.Status

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}

	// Re-fetch resume to get updated timestamp from database
	updatedResume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Fallback: return the local resume with updated status if re-fetch fails
		resume.Status = status
		return resume, nil
	}

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishStatusChanged(ctx, id.String(), oldStatus, status)
	}

	return updatedResume, nil
}

// UploadFile uploads a file for an existing resume and updates the file URL
func (s *resumeService) UploadFile(ctx context.Context, id uuid.UUID, filename string, reader io.Reader, size int64) (*model.Resume, error) {
	// Check if resume exists
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrResumeNotFound
		}
		return nil, err
	}

	// Validate file type
	if !storage.IsAllowedFileType(filename) {
		return nil, ErrInvalidFileType
	}

	// Upload file to storage
	contentType := storage.GetContentType(filename)
	objectKey, err := s.storage.UploadFile(ctx, reader, filename, contentType, size)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Get the file URL
	fileURL := s.storage.GetFileURL(objectKey)

	// Update resume with file URL
	if err := s.repo.UpdateFileURL(ctx, id, fileURL); err != nil {
		return nil, fmt.Errorf("failed to update resume file URL: %w", err)
	}

	// Re-fetch resume to get updated timestamp
	updatedResume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Return the local resume if re-fetch fails
		resume.FileURL = fileURL
		return resume, nil
	}

	// Publish updated event
	if s.publisher != nil {
		_ = s.publisher.PublishUpdated(ctx, id.String(), updatedResume)
	}

	return updatedResume, nil
}

// ParseResume parses the resume file and extracts structured data
func (s *resumeService) ParseResume(ctx context.Context, id uuid.UUID) (*ParsedResume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrResumeNotFound
		}
		return nil, err
	}

	if resume.FileURL == "" {
		return nil, errors.New("resume has no file to parse")
	}

	oldStatus := resume.Status

	// Atomically update status to processing (only if current status is pending or failed)
	// This prevents race conditions when multiple requests try to parse the same resume
	updated, err := s.repo.UpdateStatusIf(ctx, id, model.StatusProcessing, []string{model.StatusPending, model.StatusFailed})
	if err != nil {
		return nil, fmt.Errorf("failed to update status to processing: %w", err)
	}
	if !updated {
		return nil, fmt.Errorf("cannot parse resume with current status, only pending or failed resumes can be parsed")
	}
	resume.Status = model.StatusProcessing

	// Publish status changed event
	if s.publisher != nil {
		_ = s.publisher.PublishStatusChanged(ctx, id.String(), oldStatus, model.StatusProcessing)
	}

	// Extract object key from FileURL
	// FileURL format: http://endpoint/bucket/objectKey
	// We need to extract just the objectKey part
	objectKey := extractObjectKeyFromURL(resume.FileURL)

	// Download file from storage
	reader, err := s.storage.DownloadFile(ctx, objectKey)
	if err != nil {
		// Set status to failed on download error
		_ = s.repo.UpdateStatus(ctx, id, model.StatusFailed)
		// Publish status changed event
		if s.publisher != nil {
			_ = s.publisher.PublishStatusChanged(ctx, id.String(), model.StatusProcessing, model.StatusFailed)
		}
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	// Parse the file
	parsed, err := s.parser.Parse(ctx, reader, resume.FileURL)
	if err != nil {
		// Set status to failed on parse error
		_ = s.repo.UpdateStatus(ctx, id, model.StatusFailed)
		// Publish status changed event
		if s.publisher != nil {
			_ = s.publisher.PublishStatusChanged(ctx, id.String(), model.StatusProcessing, model.StatusFailed)
		}
		return nil, fmt.Errorf("failed to parse resume: %w", err)
	}

	// Update resume with parsed data
	if parsed.Name != "" && resume.Name == "" {
		resume.Name = parsed.Name
	}
	if parsed.Email != "" && resume.Email == "" {
		resume.Email = parsed.Email
	}
	if parsed.Phone != "" && resume.Phone == "" {
		resume.Phone = parsed.Phone
	}

	// Store parsed data as JSON
	if len(parsed.Skills) > 0 || len(parsed.WorkExperience) > 0 || len(parsed.Education) > 0 {
		parsedData := make(map[string]any)
		if len(parsed.Skills) > 0 {
			parsedData["skills"] = parsed.Skills
		}
		if len(parsed.WorkExperience) > 0 {
			parsedData["work_experience"] = parsed.WorkExperience
		}
		if len(parsed.Education) > 0 {
			parsedData["education"] = parsed.Education
		}
		resume.ParsedData = parsedData
		if err := s.repo.Update(ctx, resume); err != nil {
			// Set status to failed on update error
			_ = s.repo.UpdateStatus(ctx, id, model.StatusFailed)
			// Publish status changed event
			if s.publisher != nil {
				_ = s.publisher.PublishStatusChanged(ctx, id.String(), model.StatusProcessing, model.StatusFailed)
			}
			return nil, fmt.Errorf("failed to update parsed data: %w", err)
		}
	}

	// Update status to parsed
	if err := s.repo.UpdateStatus(ctx, id, model.StatusParsed); err != nil {
		// Set status to failed on status update error
		_ = s.repo.UpdateStatus(ctx, id, model.StatusFailed)
		// Publish status changed event
		if s.publisher != nil {
			_ = s.publisher.PublishStatusChanged(ctx, id.String(), model.StatusProcessing, model.StatusFailed)
		}
		return nil, fmt.Errorf("failed to update status: %w", err)
	}
	resume.Status = model.StatusParsed

	// Publish parsed event
	if s.publisher != nil {
		_ = s.publisher.PublishStatusChanged(ctx, id.String(), model.StatusProcessing, model.StatusParsed)
		_ = s.publisher.PublishParsed(ctx, id.String(), parsed)
	}

	return parsed, nil
}

// UploadAndParse uploads a resume file, parses it, and creates a resume record
// This is an atomic operation that creates a resume in parsed or failed status
func (s *resumeService) UploadAndParse(ctx context.Context, filename string, reader io.Reader, size int64, source string) (*model.Resume, *ParsedResume, error) {
	// Validate file type
	if !storage.IsAllowedFileType(filename) {
		return nil, nil, ErrInvalidFileType
	}

	// Read file content for parsing
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Upload file to storage first
	contentType := storage.GetContentType(filename)
	objectKey, err := s.storage.UploadFile(ctx, bytes.NewReader(content), filename, contentType, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload file: %w", err)
	}
	fileURL := s.storage.GetFileURL(objectKey)

	// Parse the file
	parsed, parseErr := s.parser.Parse(ctx, bytes.NewReader(content), filename)

	// Determine status based on parse result
	status := model.StatusParsed
	if parseErr != nil {
		status = model.StatusFailed
	}

	// Create resume record
	// If parsing succeeded, use parsed data; if failed, create with available data
	resume := &model.Resume{
		ID:      uuid.New(),
		Source:  source,
		FileURL: fileURL,
		Status:  status,
	}

	// Fill in parsed data if available
	if parsed != nil {
		resume.Name = parsed.Name
		resume.Email = parsed.Email
		resume.Phone = parsed.Phone

		// Store parsed data
		if len(parsed.Skills) > 0 || len(parsed.WorkExperience) > 0 || len(parsed.Education) > 0 || parsed.RawText != "" {
			parsedData := make(map[string]any)
			if len(parsed.Skills) > 0 {
				parsedData["skills"] = parsed.Skills
			}
			if len(parsed.WorkExperience) > 0 {
				parsedData["work_experience"] = parsed.WorkExperience
			}
			if len(parsed.Education) > 0 {
				parsedData["education"] = parsed.Education
			}
			if parsed.Summary != "" {
				parsedData["summary"] = parsed.Summary
			}
			if len(parsed.Languages) > 0 {
				parsedData["languages"] = parsed.Languages
			}
			if len(parsed.Certifications) > 0 {
				parsedData["certifications"] = parsed.Certifications
			}
			if parsed.RawText != "" {
				parsedData["raw_text"] = parsed.RawText
			}
			resume.ParsedData = parsedData
		}
	}

	// Store error message in summary if parsing failed
	if parseErr != nil {
		if resume.ParsedData == nil {
			resume.ParsedData = make(map[string]any)
		}
		resume.ParsedData["summary"] = fmt.Sprintf("Parse failed: %s", parseErr.Error())
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		// Clean up uploaded file on database error
		_ = s.storage.DeleteFile(ctx, objectKey)
		return nil, nil, fmt.Errorf("failed to create resume: %w", err)
	}

	// Publish events
	if s.publisher != nil {
		_ = s.publisher.PublishCreated(ctx, resume.ID.String(), resume)
		if parseErr != nil {
			// Publish failed status
			_ = s.publisher.PublishStatusChanged(ctx, resume.ID.String(), model.StatusPending, model.StatusFailed)
		} else {
			// Publish parsed status
			_ = s.publisher.PublishStatusChanged(ctx, resume.ID.String(), model.StatusPending, model.StatusParsed)
			_ = s.publisher.PublishParsed(ctx, resume.ID.String(), parsed)
		}
	}

	// Return resume even on parse failure (for tracking/audit purposes)
	return resume, parsed, parseErr
}

// extractObjectKeyFromURL extracts the MinIO object key from a full URL
// URL format: http://endpoint/bucket/objectKey
func extractObjectKeyFromURL(fileURL string) string {
	// Find the bucket name position
	// URL format: http://host:port/bucket/objectKey
	idx := strings.Index(fileURL, "://")
	if idx == -1 {
		return fileURL // Already an object key
	}

	// Get the path part after host:port
	path := fileURL[idx+3:] // Skip "://"
	slashIdx := strings.Index(path, "/")
	if slashIdx == -1 {
		return fileURL
	}

	// Skip bucket name to get object key
	path = path[slashIdx+1:] // Skip first slash and host
	slashIdx = strings.Index(path, "/")
	if slashIdx == -1 {
		return path
	}

	return path[slashIdx+1:] // Return object key after bucket
}
