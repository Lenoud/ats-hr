# ATS 招聘管理平台设计文档

## 概述

一个面向 HR 的招聘管理系统（Applicant Tracking System），用于整合多平台简历、追踪面试流程、管理面评和作品集。

**项目定位**：学习分布式系统设计，注重稳定性但不需要极高可用性。

## 系统架构

### 整体架构

```
                    ┌─────────────────┐
                    │   Web Frontend  │
                    │   (React/Vue)   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  API Gateway    │
                    │  (Kong/Traefik) │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
┌───────▼───────┐   ┌────────▼────────┐  ┌───────▼───────┐
│  Resume Svc   │   │  Interview Svc  │  │  Search Svc   │
│  (简历服务)    │   │  (面试流程服务)  │  │  (搜索服务)    │
│   Port: 8081  │   │   Port: 8082    │  │   Port: 8083  │
└───────┬───────┘   └────────┬────────┘  └───────┬───────┘
        │                    │                    │
        │            ┌───────┴───────┐            │
        │            │               │            │
        ▼            ▼               ▼            ▼
   ┌─────────┐  ┌─────────┐    ┌─────────┐  ┌─────────┐
   │PostgreSQL│  │Redis    │    │  Redis  │  │Elastic  │
   │(主存储)  │  │(缓存)   │    │ (队列)  │  │search   │
   └─────────┘  └─────────┘    └─────────┘  └─────────┘
```

### 服务职责

| 服务 | 职责 | HTTP 端口 | gRPC 端口 |
|------|------|-----------|-----------|
| API Gateway | 路由、认证、限流 | 8080 | - |
| Resume Service | 简历上传、解析、CRUD | 8081 | 9081 |
| Interview Service | 面试流程、面评、作品集 | 8082 | 9082 |
| Search Service | 简历搜索、筛选 | 8083 | 9083 |

### 通信方式

- **同步通信**：gRPC（服务间调用）
- **对外接口**：REST API
- **异步通信**：Redis Streams（事件通知）

## 数据模型

### PostgreSQL 表结构

```sql
-- 简历表
CREATE TABLE resumes (
    id              UUID PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    email           VARCHAR(100),
    phone           VARCHAR(20),
    source          VARCHAR(50),        -- 来源平台：Boss、拉勾等
    file_url        TEXT,               -- 原始文件存储路径
    parsed_data     JSONB,              -- 解析后的结构化数据
    status          VARCHAR(20) DEFAULT 'pending',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- 面试记录表
CREATE TABLE interviews (
    id              UUID PRIMARY KEY,
    resume_id       UUID REFERENCES resumes(id),
    round           INT DEFAULT 1,
    interviewer     VARCHAR(100),
    scheduled_at    TIMESTAMP,
    status          VARCHAR(20),        -- scheduled/completed/cancelled
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 面评表
CREATE TABLE feedbacks (
    id              UUID PRIMARY KEY,
    interview_id    UUID REFERENCES interviews(id),
    rating          INT CHECK (rating >= 1 AND rating <= 5),
    content         TEXT,
    recommendation  VARCHAR(20),        -- strong_yes/yes/no/strong_no
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 作品集表
CREATE TABLE portfolios (
    id              UUID PRIMARY KEY,
    resume_id       UUID REFERENCES resumes(id),
    title           VARCHAR(200),
    file_url        TEXT,
    file_type       VARCHAR(50),        -- pdf/link/image
    created_at      TIMESTAMP DEFAULT NOW()
);
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

### 数据同步流程

```
简历上传 → Resume Service 写入 PostgreSQL
         → 发送事件到 Redis Stream
         → Search Service 消费事件
         → 写入 Elasticsearch 索引
```

## API 设计

### RESTful API（对外）

#### 简历模块

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/resumes` | 上传简历 |
| GET | `/api/v1/resumes/{id}` | 获取简历详情 |
| GET | `/api/v1/resumes` | 简历列表（分页） |
| PUT | `/api/v1/resumes/{id}` | 更新简历信息 |
| DELETE | `/api/v1/resumes/{id}` | 删除简历 |
| PUT | `/api/v1/resumes/{id}/status` | 更新状态 |

#### 搜索模块

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/search` | 搜索简历 |
| POST | `/api/v1/search/advanced` | 高级搜索（多条件） |

#### 面试模块

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/interviews` | 创建面试安排 |
| GET | `/api/v1/interviews/{id}` | 获取面试详情 |
| GET | `/api/v1/resumes/{id}/interviews` | 某简历的所有面试 |
| PUT | `/api/v1/interviews/{id}/status` | 更新面试状态 |
| POST | `/api/v1/interviews/{id}/feedback` | 提交面评 |
| GET | `/api/v1/interviews/{id}/feedback` | 获取面评 |

#### 作品集模块

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/resumes/{id}/portfolios` | 上传作品集 |
| GET | `/api/v1/resumes/{id}/portfolios` | 获取作品集列表 |
| DELETE | `/api/v1/portfolios/{id}` | 删除作品集 |

### gRPC 接口（服务间）

```protobuf
// resume.proto
service ResumeService {
  rpc GetResume(GetResumeRequest) returns (Resume);
  rpc CreateResume(CreateResumeRequest) returns (Resume);
  rpc UpdateStatus(UpdateStatusRequest) returns (Resume);
}

message Resume {
  string id = 1;
  string name = 2;
  string email = 3;
  string phone = 4;
  string status = 5;
  string source = 6;
  string file_url = 7;
  bytes parsed_data = 8;
  int64 created_at = 9;
  int64 updated_at = 10;
}
```

## 简历解析方案

采用两阶段解析：基础提取 + LLM 增强。

### 解析流程

```
简历上传
    │
    ▼
┌─────────────────┐
│ 第一阶段：基础解析 │
│ - PDF/Word → 文本│
│ - 正则提取：      │
│   姓名、邮箱、电话│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 第二阶段：LLM 增强 │
│ - 调用 LLM API   │
│ - 提取：技能、经历│
│   教育、项目等    │
└─────────────────┘
         │
         ▼
   存入 JSONB 字段
```

### LLM Prompt 模板

```
解析以下简历文本，提取结构化信息，返回 JSON：
{
  "skills": ["Go", "MySQL"],
  "experience_years": 3,
  "education": [{"school": "xx", "degree": "本科"}],
  "work_history": [...]
}

简历文本：
{resume_text}
```

## 技术选型

### Go 框架与库

| 组件 | 选型 | 说明 |
|------|------|------|
| HTTP 框架 | Gin | 轻量、高性能、生态丰富 |
| gRPC | grpc-go | 官方库，服务间通信 |
| ORM | GORM | 成熟的 Go ORM，支持 PostgreSQL |
| ES 客户端 | go-elasticsearch | 官方客户端 |
| Redis | go-redis | 支持 Streams |
| 配置管理 | Viper | 支持多格式配置文件 |
| 日志 | Zap | 高性能结构化日志 |
| 文件存储 | MinIO / 本地 | 简历文件存储 |
| PDF 解析 | unidoc/pdf | Go 原生 PDF 解析 |
| Word 解析 | docx | 简单 docx 解析 |

### 项目目录结构

```
ats-platform/
├── cmd/
│   ├── gateway/          # API Gateway
│   ├── resume-service/   # 简历服务
│   ├── interview-service/# 面试服务
│   └── search-service/   # 搜索服务
├── internal/
│   ├── resume/           # 简历领域逻辑
│   ├── interview/        # 面试领域逻辑
│   ├── search/           # 搜索领域逻辑
│   └── shared/           # 共享代码
│       ├── config/
│       ├── logger/
│       ├── middleware/
│       └── pb/           # protobuf 生成的代码
├── pkg/
│   ├── parser/           # 简历解析器
│   └── llm/              # LLM 调用封装
├── deployments/
│   └── docker-compose.yml
├── proto/
│   ├── resume.proto
│   └── interview.proto
├── migrations/           # 数据库迁移
├── configs/              # 配置文件
├── Makefile
└── go.mod
```

## 部署方案

### Docker Compose 配置

```yaml
# deployments/docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: ats
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - pg_data:/var/lib/postgresql/data

  elasticsearch:
    image: elasticsearch:8.11.0
    ports:
      - "9200:9200"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    volumes:
      - es_data:/usr/share/elasticsearch/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data

volumes:
  pg_data:
  es_data:
  minio_data:
```

### 常用命令

```makefile
.PHONY: infra-up infra-down run-all migrate-up

infra-up:
	docker-compose -f deployments/docker-compose.yml up -d

infra-down:
	docker-compose -f deployments/docker-compose.yml down

migrate-up:
	go run cmd/migrate/main.go up

run-gateway:
	go run cmd/gateway/main.go

run-resume:
	go run cmd/resume-service/main.go

run-interview:
	go run cmd/interview-service/main.go

run-search:
	go run cmd/search-service/main.go

run-all:
	@make run-resume & make run-interview & make run-search & make run-gateway

proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

test:
	go test ./... -v
```

### 健康检查

每个服务提供：
```
GET /health    → { "status": "ok", "service": "resume-service" }
GET /ready     → 检查数据库、Redis 连接
```

## 实现计划

### 功能优先级

```
第一阶段：简历上传 + 解析
第二阶段：简历搜索 + 筛选
第三阶段：面试流程管理
第四阶段：面评系统
第五阶段：作品集管理
```

### 阶段划分

| 阶段 | 功能 | 核心任务 |
|------|------|----------|
| P1 | 项目初始化 | 目录结构、Docker 环境、数据库迁移 |
| P2 | Resume Service | 简历上传、解析、CRUD |
| P3 | Search Service | ES 索引、搜索接口、数据同步 |
| P4 | Interview Service | 面试流程、面评、作品集 |
| P5 | API Gateway | 路由聚合、统一入口 |
| P6 | 集成测试 | 端到端测试、性能测试 |
