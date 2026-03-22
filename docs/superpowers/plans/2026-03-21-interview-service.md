# Interview Service 实现计划

## 概述

面试流程服务，管理面试安排、面评和作品集。

## 服务信息

- **端口**: 8082 (HTTP), 9082 (gRPC)
- **数据库**: PostgreSQL (interviews, feedbacks, portfolios 表)
- **依赖**: Resume Service (通过 gRPC 获取简历信息)

## 任务分解

### Task 1: 数据模型定义

**文件**: `internal/interview/model/`

```go
// interview.go
type Interview struct {
    ID           uuid.UUID `json:"id"`
    ResumeID     uuid.UUID `json:"resume_id"`
    Round        int       `json:"round"`
    Interviewer  string    `json:"interviewer"`
    ScheduledAt  time.Time `json:"scheduled_at"`
    Status       string    `json:"status"` // scheduled/completed/cancelled
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// feedback.go
type Feedback struct {
    ID             uuid.UUID `json:"id"`
    InterviewID    uuid.UUID `json:"interview_id"`
    Rating         int       `json:"rating"` // 1-5
    Content        string    `json:"content"`
    Recommendation string    `json:"recommendation"` // strong_yes/yes/no/strong_no
    CreatedAt      time.Time `json:"created_at"`
}

// portfolio.go
type Portfolio struct {
    ID        uuid.UUID `json:"id"`
    ResumeID  uuid.UUID `json:"resume_id"`
    Title     string    `json:"title"`
    FileURL   string    `json:"file_url"`
    FileType  string    `json:"file_type"` // pdf/link/image
    CreatedAt time.Time `json:"created_at"`
}
```

### Task 2: Repository 层

**文件**: `internal/interview/repository/`

接口定义:
```go
type InterviewRepository interface {
    Create(ctx context.Context, interview *model.Interview) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.Interview, error)
    ListByResumeID(ctx context.Context, resumeID uuid.UUID) ([]*model.Interview, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type FeedbackRepository interface {
    Create(ctx context.Context, feedback *model.Feedback) error
    GetByInterviewID(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error)
    Update(ctx context.Context, feedback *model.Feedback) error
}

type PortfolioRepository interface {
    Create(ctx context.Context, portfolio *model.Portfolio) error
    GetByResumeID(ctx context.Context, resumeID uuid.UUID) ([]*model.Portfolio, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### Task 3: Service 层

**文件**: `internal/interview/service/`

```go
type InterviewService interface {
    CreateInterview(ctx context.Context, req *CreateInterviewRequest) (*model.Interview, error)
    GetInterview(ctx context.Context, id uuid.UUID) (*model.Interview, error)
    ListInterviewsByResume(ctx context.Context, resumeID uuid.UUID) ([]*model.Interview, error)
    UpdateInterviewStatus(ctx context.Context, id uuid.UUID, status string) error
    DeleteInterview(ctx context.Context, id uuid.UUID) error
}

type FeedbackService interface {
    SubmitFeedback(ctx context.Context, req *SubmitFeedbackRequest) (*model.Feedback, error)
    GetFeedback(ctx context.Context, interviewID uuid.UUID) (*model.Feedback, error)
}

type PortfolioService interface {
    CreatePortfolio(ctx context.Context, req *CreatePortfolioRequest) (*model.Portfolio, error)
    ListPortfolios(ctx context.Context, resumeID uuid.UUID) ([]*model.Portfolio, error)
    DeletePortfolio(ctx context.Context, id uuid.UUID) error
}
```

### Task 4: HTTP Handler 层

**文件**: `internal/interview/handler/`

API 端点:
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/interviews` | 创建面试 |
| GET | `/api/v1/interviews/:id` | 获取面试详情 |
| GET | `/api/v1/resumes/:id/interviews` | 某简历的所有面试 |
| PUT | `/api/v1/interviews/:id/status` | 更新面试状态 |
| DELETE | `/api/v1/interviews/:id` | 删除面试 |
| POST | `/api/v1/interviews/:id/feedback` | 提交面评 |
| GET | `/api/v1/interviews/:id/feedback` | 获取面评 |
| POST | `/api/v1/resumes/:id/portfolios` | 上传作品集 |
| GET | `/api/v1/resumes/:id/portfolios` | 获取作品集列表 |
| DELETE | `/api/v1/portfolios/:id` | 删除作品集 |

### Task 5: 服务入口

**文件**: `cmd/interview-service/main.go`

- 初始化 Gin 路由
- 数据库连接
- 注册所有 Handler
- 健康检查端点
- 嵌入测试前端页面
- 监听 0.0.0.0:8082

## 验收标准

1. 所有 API 端点正常响应
2. 面试创建、查询、状态更新功能正常
3. 面评提交和查询功能正常
4. 作品集上传、查询、删除功能正常
5. 提供前端测试界面

---

## 招聘流程设计 (待实现)

### 整体流程状态

候选人级别的招聘流程状态（建议添加到 `resumes` 表或新建 `hiring_processes` 表）：

```
┌─────────────┐
│   pending   │  ← 简历初筛阶段
└──────┬──────┘
       │
       ▼ pass
┌─────────────┐
│  screening  │  ← 二次筛选/技术评估
└──────┬──────┘
       │
       ▼ pass
┌─────────────┐
│ interviewing│  ← 面试流程中 (可多轮)
└──────┬──────┘
       │
       ├─────────────────┬─────────────────┐
       ▼                 ▼                 ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   offered   │   │   rejected   │   │  withdrawn  │
└─────────────┘   └─────────────┘   └─────────────┘
   (发offer)        (淘汰)          (候选人撤回)
```

### 状态转换规则

| From | To | Trigger |
|------|-----|---------|
| `pending` | `screening` | 简历通过初筛 |
| `pending` | `rejected` | 简历不符合要求 |
| `screening` | `interviewing` | 通过技术评估 |
| `screening` | `rejected` | 技术评估不通过 |
| `interviewing` | `offered` | 所有面试通过，发放offer |
| `interviewing` | `rejected` | 面试未通过 |
| `interviewing` | `interviewing` | 进入下一轮面试 |
| `offered` | `hired` | 候选人接受offer |
| `offered` | `rejected` | 候选人拒绝offer |
| * | `withdrawn` | 候选人主动撤回 |

### 面评推荐与流程推进

| Recommendation | Action |
|----------------|--------|
| `strong_yes` | 自动推进到下一轮 |
| `yes` | 需要下一轮确认，或推进 |
| `no` | 终止流程 (rejected) |
| `strong_no` | 立即终止流程 (rejected) |

### 数据模型变更

```go
// 新增字段到 resumes 表或新建 hiring_processes 表
type HiringProcess struct {
    ID           uuid.UUID
    ResumeID     uuid.UUID
    Status       string    // pending/screening/interviewing/offered/hired/rejected/withdrawn
    CurrentRound int        // 当前面试轮次 (0=筛选中)
    TotalRounds  int        // 计划面试轮次
    RejectReason string    // 淘汰原因
    OfferAmount  float64   // offer金额
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 新增 API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/resumes/:id/process` | 获取招聘流程状态 |
| POST | `/api/v1/resumes/:id/process/advance` | 推进到下一阶段 |
| POST | `/api/v1/resumes/:id/process/reject` | 淘汰候选人 |
| POST | `/api/v1/resumes/:id/process/offer` | 发放offer |
| POST | `/api/v1/resumes/:id/process/hire` | 确认入职 |
