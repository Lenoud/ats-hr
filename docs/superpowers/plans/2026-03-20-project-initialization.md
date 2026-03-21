# ATS Platform - Project Initialization Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 ATS 招聘管理平台的基础设施，包括目录结构、Docker 环境、数据库 schema 和共享代码模块。

**Architecture:** 采用 Go 微服务架构，通过 gRPC 进行服务间通信，使用 PostgreSQL 作为主存储、Redis 作为缓存/消息队列、Elasticsearch 作为搜索引擎、MinIO 作为文件存储。

**Tech Stack:** Go 1.21+, Gin, gRPC, GORM, PostgreSQL 15, Redis 7, Elasticsearch 8.11, MinIO, Docker Compose

---

## File Structure

```
ats-platform/
├── cmd/
│   ├── gateway/main.go              # API Gateway 入口
│   ├── resume-service/main.go       # 简历服务入口
│   ├── interview-service/main.go    # 面试服务入口
│   └── search-service/main.go       # 搜索服务入口
├── internal/
│   ├── resume/
│   │   ├── handler/                 # HTTP handlers
│   │   ├── repository/              # 数据访问层
│   │   └── service/                 # 业务逻辑层
│   ├── interview/
│   │   ├── handler/
│   │   ├── repository/
│   │   └── service/
│   ├── search/
│   │   ├── handler/
│   │   ├── repository/
│   │   └── service/
│   └── shared/
│       ├── config/config.go         # 配置管理
│       ├── logger/logger.go         # 日志封装
│       ├── middleware/auth.go       # 认证中间件
│       ├── middleware/logging.go    # 日志中间件
│       ├── middleware/recovery.go   # 恢复中间件
│       └── response/response.go     # 统一响应
├── pkg/
│   ├── parser/parser.go             # 简历解析器
│   └── llm/client.go                # LLM 调用封装
├── proto/
│   ├── resume.proto                 # 简历服务 proto
│   └── interview.proto              # 面试服务 proto
├── migrations/
│   └── 001_init.up.sql              # 初始化迁移
├── deployments/
│   └── docker-compose.yml           # Docker Compose 配置
├── configs/
│   ├── config.yaml                  # 主配置文件
│   └── config.example.yaml          # 配置示例
├── Makefile
├── go.mod
└── go.sum
```

---

## Task 1: 项目目录结构与 Go Module 初始化

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.gitignore`

- [ ] **Step 1: 创建项目根目录并初始化 Go Module**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower
mkdir -p ats-platform && cd ats-platform
go mod init github.com/example/ats-platform
```

- [ ] **Step 2: 创建目录结构**

```bash
mkdir -p cmd/{gateway,resume-service,interview-service,search-service}
mkdir -p internal/{resume,interview,search}/{handler,repository,service}
mkdir -p internal/shared/{config,logger,middleware,response}
mkdir -p pkg/{parser,llm}
mkdir -p proto
mkdir -p migrations
mkdir -p deployments
mkdir -p configs
```

- [ ] **Step 3: 创建 .gitignore**

```gitignore
# .gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
dist/

# Test coverage
*.out
coverage.html

# IDE
.idea/
.vscode/
*.swp
*.swo

# Environment
.env
.env.local
*.local.yaml

# Logs
logs/
*.log

# OS
.DS_Store
Thumbs.db

# Dependencies
vendor/

# Generated
internal/shared/pb/
```

- [ ] **Step 4: 创建 Makefile**

```makefile
.PHONY: infra-up infra-down proto migrate-up test run-gateway run-resume run-interview run-search run-all clean

# Infrastructure
infra-up:
	docker-compose -f deployments/docker-compose.yml up -d

infra-down:
	docker-compose -f deployments/docker-compose.yml down

infra-logs:
	docker-compose -f deployments/docker-compose.yml logs -f

# Database
migrate-up:
	@echo "Run migrations manually or use migrate tool"

# Protobuf
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

# Services
run-gateway:
	go run ./cmd/gateway

run-resume:
	go run ./cmd/resume-service

run-interview:
	go run ./cmd/interview-service

run-search:
	go run ./cmd/search-service

run-all:
	@make run-resume & make run-interview & make run-search & make run-gateway

# Development
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	go clean
```

- [ ] **Step 5: 验证目录结构创建成功**

```bash
ls -la
tree -L 3 || find . -type d | head -30
```

- [ ] **Step 6: 提交**

```bash
git add .
git commit -m "chore: initialize project structure and go module

- Create directory structure for microservices
- Add Makefile with common commands
- Add .gitignore for Go project"
```

---

## Task 2: Docker Compose 基础设施配置

**Files:**
- Create: `deployments/docker-compose.yml`
- Create: `deployments/.env.example`

- [ ] **Step 1: 创建 Docker Compose 配置**

```yaml
# deployments/docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: ats-postgres
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: ats
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
    volumes:
      - pg_data:/var/lib/postgresql/data
      - ../migrations:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    container_name: ats-elasticsearch
    ports:
      - "9200:9200"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    volumes:
      - es_data:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:9200/_cluster/health || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: ats-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: ats-minio
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_PASSWORD:-minioadmin}
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  pg_data:
  es_data:
  redis_data:
  minio_data:

networks:
  default:
    name: ats-network
```

- [ ] **Step 2: 创建环境变量示例文件**

```bash
# deployments/.env.example
# PostgreSQL
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# MinIO
MINIO_USER=minioadmin
MINIO_PASSWORD=minioadmin
```

- [ ] **Step 3: 复制环境变量文件**

```bash
cp deployments/.env.example deployments/.env
```

- [ ] **Step 4: 启动基础设施并验证**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
make infra-up
sleep 10
docker ps
```

- [ ] **Step 5: 验证各服务健康状态**

```bash
# PostgreSQL
docker exec ats-postgres pg_isready -U postgres

# Elasticsearch
curl -s http://localhost:9200/_cluster/health | head -20

# Redis
docker exec ats-redis redis-cli ping

# MinIO
curl -s http://localhost:9000/minio/health/live
```

- [ ] **Step 6: 提交**

```bash
git add deployments/
git commit -m "chore: add docker compose configuration for infrastructure

- PostgreSQL 15 for main storage
- Elasticsearch 8.11 for search
- Redis 7 for caching and message queue
- MinIO for file storage"
```

---

## Task 3: 数据库迁移脚本

**Files:**
- Create: `migrations/001_init.up.sql`
- Create: `migrations/001_init.down.sql`

- [ ] **Step 1: 创建初始化迁移脚本 (up)**

```sql
-- migrations/001_init.up.sql
-- 简历表
CREATE TABLE IF NOT EXISTS resumes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    email           VARCHAR(100),
    phone           VARCHAR(20),
    source          VARCHAR(50),
    file_url        TEXT,
    parsed_data     JSONB,
    status          VARCHAR(20) DEFAULT 'pending',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 面试记录表
CREATE TABLE IF NOT EXISTS interviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id       UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    round           INT DEFAULT 1,
    interviewer     VARCHAR(100),
    scheduled_at    TIMESTAMP WITH TIME ZONE,
    status          VARCHAR(20) DEFAULT 'scheduled',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 面评表
CREATE TABLE IF NOT EXISTS feedbacks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id    UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    rating          INT CHECK (rating >= 1 AND rating <= 5),
    content         TEXT,
    recommendation  VARCHAR(20),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 作品集表
CREATE TABLE IF NOT EXISTS portfolios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id       UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    title           VARCHAR(200),
    description     TEXT,
    file_url        TEXT,
    file_type       VARCHAR(50),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_resumes_status ON resumes(status);
CREATE INDEX idx_resumes_source ON resumes(source);
CREATE INDEX idx_resumes_created_at ON resumes(created_at DESC);
CREATE INDEX idx_resumes_email ON resumes(email);

CREATE INDEX idx_interviews_resume_id ON interviews(resume_id);
CREATE INDEX idx_interviews_status ON interviews(status);
CREATE INDEX idx_interviews_scheduled_at ON interviews(scheduled_at);

CREATE INDEX idx_feedbacks_interview_id ON feedbacks(interview_id);

CREATE INDEX idx_portfolios_resume_id ON portfolios(resume_id);

-- 更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为每个表添加更新时间触发器
CREATE TRIGGER update_resumes_updated_at BEFORE UPDATE ON resumes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_interviews_updated_at BEFORE UPDATE ON interviews
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feedbacks_updated_at BEFORE UPDATE ON feedbacks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_portfolios_updated_at BEFORE UPDATE ON portfolios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

- [ ] **Step 2: 创建回滚迁移脚本 (down)**

```sql
-- migrations/001_init.down.sql
-- Drop triggers
DROP TRIGGER IF EXISTS update_resumes_updated_at ON resumes;
DROP TRIGGER IF EXISTS update_interviews_updated_at ON interviews;
DROP TRIGGER IF EXISTS update_feedbacks_updated_at ON feedbacks;
DROP TRIGGER IF EXISTS update_portfolios_updated_at ON portfolios;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS portfolios;
DROP TABLE IF EXISTS feedbacks;
DROP TABLE IF EXISTS interviews;
DROP TABLE IF EXISTS resumes;
```

- [ ] **Step 3: 执行迁移脚本**

```bash
# 连接 PostgreSQL 执行迁移
docker exec -i ats-postgres psql -U postgres -d ats < migrations/001_init.up.sql
```

- [ ] **Step 4: 验证表结构创建成功**

```bash
docker exec ats-postgres psql -U postgres -d ats -c "\dt"
docker exec ats-postgres psql -U postgres -d ats -c "\d resumes"
```

Expected output: 应显示 resumes, interviews, feedbacks, portfolios 四张表及其字段结构。

- [ ] **Step 5: 提交**

```bash
git add migrations/
git commit -m "chore: add database migration scripts

- Create tables: resumes, interviews, feedbacks, portfolios
- Add indexes for query optimization
- Add auto-update timestamp triggers"
```

---

## Task 4: 共享配置模块

**Files:**
- Create: `internal/shared/config/config.go`
- Create: `configs/config.yaml`
- Create: `configs/config.example.yaml`

- [ ] **Step 1: 添加依赖**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go get github.com/spf13/viper
```

- [ ] **Step 2: 创建配置结构体**

```go
// internal/shared/config/config.go
package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Elastic  ElasticConfig
	Minio    MinioConfig
}

type ServerConfig struct {
	Name         string        `mapstructure:"name"`
	HTTPPort     int           `mapstructure:"http_port"`
	GRPCPort     int           `mapstructure:"grpc_port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ElasticConfig struct {
	Hosts    []string `mapstructure:"hosts"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
}

type MinioConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(configPath)

	// Read from environment variables
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadFromPath loads configuration from a specific path with fallback
func LoadFromPath(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

- [ ] **Step 3: 创建配置文件**

```yaml
# configs/config.yaml
server:
  name: "ats-platform"
  http_port: 8081
  grpc_port: 9081
  read_timeout: 10s
  write_timeout: 10s

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "ats"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

elastic:
  hosts:
    - "http://localhost:9200"
  username: ""
  password: ""

minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  bucket: "resumes"
```

- [ ] **Step 4: 创建配置示例文件**

```bash
cp configs/config.yaml configs/config.example.yaml
```

- [ ] **Step 5: 验证配置模块编译**

```bash
go build ./internal/shared/config/...
```

Expected: 无错误输出

- [ ] **Step 6: 提交**

```bash
git add go.mod go.sum internal/shared/config/ configs/
git commit -m "feat: add shared configuration module

- Add Viper-based configuration loading
- Support for server, database, redis, elastic, minio configs
- Add example configuration file"
```

---

## Task 5: 共享日志模块

**Files:**
- Create: `internal/shared/logger/logger.go`

- [ ] **Step 1: 添加 Zap 日志依赖**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go get go.uber.org/zap
```

- [ ] **Step 2: 创建日志模块**

```go
// internal/shared/logger/logger.go
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Development bool
	Encoding    string // json or console
}

// Init initializes the global logger
func Init(cfg Config) error {
	var config zap.Config

	if cfg.Development {
		config = zap.NewDevelopmentConfig()
		config.Encoding = "console"
	} else {
		config = zap.NewProductionConfig()
		config.Encoding = cfg.Encoding
	}

	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return err
	}

	Log = logger.Sugar()
	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	Log.Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(template string, args ...interface{}) {
	Log.Debugf(template, args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	Log.Info(args...)
}

// Infof logs a formatted info message
func Infof(template string, args ...interface{}) {
	Log.Infof(template, args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	Log.Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(template string, args ...interface{}) {
	Log.Warnf(template, args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	Log.Error(args...)
}

// Errorf logs a formatted error message
func Errorf(template string, args ...interface{}) {
	Log.Errorf(template, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	Log.Fatal(args...)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(template string, args ...interface{}) {
	Log.Fatalf(template, args...)
	os.Exit(1)
}

// With returns a logger with additional context fields
func With(args ...interface{}) *zap.SugaredLogger {
	return Log.With(args...)
}
```

- [ ] **Step 3: 验证日志模块编译**

```bash
go build ./internal/shared/logger/...
```

Expected: 无错误输出

- [ ] **Step 4: 提交**

```bash
git add go.mod go.sum internal/shared/logger/
git commit -m "feat: add shared logger module with Zap

- Support debug/info/warn/error/fatal levels
- Development and production configurations
- Sugared logger for convenient usage"
```

---

## Task 6: 统一响应模块

**Files:**
- Create: `internal/shared/response/response.go`

- [ ] **Step 1: 创建统一响应结构**

```go
// internal/shared/response/response.go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData represents paginated data
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success returns a success response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage returns a success response with custom message
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// SuccessPage returns a success response with pagination
func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// Error returns an error response
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest returns a 400 error
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    http.StatusBadRequest,
		Message: message,
	})
}

// Unauthorized returns a 401 error
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	c.JSON(http.StatusUnauthorized, Response{
		Code:    http.StatusUnauthorized,
		Message: message,
	})
}

// Forbidden returns a 403 error
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	c.JSON(http.StatusForbidden, Response{
		Code:    http.StatusForbidden,
		Message: message,
	})
}

// NotFound returns a 404 error
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	c.JSON(http.StatusNotFound, Response{
		Code:    http.StatusNotFound,
		Message: message,
	})
}

// InternalError returns a 500 error
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code:    http.StatusInternalServerError,
		Message: message,
	})
}
```

- [ ] **Step 2: 添加 Gin 依赖并验证编译**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go get github.com/gin-gonic/gin
go build ./internal/shared/response/...
```

Expected: 无错误输出

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum internal/shared/response/
git commit -m "feat: add unified response module

- Standard JSON response structure
- Pagination support
- Common error response helpers"
```

---

## Task 7: HTTP 中间件模块

**Files:**
- Create: `internal/shared/middleware/logging.go`
- Create: `internal/shared/middleware/recovery.go`
- Create: `internal/shared/middleware/cors.go`

- [ ] **Step 1: 创建日志中间件**

```go
// internal/shared/middleware/logging.go
package middleware

import (
	"time"

	"ats-platform/internal/shared/logger"

	"github.com/gin-gonic/gin"
)

// Logging returns a logging middleware
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if query != "" {
			path = path + "?" + query
		}

		logger.Infof("[%s] %s %d %v %s",
			method,
			path,
			status,
			latency,
			clientIP,
		)
	}
}
```

- [ ] **Step 2: 创建恢复中间件**

```go
// internal/shared/middleware/recovery.go
package middleware

import (
	"net/http"
	"runtime/debug"

	"ats-platform/internal/shared/logger"

	"github.com/gin-gonic/gin"
)

// Recovery returns a recovery middleware
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("Panic recovered: %v\n%s", err, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
```

- [ ] **Step 3: 创建 CORS 中间件**

```go
// internal/shared/middleware/cors.go
package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS returns a CORS middleware
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-Requested-With, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
```

- [ ] **Step 4: 验证中间件模块编译**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go build ./internal/shared/middleware/...
```

Expected: 无错误输出

- [ ] **Step 5: 提交**

```bash
git add internal/shared/middleware/
git commit -m "feat: add HTTP middleware module

- Request logging middleware
- Panic recovery middleware
- CORS middleware"
```

---

## Task 8: 服务入口模板

**Files:**
- Create: `cmd/resume-service/main.go`

- [ ] **Step 1: 创建 Resume Service 入口**

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

	"ats-platform/internal/shared/config"
	"ats-platform/internal/shared/logger"
	"ats-platform/internal/shared/middleware"

	"github.com/gin-gonic/gin"
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
		// TODO: Check database and redis connections
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": cfg.Server.Name,
		})
	})

	// API routes will be added here
	// api := router.Group("/api/v1")
	// {
	//     // Resume routes
	// }

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

	logger.Info("Server stopped")
}
```

- [ ] **Step 2: 验证服务入口编译**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go build ./cmd/resume-service/...
```

Expected: 无错误输出

- [ ] **Step 3: 提交**

```bash
git add cmd/resume-service/ go.mod go.sum
git commit -m "feat: add resume service entry point

- HTTP server with Gin
- Graceful shutdown support
- Health check endpoints
- gRPC listener placeholder"
```

---

## Task 9: Protobuf 定义

**Files:**
- Create: `proto/resume.proto`
- Create: `proto/interview.proto`

- [ ] **Step 1: 创建 Resume proto 定义**

```protobuf
// proto/resume.proto
syntax = "proto3";

package resume;

option go_package = "github.com/example/ats-platform/internal/shared/pb/resume";

// Resume represents a candidate's resume
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

// GetResumeRequest
message GetResumeRequest {
  string id = 1;
}

// CreateResumeRequest
message CreateResumeRequest {
  string name = 1;
  string email = 2;
  string phone = 3;
  string source = 4;
  string file_url = 5;
  bytes parsed_data = 6;
}

// UpdateResumeRequest
message UpdateResumeRequest {
  string id = 1;
  string name = 2;
  string email = 3;
  string phone = 4;
  bytes parsed_data = 5;
}

// UpdateStatusRequest
message UpdateStatusRequest {
  string id = 1;
  string status = 2;
}

// ListResumesRequest
message ListResumesRequest {
  int32 page = 1;
  int32 page_size = 2;
  string status = 3;
  string source = 4;
}

// ListResumesResponse
message ListResumesResponse {
  repeated Resume resumes = 1;
  int64 total = 2;
}

// ResumeService provides gRPC methods for resume management
service ResumeService {
  rpc GetResume(GetResumeRequest) returns (Resume);
  rpc CreateResume(CreateResumeRequest) returns (Resume);
  rpc UpdateResume(UpdateResumeRequest) returns (Resume);
  rpc UpdateStatus(UpdateStatusRequest) returns (Resume);
  rpc ListResumes(ListResumesRequest) returns (ListResumesResponse);
}
```

- [ ] **Step 2: 创建 Interview proto 定义**

```protobuf
// proto/interview.proto
syntax = "proto3";

package interview;

option go_package = "github.com/example/ats-platform/internal/shared/pb/interview";

// Interview represents an interview record
message Interview {
  string id = 1;
  string resume_id = 2;
  int32 round = 3;
  string interviewer = 4;
  int64 scheduled_at = 5;
  string status = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
}

// Feedback represents interview feedback
message Feedback {
  string id = 1;
  string interview_id = 2;
  int32 rating = 3;
  string content = 4;
  string recommendation = 5;
  int64 created_at = 6;
}

// Portfolio represents a candidate's portfolio item
message Portfolio {
  string id = 1;
  string resume_id = 2;
  string title = 3;
  string description = 4;
  string file_url = 5;
  string file_type = 6;
  int64 created_at = 7;
}

// GetInterviewRequest
message GetInterviewRequest {
  string id = 1;
}

// CreateInterviewRequest
message CreateInterviewRequest {
  string resume_id = 1;
  int32 round = 2;
  string interviewer = 3;
  int64 scheduled_at = 4;
}

// UpdateInterviewStatusRequest
message UpdateInterviewStatusRequest {
  string id = 1;
  string status = 2;
}

// ListInterviewsRequest
message ListInterviewsRequest {
  string resume_id = 1;
  string status = 2;
  int32 page = 3;
  int32 page_size = 4;
}

// ListInterviewsResponse
message ListInterviewsResponse {
  repeated Interview interviews = 1;
  int64 total = 2;
}

// CreateFeedbackRequest
message CreateFeedbackRequest {
  string interview_id = 1;
  int32 rating = 2;
  string content = 3;
  string recommendation = 4;
}

// GetFeedbackRequest
message GetFeedbackRequest {
  string interview_id = 1;
}

// CreatePortfolioRequest
message CreatePortfolioRequest {
  string resume_id = 1;
  string title = 2;
  string description = 3;
  string file_url = 4;
  string file_type = 5;
}

// ListPortfoliosRequest
message ListPortfoliosRequest {
  string resume_id = 1;
}

// ListPortfoliosResponse
message ListPortfoliosResponse {
  repeated Portfolio portfolios = 1;
  int64 total = 2;
}

// InterviewService provides gRPC methods for interview management
service InterviewService {
  // Interview methods
  rpc GetInterview(GetInterviewRequest) returns (Interview);
  rpc CreateInterview(CreateInterviewRequest) returns (Interview);
  rpc UpdateInterviewStatus(UpdateInterviewStatusRequest) returns (Interview);
  rpc ListInterviews(ListInterviewsRequest) returns (ListInterviewsResponse);

  // Feedback methods
  rpc CreateFeedback(CreateFeedbackRequest) returns (Feedback);
  rpc GetFeedback(GetFeedbackRequest) returns (Feedback);

  // Portfolio methods
  rpc CreatePortfolio(CreatePortfolioRequest) returns (Portfolio);
  rpc ListPortfolios(ListPortfoliosRequest) returns (ListPortfoliosResponse);
}
```

- [ ] **Step 3: 提交**

```bash
git add proto/
git commit -m "feat: add protobuf definitions for resume and interview services

- Resume service: CRUD operations and status management
- Interview service: interviews, feedback, and portfolios"
```

---

## Task 10: 最终验证

**Files:**
- Modify: `Makefile` (add tidy command)

- [ ] **Step 1: 整理依赖**

```bash
cd /private/var/folders/7d/rgkb2h7n7dn33zwlrk3zrjm00000gn/T/vibe-kanban/worktrees/39e1-/superpower/ats-platform
go mod tidy
```

- [ ] **Step 2: 验证所有模块编译**

```bash
go build ./...
```

Expected: 无错误输出

- [ ] **Step 3: 测试启动 Resume Service**

```bash
# 在后台启动服务
timeout 5 go run ./cmd/resume-service || true

# 或者使用 curl 测试健康检查（如果服务仍在运行）
curl -s http://localhost:8081/health || echo "Service not running (expected in timeout)"
```

- [ ] **Step 4: 查看最终目录结构**

```bash
tree -L 3 || find . -type f -name "*.go" -o -name "*.yaml" -o -name "*.sql" -o -name "*.proto" | head -40
```

- [ ] **Step 5: 最终提交**

```bash
git add .
git commit -m "chore: finalize project initialization

- All modules compile successfully
- Directory structure complete
- Ready for service implementation"
```

---

## Summary

完成此计划后，项目将具备：

| 组件 | 状态 |
|------|------|
| 目录结构 | ✅ 完成 |
| Docker 基础设施 | ✅ PostgreSQL, Redis, ES, MinIO |
| 数据库 Schema | ✅ 4 张核心表 + 索引 + 触发器 |
| 共享配置模块 | ✅ Viper 配置加载 |
| 共享日志模块 | ✅ Zap 结构化日志 |
| 统一响应模块 | ✅ 标准 API 响应格式 |
| HTTP 中间件 | ✅ 日志/恢复/CORS |
| 服务入口模板 | ✅ Resume Service 示例 |
| Protobuf 定义 | ✅ Resume + Interview |

**下一步：** 执行 Plan 2 - Resume Service 完整实现
