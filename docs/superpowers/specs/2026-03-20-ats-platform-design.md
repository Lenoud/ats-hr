# ATS 招聘管理平台设计文档

> **参考实现**: `internal/resume/` 模块
> **开发指南**: `docs/SERVICE_DEVELOPMENT_GUIDE.md`

## 概述

一个面向 HR 的招聘管理系统（Applicant Tracking System），用于整合多平台简历、追踪面试流程、管理面评和作品集。

**项目定位**：学习分布式系统设计，注重稳定性但不需要极高可用性。

**架构原则**：
- 分层架构：Handler → Service → Repository
- 共享逻辑：通过 `internal/shared/` 模块复用基础设施
- 依赖注入：所有依赖通过构造函数注入
- 双协议支持：HTTP REST (对外) + gRPC (服务间)

---

## 系统架构

### 整体架构

```
                         ┌─────────────────┐
                         │   Web Frontend  │
                         │   (React/Vue)   │
                         └────────┬────────┘
                                  │
                         ┌────────▼────────┐
                         │   API Gateway   │
                         │  (Gin Reverse)  │
                         │   Port: 8080    │
                         └────────┬────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Resume Service │    │ Interview Svc   │    │  Search Service │
│  (简历服务)      │    │  (面试流程服务)  │    │  (搜索服务)      │
│  HTTP: 8081     │    │  HTTP: 8082     │    │  HTTP: 8083     │
│  gRPC: 9090     │    │  gRPC: 9091     │    │  gRPC: 9092     │
└────────┬────────┘    └────────┬────────┘    └────────┬────────┘
         │                      │                      │
         │    ┌─────────────────┴─────────────────┐    │
         │    │           Shared Modules           │    │
         │    │  ┌─────────┐ ┌─────────┐ ┌──────┐ │    │
         │    │  │database │ │ storage │ │ llm  │ │    │
         │    │  └─────────┘ └─────────┘ └──────┘ │    │
         │    │  ┌─────────┐ ┌─────────┐ ┌──────┐ │    │
         │    │  │ events  │ │ logger  │ │  pb  │ │    │
         │    │  └─────────┘ └─────────┘ └──────┘ │    │
         │    └─────────────────┬─────────────────┘    │
         │                      │                      │
         ▼                      ▼                      ▼
   ┌───────────┐         ┌───────────┐         ┌───────────┐
   │ PostgreSQL│         │   MinIO   │         │   Redis   │
   │  (主存储)  │         │ (文件存储) │         │ (事件流)   │
   └───────────┘         └───────────┘         └───────────┘
                                                       │
                                                       ▼
                                                ┌───────────┐
                                                │Elasticsearch│
                                                │  (搜索索引) │
                                                └───────────┘
```

### 服务职责

| 服务 | 职责 | HTTP 端口 | gRPC 端口 | 状态 |
|------|------|-----------|-----------|------|
| API Gateway | 静态首页、健康聚合、基于路径前缀的反向代理与 Consul HTTP 服务发现 | 8080 | - | ✅ 已实现 |
| Resume Service | 简历上传、解析、CRUD、事件发布 | 8081 | 9090 | ✅ 已实现 |
| Interview Service | 面试流程、面评、作品集 | 8082 | 9091 | ✅ 已实现 |
| Search Service | 简历搜索、筛选、Redis Stream 消费、Elasticsearch 索引维护 | 8083 | - | ✅ 已实现 |

### 通信方式

| 场景 | 协议 | 说明 |
|------|------|------|
| 前端 → 后端 | HTTP REST | Gin 框架，统一响应格式 |
| 服务间同步调用 | gRPC | `resume-service` 与 `interview-service` 已提供 gRPC 接口 |
| 事件通知 | Redis Streams | `resume-service` 与 `search-service` 通过 `internal/shared/events/` 中的共享事件契约协作 |

---

## 分层架构

每个微服务遵循相同的分层架构，确保代码风格一致：

```
┌─────────────────────────────────────────────────────────────────┐
│                          main.go                                 │
│  • 配置加载、依赖注入                                             │
│  • HTTP/gRPC 服务启动                                            │
│  • 中间件注册 (CORS, Recovery, Logging)                          │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                        Handler Layer                             │
│  • HTTP 请求处理                                                  │
│  • 参数绑定与验证                                                  │
│  • 调用 Service 层                                                │
│  • 统一响应格式化 (response.Success/BadRequest/NotFound)          │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                        Service Layer                             │
│  • 业务逻辑实现                                                   │
│  • 领域规则验证 (如状态机)                                         │
│  • 事务协调                                                       │
│  • 事件发布 (events.EventPublisher)                              │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                       Repository Layer                           │
│  • 数据持久化 (GORM + 原生 SQL)                                   │
│  • 查询封装                                                       │
│  • JSONB 字段处理                                                 │
│  • 原子操作 (如 UpdateStatusIf)                                  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                         ┌──────┴──────┐
                         ▼             ▼
                   ┌──────────┐  ┌──────────┐
                   │PostgreSQL│  │  MinIO   │
                   └──────────┘  └──────────┘
```

---

## 共享模块设计

所有服务共享 `ats-platform/internal/shared/` 下的基础设施模块：

### 目录结构

```
ats-platform/internal/shared/
├── database/
│   ├── postgres.go          # PostgreSQL 连接池管理
│   └── elasticsearch.go     # Elasticsearch 客户端封装
├── storage/
│   └── minio.go             # MinIO 文件存储 (上传/下载/预签名)
├── events/
│   ├── contracts.go         # 共享事件 action / payload 契约
│   ├── publisher.go         # Redis Streams 事件发布
│   └── consumer.go          # Redis Streams Consumer Group 消费
├── llm/
│   └── client.go            # LLM API 客户端 (Moonshot/OpenAI 兼容)
├── logger/
│   └── logger.go            # Zap 结构化日志
├── middleware/
│   ├── cors.go              # CORS 跨域
│   ├── logging.go           # 请求日志
│   └── recovery.go          # Panic 恢复
├── response/
│   └── response.go          # 统一 HTTP 响应
└── pb/
    ├── resume/              # Resume Service protobuf 生成代码
    └── interview/           # Interview Service protobuf 生成代码
```

### 当前实现状态说明

- `resume-service` 同时提供 HTTP 和 gRPC，负责简历主数据、文件上传、解析和事件发布。
- `interview-service` 同时提供 HTTP 和 gRPC，覆盖面试、面评和作品集管理。
- `search-service` 当前只提供 HTTP 接口，不提供 gRPC；它通过共享事件契约消费 `resume:events` 并将简历索引到 Elasticsearch，并在 Consul 中注册为 `search-service-http`。
- `gateway` 当前是轻量级路径代理，按路径前缀决定目标服务，并通过 Consul 动态解析可用实例地址。

### 共享模块使用示例

```go
// main.go 中的依赖注入
func main() {
    ctx := context.Background()

    // 1. 数据库连接
    postgresClient, _ := database.NewPostgresClient(database.PostgresConfig{
        Host:     "localhost",
        Port:     "5432",
        User:     "postgres",
        Password: "postgres",
        DBName:   "ats",
    })
    defer postgresClient.Close()

    // 2. 文件存储
    minioStorage, _ := storage.NewMinIOClient(storage.MinIOConfig{
        Endpoint:  "localhost:9000",
        AccessKey: "minioadmin",
        SecretKey: "minioadmin",
        Bucket:    "resumes",
    })

    // 3. 事件发布
    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    publisher := events.NewEventPublisher(redisClient, "resume:events")

    // 4. LLM 客户端
    llmClient := llm.NewClient(llm.Config{
        BaseURL: "https://api.moonshot.cn/v1",
        APIKey:  os.Getenv("LLM_API_KEY"),
        Model:   "moonshot-v1-8k",
    })

    // 5. 分层架构初始化
    resumeRepo := repository.NewGormRepository(postgresClient.GetDB())
    resumeSvc := service.NewResumeServiceWithLLM(resumeRepo, minioStorage, publisher, llmClient)
    resumeHandler := handler.NewResumeHandler(resumeSvc)
}
```

---

## 数据模型

### PostgreSQL 表结构

```sql
-- 简历表 (已实现)
CREATE TABLE resumes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255),
    phone           VARCHAR(50),
    source          VARCHAR(100),        -- 来源平台：Boss、拉勾等
    file_url        TEXT,                -- MinIO 文件 URL
    parsed_data     JSONB,               -- LLM 解析后的结构化数据
    status          VARCHAR(50) DEFAULT 'pending',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    deleted_at      TIMESTAMP            -- 软删除
);

-- 面试记录表
CREATE TABLE interviews (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_id       UUID REFERENCES resumes(id),
    round           INT DEFAULT 1,
    interviewer     VARCHAR(100),
    scheduled_at    TIMESTAMP,
    status          VARCHAR(50) DEFAULT 'scheduled',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- 面评表
CREATE TABLE feedbacks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    interview_id    UUID REFERENCES interviews(id),
    rating          INT CHECK (rating >= 1 AND rating <= 5),
    content         TEXT,
    recommendation  VARCHAR(50),        -- strong_yes/yes/no/strong_no
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 作品集表
CREATE TABLE portfolios (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_id       UUID REFERENCES resumes(id),
    title           VARCHAR(200),
    file_url        TEXT,
    file_type       VARCHAR(50),        -- pdf/link/image
    created_at      TIMESTAMP DEFAULT NOW()
);
```

### 状态机设计

```
Resume Status:
┌─────────┐     process      ┌────────────┐     success    ┌────────┐
│ pending │ ───────────────► │ processing │ ─────────────► │ parsed │
└────┬────┘                  └─────┬──────┘               └───┬────┘
     │                             │                          │
     │ archive                     │ fail                     │ archive
     │                             ▼                          ▼
     │                        ┌────────┐                 ┌──────────┐
     └───────────────────────►│ failed │                 │ archived │
                              └───┬────┘                 └──────────┘
                                  │ retry
                                  │
                                  └──────────► (back to pending)
```

### Elasticsearch 索引

```json
{
  "mappings": {
    "properties": {
      "resume_id": { "type": "keyword" },
      "name": { "type": "text", "analyzer": "ik_max_word" },
      "email": { "type": "keyword" },
      "skills": { "type": "keyword" },
      "experience_years": { "type": "integer" },
      "education": { "type": "keyword" },
      "work_history": { "type": "text", "analyzer": "ik_max_word" },
      "status": { "type": "keyword" },
      "created_at": { "type": "date" }
    }
  }
}
```

---

## API 设计

### RESTful API（对外）

#### 简历模块 ✅ 已实现

| 方法 | 路径 | 描述 | 状态 |
|------|------|------|------|
| POST | `/api/v1/resumes` | 创建简历 | ✅ |
| POST | `/api/v1/resumes/upload` | 上传+解析+创建 | ✅ |
| GET | `/api/v1/resumes` | 列表（分页+过滤） | ✅ |
| GET | `/api/v1/resumes/:id` | 获取详情 | ✅ |
| PUT | `/api/v1/resumes/:id` | 更新基本信息 | ✅ |
| DELETE | `/api/v1/resumes/:id` | 删除简历 | ✅ |
| PUT | `/api/v1/resumes/:id/status` | 更新状态 | ✅ |
| POST | `/api/v1/resumes/:id/file` | 上传文件 | ✅ |
| POST | `/api/v1/resumes/:id/parse` | 触发解析 | ✅ |

#### 搜索模块 (待实现)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/search` | 搜索简历 |
| POST | `/api/v1/search/advanced` | 高级搜索 |

#### 面试模块 (待实现)

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/interviews` | 创建面试 |
| GET | `/api/v1/interviews/:id` | 获取详情 |
| GET | `/api/v1/resumes/:id/interviews` | 简历的所有面试 |
| PUT | `/api/v1/interviews/:id/status` | 更新状态 |
| POST | `/api/v1/interviews/:id/feedback` | 提交面评 |
| GET | `/api/v1/interviews/:id/feedback` | 获取面评 |

#### 作品集模块 (待实现)

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/resumes/:id/portfolios` | 上传作品集 |
| GET | `/api/v1/resumes/:id/portfolios` | 获取作品集列表 |
| DELETE | `/api/v1/portfolios/:id` | 删除作品集 |

### gRPC 接口（服务间）✅ 已实现

```protobuf
// proto/resume.proto
syntax = "proto3";
package resume;

service ResumeService {
  rpc GetResume(GetResumeRequest) returns (Resume);
  rpc CreateResume(CreateResumeRequest) returns (Resume);
  rpc UpdateResume(UpdateResumeRequest) returns (Resume);
  rpc UpdateStatus(UpdateStatusRequest) returns (Resume);
  rpc ListResumes(ListResumesRequest) returns (ListResumesResponse);
}

message Resume {
  string id = 1;
  string name = 2;
  string email = 3;
  string phone = 4;
  string source = 5;
  string file_url = 6;
  bytes parsed_data = 7;
  string status = 8;
  int64 created_at = 9;
  int64 updated_at = 10;
}
```

---

## 简历解析方案 ✅ 已实现

### 两阶段解析

```
简历文件上传
      │
      ▼
┌─────────────────────────────────┐
│ 第一阶段：文本提取 (parser.go)    │
│ • PDF  → ledongthuc/pdf         │
│ • DOCX → nguyenthenguyen/docx   │
│ • DOC  → 不支持                  │
└───────────────┬─────────────────┘
                │
                ▼
┌─────────────────────────────────┐
│ 第二阶段：LLM 智能提取           │
│ • Moonshot API (OpenAI 兼容)    │
│ • 提取：姓名、邮箱、电话、技能    │
│         工作经历、教育背景等      │
└───────────────┬─────────────────┘
                │
                ▼
         存入 parsed_data (JSONB)
```

### LLM Prompt 模板

```go
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
```

---

## 事件驱动架构

### Redis Streams 事件流

```
Stream: resume:events

事件类型:
├── created        # 简历创建
├── updated        # 简历更新
├── deleted        # 简历删除
├── status_changed # 状态变更
└── parsed         # 解析完成
```

### 事件发布示例

```go
// 创建事件
publisher.PublishCreated(ctx, resumeID, resume)

// 状态变更事件
publisher.PublishStatusChanged(ctx, resumeID, oldStatus, newStatus)

// 解析完成事件
publisher.PublishParsed(ctx, resumeID, parsedData)

// 删除事件
publisher.PublishDeleted(ctx, resumeID)
```

### 数据同步流程

```
简历上传 → Resume Service
              │
              ├─► 写入 PostgreSQL
              │
              ├─► 上传文件到 MinIO
              │
              ├─► LLM 解析
              │
              └─► 发布事件到 Redis Stream
                         │
                         ▼
                  Search Service (消费者)
                         │
                         └─► 写入 Elasticsearch
```

---

## 技术选型

### Go 框架与库

| 组件 | 选型 | 实际使用 |
|------|------|----------|
| HTTP 框架 | Gin | ✅ `github.com/gin-gonic/gin` |
| gRPC | grpc-go | ✅ `google.golang.org/grpc` |
| ORM | GORM | ✅ `gorm.io/gorm` + `gorm.io/driver/postgres` |
| PostgreSQL 驱动 | pgx | ✅ 通过 GORM |
| Redis | go-redis | ✅ `github.com/redis/go-redis/v9` |
| 文件存储 | MinIO | ✅ `github.com/minio/minio-go/v7` |
| PDF 解析 | ledongthuc/pdf | ✅ `github.com/ledongthuc/pdf` |
| DOCX 解析 | nguyenthenguyen/docx | ✅ `github.com/nguyenthenguyen/docx` |
| 日志 | Zap | ✅ `go.uber.org/zap` |
| UUID | google/uuid | ✅ `github.com/google/uuid` |

### 项目目录结构 (实际)

```
ats-platform/
├── cmd/
│   ├── gateway/              # API Gateway (规划中)
│   │   └── main.go
│   ├── resume-service/       # ✅ 简历服务
│   │   ├── main.go
│   │   └── static/
│   │       └── index.html
│   ├── interview-service/    # 面试服务 (待完善)
│   │   └── main.go
│   └── search-service/       # 搜索服务 (待完善)
│       └── main.go
│
├── internal/
│   ├── resume/               # ✅ 简历领域 (完整实现)
│   │   ├── handler/
│   │   │   └── resume_handler.go
│   │   ├── service/
│   │   │   ├── resume_service.go
│   │   │   └── parser.go
│   │   ├── repository/
│   │   │   └── resume_repository.go
│   │   ├── model/
│   │   │   └── resume.go
│   │   └── grpc/
│   │       └── server.go
│   │
│   ├── interview/            # 面试领域 (结构存在，待实现)
│   │   ├── handler/
│   │   ├── service/
│   │   ├── repository/
│   │   └── model/
│   │
│   ├── search/               # 搜索领域 (结构存在，待实现)
│   │   ├── handler/
│   │   ├── service/
│   │   ├── repository/
│   │   └── model/
│   │
│   └── shared/               # ✅ 共享模块 (完整实现)
│       ├── database/
│       │   └── postgres.go
│       ├── storage/
│       │   └── minio.go
│       ├── events/
│       │   ├── publisher.go
│       │   └── consumer.go
│       ├── llm/
│       │   └── client.go
│       ├── logger/
│       │   └── logger.go
│       ├── middleware/
│       │   ├── cors.go
│       │   ├── logging.go
│       │   └── recovery.go
│       ├── response/
│       │   └── response.go
│       └── pb/
│           └── resume/
│               ├── resume.pb.go
│               └── resume_grpc.pb.go
│
├── proto/
│   └── resume.proto          # ✅ Proto 定义
│
├── configs/
│   └── config.yaml
│
├── go.mod
└── go.sum
```

---

## 健康检查 ✅ 已实现

每个服务提供统一的健康检查端点：

```
GET /health
{
  "service": "resume-service",
  "status": "ok",
  "db": "ok",
  "minio": "ok",
  "redis": "ok",
  "time": "2026-03-22T10:00:00Z"
}

GET /ready
{
  "status": "ready"
}
```

---

## 实现进度

### 功能优先级

| 优先级 | 功能 | 状态 |
|--------|------|------|
| P1 | Resume Service | ✅ 完成 |
| P2 | Search Service | 🔲 待开发 |
| P3 | Interview Service | 🔲 待开发 |
| P4 | API Gateway | 🔲 待开发 |
| P5 | 集成测试 | 🔲 待开发 |

### Resume Service 完成度

```
Resume Service:
├── Model 层           ████████████ 100%
├── Repository 层      ████████████ 100%
├── Service 层         ████████████ 100%
├── Handler 层         ████████████ 100%
├── gRPC Server        ████████████ 100%
├── 文件解析 (PDF/DOCX) ████████████ 100%
├── LLM 智能提取       ████████████ 100%
├── 事件发布           ████████████ 100%
└── 单元测试           ░░░░░░░░░░░░   0%
```

---

## 开发规范

详细开发规范请参考: [SERVICE_DEVELOPMENT_GUIDE.md](../SERVICE_DEVELOPMENT_GUIDE.md)

### 核心原则

1. **分层架构**: 严格遵循 Handler → Service → Repository
2. **依赖注入**: 所有依赖通过构造函数注入
3. **共享逻辑**: 复用 `internal/shared/` 模块
4. **错误处理**: 使用业务错误 + 统一响应格式
5. **事件驱动**: 状态变更发布领域事件

### 命名约定

| 类型 | 规则 | 示例 |
|------|------|------|
| Handler | `{Entity}Handler` | `ResumeHandler` |
| Service | `{Entity}Service` | `ResumeService` |
| Repository | `{Entity}Repository` | `ResumeRepository` |
| 错误 | `Err{Description}` | `ErrResumeNotFound` |
| 接口方法 | 动词开头 | `GetByID`, `Create`, `UpdateStatus` |
