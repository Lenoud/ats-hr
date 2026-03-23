package events

import "time"

const (
	ActionCreated       = "created"
	ActionUpdated       = "updated"
	ActionDeleted       = "deleted"
	ActionStatusChanged = "status_changed"
	ActionParsed        = "parsed"
)

// ResumeDocumentPayload is the shared payload used for create/update style indexing events.
type ResumeDocumentPayload struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Email      string                 `json:"email"`
	Source     string                 `json:"source"`
	ParsedData map[string]interface{} `json:"parsed_data"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// ResumeStatusChangedPayload is the shared payload for status transitions.
type ResumeStatusChangedPayload struct {
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

// ResumeParsedPayload is the shared payload emitted after parsing.
type ResumeParsedPayload struct {
	Name           string        `json:"name,omitempty"`
	Email          string        `json:"email,omitempty"`
	Phone          string        `json:"phone,omitempty"`
	Summary        string        `json:"summary,omitempty"`
	Skills         []string      `json:"skills,omitempty"`
	WorkExperience []interface{} `json:"work_experience,omitempty"`
	Education      []interface{} `json:"education,omitempty"`
	Languages      []string      `json:"languages,omitempty"`
	Certifications []string      `json:"certifications,omitempty"`
	Source         string        `json:"source,omitempty"`
	UploadDate     string        `json:"upload_date,omitempty"`
	RawText        string        `json:"raw_text,omitempty"`
}
