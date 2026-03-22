package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/ats-platform/internal/interview/model"
)

var (
	// ErrNotFound is returned when a record is not found in the database
	ErrNotFound = errors.New("record not found")
)

// InterviewRepository defines the interface for interview data operations
type InterviewRepository interface {
	Create(ctx context.Context, interview *model.Interview) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error)
	ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Interview, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// interviewRepo implements InterviewRepository using GORM with raw SQL
type interviewRepo struct {
	db *gorm.DB
}

// NewInterviewRepository creates a new interview repository
func NewInterviewRepository(db *gorm.DB) InterviewRepository {
	return &interviewRepo{db: db}
}

// Create inserts a new interview into the database
func (r *interviewRepo) Create(ctx context.Context, interview *model.Interview) error {
	if interview.ID == uuid.Nil {
		interview.ID = uuid.New()
	}
	if interview.CreatedAt.IsZero() {
		interview.CreatedAt = time.Now()
	}
	if interview.UpdatedAt.IsZero() {
		interview.UpdatedAt = time.Now()
	}

	query := `INSERT INTO interviews (id, resume_id, round, interviewer, scheduled_at, status, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	return r.db.WithContext(ctx).Exec(query,
		interview.ID.String(),
		interview.ResumeID.String(),
		interview.Round,
		interview.Interviewer,
		interview.ScheduledAt,
		interview.Status,
		interview.CreatedAt,
		interview.UpdatedAt,
	).Error
}

// GetByID retrieves an interview by its ID
func (r *interviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error) {
	var interview model.Interview
	query := `SELECT id, resume_id, round, interviewer, scheduled_at, status, created_at, updated_at
			  FROM interviews WHERE id = ?`

	result := r.db.WithContext(ctx).Raw(query, id).Scan(&interview)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &interview, nil
}

// ListByResumeID retrieves all interviews for a resume
func (r *interviewRepo) ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]model.Interview, error) {
	query := `SELECT id, resume_id, round, interviewer, scheduled_at, status, created_at, updated_at
			  FROM interviews WHERE resume_id = ? ORDER BY round ASC, created_at DESC`

	var interviews []model.Interview
	result := r.db.WithContext(ctx).Raw(query, resumeID).Scan(&interviews)
	if result.Error != nil {
		return nil, result.Error
	}

	return interviews, nil
}

// UpdateStatus updates only the status of an interview
func (r *interviewRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE interviews SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result := r.db.WithContext(ctx).Exec(query, status, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an interview from the database
func (r *interviewRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM interviews WHERE id = ?`
	result := r.db.WithContext(ctx).Exec(query, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
