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
DB_HOST=192.168.250.233
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ats
REDIS_ADDR=192.168.250.233:6379
REDIS_STREAM=interview:events
```

## 前置条件

需要先创建一个有效的简历记录（通过 resume-service）：

```bash
# 创建测试简历
curl -s -X POST http://localhost:8081/api/v1/resumes \
  -H "Content-Type: application/json" \
  -d '{"name":"测试候选人","email":"test@example.com","phone":"13800138000","source":"测试"}'

# 返回示例
# {"code":0,"message":"success","data":{"id":"14871617-18cb-419f-b057-dab342c896fb",...}}

# 设置环境变量
RESUME_ID="14871617-18cb-419f-b057-dab342c896fb"
```

---

## 1. 健康检查接口

### 1.1 服务健康检查

**请求**:
```bash
curl -s http://localhost:8082/health
```

**响应**:
```json
{
  "db": "ok",
  "redis": "ok",
  "service": "interview-service",
  "status": "ok",
  "time": "2026-03-22T20:08:22+08:00"
}
```

### 1.2 服务就绪检查

**请求**:
```bash
curl -s http://localhost:8082/ready
```

**响应**:
```json
{
  "status": "ready"
}
```

### 1.3 首页 (HTML测试界面)

**请求**:
```bash
curl -s http://localhost:8082/
```

**响应**: 返回 HTML 页面（面试管理界面）

---

## 2. 面试管理接口

### 2.1 创建面试

**接口**: `POST /api/v1/interviews`

**请求**:
```bash
curl -s -X POST http://localhost:8082/api/v1/interviews \
  -H "Content-Type: application/json" \
  -d '{
    "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
    "round": 1,
    "interviewer": "张三",
    "scheduled_at": "2026-03-23T10:00:00+08:00"
  }'
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| resume_id | string(uuid) | 是 | 简历ID，必须存在于 resumes 表 |
| round | int | 是 | 面试轮次 (1, 2, 3...) |
| interviewer | string | 是 | 面试官姓名 |
| scheduled_at | string(ISO8601) | 是 | 预约时间 |

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "interview created successfully",
  "data": {
    "id": "f43170b9-2b0b-402a-93c1-d6df887a4e92",
    "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
    "round": 1,
    "interviewer": "张三",
    "scheduled_at": "2026-03-23T10:00:00+08:00",
    "status": "scheduled",
    "created_at": "2026-03-22T20:09:58.9109702+08:00",
    "updated_at": "2026-03-22T20:09:58.9109702+08:00"
  }
}
```

### 2.2 获取面试详情

**接口**: `GET /api/v1/interviews/:id`

**请求**:
```bash
INTERVIEW_ID="f43170b9-2b0b-402a-93c1-d6df887a4e92"

curl -s "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID"
```

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "f43170b9-2b0b-402a-93c1-d6df887a4e92",
    "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
    "round": 1,
    "interviewer": "张三",
    "scheduled_at": "2026-03-23T10:00:00+08:00",
    "status": "scheduled",
    "created_at": "2026-03-22T20:09:58.91097+08:00",
    "updated_at": "2026-03-22T20:09:58.91097+08:00"
  }
}
```

### 2.3 获取简历的所有面试

**接口**: `GET /api/v1/resumes/:id/interviews`

**请求**:
```bash
RESUME_ID="14871617-18cb-419f-b057-dab342c896fb"

curl -s "http://localhost:8082/api/v1/resumes/$RESUME_ID/interviews"
```

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "f43170b9-2b0b-402a-93c1-d6df887a4e92",
      "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
      "round": 1,
      "interviewer": "张三",
      "scheduled_at": "2026-03-23T10:00:00+08:00",
      "status": "scheduled",
      "created_at": "2026-03-22T20:09:58.91097+08:00",
      "updated_at": "2026-03-22T20:09:58.91097+08:00"
    }
  ]
}
```

### 2.4 更新面试状态

**接口**: `PUT /api/v1/interviews/:id/status`

**请求**:
```bash
INTERVIEW_ID="f43170b9-2b0b-402a-93c1-d6df887a4e92"

# 完成面试
curl -s -X PUT "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID/status" \
  -H "Content-Type: application/json" \
  -d '{"status": "completed"}'
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | string | 是 | 状态: scheduled/completed/cancelled |

**状态转换规则**:
- `scheduled` → `completed` ✅
- `scheduled` → `cancelled` ✅
- `completed` → `scheduled` ❌ (不允许)

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "f43170b9-2b0b-402a-93c1-d6df887a4e92",
    "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
    "round": 1,
    "interviewer": "张三",
    "scheduled_at": "2026-03-23T10:00:00+08:00",
    "status": "completed"
  }
}
```
