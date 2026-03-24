# Resume Service API 文档

> 开发模式与基础设施约定见 `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md`。

## 服务信息

- **服务名称**: `resume-service`
- **HTTP 端口**: `8081`
- **gRPC 端口**: `9090`
- **基础 URL**: `http://localhost:8081`
- **当前状态**: HTTP + gRPC 已实现

## 常用环境变量

```bash
HTTP_HOST=0.0.0.0
HTTP_PORT=8081
GRPC_HOST=0.0.0.0
GRPC_PORT=9090
CONSUL_HOST=127.0.0.1
CONSUL_PORT=8500
SERVICE_ADDRESS=
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ats
MINIO_ENDPOINT=127.0.0.1:9000
MINIO_USER=minioadmin
MINIO_PASSWORD=minioadmin
MINIO_BUCKET=resumes
MINIO_USE_SSL=false
REDIS_ADDR=127.0.0.1:6379
REDIS_STREAM=resume:events
```

## 健康检查

### `GET /health`

```bash
curl -s http://localhost:8081/health
```

示例响应：

```json
{
  "service": "resume-service",
  "status": "ok",
  "time": "2026-03-24T10:00:00+08:00"
}
```

### `GET /ready`

```bash
curl -s http://localhost:8081/ready
```

示例响应：

```json
{
  "status": "ready"
}
```

## 主要 HTTP 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/resumes` | 创建简历 |
| GET | `/api/v1/resumes` | 查询简历列表 |
| GET | `/api/v1/resumes/:id` | 获取简历详情 |
| PUT | `/api/v1/resumes/:id` | 更新简历 |
| DELETE | `/api/v1/resumes/:id` | 删除简历 |
| PUT | `/api/v1/resumes/:id/status` | 更新简历状态 |
| POST | `/api/v1/resumes/:id/file` | 上传简历文件 |
| POST | `/api/v1/resumes/:id/parse` | 解析已上传简历 |
| POST | `/api/v1/resumes/upload` | 一键上传并解析 |

## 示例

### 创建简历

```bash
curl -s -X POST http://localhost:8081/api/v1/resumes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三",
    "email": "zhangsan@example.com",
    "phone": "13800138000",
    "source": "LinkedIn"
  }'
```

### 获取简历详情

```bash
RESUME_ID="550e8400-e29b-41d4-a716-446655440000"
curl -s "http://localhost:8081/api/v1/resumes/$RESUME_ID"
```

### 查询简历列表

```bash
curl -s "http://localhost:8081/api/v1/resumes?page=1&page_size=10&status=parsed&source=LinkedIn"
```

### 更新简历

```bash
RESUME_ID="550e8400-e29b-41d4-a716-446655440000"
curl -s -X PUT "http://localhost:8081/api/v1/resumes/$RESUME_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "李四",
    "email": "lisi@example.com",
    "phone": "13900139000"
  }'
```

### 删除简历

```bash
RESUME_ID="550e8400-e29b-41d4-a716-446655440000"
curl -s -X DELETE "http://localhost:8081/api/v1/resumes/$RESUME_ID"
```

### 更新状态

```bash
RESUME_ID="550e8400-e29b-41d4-a716-446655440000"
curl -s -X PUT "http://localhost:8081/api/v1/resumes/$RESUME_ID/status" \
  -H "Content-Type: application/json" \
  -d '{"status": "archived"}'
```

### 上传文件

```bash
RESUME_ID="550e8400-e29b-41d4-a716-446655440000"
curl -s -X POST "http://localhost:8081/api/v1/resumes/$RESUME_ID/file" \
  -F "file=@/path/to/resume.pdf"
```

### 一键上传并解析

```bash
curl -s -X POST "http://localhost:8081/api/v1/resumes/upload" \
  -F "file=@/path/to/resume.pdf" \
  -F "source=LinkedIn"
```

### 解析已上传简历

```bash
RESUME_ID="550e8400-e29b-41d4-a716-446655440000"
curl -s -X POST "http://localhost:8081/api/v1/resumes/$RESUME_ID/parse"
```

## 状态机

| 状态 | 说明 | 可转换目标 |
|------|------|-----------|
| `pending` | 待处理 | `processing`, `archived` |
| `processing` | 处理中 | `parsed`, `failed` |
| `parsed` | 已解析 | `archived` |
| `failed` | 解析失败 | `pending`, `archived` |
| `archived` | 已归档 | - |

## 常见错误

| HTTP Code | 说明 |
|-----------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

常见错误消息：

- `invalid resume id`
- `resume not found`
- `invalid status transition`
- `invalid file type, only PDF, DOC, DOCX are allowed`
- `file size exceeds 10MB limit`

## gRPC

`resume-service` 同时提供 gRPC 服务，定义见：

- `ats-platform/proto/resume.proto`

如需重新生成：

```bash
cd ats-platform
protoc --go_out=. --go-grpc_out=. proto/*.proto
```
