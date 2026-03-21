package model

import "time"

// ResumeDocument represents a resume document stored in Elasticsearch
type ResumeDocument struct {
	ResumeID       string    `json:"resume_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Skills         []string  `json:"skills"`
	ExperienceYears int      `json:"experience_years"`
	Education      string    `json:"education"`
	WorkHistory    string    `json:"work_history"`
	Status         string    `json:"status"`
	Source         string    `json:"source"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IndexName returns the Elasticsearch index name for resume documents
func (r ResumeDocument) IndexName() string {
	return "resumes"
}

// DocumentID returns the unique document ID for Elasticsearch
func (r ResumeDocument) DocumentID() string {
	return r.ResumeID
}

// NewResumeDocument creates a new ResumeDocument instance
func NewResumeDocument(
	resumeID string,
	name string,
	email string,
	skills []string,
	experienceYears int,
	education string,
	workHistory string,
	status string,
	source string,
	createdAt time.Time,
	updatedAt time.Time,
) *ResumeDocument {
	return &ResumeDocument{
		ResumeID:       resumeID,
		Name:           name,
		Email:          email,
		Skills:         skills,
		ExperienceYears: experienceYears,
		Education:      education,
		WorkHistory:    workHistory,
		Status:         status,
		Source:         source,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}
