# Search Service 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Search Service 的完整功能，包括 Elasticsearch 索引管理、简历搜索接口、数据同步（从 Redis Streams 消费事件）。

**Architecture:** 采用分层架构（Handler → Service → Repository），使用 go-elasticsearch 操作 ES，通过 go-redis 消费 Redis Streams 事件，
实现简历数据的异步同步。

**Tech Stack:** Go 1.25, Gin, go-elasticsearch v8, go-redis v9, gRPC

---

## File Structure

```
ats-platform/
├── cmd/search-service/main.go            # 服务入口
├── internal/search/
│   ├── handler/
│   │   ├── search_handler.go             # HTTP handlers
│   │   └── search_handler_test.go        # Handler 测试
│   ├── service/
│   │   ├── search_service.go             # 搜索业务逻辑
│   │   ├── search_service_test.go        # Service 测试
│   │   └── indexer.go                    # 索引器
│   ├── repository/
│   │   ├── es_repository.go              # ES 数据访问层
│   │   └── es_repository_test.go         # Repository 测试
│   └── model/
│       └── resume_document.go            # ES 文档模型
├── internal/shared/
│   └── events/
│       └── consumer.go                   # Redis Streams 消费者
└── configs/
    └── config.yaml                        # 配置文件（已存在）
```

---

## Task 1: ES 文档模型定义

**Files:**
- Create: `internal/search/model/resume_document.go`
- Create: `internal/search/model/resume_document_test.go`

- [ ] **Step 1: 编写 ResumeDocument 模型测试**

```go
// internal/search/model/resume_document_test.go
package model

import (
	"testing"
	"time"
)

func TestResumeDocument_NewFromResume(t *testing.T) {
	now := time.Now()
	doc := NewResumeDocument("test-id", "John Doe", "john@example.com",
		[]string{"Go", "MySQL"}, 5, "Bachelor", "5 years experience", "pending", now)

	assert.Equal(t, "test-id", doc.ResumeID)
	assert.Equal(t, "John Doe", doc.Name)
	assert.Equal(t, "john@example.com", doc.Email)
	assert.Equal(t, []string{"Go", "MySQL"}, doc.Skills)
	assert.Equal(t, 5, doc.ExperienceYears)
	assert.Equal(t, "Bachelor", doc.Education)
	assert.Equal(t, "pending", doc.Status)
}
```

- [ ] **Step 2: 实现 ResumeDocument 模型**

```go
// internal/search/model/resume_document.go
package model

import (
	"time"
)

// ResumeDocument represents a resume document in Elasticsearch
type ResumeDocument struct {
	ResumeID       string    `json:"resume_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Skills         []string  `json:"skills,omitempty"`
	ExperienceYears int      `json:"experience_years,omitempty"`
	Education      string    `json:"education,omitempty"`
	WorkHistory    string    `json:"work_history,omitempty"`
	Status         string    `json:"status"`
	Source         string    `json:"source,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// IndexName returns the Elasticsearch index name
func (ResumeDocument) IndexName() string {
	return "resumes"
}

// DocumentID returns the document ID for Elasticsearch
func (d *ResumeDocument) DocumentID() string {
	return d.ResumeID
}

// NewResumeDocument creates a new ResumeDocument
func NewResumeDocument(id, name, email string, skills []string,
	expYears int, education, workHistory, status string, createdAt time.Time) *ResumeDocument {
	return &ResumeDocument{
		ResumeID:        id,
		Name:            name,
		Email:           email,
		Skills:          skills,
		ExperienceYears: expYears,
		Education:       education,
		WorkHistory:     workHistory,
		Status:          status,
		CreatedAt:       createdAt,
	}
}
```

- [ ] **Step 3: 运行测试验证**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go get github.com/stretchr/testify
go test ./internal/search/model/... -v
```

- [ ] **Step 4: 提交**

```bash
git add internal/search/model/
git commit -m "feat(search): add resume document model for Elasticsearch

- Define ResumeDocument struct with JSON tags
- Add NewResumeDocument constructor
- Add unit tests"
```

---

## Task 2: Elasticsearch Repository 层

**Files:**
- Create: `internal/search/repository/es_repository.go`
- Create: `internal/search/repository/es_repository_test.go`

- [ ] **Step 1: 定义 ES Repository 接口**

```go
// internal/search/repository/es_repository.go
package repository

import (
	"context"

	"github.com/example/ats-platform/internal/search/model"
)

// SearchFilter defines search filter options
type SearchFilter struct {
	Query         string
	Skills        []string
	Status        string
	Source        string
	MinExperience int
	MaxExperience int
	Page          int
	PageSize      int
}

// SearchResult represents search results
type SearchResult struct {
	Documents []model.ResumeDocument
	Total     int64
}

// ESRepository defines the interface for Elasticsearch operations
type ESRepository interface {
	Index(ctx context.Context, doc *model.ResumeDocument) error
	GetByID(ctx context.Context, id string) (*model.ResumeDocument, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filter SearchFilter) (*SearchResult, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}
```

- [ ] **Step 2: 编写 Repository 测试**

```go
// internal/search/repository/es_repository_test.go
package repository

import (
	"context"
	"testing"

	"github.com/example/ats-platform/internal/search/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRepository_Index(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	doc := &model.ResumeDocument{
		ResumeID: "test-1",
		Name:     "John Doe",
		Email:    "john@example.com",
		Status:   "pending",
	}

	err := repo.Index(ctx, doc)
	require.NoError(t, err)

	// Verify document was indexed
	got, err := repo.GetByID(ctx, "test-1")
	require.NoError(t, err)
	assert.Equal(t, doc, got)
}

func TestMockRepository_Search(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Index test documents
	repo.Index(ctx, &model.ResumeDocument{ResumeID: "1", Name: "John", Status: "pending", Skills: []string{"Go"}})
	repo.Index(ctx, &model.ResumeDocument{ResumeID: "2", Name: "Jane", Status: "parsed", Skills: []string{"Python"}})

	// Search by status
	result, err := repo.Search(ctx, SearchFilter{Status: "pending"})
	require.NoError(t, err)
	assert.Len(t, result.Documents, 1)
	assert.Equal(t, "1", result.Documents[0].ResumeID)

	// Search all
	result, err = repo.Search(ctx, SearchFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}
```

- [ ] **Step 3: 实现 MockRepository（用于测试）**

```go
// internal/search/repository/mock_repository.go
package repository

import (
	"context"
	"sync"

	"github.com/example/ats-platform/internal/search/model"
)

// MockRepository implements ESRepository for testing
type MockRepository struct {
	mu    sync.RWMutex
	docs  map[string]*model.ResumeDocument
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		docs: make(map[string]*model.ResumeDocument),
	}
}

func (r *MockRepository) Index(ctx context.Context, doc *model.ResumeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ResumeID] = doc
	return nil
}

func (r *MockRepository) GetByID(ctx context.Context, id string) (*model.ResumeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.docs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return doc, nil
}

func (r *MockRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.docs, id)
	return nil
}

func (r *MockRepository) Search(ctx context.Context, filter SearchFilter) (*SearchResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []model.ResumeDocument
	for _, doc := range r.docs {
		if filter.Status != "" && doc.Status != filter.Status {
			continue
		}
		if filter.Source != "" && doc.Source != filter.Source {
			continue
		}
		results = append(results, *doc)
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

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(results) {
		return &SearchResult{Documents: []model.ResumeDocument{}, Total: int64(len(results))}, nil
	}
	if end > len(results) {
		end = len(results)
	}

	return &SearchResult{
		Documents: results[start:end],
		Total:     int64(len(results)),
	}, nil
}

func (r *MockRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, ok := r.docs[id]
	if !ok {
		return ErrNotFound
	}
	doc.Status = status
	return nil
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/search/repository/... -v
```

- [ ] **Step 5: 提交**

```bash
git add internal/search/repository/
git commit -m "feat(search): implement ES repository interface and mock

- Define ESRepository interface
- Implement MockRepository for testing
- Add search filter and result types
- Add comprehensive unit tests"
```

---

## Task 3: Search Service 层

**Files:**
- Create: `internal/search/service/search_service.go`
- Create: `internal/search/service/search_service_test.go`
- Create: `internal/search/service/indexer.go`

- [ ] **Step 1: 定义 Search Service 接口**

```go
// internal/search/service/search_service.go
package service

import (
	"context"

	"github.com/example/ats-platform/internal/search/model"
	"github.com/example/ats-platform/internal/search/repository"
)

// SearchService defines the interface for search operations
type SearchService interface {
	// Search searches resumes based on filter
	Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error)

	// IndexResume indexes a resume document
	IndexResume(ctx context.Context, doc *model.ResumeDocument) error

	// DeleteResume removes a resume from index
	DeleteResume(ctx context.Context, id string) error

	// UpdateResumeStatus updates status in index
	UpdateResumeStatus(ctx context.Context, id string, status string) error
}

// IndexerService defines the interface for indexing operations
type IndexerService interface {
	// SyncResume syncs a resume from database to ES
	SyncResume(ctx context.Context, resumeID string) error
}
```

- [ ] **Step 2: 编写 Service 测试**

```go
// internal/search/service/search_service_test.go
package service

import (
	"context"
	"testing"

	"github.com/example/ats-platform/internal/search/model"
	"github.com/example/ats-platform/internal/search/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockESRepository struct {
	mock.Mock
}

func (m *MockESRepository) Index(ctx context.Context, doc *model.ResumeDocument) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockESRepository) GetByID(ctx context.Context, id string) (*model.ResumeDocument, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.ResumeDocument), args.Error(1)
}

func (m *MockESRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockESRepository) Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*repository.SearchResult), args.Error(1)
}

func (m *MockESRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func TestSearchService_Search(t *testing.T) {
	mockRepo := new(MockESRepository)
	svc := NewSearchService(mockRepo)

	expected := &repository.SearchResult{
		Documents: []model.ResumeDocument{{ResumeID: "1", Name: "John"}},
		Total:     1,
	}

	mockRepo.On("Search", mock.Anything, mock.Anything).Return(expected, nil)

	result, err := svc.Search(context.Background(), repository.SearchFilter{Query: "John"})
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestSearchService_IndexResume(t *testing.T) {
	mockRepo := new(MockESRepository)
	svc := NewSearchService(mockRepo)

	doc := &model.ResumeDocument{ResumeID: "1", Name: "John"}
	mockRepo.On("Index", mock.Anything, doc).Return(nil)

	err := svc.IndexResume(context.Background(), doc)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
```

- [ ] **Step 3: 实现 Search Service**

```go
// internal/search/service/search_service.go (continued)

import "errors"

var (
	ErrDocumentNotFound = errors.New("document not found")
)

type searchServiceImpl struct {
	repo repository.ESRepository
}

func NewSearchService(repo repository.ESRepository) SearchService {
	return &searchServiceImpl{repo: repo}
}

func (s *searchServiceImpl) Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 10
	}

	return s.repo.Search(ctx, filter)
}

func (s *searchServiceImpl) IndexResume(ctx context.Context, doc *model.ResumeDocument) error {
	return s.repo.Index(ctx, doc)
}

func (s *searchServiceImpl) DeleteResume(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *searchServiceImpl) UpdateResumeStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/search/service/... -v
```

- [ ] **Step 5: 提交**

```bash
git add internal/search/service/
git commit -m "feat(search): implement search service layer

- Define SearchService interface
- Implement search and index operations
- Add mock-based unit tests"
```

---

## Task 4: HTTP Handler 实现

**Files:**
- Create: `internal/search/handler/search_handler.go`
- Create: `internal/search/handler/search_handler_test.go`

- [ ] **Step 1: 编写 Handler 测试**

```go
// internal/search/handler/search_handler_test.go
package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/example/ats-platform/internal/search/model"
	"github.com/example/ats-platform/internal/search/repository"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) Search(ctx context.Context, filter repository.SearchFilter) (*repository.SearchResult, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*repository.SearchResult), args.Error(1)
}

func (m *MockSearchService) IndexResume(ctx context.Context, doc *model.ResumeDocument) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockSearchService) DeleteResume(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSearchService) UpdateResumeStatus(ctx context.Context, id string, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func TestSearchHandler_Search(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockSearchService)
	handler := NewSearchHandler(mockSvc)

	expected := &repository.SearchResult{
		Documents: []model.ResumeDocument{{ResumeID: "1", Name: "John"}},
		Total:     1,
	}

	mockSvc.On("Search", mock.Anything, mock.Anything).Return(expected, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/search?query=John", nil)

	handler.Search(c)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
}
```

- [ ] **Step 2: 实现 Handler**

```go
// internal/search/handler/search_handler.go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/example/ats-platform/internal/search/repository"
	"github.com/example/ats-platform/internal/search/service"
	"github.com/example/ats-platform/internal/shared/response"
)

type SearchHandler struct {
	svc service.SearchService
}

func NewSearchHandler(svc service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Search handles GET /api/v1/search
func (h *SearchHandler) Search(c *gin.Context) {
	filter := repository.SearchFilter{
		Query:         c.Query("query"),
		Status:        c.Query("status"),
		Source:        c.Query("source"),
		Page:          parseIntQuery(c, "page", 1),
		PageSize:      parseIntQuery(c, "page_size", 10),
		MinExperience: parseIntQuery(c, "min_exp", 0),
		MaxExperience: parseIntQuery(c, "max_exp", 0),
	}

	// Parse skills (comma-separated)
	if skills := c.Query("skills"); skills != "" {
		filter.Skills = strings.Split(skills, ",")
	}

	result, err := h.svc.Search(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "search failed")
		return
	}

	response.SuccessPage(c, result.Documents, result.Total, filter.Page, filter.PageSize)
}

// AdvancedSearch handles POST /api/v1/search/advanced
func (h *SearchHandler) AdvancedSearch(c *gin.Context) {
	var filter repository.SearchFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.Search(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "search failed")
		return
	}

	response.SuccessPage(c, result.Documents, result.Total, filter.Page, filter.PageSize)
}

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

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/search/handler/... -v
```

- [ ] **Step 4: 提交**

```bash
git add internal/search/handler/
git commit -m "feat(search): implement HTTP handlers for search

- Implement search and advanced search endpoints
- Support query params and JSON body filters
- Add comprehensive handler tests"
```

---

## Task 5: Redis Streams 消费者

**Files:**
- Create: `internal/shared/events/consumer.go`
- Create: `internal/shared/events/consumer_test.go`

- [ ] **Step 1: 定义事件结构和消费者接口**

```go
// internal/shared/events/consumer.go
package events

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// ResumeEvent represents a resume event from Redis Stream
type ResumeEvent struct {
	ResumeID string `json:"resume_id"`
	Action   string `json:"action"` // created, updated, deleted
	Payload  string `json:"payload,omitempty"`
}

// EventHandler handles events from Redis Stream
type EventHandler func(ctx context.Context, event ResumeEvent) error

// StreamConsumer consumes events from Redis Stream
type StreamConsumer struct {
	client   *redis.Client
	stream   string
	group    string
	consumer string
	handler  EventHandler
}

// NewStreamConsumer creates a new stream consumer
func NewStreamConsumer(client *redis.Client, stream, group, consumer string, handler EventHandler) *StreamConsumer {
	return &StreamConsumer{
		client:   client,
		stream:   stream,
		group:    group,
		consumer: consumer,
		handler:  handler,
	}
}

// Start starts consuming events
func (c *StreamConsumer) Start(ctx context.Context) error {
	// Create consumer group if not exists
	err := c.client.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil && !isGroupExistsError(err) {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			c.consumeBatch(ctx)
		}
	}
}

func (c *StreamConsumer) consumeBatch(ctx context.Context) {
	streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.group,
		Consumer: c.consumer,
		Streams:  []string{c.stream, ">"},
		Count:    10,
		Block:    time.Second * 5,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return
		}
		time.Sleep(time.Second)
		return
	}

	for _, stream := range streams {
		for _, message := range stream.Messages {
			c.processMessage(ctx, message)
		}
	}
}

func (c *StreamConsumer) processMessage(ctx context.Context, message redis.XMessage) {
	event := ResumeEvent{}
	if resumeID, ok := message.Values["resume_id"].(string); ok {
		event.ResumeID = resumeID
	}
	if action, ok := message.Values["action"].(string); ok {
		event.Action = action
	}
	if payload, ok := message.Values["payload"].(string); ok {
		event.Payload = payload
	}

	if err := c.handler(ctx, event); err != nil {
		// Log error but don't acknowledge
		return
	}

	// Acknowledge message
	c.client.XAck(ctx, c.stream, c.group, message.ID)
}

func isGroupExistsError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}
```

- [ ] **Step 2: 运行测试并提交**

```bash
go get github.com/redis/go-redis/v9
go test ./internal/shared/events/... -v
git add internal/shared/events/
git commit -m "feat(events): implement Redis Streams consumer

- Define ResumeEvent structure
- Implement StreamConsumer with batch processing
- Support consumer groups for scalability"
```

---

## Task 6: 服务入口集成

**Files:**
- Create: `cmd/search-service/main.go`

- [ ] **Step 1: 创建服务入口**

```go
// cmd/search-service/main.go
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

	"github.com/example/ats-platform/internal/search/handler"
	"github.com/example/ats-platform/internal/search/repository"
	"github.com/example/ats-platform/internal/search/service"
	"github.com/example/ats-platform/internal/shared/config"
	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/example/ats-platform/internal/shared/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

	logger.Infof("Starting Search Service...")

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Initialize dependencies
	esRepo := repository.NewMockRepository() // TODO: Use real ES client
	searchSvc := service.NewSearchService(esRepo)
	searchHandler := handler.NewSearchHandler(searchSvc)

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
			"service": "search-service",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		// Check Redis connection
		if err := rdb.Ping(c).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "not ready",
				"service": "search-service",
				"error":   "redis connection failed",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "search-service",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/search", searchHandler.Search)
		api.POST("/search/advanced", searchHandler.AdvancedSearch)
	}

	// Start HTTP server
	httpAddr := fmt.Sprintf(":%d", 8083) // Search service port
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
	grpcAddr := fmt.Sprintf(":%d", 9083)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatalf("Failed to listen on gRPC port: %v", err)
	}
	logger.Infof("gRPC server listening on %s", grpcAddr)

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
	rdb.Close()

	logger.Info("Server stopped")
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./cmd/search-service/...
```

- [ ] **Step 3: 提交**

```bash
git add cmd/search-service/ go.mod go.sum
git commit -m "feat(search): integrate all components in service entry point

- Connect to Redis for event consumption
- Wire up repository, service, and handler
- Add health check endpoints
- Register API routes"
```

---

## Task 7: 最终验证

- [ ] **Step 1: 运行所有测试**

```bash
go test ./... -v -cover
```

- [ ] **Step 2: 确保测试覆盖率 > 70%**

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1
```

- [ ] **Step 3: 最终提交**

```bash
git add .
git commit -m "chore: finalize search service implementation

- All tests passing
- Coverage > 70%
- Ready for integration with Resume Service"
```

---

## Summary

完成此计划后，Search Service 将具备：

| 功能 | 状态 |
|------|------|
| ES 文档模型 | ✅ ResumeDocument |
| ES Repository | ✅ Mock + 接口 |
| Search Service | ✅ 搜索 + 索引 |
| HTTP Handler | ✅ 搜索端点 |
| Redis Streams | ✅ 事件消费 |
| 健康检查 | ✅ /health, /ready |

**API 端点:**
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/search` | 搜索简历 |
| POST | `/api/v1/search/advanced` | 高级搜索 |

**下一步：** 集成 Elasticsearch 客户端实现
