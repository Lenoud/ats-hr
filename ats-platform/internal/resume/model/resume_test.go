package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestResume_TableName(t *testing.T) {
	resume := Resume{}
	if got := resume.TableName(); got != "resumes" {
		t.Errorf("Resume.TableName() = %v, want %v", got, "resumes")
	}
}

func TestResume_IsParsed(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "parsed status returns true",
			status: StatusParsed,
			want:   true,
		},
		{
			name:   "pending status returns false",
			status: StatusPending,
			want:   false,
		},
		{
			name:   "processing status returns false",
			status: StatusProcessing,
			want:   false,
		},
		{
			name:   "failed status returns false",
			status: StatusFailed,
			want:   false,
		},
		{
			name:   "archived status returns false",
			status: StatusArchived,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Resume{Status: tt.status}
			if got := r.IsParsed(); got != tt.want {
				t.Errorf("Resume.IsParsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResume_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name        string
		currentStatus string
		newStatus     string
		want         bool
	}{
		// Valid transitions from pending
		{
			name:        "pending to processing is valid",
			currentStatus: StatusPending,
			newStatus:     StatusProcessing,
			want:         true,
		},
		{
			name:        "pending to archived is valid",
			currentStatus: StatusPending,
			newStatus:     StatusArchived,
			want:         true,
		},
		{
			name:        "pending to parsed is invalid",
			currentStatus: StatusPending,
			newStatus:     StatusParsed,
			want:         false,
		},
		{
			name:        "pending to failed is invalid",
			currentStatus: StatusPending,
			newStatus:     StatusFailed,
			want:         false,
		},
		// Valid transitions from processing
		{
			name:        "processing to parsed is valid",
			currentStatus: StatusProcessing,
			newStatus:     StatusParsed,
			want:         true,
		},
		{
			name:        "processing to failed is valid",
			currentStatus: StatusProcessing,
			newStatus:     StatusFailed,
			want:         true,
		},
		{
			name:        "processing to pending is invalid",
			currentStatus: StatusProcessing,
			newStatus:     StatusPending,
			want:         false,
		},
		{
			name:        "processing to archived is invalid",
			currentStatus: StatusProcessing,
			newStatus:     StatusArchived,
			want:         false,
		},
		// Valid transitions from parsed
		{
			name:        "parsed to archived is valid",
			currentStatus: StatusParsed,
			newStatus:     StatusArchived,
			want:         true,
		},
		{
			name:        "parsed to pending is invalid",
			currentStatus: StatusParsed,
			newStatus:     StatusPending,
			want:         false,
		},
		{
			name:        "parsed to processing is invalid",
			currentStatus: StatusParsed,
			newStatus:     StatusProcessing,
			want:         false,
		},
		// Valid transitions from failed
		{
			name:        "failed to pending is valid",
			currentStatus: StatusFailed,
			newStatus:     StatusPending,
			want:         true,
		},
		{
			name:        "failed to archived is valid",
			currentStatus: StatusFailed,
			newStatus:     StatusArchived,
			want:         true,
		},
		{
			name:        "failed to processing is invalid",
			currentStatus: StatusFailed,
			newStatus:     StatusProcessing,
			want:         false,
		},
		// Valid transitions from archived
		{
			name:        "archived has no valid transitions",
			currentStatus: StatusArchived,
			newStatus:     StatusPending,
			want:         false,
		},
		{
			name:        "archived to processing is invalid",
			currentStatus: StatusArchived,
			newStatus:     StatusProcessing,
			want:         false,
		},
		// Invalid status transitions
		{
			name:        "invalid current status",
			currentStatus: "invalid_status",
			newStatus:     StatusPending,
			want:         false,
		},
		{
			name:        "invalid target status",
			currentStatus: StatusPending,
			newStatus:     "invalid_status",
			want:         false,
		},
		// Same status
		{
			name:        "same status is invalid",
			currentStatus: StatusPending,
			newStatus:     StatusPending,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Resume{Status: tt.currentStatus}
			if got := r.CanTransitionTo(tt.newStatus); got != tt.want {
				t.Errorf("Resume.CanTransitionTo(%v) = %v, want %v", tt.newStatus, got, tt.want)
			}
		})
	}
}

func TestResume_DefaultValues(t *testing.T) {
	r := &Resume{}

	// Test default status
	if r.Status != "" {
		t.Errorf("Expected empty status for new resume, got %v", r.Status)
	}

	// Test with default status set
	r.Status = StatusPending
	if r.Status != StatusPending {
		t.Errorf("Expected status %v, got %v", StatusPending, r.Status)
	}
}

func TestResume_IDGeneration(t *testing.T) {
	id := uuid.New()
	r := Resume{ID: id}

	if r.ID != id {
		t.Errorf("Expected ID %v, got %v", id, r.ID)
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"StatusPending constant", StatusPending},
		{"StatusProcessing constant", StatusProcessing},
		{"StatusParsed constant", StatusParsed},
		{"StatusFailed constant", StatusFailed},
		{"StatusArchived constant", StatusArchived},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status == "" {
				t.Errorf("Status constant %v is empty", tt.name)
			}
		})
	}
}

func TestResume_AllFields(t *testing.T) {
	id := uuid.New()
	r := Resume{
		ID:         id,
		Name:       "John Doe",
		Email:      "john@example.com",
		Phone:      "+1234567890",
		Source:     "LinkedIn",
		FileURL:    "https://example.com/resume.pdf",
		ParsedData: map[string]any{"experience": "5 years"},
		Status:     StatusParsed,
	}

	if r.ID != id {
		t.Errorf("Expected ID %v, got %v", id, r.ID)
	}
	if r.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", r.Name)
	}
	if r.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %v", r.Email)
	}
	if r.Phone != "+1234567890" {
		t.Errorf("Expected phone '+1234567890', got %v", r.Phone)
	}
	if r.Source != "LinkedIn" {
		t.Errorf("Expected source 'LinkedIn', got %v", r.Source)
	}
	if r.FileURL != "https://example.com/resume.pdf" {
		t.Errorf("Expected file URL 'https://example.com/resume.pdf', got %v", r.FileURL)
	}
	if r.ParsedData == nil {
		t.Error("Expected ParsedData to be set, got nil")
	}
	if r.Status != StatusParsed {
		t.Errorf("Expected status %v, got %v", StatusParsed, r.Status)
	}
}
