# API Gateway 实现计划

## 概述

API Gateway 作为统一入口，路由请求到各个微服务。

## 服务信息

- **端口**: 8080 (HTTP)
- **职责**: 路由聚合、反向代理、CORS

## 后端服务

| 服务 | 端口 | 路由前缀 |
|------|------|----------|
| Resume Service | 8081 | /api/v1/resumes |
| Interview Service | 8082 | /api/v1/interviews, /api/v1/resumes/:id/interviews, /api/v1/resumes/:id/portfolios |
| Search Service | 8083 | /api/v1/search |

## 任务分解

### Task 1: 创建 Gateway 入口

**文件**: `cmd/gateway/main.go`

- 初始化 Gin 路由
- 配置反向代理
- 健康检查端点
- 嵌入统一测试前端
- 监听 0.0.0.0:8080

### Task 2: 实现代理配置

```go
// 服务配置
var services = map[string]string{
    "resume":     "http://localhost:8081",
    "interview":  "http://localhost:8082",
    "search":     "http://localhost:8083",
}
```

### Task 3: 路由规则

| 路径 | 目标服务 |
|------|----------|
| /api/v1/search* | Search Service |
| /api/v1/search/advanced* | Search Service |
| /api/v1/resumes/:id/interviews* | Interview Service |
| /api/v1/resumes/:id/portfolios* | Interview Service |
| /api/v1/interviews* | Interview Service |
| /api/v1/portfolios* | Interview Service |
| /api/v1/resumes* | Resume Service |

### Task 4: 统一前端页面

创建聚合所有功能的测试页面：
- 简历管理
- 简历搜索
- 面试管理
- 作品集管理

## 验收标准

1. 所有路由正确转发
2. 统一入口可访问所有服务功能
3. 提供集成测试前端
