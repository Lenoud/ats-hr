# Search Service 设计规格

## 概述

`search-service` 是 ATS 平台中负责简历搜索与索引维护的服务。当前实现基于 Elasticsearch 提供全文检索能力，并通过 Redis Stream 消费 `resume-service` 发布的事件来维护搜索索引。

本文档描述的是仓库中的当前实现状态，而不是实施前规划。

## 当前实现摘要

- HTTP 端口：`8083`
- gRPC：当前未提供
- 搜索引擎：Elasticsearch 8.x
- 事件来源：Redis Stream `resume:events`
- 服务发现：启动时注册到 Consul

## 架构设计

### 分层结构

```text
cmd/search-service/main.go
    ├── 配置加载
    ├── Elasticsearch / Redis / Consul 初始化
    ├── Redis Stream consumer 启动
    └── HTTP Server (Gin)

internal/search/
├── handler/
│   └── search_handler.go
├── service/
│   └── search_service.go
├── repository/
│   ├── es_repository.go
│   └── mock_repository.go
└── model/
    └── resume_document.go
```

### 运行职责

- `handler`：解析 HTTP 查询参数或 JSON 请求体，调用 `SearchService`
- `service`：处理分页默认值，封装索引、删除、状态更新等领域操作
- `repository`：直接对 Elasticsearch 发起索引、搜索、删除和更新请求
- `main.go`：负责事件消费、服务注册、HTTP 服务启动与优雅退出

## HTTP API

### 路由列表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | HTML 首页 |
| GET | `/health` | 健康检查 |
| GET | `/ready` | 就绪检查 |
| GET | `/api/v1/search` | 查询字符串搜索 |
| POST | `/api/v1/search/advanced` | JSON 条件搜索 |

### 查询参数

`GET /api/v1/search` 支持以下参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `query` | string | 全文查询 |
| `skills` | string | 逗号分隔技能列表 |
| `status` | string | 简历状态过滤 |
| `source` | string | 来源过滤 |
| `min_exp` | int | 最小工作年限 |
| `max_exp` | int | 最大工作年限 |
| `page` | int | 页码，默认 `1` |
| `page_size` | int | 每页数量，默认 `10` |

### 响应格式

HTTP 响应通过共享 `response` 包统一封装，分页搜索接口返回：

```json
{
  "code": 0,
  "message": "success",
  "data": [],
  "total": 0,
  "page": 1,
  "page_size": 10
}
```

## Elasticsearch 设计

### 连接配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `ES_ADDRESSES` | `http://localhost:9200` | ES 地址列表，支持逗号分隔 |
| `ES_USERNAME` | 空 | 基础认证用户名 |
| `ES_PASSWORD` | 空 | 基础认证密码 |
| `ES_API_KEY` | 空 | API Key |
| `ES_CLOUD_ID` | 空 | Elastic Cloud ID |
| `ES_INDEX` | `resumes` | 索引名 |
| `ES_INSECURE_SKIP_VERIFY` | `false` | 是否跳过 TLS 校验 |

### 索引映射

服务启动时会调用 `EnsureIndex` 自动检查并创建索引。当前实现包含以下核心字段：

- `resume_id`
- `name`
- `email`
- `skills`
- `experience_years`
- `education`
- `work_history`
- `status`
- `source`
- `created_at`
- `updated_at`

### 搜索能力

当前 repository 层支持：

- 文档索引
- 按文档 ID 查询
- 删除文档
- 更新简历状态
- 多条件搜索

搜索过滤器结构定义在 `internal/search/repository/es_repository.go` 中，字段为：

- `Query`
- `Skills`
- `Status`
- `Source`
- `MinExperience`
- `MaxExperience`
- `Page`
- `PageSize`

## 事件消费

### 消费模型

`search-service` 使用 Redis Stream Consumer Group 模式消费 `resume:events`。
事件 action 与 payload 结构统一由 `internal/shared/events/contracts.go` 定义，消费端不再维护本地私有 payload 结构。

相关配置：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `REDIS_ADDR` | `localhost:6379` | Redis 地址 |
| `REDIS_STREAM` | `resume:events` | 事件流名称 |
| `REDIS_GROUP` | `search-service` | Consumer Group 名称 |
| `REDIS_CONSUMER` | 自动生成 | Consumer 实例名 |
| `REDIS_CONSUMER_ENABLED` | `true` | 是否启用 consumer |

### 事件处理职责

当前实现会根据 `resume-service` 发布的事件更新 Elasticsearch：

| Action | 处理逻辑 |
|--------|----------|
| `created` | 解析 payload 并建立索引文档 |
| `updated` | 重新索引文档 |
| `deleted` | 删除索引文档 |
| `status_changed` | 更新文档状态 |
| `parsed` | 当前忽略，不直接触发索引写入 |

### ACK 行为

- handler 返回成功时，consumer 会对消息执行 `XAck`
- handler 返回错误时，不确认消息，保留后续重试机会

## 服务注册与健康检查

### Consul

服务启动时会注册到 Consul，默认配置：

| 环境变量 | 默认值 |
|----------|--------|
| `SERVICE_NAME` | `search-service` |
| `CONSUL_HOST` | `127.0.0.1` |
| `CONSUL_PORT` | `8500` |

### 健康接口

- `/health`：返回服务状态和 Redis 降级状态
- `/ready`：通过 Elasticsearch `Ping` 判断就绪状态

## 配置总览

| 环境变量 | 默认值 |
|----------|--------|
| `SERVICE_NAME` | `search-service` |
| `HTTP_HOST` | `0.0.0.0` |
| `HTTP_PORT` | `8083` |
| `CONSUL_HOST` | `127.0.0.1` |
| `CONSUL_PORT` | `8500` |
| `ES_ADDRESSES` | `http://localhost:9200` |
| `ES_INDEX` | `resumes` |
| `REDIS_ADDR` | `localhost:6379` |
| `REDIS_STREAM` | `resume:events` |

## 边界与已知限制

- 当前不提供 gRPC 接口。
- 服务依赖共享事件契约中的 `ResumeDocumentPayload` 与 `ResumeStatusChangedPayload` 来建立搜索文档和同步状态。
- 当前网关通过路径前缀将 `/api/v1/search*` 请求转发到本服务，但并未通过 Consul 做动态路由。
- 本文档不包含未实现的测试计划或未来重构方案，只描述仓库现状。
