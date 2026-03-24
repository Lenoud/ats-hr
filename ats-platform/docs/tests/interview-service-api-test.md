# Interview Service API 接口测试文档

## 测试信息

- **服务名称**: interview-service
- **测试日期**: 2026-03-22
- **服务端口**: 8082 (HTTP)
- **基础URL**: `http://localhost:8082`

## 环境变量

```bash
# 默认配置
HTTP_HOST=0.0.0.0
HTTP_PORT=8082
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ats
REDIS_ADDR=127.0.0.1:6379
CONSUL_HOST=127.0.0.1
CONSUL_PORT=8500
```

如果使用 `./scripts/run-services.sh --no-infra --gateway`，本地开发默认会按“Docker Consul + 宿主机服务”模型设置环境，并自动导出 `SERVICE_ADDRESS=host.docker.internal`。

## 前置条件

需要先创建一个有效的简历记录（通过 `resume-service`）：

```bash
curl -s -X POST http://localhost:8081/api/v1/resumes \
  -H "Content-Type: application/json" \
  -d '{"name":"测试候选人","email":"test@example.com","phone":"13800138000","source":"测试"}'
```

将返回结果中的 `data.id` 记为 `RESUME_ID`。

---

## 1. 健康检查接口

### 1.1 服务健康检查

```bash
curl -s http://localhost:8082/health
```

### 1.2 服务就绪检查

```bash
curl -s http://localhost:8082/ready
```

### 1.3 首页

```bash
curl -s http://localhost:8082/
```

---

## 2. 面试管理接口

### 2.1 创建面试

接口：`POST /api/v1/interviews`

```bash
curl -s -X POST http://localhost:8082/api/v1/interviews \
  -H "Content-Type: application/json" \
  -d '{
    "resume_id": "'"$RESUME_ID"'",
    "round": 1,
    "interviewer": "张三",
    "scheduled_at": "2026-03-23T10:00:00+08:00"
  }'
```

### 2.2 获取面试详情

接口：`GET /api/v1/interviews/:id`

```bash
curl -s "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID"
```

### 2.3 获取简历的所有面试

接口：`GET /api/v1/resumes/:id/interviews`

```bash
curl -s "http://localhost:8082/api/v1/resumes/$RESUME_ID/interviews"
```

### 2.4 更新面试状态

接口：`PUT /api/v1/interviews/:id/status`

```bash
curl -s -X PUT "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}'
```

### 2.5 通过 gateway 验证嵌套路由转发

`gateway` 已支持将 `/api/v1/resumes/:id/interviews` 正确转发到 `interview-service`：

```bash
curl -s "http://localhost:8080/api/v1/resumes/$RESUME_ID/interviews"
```

如果该接口返回业务 JSON，而不是 `404 page not found`，说明 gateway 的嵌套路由分发正常。

---

## 维护说明

- 本文档属于测试/实验文档，不替代正式 API 说明。
- 若默认本地环境、gateway 转发规则或接口返回结构发生变化，需要同步更新本文档。
