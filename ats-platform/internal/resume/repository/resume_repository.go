package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/ats-platform/internal/resume/model"
)

// JSONMap is a custom type for handling map[string]any with GORM and SQLite
type JSONMap map[string]any

// Value implements driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		*j = make(JSONMap)
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// GormDataType implements schema.GormDataTypeInterface
func (JSONMap) GormDataType() string {
	return "json"
}

var (
	// ErrNotFound is returned when a record is not found in the database
	ErrNotFound = errors.New("record not found")
)

// ListFilter defines the filter options for listing resumes
type ListFilter struct {
	Page     int
	PageSize int
	Status   string
	Source   string
}

// ResumeRepository defines the interface for resume data operations
type ResumeRepository interface {
	Create(ctx context.Context, resume *model.Resume) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error)
	List(ctx context.Context, filter ListFilter) ([]model.Resume, int64, error)
	Update(ctx context.Context, resume *model.Resume) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateStatusIf(ctx context.Context, id uuid.UUID, newStatus string, expectedStatuses []string) (bool, error)
	UpdateFileURL(ctx context.Context, id uuid.UUID, fileURL string) error
}

// gormRepository implements ResumeRepository using GORM
type gormRepository struct {
	db *gorm.DB
}

// NewGormRepository creates a new GORM-based resume repository
func NewGormRepository(db *gorm.DB) ResumeRepository {
	return &gormRepository{db: db}
}

// Create inserts a new resume into the database
func (r *gormRepository) Create(ctx context.Context, resume *model.Resume) error {
	// Set timestamps if not already set
	if resume.CreatedAt.IsZero() {
		resume.CreatedAt = time.Now()
	}
	if resume.UpdatedAt.IsZero() {
		resume.UpdatedAt = time.Now()
	}

	// Convert ParsedData to JSON
	parsedDataJSON, _ := json.Marshal(resume.ParsedData)
	if len(parsedDataJSON) == 0 {
		parsedDataJSON = []byte("{}")
	}

	// Use raw SQL to insert with proper JSON handling
	query := `INSERT INTO resumes (id, name, email, phone, source, file_url, parsed_data, status, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	return r.db.WithContext(ctx).Exec(query,
		resume.ID.String(),
		resume.Name,
		resume.Email,
		resume.Phone,
		resume.Source,
		resume.FileURL,
		string(parsedDataJSON),
		resume.Status,
		resume.CreatedAt,
		resume.UpdatedAt,
	).Error
}

// GetByID retrieves a resume by its ID
func (r *gormRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	type dbResume struct {
		ID         uuid.UUID
		Name       string
		Email      string
		Phone      string
		Source     string
		FileURL    string
		ParsedData string
		Status     string
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}

	var resume dbResume
	result := r.db.WithContext(ctx).Raw(`
		SELECT id, name, email, phone, source, file_url, parsed_data, status, created_at, updated_at
		FROM resumes
		WHERE id = ?
	`, id).Scan(&resume)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	// Parse ParsedData JSON
	var parsedData map[string]any
	if resume.ParsedData != "" && resume.ParsedData != "{}" {
		_ = json.Unmarshal([]byte(resume.ParsedData), &parsedData)
	}

	return &model.Resume{
		ID:         resume.ID,
		Name:       resume.Name,
		Email:      resume.Email,
		Phone:      resume.Phone,
		Source:     resume.Source,
		FileURL:    resume.FileURL,
		ParsedData: parsedData,
		Status:     resume.Status,
		CreatedAt:  resume.CreatedAt,
		UpdatedAt:  resume.UpdatedAt,
	}, nil
}

// List retrieves resumes with filtering and pagination
func (r *gormRepository) List(ctx context.Context, filter ListFilter) ([]model.Resume, int64, error) {
	type dbResume struct {
		ID         uuid.UUID
		Name       string
		Email      string
		Phone      string
		Source     string
		FileURL    string
		ParsedData string
		Status     string
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}

	// Build WHERE clause
	whereClause := "1=1"
	args := []interface{}{}

	if filter.Status != "" {
		whereClause += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Source != "" {
		whereClause += " AND source = ?"
		args = append(args, filter.Source)
	}

	// Count total records matching the filter
	var total int64
	countQuery := "SELECT COUNT(*) FROM resumes WHERE " + whereClause
	err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// Fetch paginated results
	query := `SELECT id, name, email, phone, source, file_url, parsed_data, status, created_at, updated_at
			  FROM resumes WHERE ` + whereClause + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	var dbResumes []dbResume
	err = r.db.WithContext(ctx).Raw(query, args...).Scan(&dbResumes).Error
	if err != nil {
		return nil, 0, err
	}

	// Convert to model.Resume
	resumes := make([]model.Resume, len(dbResumes))
	for i, dr := range dbResumes {
		var parsedData map[string]any
		if dr.ParsedData != "" && dr.ParsedData != "{}" {
			_ = json.Unmarshal([]byte(dr.ParsedData), &parsedData)
		}

		resumes[i] = model.Resume{
			ID:         dr.ID,
			Name:       dr.Name,
			Email:      dr.Email,
			Phone:      dr.Phone,
			Source:     dr.Source,
			FileURL:    dr.FileURL,
			ParsedData: parsedData,
			Status:     dr.Status,
			CreatedAt:  dr.CreatedAt,
			UpdatedAt:  dr.UpdatedAt,
		}
	}

	return resumes, total, nil
}

// Update modifies an existing resume in the database
func (r *gormRepository) Update(ctx context.Context, resume *model.Resume) error {
	// Convert ParsedData to JSON
	parsedDataJSON, _ := json.Marshal(resume.ParsedData)
	if len(parsedDataJSON) == 0 {
		parsedDataJSON = []byte("{}")
	}

	query := `UPDATE resumes
			  SET name = ?, email = ?, phone = ?, source = ?, file_url = ?, parsed_data = ?, status = ?, updated_at = ?
			  WHERE id = ?`

	result := r.db.WithContext(ctx).Exec(query,
		resume.Name,
		resume.Email,
		resume.Phone,
		resume.Source,
		resume.FileURL,
		string(parsedDataJSON),
		resume.Status,
		resume.UpdatedAt,
		resume.ID,
	)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a resume from the database (hard delete)
func (r *gormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM resumes WHERE id = ?`
	result := r.db.WithContext(ctx).Exec(query, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateStatus updates only the status of a resume
func (r *gormRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE resumes SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result := r.db.WithContext(ctx).Exec(query, status, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateStatusIf updates status only if current status matches one of expected statuses
// Returns true if update was performed, false if status didn't match
func (r *gormRepository) UpdateStatusIf(ctx context.Context, id uuid.UUID, newStatus string, expectedStatuses []string) (bool, error) {
	query := `UPDATE resumes SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status IN ?`
	result := r.db.WithContext(ctx).Exec(query, newStatus, id, expectedStatuses)

	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// UpdateFileURL updates only the file URL of a resume
func (r *gormRepository) UpdateFileURL(ctx context.Context, id uuid.UUID, fileURL string) error {
	query := `UPDATE resumes SET file_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result := r.db.WithContext(ctx).Exec(query, fileURL, id)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
