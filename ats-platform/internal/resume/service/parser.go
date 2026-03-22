package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/example/ats-platform/internal/shared/llm"
)

var (
	ErrParseFailed       = errors.New("failed to parse resume")
	ErrUnsupportedFormat = errors.New("unsupported file format")
)

// ParsedResume represents structured resume data
type ParsedResume struct {
	Name           string           `json:"name,omitempty"`
	Email          string           `json:"email,omitempty"`
	Phone          string           `json:"phone,omitempty"`
	Summary        string           `json:"summary,omitempty"`
	Skills         []string         `json:"skills,omitempty"`
	WorkExperience []WorkExperience `json:"work_experience,omitempty"`
	Education      []Education      `json:"education,omitempty"`
	Languages      []string         `json:"languages,omitempty"`
	Certifications []string         `json:"certifications,omitempty"`
	Source         string           `json:"source,omitempty"`
	UploadDate     string           `json:"upload_date,omitempty"`
	RawText        string           `json:"raw_text,omitempty"`
}

// WorkExperience represents a work experience entry
type WorkExperience struct {
	Company     string `json:"company,omitempty"`
	Position    string `json:"position,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	Description string `json:"description,omitempty"`
}

// Education represents an education entry
type Education struct {
	School    string `json:"school,omitempty"`
	Degree    string `json:"degree,omitempty"`
	Major     string `json:"major,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// ResumeParser defines the interface for parsing resumes
type ResumeParser interface {
	Parse(ctx context.Context, reader io.Reader, filename string) (*ParsedResume, error)
}

// resumeParser implements ResumeParser
type resumeParser struct {
	llmClient *llm.Client
}

// NewResumeParser creates a new resume parser
func NewResumeParser() ResumeParser {
	return &resumeParser{}
}

// NewResumeParserWithLLM creates a new resume parser with LLM support
func NewResumeParserWithLLM(client *llm.Client) ResumeParser {
	return &resumeParser{llmClient: client}
}

// Parse parses a resume file and extracts structured data
func (p *resumeParser) Parse(ctx context.Context, reader io.Reader, filename string) (*ParsedResume, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var text string
	lowerFilename := strings.ToLower(filename)

	switch {
	case strings.HasSuffix(lowerFilename, ".pdf"):
		text, err = p.extractTextFromPDF(content)
	case strings.HasSuffix(lowerFilename, ".docx"):
		text, err = p.extractTextFromDOCX(content)
	case strings.HasSuffix(lowerFilename, ".doc"):
		text, err = p.extractTextFromDOC(content)
	default:
		return nil, ErrUnsupportedFormat
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	return p.parseWithLLM(ctx, text, filename)
}

// parseWithLLM sends text to LLM for structured parsing
func (p *resumeParser) parseWithLLM(ctx context.Context, text, filename string) (*ParsedResume, error) {
	result := &ParsedResume{
		RawText: text,
	}

	if p.llmClient == nil || strings.TrimSpace(text) == "" {
		return result, nil
	}

	systemPrompt := `You are a resume parser. Extract structured information from the resume text.
Return ONLY valid JSON with the following structure (omit fields if not found):
{
  "name": "full name",
  "email": "email address",
  "phone": "phone number",
  "summary": "professional summary",
  "skills": ["skill1", "skill2"],
  "work_experience": [
    {"company": "name", "position": "title", "start_date": "YYYY-MM", "end_date": "YYYY-MM or present", "description": "brief description"}
  ],
  "education": [
    {"school": "name", "degree": "degree type", "major": "field of study", "start_date": "YYYY", "end_date": "YYYY"}
  ],
  "languages": ["language1"],
  "certifications": ["cert1"]
}`

	userPrompt := fmt.Sprintf("Parse this resume text and return structured JSON:\n\n%s", text)

	response, err := p.llmClient.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return result, nil // Return raw text on LLM failure
	}

	// Clean up response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var parsed ParsedResume
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return result, nil // Return raw text on parse failure
	}

	parsed.RawText = text
	return &parsed, nil
}

// extractTextFromPDF extracts plain text from PDF content
func (p *resumeParser) extractTextFromPDF(content []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("create pdf reader: %w", err)
	}

	var result strings.Builder
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		result.WriteString(text)
		if i < numPages {
			result.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(result.String()), nil
}

// extractTextFromDOCX extracts plain text from DOCX content
func (p *resumeParser) extractTextFromDOCX(content []byte) (string, error) {
	doc, err := docx.ReadDocxFromMemory(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("read docx: %w", err)
	}
	defer doc.Close()

	return doc.Editable().GetContent(), nil
}

// extractTextFromDOC extracts plain text from DOC content
func (p *resumeParser) extractTextFromDOC(content []byte) (string, error) {
	return "", ErrUnsupportedFormat
}
