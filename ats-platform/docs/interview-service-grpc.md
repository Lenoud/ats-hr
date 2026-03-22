# Interview Service gRPC API 文档

## 服务信息

- **HTTP 端口**: 8082
- **gRPC 端口**: 9091

## gRPC 服务列表

| 服务名 | 描述 |
|--------|------|
| `interview.InterviewService` | 面试管理 |
| `interview.FeedbackService` | 面试反馈管理 |
| `interview.PortfolioService` | 作品集管理 |

## 测试命令

### 列出所有服务

```bash
grpcurl -plaintext localhost:9091 list
```

**输出:**
```
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
interview.FeedbackService
interview.InterviewService
interview.PortfolioService
```

### 查看服务方法

```bash
# InterviewService
grpcurl -plaintext localhost:9091 describe interview.InterviewService

# FeedbackService
grpcurl -plaintext localhost:9091 describe interview.FeedbackService

# PortfolioService
grpcurl -plaintext localhost:9091 describe interview.PortfolioService
```

### 查看请求/响应结构

```bash
grpcurl -plaintext localhost:9091 describe interview.CreateInterviewRequest
grpcurl -plaintext localhost:9091 describe interview.Interview
```

## InterviewService 方法

### CreateInterview - 创建面试

```bash
grpcurl -plaintext -d '{
  "resume_id": "<有效的resume UUID>",
  "round": 1,
  "interviewer": "张三",
  "scheduled_at": 1735689600
}' localhost:9091 interview.InterviewService/CreateInterview
```

**字段说明:**
- `resume_id`: 关联的简历 ID (必填)
- `round`: 面试轮次，从 1 开始 (必填)
- `interviewer`: 面试官姓名
- `scheduled_at`: 预约时间 (Unix 时间戳)

### GetInterview - 获取面试详情

```bash
grpcurl -plaintext -d '{
  "id": "<面试UUID>"
}' localhost:9091 interview.InterviewService/GetInterview
```

### ListInterviews - 查询面试列表

```bash
grpcurl -plaintext -d '{
  "page": 1,
  "page_size": 10,
  "resume_id": "<简历UUID>",
  "status": "scheduled"
}' localhost:9091 interview.InterviewService/ListInterviews
```

**字段说明:**
- `page`: 页码，从 1 开始
- `page_size`: 每页数量，最大 100
- `resume_id`: 按简历 ID 筛选
- `status`: 按状态筛选 (scheduled/completed/cancelled)

### UpdateInterviewStatus - 更新面试状态

```bash
grpcurl -plaintext -d '{
  "id": "<面试UUID>",
  "status": "completed"
}' localhost:9091 interview.InterviewService/UpdateInterviewStatus
```

**状态值:**
- `scheduled` - 已安排
- `completed` - 已完成
- `cancelled` - 已取消

## FeedbackService 方法

### CreateFeedback - 提交面试反馈

```bash
grpcurl -plaintext -d '{
  "interview_id": "<面试UUID>",
  "rating": 4,
  "content": "候选人技术能力扎实，沟通良好",
  "recommendation": "hire"
}' localhost:9091 interview.FeedbackService/CreateFeedback
```

**字段说明:**
- `interview_id`: 关联的面试 ID (必填)
- `rating`: 评分 1-5 (必填)
- `content`: 反馈内容
- `recommendation`: 推荐意见 (hire/no_hire/consider)

### GetFeedbackByInterview - 获取面试反馈

```bash
grpcurl -plaintext -d '{
  "interview_id": "<面试UUID>"
}' localhost:9091 interview.FeedbackService/GetFeedbackByInterview
```

## PortfolioService 方法

### CreatePortfolio - 创建作品集

```bash
grpcurl -plaintext -d '{
  "resume_id": "<简历UUID>",
  "title": "项目展示",
  "file_url": "https://example.com/portfolio.pdf",
  "file_type": "pdf"
}' localhost:9091 interview.PortfolioService/CreatePortfolio
```

**支持的文件类型:** pdf, doc, docx, ppt, pptx, link

### ListPortfolios - 查询作品集列表

```bash
grpcurl -plaintext -d '{
  "page": 1,
  "page_size": 10,
  "resume_id": "<简历UUID>"
}' localhost:9091 interview.PortfolioService/ListPortfolios
```

## 错误码

| gRPC Code | 说明 |
|-----------|------|
| `InvalidArgument` | 参数错误 (如无效的 UUID 格式) |
| `NotFound` | 资源不存在 |
| `AlreadyExists` | 资源已存在 (如重复提交反馈) |
| `FailedPrecondition` | 状态转换无效 |
| `Internal` | 服务器内部错误 |
| `Unimplemented` | 方法未实现 |

## Proto 文件位置

```
proto/interview.proto
```

## 重新生成 Proto

```bash
protoc --go_out=. --go-grpc_out=. proto/interview.proto
```
