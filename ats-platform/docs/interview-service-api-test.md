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
    "status": "completed",
    "created_at": "2026-03-22T20:09:58.91097+08:00",
    "updated_at": "2026-03-22T20:10:22.611799+08:00"
  }
}
```

### 2.5 删除面试

**接口**: `DELETE /api/v1/interviews/:id`

**请求**:
```bash
INTERVIEW_ID="589c3545-88c0-4b47-8ea0-ce34e4a84cd1"

curl -s -X DELETE "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID"
```

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "interview deleted successfully"
}
```

---

## 3. 面评管理接口

### 3.1 提交面评

**接口**: `POST /api/v1/interviews/:id/feedback`

**请求**:
```bash
INTERVIEW_ID="f43170b9-2b0b-402a-93c1-d6df887a4e92"

curl -s -X POST "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID/feedback" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 4,
    "content": "候选人技术能力扎实，沟通良好，建议进入下一轮面试。",
    "recommendation": "yes"
  }'
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| rating | int | 是 | 评分 1-5 (1=不合格, 5=优秀) |
| content | string | 是 | 面评内容 |
| recommendation | string | 是 | 推荐意见: strong_yes/yes/no/strong_no |

**推荐意见选项**:
- `strong_yes`: 强烈推荐
- `yes`: 推荐
- `no`: 不推荐
- `strong_no`: 强烈不推荐

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "feedback submitted successfully",
  "data": {
    "id": "711b0cf3-49ed-44d9-aa53-ec26eea404d8",
    "interview_id": "f43170b9-2b0b-402a-93c1-d6df887a4e92",
    "rating": 4,
    "content": "候选人技术能力扎实，沟通良好，建议进入下一轮面试。",
    "recommendation": "yes",
    "created_at": "2026-03-22T20:10:45.6876534+08:00"
  }
}
```

### 3.2 获取面评

**接口**: `GET /api/v1/interviews/:id/feedback`

**请求**:
```bash
INTERVIEW_ID="f43170b9-2b0b-402a-93c1-d6df887a4e92"

curl -s "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID/feedback"
```

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "711b0cf3-49ed-44d9-aa53-ec26eea404d8",
    "interview_id": "f43170b9-2b0b-402a-93c1-d6df887a4e92",
    "rating": 4,
    "content": "候选人技术能力扎实，沟通良好，建议进入下一轮面试。",
    "recommendation": "yes",
    "created_at": "2026-03-22T20:10:45.687653+08:00"
  }
}
```

---

## 4. 作品集管理接口

### 4.1 创建作品集

**接口**: `POST /api/v1/resumes/:id/portfolios`

**请求**:
```bash
RESUME_ID="14871617-18cb-419f-b057-dab342c896fb"

curl -s -X POST "http://localhost:8082/api/v1/resumes/$RESUME_ID/portfolios" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "GitHub项目展示",
    "file_url": "https://github.com/example/project",
    "file_type": "link"
  }'
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 作品标题 |
| file_url | string | 是 | 文件或链接URL |
| file_type | string | 是 | 类型: pdf/link/image |

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "portfolio created successfully",
  "data": {
    "id": "88859e7c-954c-431c-8846-48f78e1cb547",
    "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
    "title": "GitHub项目展示",
    "file_url": "https://github.com/example/project",
    "file_type": "link",
    "created_at": "2026-03-22T20:10:46.1697216+08:00"
  }
}
```

### 4.2 获取作品集列表

**接口**: `GET /api/v1/resumes/:id/portfolios`

**请求**:
```bash
RESUME_ID="14871617-18cb-419f-b057-dab342c896fb"

curl -s "http://localhost:8082/api/v1/resumes/$RESUME_ID/portfolios"
```

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "88859e7c-954c-431c-8846-48f78e1cb547",
      "resume_id": "14871617-18cb-419f-b057-dab342c896fb",
      "title": "GitHub项目展示",
      "file_url": "https://github.com/example/project",
      "file_type": "link",
      "created_at": "2026-03-22T20:10:46.169721+08:00"
    }
  ]
}
```

### 4.3 删除作品集

**接口**: `DELETE /api/v1/portfolios/:id`

**请求**:
```bash
PORTFOLIO_ID="88859e7c-954c-431c-8846-48f78e1cb547"

curl -s -X DELETE "http://localhost:8082/api/v1/portfolios/$PORTFOLIO_ID"
```

**成功响应** (code: 0):
```json
{
  "code": 0,
  "message": "portfolio deleted successfully"
}
```

---

## 5. 错误处理测试

### 5.1 获取不存在的面试

**请求**:
```bash
curl -s "http://localhost:8082/api/v1/interviews/00000000-0000-0000-0000-000000000000"
```

**响应** (code: 404):
```json
{
  "code": 404,
  "message": "interview not found"
}
```

### 5.2 无效的面试ID格式

**请求**:
```bash
curl -s "http://localhost:8082/api/v1/interviews/invalid-id"
```

**响应** (code: 400):
```json
{
  "code": 400,
  "message": "invalid interview id"
}
```

### 5.3 创建面试时使用不存在的简历ID

**请求**:
```bash
curl -s -X POST http://localhost:8082/api/v1/interviews \
  -H "Content-Type: application/json" \
  -d '{"resume_id":"00000000-0000-0000-0000-000000000000","round":1,"interviewer":"测试","scheduled_at":"2026-03-23T10:00:00+08:00"}'
```

**响应** (code: 500):
```json
{
  "code": 500,
  "message": "ERROR: insert or update on table \"interviews\" violates foreign key constraint \"interviews_resume_id_fkey\" (SQLSTATE 23503)"
}
```

### 5.4 重复提交面评

**请求**:
```bash
# 同一面试已有面评时再次提交
INTERVIEW_ID="f43170b9-2b0b-402a-93c1-d6df887a4e92"

curl -s -X POST "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID/feedback" \
  -H "Content-Type: application/json" \
  -d '{"rating":5,"content":"重复提交测试","recommendation":"strong_yes"}'
```

**响应** (code: 400):
```json
{
  "code": 400,
  "message": "feedback already exists for this interview"
}
```

### 5.5 无效的状态转换

**请求**:
```bash
# 已完成的面试不能改回 scheduled
INTERVIEW_ID="f43170b9-2b0b-402a-93c1-d6df887a4e92"

curl -s -X PUT "http://localhost:8082/api/v1/interviews/$INTERVIEW_ID/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"scheduled"}'
```

**响应** (code: 400):
```json
{
  "code": 400,
  "message": "invalid status transition"
}
```

---

## 6. 完整测试脚本

```bash
#!/bin/bash
# Interview Service API 完整测试脚本

BASE_URL="http://localhost:8082"
RESUME_SERVICE="http://localhost:8081"

echo "=========================================="
echo "     Interview Service API Testing"
echo "=========================================="

# 1. 创建测试简历
echo ""
echo "=== 准备: 创建测试简历 ==="
RESUME_RESULT=$(curl -s -X POST "$RESUME_SERVICE/api/v1/resumes" \
  -H "Content-Type: application/json" \
  -d '{"name":"测试候选人","email":"test@example.com","phone":"13800138000","source":"测试"}')
RESUME_ID=$(echo "$RESUME_RESULT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Resume ID: $RESUME_ID"

# 2. 健康检查
echo ""
echo "=== 1. Health Check ==="
curl -s "$BASE_URL/health"

# 3. 创建面试
echo ""
echo "=== 2. Create Interview ==="
INTERVIEW_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/interviews" \
  -H "Content-Type: application/json" \
  -d "{\"resume_id\":\"$RESUME_ID\",\"round\":1,\"interviewer\":\"张三\",\"scheduled_at\":\"2026-03-23T10:00:00+08:00\"}")
echo "$INTERVIEW_RESULT"
INTERVIEW_ID=$(echo "$INTERVIEW_RESULT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

# 4. 获取面试
echo ""
echo "=== 3. Get Interview ==="
curl -s "$BASE_URL/api/v1/interviews/$INTERVIEW_ID"

# 5. 更新状态
echo ""
echo "=== 4. Update Status ==="
curl -s -X PUT "$BASE_URL/api/v1/interviews/$INTERVIEW_ID/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}'

# 6. 提交面评
echo ""
echo "=== 5. Submit Feedback ==="
curl -s -X POST "$BASE_URL/api/v1/interviews/$INTERVIEW_ID/feedback" \
  -H "Content-Type: application/json" \
  -d '{"rating":4,"content":"候选人技术能力扎实","recommendation":"yes"}'

# 7. 获取面评
echo ""
echo "=== 6. Get Feedback ==="
curl -s "$BASE_URL/api/v1/interviews/$INTERVIEW_ID/feedback"

# 8. 创建作品集
echo ""
echo "=== 7. Create Portfolio ==="
PORTFOLIO_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/resumes/$RESUME_ID/portfolios" \
  -H "Content-Type: application/json" \
  -d '{"title":"GitHub项目","file_url":"https://github.com/example","file_type":"link"}')
echo "$PORTFOLIO_RESULT"
PORTFOLIO_ID=$(echo "$PORTFOLIO_RESULT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

# 9. 获取作品集
echo ""
echo "=== 8. List Portfolios ==="
curl -s "$BASE_URL/api/v1/resumes/$RESUME_ID/portfolios"

# 10. 删除作品集
echo ""
echo "=== 9. Delete Portfolio ==="
curl -s -X DELETE "$BASE_URL/api/v1/portfolios/$PORTFOLIO_ID"

# 11. 删除面试
echo ""
echo "=== 10. Delete Interview ==="
curl -s -X DELETE "$BASE_URL/api/v1/interviews/$INTERVIEW_ID"

echo ""
echo "=========================================="
echo "     All Tests Completed"
echo "=========================================="
```

---

## 7. API 端点汇总

| 方法 | 端点 | 描述 | 状态码 |
|------|------|------|--------|
| GET | `/health` | 服务健康检查 | 200 |
| GET | `/ready` | 服务就绪检查 | 200/503 |
| GET | `/` | 首页(HTML测试界面) | 200 |
| POST | `/api/v1/interviews` | 创建面试 | 200 |
| GET | `/api/v1/interviews/:id` | 获取面试详情 | 200/404 |
| PUT | `/api/v1/interviews/:id/status` | 更新面试状态 | 200/400/404 |
| DELETE | `/api/v1/interviews/:id` | 删除面试 | 200/404 |
| GET | `/api/v1/resumes/:id/interviews` | 获取简历的所有面试 | 200 |
| POST | `/api/v1/interviews/:id/feedback` | 提交面评 | 200/400/404 |
| GET | `/api/v1/interviews/:id/feedback` | 获取面评 | 200/404 |
| POST | `/api/v1/resumes/:id/portfolios` | 创建作品集 | 200/400 |
| GET | `/api/v1/resumes/:id/portfolios` | 获取作品集列表 | 200 |
| DELETE | `/api/v1/portfolios/:id` | 删除作品集 | 200/404 |

---

## 8. 测试结果

| 测试项 | 结果 |
|--------|------|
| 健康检查 | ✅ 通过 |
| 创建面试 | ✅ 通过 |
| 获取面试详情 | ✅ 通过 |
| 列出简历的面试 | ✅ 通过 |
| 更新面试状态 | ✅ 通过 |
| 删除面试 | ✅ 通过 |
| 提交面评 | ✅ 通过 |
| 获取面评 | ✅ 通过 |
| 创建作品集 | ✅ 通过 |
| 列出作品集 | ✅ 通过 |
| 删除作品集 | ✅ 通过 |
| 错误处理 | ✅ 通过 |
