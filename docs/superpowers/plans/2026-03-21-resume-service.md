# Resume Service 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Resume Service 的完整功能，包括简历 CRUD 操作、简历解析（基础+LLM增强）、gRPC 接口和数据同步到 Redis Streams。

**Architecture:** 采用分层架构（Handler → Service → Repository），使用 GORM 操作 PostgreSQL，通过 Redis Streams 发送事件通知，支持 gRPC 服务间调用。

**Tech Stack:** Go 1.25, Gin, GORM, pgx, go-redis, grpc-go, testify

---

## File Structure

```
ats-platform/
├── cmd/resume-service/main.go          # 服务入口（已存在，需更新）
├── internal/resume/
│   ├── handler/
│   │   ├── resume_handler.go           # HTTP handlers
│   │   └── resume_handler_test.go      # Handler 测试
│   ├── service/
│   │   ├── resume_service.go           # 业务逻辑
│   │   ├── resume_service_test.go      # Service 测试
│   │   └── parser.go                   # 简历解析逻辑
│   ├── repository/
│   │   ├── resume_repository.go        # 数据访问层
│   │   └── resume_repository_test.go   # Repository 测试
│   └── model/
│       └── resume.go                   # 数据模型
├── internal/shared/
│   ├── database/
│   │   └── postgres.go                 # 数据库连接
│   └── events/
│       └── publisher.go                # Redis Streams 事件发布
└── tests/
    └── integration/
        └── resume_test.go              # 集成测试
```

---

## Task 1: 数据模型定义

**Files:**
- Create: `internal/resume/model/resume.go`
- Create: `internal/resume/model/resume_test.go`

- [ ] **Step 1: 编写 Resume 模型测试**

```go
// internal/resume/model/resume_test.go
package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestResume_TableName(t *testing.T) {
	r := Resume{}
	if r.TableName() != "resumes" {
		t.Errorf("expected table name 'resumes', got '%s'", r.TableName())
	}
}

func TestResume_DefaultStatus(t *testing.T) {
	r := Resume{}
	if r.Status != "pending" {
		t.Errorf("expected default status 'pending', got '%s'", r.Status)
	}
}

func TestResume_StatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Pending", StatusPending, "pending"},
		{"Processing", StatusProcessing, "processing"},
		{"Parsed", StatusParsed, "parsed"},
		{"Failed", StatusFailed, "failed"},
		{"Archived", StatusArchived, "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go test ./internal/resume/model/... -v
```

Expected: 测试失败，因为 Resume 类型未定义

- [ ] **Step 3: 实现 Resume 模型**

```go
// internal/resume/model/resume.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// Resume status constants
const (
	StatusPending     = "pending"
	StatusProcessing  = "processing"
	StatusParsed      = "parsed"
	StatusFailed      = "failed"
	StatusArchived    = "archived"
)

// Resume represents a candidate's resume
type Resume struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string          `json:"name" gorm:"size:100;not null"`
	Email       string          `json:"email" gorm:"size:100"`
	Phone       string          `json:"phone" gorm:"size:20"`
	Source      string          `json:"source" gorm:"size:50"`
	FileURL     string          `json:"file_url" gorm:"type:text"`
	ParsedData  map[string]any  `json:"parsed_data" gorm:"type:jsonb"`
	Status      string          `json:"status" gorm:"size:20;default:pending"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for GORM
func (Resume) TableName() string {
	return "resumes"
}

// BeforeCreate sets default values before creating
func (r *Resume) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.Status == "" {
		r.Status = StatusPending
	}
	return nil
}

// IsParsed returns true if resume has been parsed successfully
func (r *Resume) IsParsed() bool {
	return r.Status == StatusParsed
}

// CanTransitionTo checks if status transition is valid
func (r *Resume) CanTransitionTo(newStatus string) bool {
	validTransitions := map[string][]string{
		StatusPending:    {StatusProcessing, StatusArchived},
		StatusProcessing: {StatusParsed, StatusFailed},
		StatusParsed:     {StatusArchived},
		StatusFailed:     {StatusPending, StatusArchived},
		StatusArchived:   {},
	}

	allowed, exists := validTransitions[r.Status]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 添加依赖并运行测试**

```bash
go get github.com/google/uuid gorm.io/gorm
go test ./internal/resume/model/... -v
```

Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add internal/resume/model/
git commit -m "feat(resume): add resume model with status transitions

- Define Resume struct with GORM tags
- Add status constants and transitions
- Add unit tests for model"
```

---

## Task 2: Repository 层实现

**Files:**
- Create: `internal/resume/repository/resume_repository.go`
- Create: `internal/resume/repository/resume_repository_test.go`

- [ ] **Step 1: 编写 Repository 接口和测试**

```go
// internal/resume/repository/resume_repository.go
package repository

import (
	"context"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/google/uuid"
)

// ResumeRepository defines the interface for resume data access
type ResumeRepository interface {
	Create(ctx context.Context, resume *model.Resume) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error)
	List(ctx context.Context, filter ListFilter) ([]model.Resume, int64, error)
	Update(ctx context.Context, resume *model.Resume) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

// ListFilter defines filtering options for listing resumes
type ListFilter struct {
	Page     int
	PageSize int
	Status   string
	Source   string
}
```

- [ ] **Step 2: 编写 Repository 测试**

```go
// internal/resume/repository/resume_repository_test.go
package repository

import (
	"context"
	"testing"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResumeRepository_Create(t *testing.T) {
	// This test uses a test database connection
	// Setup will be handled in test main
	t.Run("should create resume with generated ID", func(t *testing.T) {
		repo := NewTestRepository(t)
		defer repo.Cleanup()

		resume := &model.Resume{
			Name:   "John Doe",
			Email:  "john@example.com",
			Phone:  "+1234567890",
			Source: "boss",
		}

		err := repo.Create(context.Background(), resume)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resume.ID)
		assert.Equal(t, model.StatusPending, resume.Status)
	})
}

func TestResumeRepository_GetByID(t *testing.T) {
	t.Run("should return resume when exists", func(t *testing.T) {
		repo := NewTestRepository(t)
		defer repo.Cleanup()

		resume := &model.Resume{
			Name:   "Jane Doe",
			Email:  "jane@example.com",
			Source: "lagou",
		}
		require.NoError(t, repo.Create(context.Background(), resume))

		got, err := repo.GetByID(context.Background(), resume.ID)
		require.NoError(t, err)
		assert.Equal(t, resume.ID, got.ID)
		assert.Equal(t, resume.Name, got.Name)
	})

	t.Run("should return error when not found", func(t *testing.T) {
		repo := NewTestRepository(t)
		defer repo.Cleanup()

		_, err := repo.GetByID(context.Background(), uuid.New())
		assert.Error(t, err)
	})
}

func TestResumeRepository_List(t *testing.T) {
	t.Run("should list resumes with pagination", func(t *testing.T) {
		repo := NewTestRepository(t)
		defer repo.Cleanup()

		// Create test data
		for i := 0; i < 15; i++ {
			err := repo.Create(context.Background(), &model.Resume{
				Name:   "User",
				Email:  "user@example.com",
				Source: "boss",
			})
			require.NoError(t, err)
		}

		resumes, total, err := repo.List(context.Background(), ListFilter{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Len(t, resumes, 10)
		assert.Equal(t, int64(15), total)
	})

	t.Run("should filter by status", func(t *testing.T) {
		repo := NewTestRepository(t)
		defer repo.Cleanup()

		// Create resumes with different statuses
		err := repo.Create(context.Background(), &model.Resume{
			Name:   "Pending User",
			Email:  "pending@example.com",
			Source: "boss",
		})
		require.NoError(t, err)

		parsed := &model.Resume{
			Name:   "Parsed User",
			Email:  "parsed@example.com",
			Source: "boss",
			Status: model.StatusParsed,
		}
		require.NoError(t, repo.Create(context.Background(), parsed))

		resumes, total, err := repo.List(context.Background(), ListFilter{
			Page:     1,
			PageSize: 10,
			Status:   model.StatusParsed,
		})
		require.NoError(t, err)
		assert.Len(t, resumes, 1)
		assert.Equal(t, int64(1), total)
	})
}

func TestResumeRepository_UpdateStatus(t *testing.T) {
	t.Run("should update status", func(t *testing.T) {
		repo := NewTestRepository(t)
		defer repo.Cleanup()

		resume := &model.Resume{
			Name:   "Status User",
			Email:  "status@example.com",
			Source: "boss",
		}
		require.NoError(t, repo.Create(context.Background(), resume))

		err := repo.UpdateStatus(context.Background(), resume.ID, model.StatusProcessing)
		require.NoError(t, err)

		got, err := repo.GetByID(context.Background(), resume.ID)
		require.NoError(t, err)
		assert.Equal(t, model.StatusProcessing, got.Status)
	})
}
```

- [ ] **Step 3: 运行测试确认失败**

```bash
go get github.com/stretchr/testify
go test ./internal/resume/repository/... -v
```

Expected: 测试失败，因为 GormRepository 未实现

- [ ] **Step 4: 实现 GORM Repository**

```go
// internal/resume/repository/gorm_repository.go
package repository

import (
	"context"
	"errors"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("resume not found")
)

// GormRepository implements ResumeRepository using GORM
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository creates a new GORM-based repository
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// Create inserts a new resume
func (r *GormRepository) Create(ctx context.Context, resume *model.Resume) error {
	return r.db.WithContext(ctx).Create(resume).Error
}

// GetByID retrieves a resume by ID
func (r *GormRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	var resume model.Resume
	err := r.db.WithContext(ctx).First(&resume, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &resume, nil
}

// List retrieves resumes with filtering and pagination
func (r *GormRepository) List(ctx context.Context, filter ListFilter) ([]model.Resume, int64, error) {
	var resumes []model.Resume
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Resume{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	if err := query.Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&resumes).Error; err != nil {
		return nil, 0, err
	}

	return resumes, total, nil
}

// Update updates an existing resume
func (r *GormRepository) Update(ctx context.Context, resume *model.Resume) error {
	return r.db.WithContext(ctx).Save(resume).Error
}

// Delete removes a resume
func (r *GormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Resume{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateStatus updates only the status field
func (r *GormRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Resume{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 5: 创建测试辅助函数**

```go
// internal/resume/repository/test_helper.go
package repository

import (
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TestRepository struct {
	*GormRepository
	db *gorm.DB
	t  *testing.T
}

func NewTestRepository(t *testing.T) *TestRepository {
	dsn := "host=localhost user=postgres password=postgres dbname=ats_test port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("database not available: %v", err)
	}

	// Auto migrate
	db.AutoMigrate(&model.Resume{})

	return &TestRepository{
		GormRepository: NewGormRepository(db),
		db:             db,
		t:              t,
	}
}

func (r *TestRepository) Cleanup() {
	r.db.Exec("TRUNCATE resumes RESTART IDENTITY CASCADE")
}
```

- [ ] **Step 6: 运行测试**

```bash
# 确保测试数据库存在
docker exec ats-postgres psql -U postgres -c "CREATE DATABASE ats_test;" 2>/dev/null || true

go test ./internal/resume/repository/... -v
```

Expected: 所有测试通过

- [ ] **Step 7: 提交**

```bash
git add internal/resume/repository/
git commit -m "feat(resume): implement repository layer with GORM

- Define ResumeRepository interface
- Implement GormRepository with CRUD operations
- Add list filtering and pagination
- Add comprehensive unit tests"
```

---

## Task 3: Service 层实现

**Files:**
- Create: `internal/resume/service/resume_service.go`
- Create: `internal/resume/service/resume_service_test.go`

- [ ] **Step 1: 定义 Service 接口**

```go
// internal/resume/service/resume_service.go
package service

import (
	"context"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/google/uuid"
)

// CreateResumeInput defines input for creating a resume
type CreateResumeInput struct {
	Name   string `json:"name" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	Phone  string `json:"phone"`
	Source string `json:"source"`
	FileURL string `json:"file_url"`
}

// UpdateResumeInput defines input for updating a resume
type UpdateResumeInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// ResumeService defines the business logic interface
type ResumeService interface {
	Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error)
	List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateResumeInput) (*model.Resume, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error)
}
```

- [ ] **Step 2: 编写 Service 测试**

```go
// internal/resume/service/resume_service_test.go
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, resume *model.Resume) error {
	args := m.Called(ctx, resume)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, filter interface{}) ([]model.Resume, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]model.Resume), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, resume *model.Resume) error {
	args := m.Called(ctx, resume)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func TestResumeService_Create(t *testing.T) {
	t.Run("should create resume successfully", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewResumeService(mockRepo)

		input := CreateResumeInput{
			Name:   "John Doe",
			Email:  "john@example.com",
			Phone:  "+1234567890",
			Source: "boss",
		}

		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		resume, err := svc.Create(context.Background(), input)

		assert.NoError(t, err)
		assert.Equal(t, input.Name, resume.Name)
		assert.Equal(t, input.Email, resume.Email)
		assert.Equal(t, model.StatusPending, resume.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewResumeService(mockRepo)

		input := CreateResumeInput{
			Name:  "John Doe",
			Email: "john@example.com",
		}

		mockRepo.On("Create", mock.Anything, mock.Anything).
			Return(errors.New("database error"))

		_, err := svc.Create(context.Background(), input)
		assert.Error(t, err)
	})
}

func TestResumeService_GetByID(t *testing.T) {
	t.Run("should return resume when found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewResumeService(mockRepo)

		id := uuid.New()
		expected := &model.Resume{
			ID:     id,
			Name:   "John Doe",
			Email:  "john@example.com",
			Status: model.StatusPending,
		}

		mockRepo.On("GetByID", mock.Anything, id).Return(expected, nil)

		got, err := svc.GetByID(context.Background(), id)

		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("should return error when not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewResumeService(mockRepo)

		id := uuid.New()
		mockRepo.On("GetByID", mock.Anything, id).
			Return(nil, errors.New("not found"))

		_, err := svc.GetByID(context.Background(), id)
		assert.Error(t, err)
	})
}

func TestResumeService_UpdateStatus(t *testing.T) {
	t.Run("should update status when valid transition", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewResumeService(mockRepo)

		id := uuid.New()
		current := &model.Resume{
			ID:     id,
			Name:   "John Doe",
			Status: model.StatusPending,
		}

		mockRepo.On("GetByID", mock.Anything, id).Return(current, nil)
		mockRepo.On("UpdateStatus", mock.Anything, id, model.StatusProcessing).Return(nil)

		got, err := svc.UpdateStatus(context.Background(), id, model.StatusProcessing)

		assert.NoError(t, err)
		assert.Equal(t, model.StatusProcessing, got.Status)
	})

	t.Run("should return error for invalid transition", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewResumeService(mockRepo)

		id := uuid.New()
		current := &model.Resume{
			ID:     id,
			Name:   "John Doe",
			Status: model.StatusArchived,
		}

		mockRepo.On("GetByID", mock.Anything, id).Return(current, nil)

		_, err := svc.UpdateStatus(context.Background(), id, model.StatusProcessing)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status transition")
	})
}
```

- [ ] **Step 3: 运行测试确认失败**

```bash
go test ./internal/resume/service/... -v
```

Expected: 测试失败

- [ ] **Step 4: 实现 Service**

```go
// internal/resume/service/resume_service.go (continued)

import (
	"errors"

	"github.com/example/ats-platform/internal/resume/repository"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrResumeNotFound          = errors.New("resume not found")
)

// resumeServiceImpl implements ResumeService
type resumeServiceImpl struct {
	repo repository.ResumeRepository
}

// NewResumeService creates a new resume service
func NewResumeService(repo repository.ResumeRepository) ResumeService {
	return &resumeServiceImpl{repo: repo}
}

func (s *resumeServiceImpl) Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error) {
	resume := &model.Resume{
		Name:    input.Name,
		Email:   input.Email,
		Phone:   input.Phone,
		Source:  input.Source,
		FileURL: input.FileURL,
		Status:  model.StatusPending,
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		return nil, err
	}

	return resume, nil
}

func (s *resumeServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *resumeServiceImpl) List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error) {
	filter := repository.ListFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Source:   source,
	}
	return s.repo.List(ctx, filter)
}

func (s *resumeServiceImpl) Update(ctx context.Context, id uuid.UUID, input UpdateResumeInput) (*model.Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		resume.Name = input.Name
	}
	if input.Email != "" {
		resume.Email = input.Email
	}
	if input.Phone != "" {
		resume.Phone = input.Phone
	}

	if err := s.repo.Update(ctx, resume); err != nil {
		return nil, err
	}

	return resume, nil
}

func (s *resumeServiceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *resumeServiceImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !resume.CanTransitionTo(status) {
		return nil, ErrInvalidStatusTransition
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}

	resume.Status = status
	return resume, nil
}
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/resume/service/... -v
```

Expected: 所有测试通过

- [ ] **Step 6: 提交**

```bash
git add internal/resume/service/
git commit -m "feat(resume): implement service layer with business logic

- Define ResumeService interface
- Implement status transition validation
- Add mock-based unit tests"
```

---

## Task 4: HTTP Handler 实现

**Files:**
- Create: `internal/resume/handler/resume_handler.go`
- Create: `internal/resume/handler/resume_handler_test.go`

- [ ] **Step 1: 编写 Handler 测试**

```go
// internal/resume/handler/resume_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/ats-platform/internal/resume/model"
	"github.com/example/ats-platform/internal/resume/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockResumeService struct {
	mock.Mock
}

func (m *MockResumeService) Create(ctx context.Context, input service.CreateResumeInput) (*model.Resume, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockResumeService) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockResumeService) List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error) {
	args := m.Called(ctx, page, pageSize, status, source)
	return args.Get(0).([]model.Resume), args.Get(1).(int64), args.Error(2)
}

func (m *MockResumeService) Update(ctx context.Context, id uuid.UUID, input service.UpdateResumeInput) (*model.Resume, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func (m *MockResumeService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockResumeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error) {
	args := m.Called(ctx, id, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Resume), args.Error(1)
}

func setupTestHandler() (*gin.Engine, *MockResumeService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(MockResumeService)
	handler := NewResumeHandler(mockSvc)

	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/resumes", handler.Create)
		api.GET("/resumes/:id", handler.GetByID)
		api.GET("/resumes", handler.List)
		api.PUT("/resumes/:id", handler.Update)
		api.DELETE("/resumes/:id", handler.Delete)
		api.PUT("/resumes/:id/status", handler.UpdateStatus)
	}

	return router, mockSvc
}

func TestResumeHandler_Create(t *testing.T) {
	t.Run("should create resume and return 200", func(t *testing.T) {
		router, mockSvc := setupTestHandler()

		input := service.CreateResumeInput{
			Name:   "John Doe",
			Email:  "john@example.com",
			Source: "boss",
		}
		body, _ := json.Marshal(input)

		expected := &model.Resume{
			ID:     uuid.New(),
			Name:   input.Name,
			Email:  input.Email,
			Source: input.Source,
			Status: model.StatusPending,
		}
		mockSvc.On("Create", mock.Anything, input).Return(expected, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/resumes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("should return 400 for invalid input", func(t *testing.T) {
		router, _ := setupTestHandler()

		body := []byte(`{"name": ""}`) // Missing required fields

		req := httptest.NewRequest(http.MethodPost, "/api/v1/resumes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResumeHandler_GetByID(t *testing.T) {
	t.Run("should return resume when found", func(t *testing.T) {
		router, mockSvc := setupTestHandler()

		id := uuid.New()
		expected := &model.Resume{
			ID:     id,
			Name:   "John Doe",
			Email:  "john@example.com",
			Status: model.StatusPending,
		}
		mockSvc.On("GetByID", mock.Anything, id).Return(expected, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/resumes/"+id.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when not found", func(t *testing.T) {
		router, mockSvc := setupTestHandler()

		id := uuid.New()
		mockSvc.On("GetByID", mock.Anything, id).Return(nil, service.ErrResumeNotFound)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/resumes/"+id.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestResumeHandler_List(t *testing.T) {
	t.Run("should return paginated list", func(t *testing.T) {
		router, mockSvc := setupTestHandler()

		resumes := []model.Resume{
			{ID: uuid.New(), Name: "User 1", Email: "user1@example.com"},
			{ID: uuid.New(), Name: "User 2", Email: "user2@example.com"},
		}
		mockSvc.On("List", mock.Anything, 1, 10, "", "").Return(resumes, int64(2), nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/resumes?page=1&page_size=10", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/resume/handler/... -v
```

Expected: 测试失败

- [ ] **Step 3: 实现 Handler**

```go
// internal/resume/handler/resume_handler.go
package handler

import (
	"net/http"

	"github.com/example/ats-platform/internal/resume/service"
	"github.com/example/ats-platform/internal/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ResumeHandler handles HTTP requests for resumes
type ResumeHandler struct {
	svc service.ResumeService
}

// NewResumeHandler creates a new resume handler
func NewResumeHandler(svc service.ResumeService) *ResumeHandler {
	return &ResumeHandler{svc: svc}
}

// Create handles POST /resumes
func (h *ResumeHandler) Create(c *gin.Context) {
	var input service.CreateResumeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resume, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		response.InternalError(c, "failed to create resume")
		return
	}

	response.Success(c, resume)
}

// GetByID handles GET /resumes/:id
func (h *ResumeHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "invalid resume ID")
		return
	}

	resume, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrResumeNotFound) {
			response.NotFound(c, "resume not found")
			return
		}
		response.InternalError(c, "failed to get resume")
		return
	}

	response.Success(c, resume)
}

// List handles GET /resumes
func (h *ResumeHandler) List(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 10)
	status := c.Query("status")
	source := c.Query("source")

	resumes, total, err := h.svc.List(c.Request.Context(), page, pageSize, status, source)
	if err != nil {
		response.InternalError(c, "failed to list resumes")
		return
	}

	response.SuccessPage(c, resumes, total, page, pageSize)
}

// Update handles PUT /resumes/:id
func (h *ResumeHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "invalid resume ID")
		return
	}

	var input service.UpdateResumeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resume, err := h.svc.Update(c.Request.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrResumeNotFound) {
			response.NotFound(c, "resume not found")
			return
		}
		response.InternalError(c, "failed to update resume")
		return
	}

	response.Success(c, resume)
}

// Delete handles DELETE /resumes/:id
func (h *ResumeHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "invalid resume ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrResumeNotFound) {
			response.NotFound(c, "resume not found")
			return
		}
		response.InternalError(c, "failed to delete resume")
		return
	}

	response.SuccessWithMessage(c, "resume deleted", nil)
}

// UpdateStatus handles PUT /resumes/:id/status
func (h *ResumeHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "invalid resume ID")
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resume, err := h.svc.UpdateStatus(c.Request.Context(), id, input.Status)
	if err != nil {
		if errors.Is(err, service.ErrResumeNotFound) {
			response.NotFound(c, "resume not found")
			return
		}
		if errors.Is(err, service.ErrInvalidStatusTransition) {
			response.BadRequest(c, "invalid status transition")
			return
		}
		response.InternalError(c, "failed to update status")
		return
	}

	response.Success(c, resume)
}

// parseIntQuery parses an integer query parameter with default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return result
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/resume/handler/... -v
```

Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add internal/resume/handler/
git commit -m "feat(resume): implement HTTP handlers with Gin

- Implement CRUD endpoints
- Add status update endpoint
- Add comprehensive handler tests with mocks"
```

---

## Task 5: 集成到服务入口

**Files:**
- Modify: `cmd/resume-service/main.go`

- [ ] **Step 1: 更新服务入口集成所有组件**

```go
// cmd/resume-service/main.go
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/ats-platform/internal/resume/handler"
	"github.com/example/ats-platform/internal/resume/repository"
	"github.com/example/ats-platform/internal/resume/service"
	"github.com/example/ats-platform/internal/shared/config"
	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/example/ats-platform/internal/shared/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:       "debug",
		Development: true,
	}); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Infof("Starting %s...", cfg.Server.Name)

	// Connect to database
	db, err := gorm.Open(postgres.Open(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)), &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Initialize dependencies
	resumeRepo := repository.NewGormRepository(db)
	resumeSvc := service.NewResumeService(resumeRepo)
	resumeHandler := handler.NewResumeHandler(resumeSvc)

	// Setup Gin router
	gin.SetMode(gin.DebugMode)
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.Logging())
	router.Use(middleware.CORS())

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": cfg.Server.Name,
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		// Check database connection
		if sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "not ready",
				"service": cfg.Server.Name,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": cfg.Server.Name,
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/resumes", resumeHandler.Create)
		api.GET("/resumes/:id", resumeHandler.GetByID)
		api.GET("/resumes", resumeHandler.List)
		api.PUT("/resumes/:id", resumeHandler.Update)
		api.DELETE("/resumes/:id", resumeHandler.Delete)
		api.PUT("/resumes/:id/status", resumeHandler.UpdateStatus)
	}

	// Start HTTP server
	httpAddr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	go func() {
		logger.Infof("HTTP server listening on %s", httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start gRPC server (placeholder)
	grpcAddr := fmt.Sprintf(":%d", cfg.Server.GRPCPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatalf("Failed to listen on gRPC port: %v", err)
	}
	logger.Infof("gRPC server listening on %s", grpcAddr)
	// TODO: Register gRPC server

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown error: %v", err)
	}

	grpcListener.Close()
	sqlDB.Close()

	logger.Info("Server stopped")
}
```

- [ ] **Step 2: 添加依赖并验证编译**

```bash
go get gorm.io/gorm gorm.io/driver/postgres
go build ./cmd/resume-service/...
```

Expected: 编译成功

- [ ] **Step 3: 启动基础设施进行集成测试**

```bash
make infra-up
sleep 10
go run ./cmd/resume-service &
```

- [ ] **Step 4: 测试 API 端点**

```bash
# 健康检查
curl -s http://localhost:8081/health

# 创建简历
curl -s -X POST http://localhost:8081/api/v1/resumes \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","source":"boss"}'

# 获取简历列表
curl -s http://localhost:8081/api/v1/resumes

# 更新状态
curl -s -X PUT http://localhost:8081/api/v1/resumes/{id}/status \
  -H "Content-Type: application/json" \
  -d '{"status":"processing"}'
```

Expected: 所有 API 返回正确响应

- [ ] **Step 5: 提交**

```bash
git add cmd/resume-service/main.go go.mod go.sum
git commit -m "feat(resume): integrate all components in service entry point

- Connect to PostgreSQL with GORM
- Wire up repository, service, and handler
- Add database health check
- Register API routes"
```

---

## Task 6: 最终验证

- [ ] **Step 1: 运行所有测试**

```bash
go test ./... -v -cover
```

- [ ] **Step 2: 确保测试覆盖率**

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1
```

Expected: 覆盖率 > 70%

- [ ] **Step 3: 最终提交**

```bash
git add .
git commit -m "chore: finalize resume service implementation

- All tests passing
- Coverage > 70%
- Ready for gRPC and search integration"
```

---

## Summary

完成此计划后，Resume Service 将具备：

| 功能 | 状态 |
|------|------|
| 数据模型 | ✅ Resume 模型 + 状态转换 |
| Repository | ✅ GORM 实现 + CRUD |
| Service | ✅ 业务逻辑 + 验证 |
| HTTP Handler | ✅ REST API 端点 |
| 健康检查 | ✅ /health, /ready |
| 测试覆盖 | ✅ 单元测试 + 集成测试 |

**下一步：**
- Plan 3: Search Service (ES 索引 + 数据同步)
- gRPC 服务实现
- 简历解析功能
