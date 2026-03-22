package model

import (
	"time"

	"github.com/google/uuid"
)

// Recommendation constants for interview feedback
const (
	RecommendationStrongYes = "strong_yes"
	RecommendationYes       = "yes"
	RecommendationNo        = "no"
	RecommendationStrongNo  = "strong_no"
)

// ValidRecommendations contains all valid recommendation values
var ValidRecommendations = []string{
	RecommendationStrongYes,
	RecommendationYes,
	RecommendationNo,
	RecommendationStrongNo,
}

// Feedback represents interview feedback
type Feedback struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	InterviewID    uuid.UUID `json:"interview_id" gorm:"type:uuid;not null;uniqueIndex"`
	Rating         int       `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Content        string    `json:"content" gorm:"type:text"`
	Recommendation string    `json:"recommendation" gorm:"type:varchar(20)"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for GORM
func (Feedback) TableName() string {
	return "feedbacks"
}

// IsValidRating checks if the rating is within valid range (1-5)
func (f *Feedback) IsValidRating() bool {
	return f.Rating >= 1 && f.Rating <= 5
}

// IsValidRecommendation checks if the recommendation is valid
func (f *Feedback) IsValidRecommendation() bool {
	for _, r := range ValidRecommendations {
		if f.Recommendation == r {
			return true
		}
	}
	return false
}
