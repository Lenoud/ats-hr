package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/ats-platform/internal/interview/model"
)

// FeedbackRepository defines the interface for feedback data operations
type FeedbackRepository interface {
	Create(ctx context.Context, feedback *model.Feedback) error
	GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error)
}

// feedbackRepo implements FeedbackRepository using GORM with raw SQL
type feedbackRepo struct {
	db *gorm.DB
}

// NewFeedbackRepository creates a new feedback repository
func NewFeedbackRepository(db *gorm.DB) FeedbackRepository {
	return &feedbackRepo{db: db}
}

// Create inserts a new feedback into the database
func (r *feedbackRepo) Create(ctx context.Context, feedback *model.Feedback) error {
	if feedback.ID == uuid.Nil {
		feedback.ID = uuid.New()
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now()
	}

	query := `INSERT INTO feedbacks (id, interview_id, rating, content, recommendation, created_at)
			  VALUES (?, ?, ?, ?, ?, ?)`

	return r.db.WithContext(ctx).Exec(query,
		feedback.ID.String(),
		feedback.InterviewID.String(),
		feedback.Rating,
		feedback.Content,
		feedback.Recommendation,
		feedback.CreatedAt,
	).Error
}

// GetByInterviewID retrieves feedback by interview ID
func (r *feedbackRepo) GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error) {
	var feedback model.Feedback
	query := `SELECT id, interview_id, rating, content, recommendation, created_at
			  FROM feedbacks WHERE interview_id = ?`

	result := r.db.WithContext(ctx).Raw(query, interviewID).Scan(&feedback)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &feedback, nil
}
