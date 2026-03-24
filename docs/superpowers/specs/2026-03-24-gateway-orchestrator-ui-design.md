# Gateway Orchestrator UI Design

## Goal

将 `gateway` 的当前简易首页重构为一个可直接调试整条 ATS 主链路的前端编排台，同时保持与 `resume-service`、`interview-service`、`search-service` 现有静态页面统一的视觉语言。

本次设计目标不是把 `gateway` 做成第四个完整业务后台，而是让它成为：

- 统一入口
- 链路健康观察面板
- 真实经过 gateway 的主流程调试台

## Scope

本次只覆盖 `gateway` 首页前端重构，包括：

- 页面信息架构
- 视觉风格统一
- 前端交互与本地状态管理
- 通过 gateway 路由执行主闭环调试

本次不包含：

- 新增后端 orchestration API
- 调整现有业务服务接口定义
- 把 gateway 扩展成完整 CRUD 控制台
- 引入前端框架或构建系统

## Current State

当前 `gateway` 首页具备以下特点：

- HTML 内联在 `cmd/gateway/main.go`
- 只有简单标题、三个服务状态卡片和三个链接
- 能表达“统一入口”的概念，但不能调试实际链路
- 页面结构和 `resume-service` / `interview-service` / `search-service` 的静态控制台风格不一致

相比之下，其他三个模块已经形成较清晰的前端规范：

- 使用衬线标题和有明显氛围感的渐变背景
- 统一采用 `hero + panel + card` 结构
- 页面内直接通过 `fetch` 调真实 API
- 支持 toast、结果预览、局部状态反馈

因此 `gateway` 当前的主要问题不是“缺少更漂亮的首页”，而是：

1. 无法承担链路调试入口的职责
2. 与其他三个模块的页面语言不统一
3. 重要的 gateway 运行时信息没有被展示出来

## Design Principles

### 1. Gateway Is An Orchestrator, Not Another CRUD App

`gateway` 页面的角色应当是“流程编排台”：

- 负责串联主闭环
- 负责展示各步状态
- 负责展示请求/响应和链路健康

而不是重复实现 `resume-service` 或 `interview-service` 的完整操作台。

### 2. Debug Through Real Gateway Routes

页面上的主流程必须通过 `gateway` 当前已有的 `/api/v1/*` 转发路径执行，而不是调用新增的内部聚合接口。

这样用户调试到的才是：

- 真实 gateway 转发行为
- 真实服务发现链路
- 真实上下游响应

而不是一套只服务于前端 demo 的旁路逻辑。

### 3. One Main Flow, Optional Step Replay

页面的主体验应当是“一键跑通主闭环”，但同时保留每一步单独执行的能力，方便局部重试和故障定位。

这意味着页面交互采用：

- 主按钮驱动的串行流程
- 步骤卡片驱动的局部调试

而不是只做其中一类。

### 4. Visual Language Must Match Existing Service Consoles

`gateway` 不应照搬任一现有页面，但需要明确对齐其共同规范：

- 衬线标题
- 暖色 / 青绿色渐变背景
- 圆角玻璃面板
- `hero + panel + card` 结构
- toast 提示
- 明确的信息层级和结果预览

## Target User Flow

本次确认的主闭环为：

1. 创建简历
2. 更新简历状态，并在可行时触发解析
3. 创建面试
4. 用搜索接口验证结果可见性
5. 展示 gateway 健康聚合与本轮链路上下文

### Step 2 Behavior

第二步不应成为整条 demo 的脆弱点。

推荐行为：

- 优先执行状态更新
- 若解析前置条件满足，再尝试触发解析
- 若解析无法执行或失败，可将其标记为“降级完成”，但主链路仍可继续

这样可以避免某些高波动能力阻塞整条 gateway 调试链路。

## Information Architecture

页面采用以下结构：

### 1. Hero

顶部 Hero 负责定义页面定位，而不是只放欢迎语。

应包含：

- `ATS Gateway Orchestrator` 标题
- 一句话说明：统一入口、链路调试、服务发现观测
- 顶部快捷动作：
  - `Run Full Demo`
  - `Refresh Health`
  - `Reset Context`
- 若适合，也可放 2 到 3 个状态 pill：
  - 当前环境
  - gateway 健康状态
  - 最后一次执行时间

### 2. Flow Overview

用 5 个连续步骤节点展示当前主闭环：

- Create Resume
- Update / Parse Resume
- Create Interview
- Search Verify
- Gateway Summary

每个节点显示：

- `idle`
- `running`
- `success`
- `error`
- `degraded`（仅用于第二步）

### 3. Step Cards

为每一步提供独立卡片。

每张卡片包含：

- 该步骤作用说明
- 必要输入项
- 当前使用的上下文值
- `Run Step` 按钮
- 最近一次结果摘要

这些卡片需要支持：

- 单独运行
- 被 `Run Full Demo` 串行驱动

### 4. Runtime Context

专门展示本轮链路上下文，而不是分散在每个卡片里。

建议字段：

- `resumeId`
- `interviewId`
- `searchQuery`
- `lastError`
- `lastSuccessStep`

### 5. Request / Response Inspector

统一展示最近一次执行的请求与响应。

建议包括：

- Request method
- Request URL
- Request payload
- Response status
- Response body

作用是让用户不必打开 devtools 就能排查链路问题。

### 6. Service Health Panel

展示 gateway 当前对各服务的理解结果：

- resume / interview / search / gateway 状态
- 当前发现到的 upstream 地址
- 状态码或错误摘要

这个面板既服务于页面调试，也服务于服务发现验证。

## Interaction Design

### 1. Run Full Demo

点击后串行执行主闭环的 5 个步骤。

规则：

- 严格按顺序推进
- 任一步失败则停止
- 停止后将失败步骤标红
- 错误同时进入 Inspector 和 `lastError`

### 2. Run Step

每个步骤卡片可单独执行。

用途：

- 局部重试
- 参数修改后重跑
- 不重复执行整条链路

### 3. Dependency Guardrails

依赖上下文的步骤必须做前置校验。

例如：

- 没有 `resumeId` 不允许创建面试
- 没有上下文数据时不允许直接跑汇总步骤
- 搜索步骤没有关键词时，应提供默认查询或明确报错

### 4. Result Surfacing

信息呈现分层：

- 卡片内只显示摘要
- Inspector 显示完整数据
- toast 只承担轻量提示

不应让 toast 成为唯一反馈来源。

## Technical Design

### 1. Move HTML Out Of `main.go`

当前内联 HTML 应迁移为独立静态文件：

- `ats-platform/cmd/gateway/static/index.html`

并通过：

```go
//go:embed static/index.html
```

嵌入。

这样更符合其他服务的现有结构，也降低后续维护成本。

### 2. Frontend State Model

页面内部维护一份轻量状态对象，包含：

- flow step statuses
- ids and current context
- latest request
- latest response
- latest errors
- service health snapshot

该状态只服务当前页面，不需要抽象成框架级状态管理。

### 3. API Calling Rules

页面通过 gateway 本身的 `/api/v1/*` 路由发起业务请求。

允许直连的只有：

- gateway 自己的 `/health`
- gateway 首页资源本身

不应绕过 gateway 直接调用其他服务 HTTP 端口来完成主 demo。

### 4. No New Backend Orchestration Endpoint

本次不新增：

- `/orchestrate`
- `/debug/run-demo`
- 任何聚合工作流 API

原因：

- 会把 gateway 从代理演变成 workflow backend
- 会让调试结果偏离真实转发路径
- 会扩大本次前端重构范围

## Risks

### Risk 1: Page Becomes Too Big

如果把所有业务能力都塞进 `gateway`，页面会快速膨胀。

处理方式：

- 只支持主闭环最小集
- 复杂操作仍留在各服务自己的页面

### Risk 2: Full Demo Is Fragile

链路中的单个波动能力可能导致整条 demo 经常失败。

处理方式：

- 第二步允许降级成功
- 错误信息必须明确指向具体步骤

### Risk 3: Visual Drift

如果 gateway 页面只追求“更炫”，会和现有三个页面脱节。

处理方式：

- 复用其共同视觉语言
- 但通过流程状态和 Inspector 区分 gateway 的产品角色

## Success Criteria

- gateway 首页视觉上与另外三个模块统一
- 页面主角色清晰是“流程编排台”
- `Run Full Demo` 能跑通或在明确步骤失败处停止
- 每一步均可单独执行
- 页面能持续显示本轮关键上下文和最近请求/响应
- 用户无需离开 gateway 页面即可判断问题属于：
  - 服务健康
  - gateway 转发
  - 业务请求失败
  - 搜索链路不可见
