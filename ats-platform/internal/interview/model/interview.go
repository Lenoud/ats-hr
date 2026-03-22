package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Interview status constants
const (
	InterviewStatusScheduled = "scheduled"
	InterviewStatusCompleted = "completed"
	InterviewStatusCancelled = "cancelled"
)

// validInterviewStatusTransitions defines allowed status transitions
var validInterviewStatusTransitions = map[string][]string{
	InterviewStatusScheduled: {InterviewStatusCompleted, InterviewStatusCancelled},
	InterviewStatusCompleted: {},
	InterviewStatusCancelled: {InterviewStatusScheduled}, // Can reschedule
}

// Interview represents an interview record
type Interview struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ResumeID    uuid.UUID `json:"resume_id" gorm:"type:uuid;not null;index"`
	Round       int       `json:"round" gorm:"not null"`
	Interviewer string    `json:"interviewer" gorm:"type:varchar(100)"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Status      string    `json:"status" gorm:"type:varchar(20);default:scheduled"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Interview) TableName() string {
	return "interviews"
}

// CanTransitionTo checks if a status transition is valid
func (i *Interview) CanTransitionTo(newStatus string) bool {
	allowedTransitions, exists := validInterviewStatusTransitions[i.Status]
	if !exists {
		return false
	}
	return slices.Contains(allowedTransitions, newStatus)
}

// IsCompleted returns true if the interview is completed
func (i *Interview) IsCompleted() bool {
	return i.Status == InterviewStatusCompleted
}
