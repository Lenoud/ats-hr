# ATS Platform 服务开发指南

> 基于 `resume-service` 实际实现总结的开发规范，供 `interview-service`、`search-service`、`gateway` 参考遵循。

## 目录

- [架构概述](#架构概述)
- [目录结构规范](#目录结构规范)
- [分层架构详解](#分层架构详解)
- [共享模块使用](#共享模块使用)
- [代码模板](#代码模板)
- [命名约定](#命名约定)
- [最佳实践](#最佳实践)

---

## 架构概述

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         main.go                              │
│  (依赖注入、路由配置、中间件、HTTP/gRPC 服务启动)              │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                      Handler Layer                           │
│  (HTTP 请求处理、参数验证、响应格式化)                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                      Service Layer                           │
│  (业务逻辑、领域规则、事务协调、事件发布)                       │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                     Repository Layer                         │
│  (数据持久化、查询封装、SQL/ORM 操作)                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
   ┌─────────┐      ┌─────────┐      ┌─────────┐
   │PostgreSQL│      │  MinIO  │      │  Redis  │
   └─────────┘      └─────────┘      └─────────┘
```

### 协议支持

平台默认优先提供 **HTTP REST**。是否同时提供 **gRPC** 取决于服务职责和当前实现阶段：

| 协议 | 端口 | 用途 |
|------|------|------|
| HTTP | 808X | 对外 API、Web 前端调用 |
| gRPC | 909X | 服务间同步调用 |

当前仓库内的实际状态：

- `resume-service`：HTTP + gRPC
- `interview-service`：HTTP + gRPC
- `search-service`：HTTP，暂未提供 gRPC 入口
- `gateway`：HTTP，仅做首页、健康检查和反向代理

---

## 目录结构规范

### 服务目录结构

```
ats-platform/
├── cmd/
│   └── {service-name}/           # 服务入口
│       ├── main.go               # 主程序
│       └── static/               # 静态文件 (可选)
│
├── internal/
│   ├── {domain}/                 # 领域模块 (resume/interview/search)
│   │   ├── handler/              # HTTP 处理器
│   │   │   └── {name}_handler.go
│   │   ├── service/              # 业务逻辑
│   │   │   ├── {name}_service.go
│   │   │   └── {submodule}.go    # 子模块 (如 parser.go)
│   │   ├── repository/           # 数据访问
│   │   │   └── {name}_repository.go
│   │   ├── model/                # 数据模型
│   │   │   └── {name}.go
│   │   └── grpc/                 # gRPC 服务端 (按需提供)
│   │       └── server.go
│   │
│   └── shared/                   # 共享模块
│       ├── database/             # PostgreSQL/Elasticsearch 连接
│       ├── events/               # 事件发布/消费
│       ├── llm/                  # LLM 客户端
│       ├── logger/               # 日志
│       ├── middleware/           # HTTP 中间件
│       ├── pb/                   # Protobuf 生成代码
│       ├── response/             # 统一响应
│       └── storage/              # 文件存储
│
└── proto/                        # Proto 定义文件
    └── {domain}.proto
```

### 文件命名规则

| 类型 | 命名格式 | 示例 |
|------|----------|------|
| Handler | `{entity}_handler.go` | `resume_handler.go` |
| Service | `{entity}_service.go` | `resume_service.go` |
| Repository | `{entity}_repository.go` | `resume_repository.go` |
| Model | `{entity}.go` | `resume.go` |
| gRPC | `server.go` | `server.go` |

---

## 分层架构详解

### 1. Model 层 (数据模型)

**职责**: 定义数据结构和常量

**示例**: `internal/resume/model/resume.go`

```go
package model

import (
    "slices"
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// 状态常量
const (
    StatusPending    = "pending"
    StatusProcessing = "processing"
    StatusParsed     = "parsed"
    StatusFailed     = "failed"
    StatusArchived   = "archived"
)

// 状态转换规则
var validStatusTransitions = map[string][]string{
    StatusPending:    {StatusProcessing, StatusArchived},
    StatusProcessing: {StatusParsed, StatusFailed},
    StatusParsed:     {StatusArchived},
    StatusFailed:     {StatusPending, StatusArchived},
    StatusArchived:   {},
}

// Resume 数据模型
type Resume struct {
    ID         uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
    Name       string         `json:"name" gorm:"type:varchar(255);not null"`
    Email      string         `json:"email" gorm:"type:varchar(255)"`
    Phone      string         `json:"phone" gorm:"type:varchar(50)"`
    Source     string         `json:"source" gorm:"type:varchar(100)"`
    FileURL    string         `json:"file_url" gorm:"type:text"`
    ParsedData map[string]any `json:"parsed_data" gorm:"type:jsonb"`
    Status     string         `json:"status" gorm:"type:varchar(50);default:pending"`
    CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt  time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
    DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName 指定表名
func (Resume) TableName() string {
    return "resumes"
}

// CanTransitionTo 验证状态转换是否合法
func (r *Resume) CanTransitionTo(newStatus string) bool {
    allowed, exists := validStatusTransitions[r.Status]
    if !exists {
        return false
    }
    return slices.Contains(allowed, newStatus)
}
```

**要点**:
- 使用 `uuid.UUID` 作为主键
- JSON 字段使用 `map[string]any` + GORM `jsonb` 类型
- 状态机逻辑封装在模型方法中
- 软删除使用 `gorm.DeletedAt`

---

### 2. Repository 层 (数据访问)

**职责**: 封装数据库操作，提供清晰的接口

**接口定义**:

```go
package repository

import (
    "context"
    "github.com/google/uuid"
    "github.com/example/ats-platform/internal/resume/model"
)

var ErrNotFound = errors.New("record not found")

type ListFilter struct {
    Page     int
    PageSize int
    Status   string
    Source   string
}

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
```

**实现要点**:

```go
type gormRepository struct {
    db *gorm.DB
}

func NewGormRepository(db *gorm.DB) ResumeRepository {
    return &gormRepository{db: db}
}

// Create - 使用原生 SQL 处理 JSONB
func (r *gormRepository) Create(ctx context.Context, resume *model.Resume) error {
    parsedDataJSON, _ := json.Marshal(resume.ParsedData)
    if len(parsedDataJSON) == 0 {
        parsedDataJSON = []byte("{}")
    }

    query := `INSERT INTO resumes (id, name, email, phone, source, file_url, parsed_data, status, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

    return r.db.WithContext(ctx).Exec(query,
        resume.ID.String(),
        resume.Name,
        resume.Email,
        // ...其他字段
    ).Error
}

// UpdateStatusIf - 原子状态更新，防止并发问题
func (r *gormRepository) UpdateStatusIf(ctx context.Context, id uuid.UUID, newStatus string, expectedStatuses []string) (bool, error) {
    query := `UPDATE resumes SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status IN ?`
    result := r.db.WithContext(ctx).Exec(query, newStatus, id, expectedStatuses)
    if result.Error != nil {
        return false, result.Error
    }
    return result.RowsAffected > 0, nil
}
```

**要点**:
- 使用接口定义，便于测试和替换实现
- JSONB 字段使用原生 SQL 处理
- 关键操作提供原子性保证 (如 `UpdateStatusIf`)
- 返回自定义错误 (`ErrNotFound`) 而非 GORM 错误

---

### 3. Service 层 (业务逻辑)

**职责**: 实现业务逻辑、协调多个 Repository、发布事件

**接口定义**:

```go
package service

import (
    "context"
    "io"
    "github.com/google/uuid"
    "github.com/example/ats-platform/internal/resume/model"
)

var (
    ErrInvalidStatusTransition = errors.New("invalid status transition")
    ErrResumeNotFound          = errors.New("resume not found")
    ErrInvalidFileType         = errors.New("invalid file type")
)

type CreateResumeInput struct {
    Name    string `json:"name" binding:"required"`
    Email   string `json:"email" binding:"required,email"`
    Phone   string `json:"phone"`
    Source  string `json:"source"`
    FileURL string `json:"file_url"`
}

type ResumeService interface {
    Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error)
    List(ctx context.Context, page, pageSize int, status, source string) ([]model.Resume, int64, error)
    Update(ctx context.Context, id uuid.UUID, input UpdateResumeInput) (*model.Resume, error)
    Delete(ctx context.Context, id uuid.UUID) error
    UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error)
    UploadFile(ctx context.Context, id uuid.UUID, filename string, reader io.Reader, size int64) (*model.Resume, error)
    ParseResume(ctx context.Context, id uuid.UUID) (*ParsedResume, error)
}
```

**实现要点**:

```go
type resumeService struct {
    repo      repository.ResumeRepository
    storage   storage.FileStorage
    parser    ResumeParser
    publisher *events.EventPublisher
}

func NewResumeService(
    repo repository.ResumeRepository,
    storage storage.FileStorage,
    publisher *events.EventPublisher,
) ResumeService {
    return &resumeService{
        repo:      repo,
        storage:   storage,
        parser:    NewResumeParser(),
        publisher: publisher,
    }
}

func (s *resumeService) Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error) {
    resume := &model.Resume{
        ID:     uuid.New(),
        Name:   input.Name,
        Email:  input.Email,
        Phone:  input.Phone,
        Source: input.Source,
        Status: model.StatusPending,
    }

    if err := s.repo.Create(ctx, resume); err != nil {
        return nil, err
    }

    // 发布事件
    if s.publisher != nil {
        _ = s.publisher.PublishCreated(ctx, resume.ID.String(), resume)
    }

    return resume, nil
}

func (s *resumeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error) {
    resume, err := s.repo.GetByID(ctx, id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, ErrResumeNotFound
        }
        return nil, err
    }

    // 验证状态转换
    if !resume.CanTransitionTo(status) {
        return nil, ErrInvalidStatusTransition
    }

    oldStatus := resume.Status
    if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
        return nil, err
    }

    // 重新获取更新后的数据
    updatedResume, _ := s.repo.GetByID(ctx, id)

    // 发布状态变更事件
    if s.publisher != nil {
        _ = s.publisher.PublishStatusChanged(ctx, id.String(), oldStatus, status)
    }

    return updatedResume, nil
}
```

**要点**:
- 使用 Input 结构体接收参数，便于验证
- 错误转换为业务错误 (如 `ErrResumeNotFound`)
- 业务操作后发布领域事件
- 依赖注入所有外部依赖

---

### 4. Handler 层 (HTTP 处理)

**职责**: 处理 HTTP 请求、参数绑定、调用 Service、格式化响应

```go
package handler

import (
    "strconv"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/example/ats-platform/internal/resume/service"
    "github.com/example/ats-platform/internal/shared/response"
)

type ResumeHandler struct {
    svc service.ResumeService
}

func NewResumeHandler(svc service.ResumeService) *ResumeHandler {
    return &ResumeHandler{svc: svc}
}

// Create POST /api/v1/resumes
func (h *ResumeHandler) Create(c *gin.Context) {
    var input service.CreateResumeInput
    if err := c.ShouldBindJSON(&input); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    resume, err := h.svc.Create(c.Request.Context(), input)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }

    response.Success(c, resume)
}

// List GET /api/v1/resumes
func (h *ResumeHandler) List(c *gin.Context) {
    page := parseIntQuery(c, "page", 1)
    pageSize := parseIntQuery(c, "page_size", 10)
    status := c.Query("status")
    source := c.Query("source")

    resumes, total, err := h.svc.List(c.Request.Context(), page, pageSize, status, source)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }

    response.SuccessPage(c, resumes, total, page, pageSize)
}

// GetByID GET /api/v1/resumes/:id
func (h *ResumeHandler) GetByID(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        response.BadRequest(c, "invalid resume id")
        return
    }

    resume, err := h.svc.GetByID(c.Request.Context(), id)
    if err != nil {
        if err == service.ErrResumeNotFound {
            response.NotFound(c, "resume not found")
            return
        }
        response.InternalError(c, err.Error())
        return
    }

    response.Success(c, resume)
}

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
    value := c.Query(key)
    if value == "" {
        return defaultValue
    }
    intValue, err := strconv.Atoi(value)
    if err != nil {
        return defaultValue
    }
    return intValue
}
```

**要点**:
- 使用统一的 `response` 包处理响应
- 参数验证失败返回 `BadRequest`
- 业务错误转换为适当的 HTTP 状态码
- Handler 只做请求处理，不包含业务逻辑

---

### 5. gRPC Server

**职责**: 实现 gRPC 服务接口

```go
package grpc

import (
    "context"
    "encoding/json"
    "github.com/google/uuid"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    pb "github.com/example/ats-platform/internal/shared/pb/resume"
    "github.com/example/ats-platform/internal/resume/service"
    "github.com/example/ats-platform/internal/resume/model"
)

type ResumeServiceServer struct {
    pb.UnimplementedResumeServiceServer
    svc service.ResumeService
}

func NewResumeServiceServer(svc service.ResumeService) *ResumeServiceServer {
    return &ResumeServiceServer{svc: svc}
}

func (s *ResumeServiceServer) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.Resume, error) {
    id, err := uuid.Parse(req.GetId())
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
    }

    resume, err := s.svc.GetByID(ctx, id)
    if err != nil {
        if err == service.ErrResumeNotFound {
            return nil, status.Errorf(codes.NotFound, "resume not found")
        }
        return nil, status.Errorf(codes.Internal, "get resume failed: %v", err)
    }

    return toProto(resume), nil
}

func toProto(r *model.Resume) *pb.Resume {
    var parsedData []byte
    if r.ParsedData != nil {
        parsedData, _ = json.Marshal(r.ParsedData)
    }
    return &pb.Resume{
        Id:         r.ID.String(),
        Name:       r.Name,
        Email:      r.Email,
        Status:     r.Status,
        ParsedData: parsedData,
        CreatedAt:  r.CreatedAt.Unix(),
        UpdatedAt:  r.UpdatedAt.Unix(),
    }
}
```

---

### 6. main.go (服务入口)

**职责**: 依赖注入、路由配置、启动服务

```go
package main

import (
    "context"
    "net"

    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"

    grpcHandler "github.com/example/ats-platform/internal/resume/grpc"
    "github.com/example/ats-platform/internal/resume/handler"
    "github.com/example/ats-platform/internal/resume/repository"
    "github.com/example/ats-platform/internal/resume/service"
    "github.com/example/ats-platform/internal/shared/database"
    "github.com/example/ats-platform/internal/shared/events"
    "github.com/example/ats-platform/internal/shared/llm"
    "github.com/example/ats-platform/internal/shared/middleware"
    "github.com/example/ats-platform/internal/shared/storage"
)

func main() {
    ctx := context.Background()
    cfg := loadConfig()

    // 1. 初始化基础设施
    postgresClient, _ := database.NewPostgresClient(database.PostgresConfig{...})
    minioStorage, _ := storage.NewMinIOClient(storage.MinIOConfig{...})
    redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
    llmClient := llm.NewClient(llm.Config{...})

    // 2. 初始化事件发布器
    publisher := events.NewEventPublisher(redisClient, cfg.RedisStream)

    // 3. 初始化分层架构 (依赖注入)
    resumeRepo := repository.NewGormRepository(postgresClient.GetDB())
    resumeSvc := service.NewResumeServiceWithLLM(resumeRepo, minioStorage, publisher, llmClient)
    resumeHandler := handler.NewResumeHandler(resumeSvc)

    // 4. 启动 gRPC 服务
    go func() {
        lis, _ := net.Listen("tcp", cfg.GRPCHost+":"+cfg.GRPCPort)
        grpcSrv := grpc.NewServer()
        pb.RegisterResumeServiceServer(grpcSrv, grpcHandler.NewResumeServiceServer(resumeSvc))
        reflection.Register(grpcSrv)
        grpcSrv.Serve(lis)
    }()

    // 5. 配置 HTTP 路由
    router := gin.New()
    router.Use(middleware.Recovery(), middleware.Logging(), middleware.CORS())

    // 健康检查
    router.GET("/health", healthHandler)
    router.GET("/ready", readyHandler)

    // API 路由
    api := router.Group("/api/v1")
    {
        api.POST("/resumes", resumeHandler.Create)
        api.GET("/resumes/:id", resumeHandler.GetByID)
        api.GET("/resumes", resumeHandler.List)
        // ...更多路由
    }

    router.Run(cfg.HTTPHost + ":" + cfg.HTTPPort)
}
```

---

## 共享模块使用

### database - PostgreSQL 连接

```go
postgresClient, err := database.NewPostgresClient(database.PostgresConfig{
    Host:     "localhost",
    Port:     "5432",
    User:     "postgres",
    Password: "postgres",
    DBName:   "ats",
})
if err != nil {
    panic(err)
}
defer postgresClient.Close()

// 获取 GORM DB
db := postgresClient.GetDB()
```

### storage - MinIO 文件存储

```go
minioStorage, err := storage.NewMinIOClient(storage.MinIOConfig{
    Endpoint:  "localhost:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    UseSSL:    false,
    Bucket:    "resumes",
})

// 上传文件
objectKey, err := minioStorage.UploadFile(ctx, reader, filename, contentType, size)

// 获取 URL
fileURL := minioStorage.GetFileURL(objectKey)

// 下载文件
reader, err := minioStorage.DownloadFile(ctx, objectKey)

// 验证文件类型
if !storage.IsAllowedFileType(filename) {
    return ErrInvalidFileType
}
```

### events - Redis Streams 事件发布

```go
publisher := events.NewEventPublisher(redisClient, "resume:events")

// 发布创建事件
publisher.PublishCreated(ctx, resumeID, resume)

// 发布状态变更事件
publisher.PublishStatusChanged(ctx, resumeID, oldStatus, newStatus)

// 发布解析完成事件
publisher.PublishParsed(ctx, resumeID, parsedData)

// 发布删除事件
publisher.PublishDeleted(ctx, resumeID)
```

### llm - LLM 客户端

```go
llmClient := llm.NewClient(llm.Config{
    BaseURL: "https://api.moonshot.cn/v1",
    APIKey:  "your-api-key",
    Model:   "moonshot-v1-8k",
})

// 调用 Chat Completion
response, err := llmClient.Complete(ctx, systemPrompt, userPrompt)
```

### response - 统一响应

```go
// 成功响应
response.Success(c, data)

// 成功响应 + 消息
response.SuccessWithMessage(c, "created successfully", data)

// 分页响应
response.SuccessPage(c, items, total, page, pageSize)

// 错误响应
response.BadRequest(c, "invalid parameter")
response.NotFound(c, "resource not found")
response.InternalError(c, "internal server error")
```

### middleware - HTTP 中间件

```go
router := gin.New()
router.Use(middleware.Recovery())   // Panic 恢复
router.Use(middleware.Logging())    // 请求日志
router.Use(middleware.CORS())       // 跨域支持
```

---

## 命名约定

### 包命名

| 类型 | 规则 | 示例 |
|------|------|------|
| 领域包 | 单数名词 | `resume`, `interview`, `search` |
| 共享包 | 功能描述 | `database`, `storage`, `events` |

### 接口命名

```go
// Repository 接口
type ResumeRepository interface { ... }

// Service 接口
type ResumeService interface { ... }

// Parser 接口
type ResumeParser interface { ... }
```

### 结构体命名

```go
// 实现 (小写开头，私有)
type resumeService struct { ... }
type gormRepository struct { ... }

// 构造函数 (New 开头)
func NewResumeService(...) ResumeService { ... }
func NewGormRepository(...) ResumeRepository { ... }
```

### 错误命名

```go
var (
    ErrNotFound          = errors.New("record not found")
    ErrInvalidStatus     = errors.New("invalid status")
    ErrInvalidTransition = errors.New("invalid status transition")
)
```

---

## 最佳实践

### 1. 依赖注入

```go
// ✅ 正确: 通过构造函数注入依赖
func NewResumeService(
    repo repository.ResumeRepository,
    storage storage.FileStorage,
    publisher *events.EventPublisher,
) ResumeService {
    return &resumeService{repo, storage, publisher}
}

// ❌ 错误: 在函数内部创建依赖
func (s *resumeService) Create(...) {
    db := database.Connect()  // 不要这样做
}
```

### 2. 错误处理

```go
// ✅ 正确: 转换为业务错误
func (s *resumeService) GetByID(ctx context.Context, id uuid.UUID) (*model.Resume, error) {
    resume, err := s.repo.GetByID(ctx, id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, ErrResumeNotFound
        }
        return nil, err
    }
    return resume, nil
}

// ✅ 正确: Handler 层转换为 HTTP 状态码
func (h *ResumeHandler) GetByID(c *gin.Context) {
    resume, err := h.svc.GetByID(ctx, id)
    if err != nil {
        if err == service.ErrResumeNotFound {
            response.NotFound(c, "resume not found")
            return
        }
        response.InternalError(c, err.Error())
        return
    }
    response.Success(c, resume)
}
```

### 3. 事件发布

```go
// ✅ 正确: 业务操作后发布事件
func (s *resumeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Resume, error) {
    // ...业务逻辑

    if s.publisher != nil {
        _ = s.publisher.PublishStatusChanged(ctx, id.String(), oldStatus, status)
    }

    return resume, nil
}

// 注意: 事件发布失败不应影响主流程，使用 _ 忽略错误
```

### 4. 状态机验证

```go
// ✅ 正确: 在模型中封装状态转换逻辑
func (r *Resume) CanTransitionTo(newStatus string) bool {
    allowed, exists := validStatusTransitions[r.Status]
    if !exists {
        return false
    }
    return slices.Contains(allowed, newStatus)
}

// 在 Service 中使用
if !resume.CanTransitionTo(status) {
    return nil, ErrInvalidStatusTransition
}
```

### 5. 并发安全

```go
// ✅ 正确: 使用原子操作防止并发问题
updated, err := s.repo.UpdateStatusIf(ctx, id, StatusProcessing, []string{StatusPending, StatusFailed})
if !updated {
    return nil, fmt.Errorf("cannot parse resume with current status")
}
```

### 6. Context 传递

```go
// ✅ 正确: 始终传递 context
func (s *resumeService) Create(ctx context.Context, input CreateResumeInput) (*model.Resume, error) {
    if err := s.repo.Create(ctx, resume); err != nil {
        return nil, err
    }
    // ...
}

// ❌ 错误: 使用 context.Background() 或 context.TODO()
func (s *resumeService) Create(input CreateResumeInput) (*model.Resume, error) {
    ctx := context.Background()  // 不要这样做
}
```

---

## 服务开发检查清单

完成新服务开发时，确保以下项目都已完成：

### 结构完整性
- [ ] `cmd/{service}/main.go` - 服务入口
- [ ] `internal/{domain}/model/` - 数据模型
- [ ] `internal/{domain}/repository/` - 数据访问层
- [ ] `internal/{domain}/service/` - 业务逻辑层
- [ ] `internal/{domain}/handler/` - HTTP 处理器
- [ ] `internal/{domain}/grpc/` - gRPC 服务端（按需提供）
- [ ] `proto/{domain}.proto` - Proto 定义

### 功能完整性
- [ ] CRUD 操作完整
- [ ] 分页列表支持
- [ ] 健康检查 (`/health`, `/ready`)
- [ ] 事件发布 (创建、更新、删除、状态变更)
- [ ] 错误处理和转换
- [ ] 请求参数验证

### 代码质量
- [ ] 接口定义清晰
- [ ] 依赖注入实现
- [ ] 错误处理统一
- [ ] 命名符合规范
- [ ] 注释完整

---

## 参考资源

- **实际实现**: `internal/resume/` 目录
- **设计文档**: `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- **Proto 定义**: `proto/resume.proto`

---

## 当前仓库补充说明

### search-service

- 当前实现包含 `handler -> service -> repository` 三层，以及 `cmd/search-service/main.go` 中的 Redis Stream consumer 启动逻辑。
- `internal/search/repository/es_repository.go` 已实现 Elasticsearch 索引创建、搜索、删除和状态更新。
- `search-service` 当前只提供 HTTP 接口：`GET /api/v1/search` 与 `POST /api/v1/search/advanced`。
- `resume-service` 与 `search-service` 之间的 Redis Stream action / payload 语义应统一收敛到 `ats-platform/internal/shared/events/contracts.go`，避免两端各自维护一份隐式约定。

### gateway

- 当前 `gateway` 实现集中在 `cmd/gateway/main.go`，没有单独的 `internal/gateway/` 领域目录。
- 它提供首页、`/health` 聚合检查和 `/api/v1/*` 的路径前缀代理。
- 当前代理目标服务通过 Consul 动态解析，路径前缀仍然决定逻辑服务名，网关本身保持轻量，不额外承担鉴权、限流或负载均衡编排。

### 本地开发启动约定

- 多服务联调统一使用 `ats-platform/scripts/run-services.sh`。
- `Makefile` 中的 `run-all`、`run-all-no-infra`、`run-all-with-gateway`、`build-services` 都只作为该脚本的薄包装入口。
- 单服务调试仍然使用 `run-resume`、`run-interview`、`run-search`、`run-gateway`。
