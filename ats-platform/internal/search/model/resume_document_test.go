package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResumeDocument_IndexName(t *testing.T) {
	doc := ResumeDocument{}
	assert.Equal(t, "resumes", doc.IndexName())
}

func TestResumeDocument_DocumentID(t *testing.T) {
	doc := ResumeDocument{
		ResumeID: "test-resume-123",
	}
	assert.Equal(t, "test-resume-123", doc.DocumentID())
}

func TestResumeDocument_EmptyDocumentID(t *testing.T) {
	doc := ResumeDocument{}
	assert.Equal(t, "", doc.DocumentID())
}

func TestNewResumeDocument(t *testing.T) {
	now := time.Now()
	doc := NewResumeDocument(
		"resume-uuid-123",
		"John Doe",
		"john@example.com",
		[]string{"Go", "Python", "Docker"},
		5,
		"Bachelor",
		"Software Engineer at Tech Corp",
		"parsed",
		"LinkedIn",
		now,
		now,
	)

	assert.Equal(t, "resume-uuid-123", doc.ResumeID)
	assert.Equal(t, "John Doe", doc.Name)
	assert.Equal(t, "john@example.com", doc.Email)
	assert.Equal(t, []string{"Go", "Python", "Docker"}, doc.Skills)
	assert.Equal(t, 5, doc.ExperienceYears)
	assert.Equal(t, "Bachelor", doc.Education)
	assert.Equal(t, "Software Engineer at Tech Corp", doc.WorkHistory)
	assert.Equal(t, "parsed", doc.Status)
	assert.Equal(t, "LinkedIn", doc.Source)
	assert.Equal(t, now, doc.CreatedAt)
	assert.Equal(t, now, doc.UpdatedAt)
}

func TestNewResumeDocument_EmptySkills(t *testing.T) {
	now := time.Now()
	doc := NewResumeDocument(
		"resume-uuid-456",
		"Jane Doe",
		"jane@example.com",
		[]string{},
		3,
		"Master",
		"",
		"pending",
		"Direct",
		now,
		now,
	)

	assert.Equal(t, "resume-uuid-456", doc.ResumeID)
	assert.Equal(t, "Jane Doe", doc.Name)
	assert.Empty(t, doc.Skills)
	assert.Equal(t, 3, doc.ExperienceYears)
	assert.Equal(t, "Master", doc.Education)
	assert.Empty(t, doc.WorkHistory)
	assert.Equal(t, "pending", doc.Status)
}

func TestResumeDocument_AllFields(t *testing.T) {
	now := time.Now()
	doc := ResumeDocument{
		ResumeID:       "full-resume-001",
		Name:           "Alice Smith",
		Email:          "alice@example.com",
		Skills:         []string{"Java", "Kubernetes", "AWS"},
		ExperienceYears: 10,
		Education:      "PhD",
		WorkHistory:    "Senior Engineer at Big Tech",
		Status:         "parsed",
		Source:         "Indeed",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	assert.Equal(t, "resumes", doc.IndexName())
	assert.Equal(t, "full-resume-001", doc.DocumentID())
	assert.Equal(t, "Alice Smith", doc.Name)
	assert.Equal(t, "alice@example.com", doc.Email)
	assert.Equal(t, []string{"Java", "Kubernetes", "AWS"}, doc.Skills)
	assert.Equal(t, 10, doc.ExperienceYears)
	assert.Equal(t, "PhD", doc.Education)
	assert.Equal(t, "Senior Engineer at Big Tech", doc.WorkHistory)
	assert.Equal(t, "parsed", doc.Status)
	assert.Equal(t, "Indeed", doc.Source)
	assert.Equal(t, now, doc.CreatedAt)
	assert.Equal(t, now, doc.UpdatedAt)
}

func TestResumeDocument_ZeroExperienceYears(t *testing.T) {
	doc := ResumeDocument{
		ResumeID:       "junior-001",
		Name:           "Junior Dev",
		ExperienceYears: 0,
	}

	assert.Equal(t, 0, doc.ExperienceYears)
}

func TestResumeDocument_NilSkills(t *testing.T) {
	doc := ResumeDocument{
		ResumeID: "no-skills-001",
		Name:     "No Skills",
		Skills:   nil,
	}

	assert.Nil(t, doc.Skills)
}
