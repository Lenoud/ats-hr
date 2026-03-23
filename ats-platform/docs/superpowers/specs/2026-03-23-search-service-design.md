# Search-Service 设计规格

## 概述

search-service 是 ATS 平台的简历搜索服务，基于 Elasticsearch 提供全文搜索能力。本文档描述 search-service 的完整实现方案。

## 1. 禂述

search-service 是 ATS 平台的简历搜索服务，基于 Elasticsearch 提供全文搜索能力。本文档描述 search-service 的完整实现方案。

## 2. 架构设计

### 2.1 系统架构
search-service 猴需修改以符合 SERVICE_DEVELOPMENT_GUIDE.md 的标准：
1. **重写 `cmd/search-service/main.go**： 修复当前损坏的代码，遵循标准服务模板
2. **实现 ES Repository**： 添加真实的 Elasticsearch 连接和操作实现
3. **扩展 Event Consumer**： 消费 resume:events 流中的关键事件
4. **添加 gRPC 支持**（暂不实现， 用户已确认暂不需要)

### 2.2 分层设计
```
cmd/search-service/main.go
    └── HTTP Server (Gin)
    └── Graceful Shutdown
    └── Event Consumer (goroutine)

internal/search/
├── handler/
│   └── search_handler.go (已存在)
├── service/
│   └── search_service.go (已存在)
├── repository/
│   ├── es_repository.go (需实现)
│   └── mock_repository.go (已存在)
├── model/
│   └── resume_document.go (已存在)
└── grpc/ (暂不实现)
```

### 2.3 技术栈
| 组件 | 技术选型 | 说明 |
|------|---------|------|
| HTTP 框架 | Gin | RESTful API |
| 搜索引擎 | Elasticsearch 8.x | 全文搜索 |
| 缓存/消息 | Redis | 事件消费 |
| 服务发现 | Consul | 服务注册与发现 |
| 日志 | Zap | 结构化日志 |

## 3. Elasticsearch 配置
### 3.1 连接配置
| 配置项 | 环境变量 | 默认值 |
|--------|---------|------|
| ES 地址 | ES_ADDRESSES | localhost:9200 |
| ES 用户名 | ES_USERNAME | elastic |
| ES 密码 | ES_PASSWORD | - |
| ES 索引名 | ES_INDEX | resumes |

### 3.2 索引映射
```json
{
  "mappings": {
    "properties": {
      "resume_id": { "type": "keyword" },
      "name": {
        "type": "text",
        "analyzer": "standard"
      },
      "email": {
        "type": "text",
        "analyzer": "standard"
      },
      "skills": {
        "type": "text",
        "analyzer": "standard"
      },
      "experience_years": { "type": "integer" },
      "education": {
        "type": "text",
        "analyzer": "standard"
      },
      "work_history": {
        "type": "text",
        "analyzer": "standard"
      },
      "status": { "type": "keyword" },
      "source": { "type": "keyword" },
      "created_at": { "type": "date" },
      "updated_at": { "type": "date" }
    }
  }
}
```
索引在服务启动时自动创建（如果不存在)。

### 3.3 搜索功能
支持以下搜索条件:
- 全文搜索 (query): 在 name, skills、 work_history 中搜索
- 技能过滤 (skills): 精确匹配技能列表
- 状态过滤 (status): 精确匹配状态
- 来源过滤 (source): 精确匹配来源
- 经验范围 (min_exp, max_exp): 数值范围过滤
- 分页 (page, page_size): 标准分页

## 4. 事件消费
### 4.1 消费的事件
从 `resume:events` 流消费以下事件:
| Action | 处理逻辑 |
|--------|----------|
| created | 创建 ES 文档， 解析 Payload 获取简历信息并索引 |
| deleted | 删除 ES 文档 |
| status_changed | 更新 ES 文档状态 |

### 4.2 事件处理流程
```
┌────────────────┐
│ resume-service │
│  PublishCreated │
│  PublishDeleted │
│  PublishStatusChanged │
└────────┬───────┬────────────────────────────┐
                   │                           │
                   ▼                           ▼
            ┌─────────────────────────────────────────┐
            │       Redis Stream (resume:events)    │
            └─────────────────────────────────────────┘
                           │
                           ▼
            ┌─────────────────────────────────────────┐
            │     search-service Event Consumer       │
            │                                        │
            │  created → IndexResumeDocument()       │
            │  deleted → DeleteResumeDocument()      │
            │  status_changed → UpdateResumeStatus() │
            └─────────────────────────────────────────┘
                           │
                           ▼
            ┌─────────────────────────────────────────┐
            │          Elasticsearch Index             │
            └─────────────────────────────────────────┘
```

### 4.3 错误处理
- 事件处理失败时不 ACK，消息会重新投递
- 实现指数退避重试机制
- 记录处理失败的详细信息

## 5. HTTP API
### 5.1 端点列表
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 健康检查 |
| GET | /ready | 就绪检查 |
| GET | /api/v1/search | 搜索简历 |
| POST | /api/v1/search/advanced | 高级搜索 |

### 5.2 请求/响应格式
#### 搜索请求
```http
GET /api/v1/search?query=golang&skills=go,docker&page=1&page_size=10
```

#### 搜索响应
```json
{
  "code": 0,
  "message": "success",
  "data": [...],
  "total": 100,
  "page": 1,
  "page_size": 10
}
```

## 6. 配置管理
### 6.1 环境变量
| 变量 | 默认值 | 说明 |
|------|---------|------|
| SERVICE_NAME | search-service | 服务名称 |
| HTTP_HOST | 0.0.0.0 | HTTP 监听地址 |
| HTTP_PORT | 8082 | HTTP 端口 |
| CONSUL_HOST | 127.0.0.1 | Consul 地址 |
| CONSUL_PORT | 8500 | Consul 端口 |
| ES_ADDRESSES | localhost:9200 | Elasticsearch 地址 |
| ES_USERNAME | elastic | Elasticsearch 用户名 |
| ES_PASSWORD | - | Elasticsearch 密码 |
| ES_INDEX | resumes | Elasticsearch 索引名 |
| REDIS_ADDR | localhost:6379 | Redis 地址 |
| REDIS_STREAM | resume:events | Redis Stream 名称 |

## 7. 优雅退出
### 7.1 关闭顺序
1. 停止接收新请求
2. 关闭 HTTP Server (5秒超时)
3. 从 Consul 注销服务
4. 关闭 Redis 连接

### 7.2 信号处理
- 监听 SIGINT (Ctrl+C)
- 监听 SIGTERM (kill 命令)

## 8. 文件变更清单
| 文件 | 操作 | 说明 |
|------|------|------|
| `cmd/search-service/main.go` | 重写 | 完全重写，遵循标准模板 |
| `cmd/search-service/static/index.html` | 新建 | 服务首页 |
| `internal/search/repository/es_repository.go` | 修改 | 添加 ES 实现结构体和方法 |
| `internal/shared/events/consumer.go` | 无需修改 | 现有实现已满足需求 |

## 9. 测试计划
### 9.1 单元测试
- MockRepository 测试（已存在）
- SearchService 测试
- ES Repository 实现（使用 mock）

### 9.2 集成测试
- 端到端搜索流程测试
- 事件消费流程测试

## 10. 部署配置
### 10.1 Docker Compose
search-service 已在 docker-compose.yml 中配置 Elasticsearch 容器。

### 10.2 服务依赖
```
search-service
    ├── elasticsearch (必需)
    ├── redis (必需)
    └── consul (必需)
```
