# P0 Completion Design

## Goal

完成当前阶段的 P0 收尾工作，在不处理硬编码便利性、不补测试的前提下，解决三个核心问题：

- `interview-service` 的 gRPC 能力与真实实现不一致
- `resume-service -> search-service` 的事件契约仍然隐式耦合
- `gateway` 需要升级为支持 Consul 动态发现的网关

## Scope

本轮只包含以下三项：

1. `interview-service gRPC` 对齐
2. `search` 事件契约固化
3. `gateway` 动态发现改造

本轮不包含：

- 硬编码默认值或敏感配置清理
- 测试补充
- 业务模型扩展
- 统一错误语义的全仓治理
- 网关鉴权、限流、熔断等增强能力

## Current State

### 1. interview-service gRPC

当前 `proto/interview.proto` 暴露的方法比 service 层真实能力更满：

- `UpdateInterview` 仅返回“更新后的对象视图”，并没有完成持久化更新
- `GetFeedback` 已定义但服务端直接返回 `Unimplemented`
- 其他接口虽可用，但文档、proto、实现并未完全保持同一语义层级

结果是调用方很容易误判哪些能力是正式可用的。

### 2. search 事件契约

`resume-service` 通过共享事件发布器写入 Redis Stream，`search-service` 再消费这些事件并更新 Elasticsearch。

但当前契约仍是“代码隐式约定”：

- action 字符串散落在发布端与消费端
- payload 结构由发布侧自由生成、消费侧自行猜测
- 事件类型和字段边界没有被明确声明为共享契约

### 3. gateway

`gateway` 目前是静态路径代理：

- 服务地址写死在代码中
- 已有 Consul 注册，但 `gateway` 并未消费服务发现结果
- 文档虽然已说明它是轻量网关，但新的需求要求它支持动态发现

## Design Decisions

### Decision 1: gRPC 对齐采取“收缩到真实能力”为主

本轮不强行把 `interview-service` 的所有 proto 方法都补成完整业务能力，而是优先让“对外承诺”和“真实能力”一致。

具体策略：

- 对已经具备稳定 service 支撑的方法保留
- 对只有半实现的方法，要么补到可用，要么从 proto 和文档中移除/降级
- 不再保留“接口存在但实际上只是占位”的状态

原因：

- 这符合 P0 收尾目标
- 避免本轮工作扩展成完整业务能力建设
- 能最快降低误用风险

### Decision 2: 事件契约进入 shared contract

将 `resume-service -> search-service` 的事件语义收敛到共享定义中。

契约至少应明确：

- action 常量集合
- 每类事件 payload 的字段结构
- 哪些事件必须带 payload，哪些不需要
- `search-service` 对各类事件的处理语义

实现原则：

- 发布方和消费方使用同一套共享结构或转换入口
- 不再允许 action 名和 payload 结构在两端分别硬编码一份

### Decision 3: gateway 升级为动态发现网关，但仍保持轻量职责

`gateway` 本轮需要接入 Consul 做服务发现，但职责仍然保持克制：

- 路径前缀判断仍保留
- 目标实例地址通过 Consul 动态解析
- 不在本轮引入鉴权、熔断、负载均衡策略扩展

这意味着：

- `gateway` 从“静态代理”升级为“动态发现代理”
- 但它仍不是完整的 API 管理层或 BFF

## Proposed Architecture

### A. interview-service gRPC alignment

涉及层次：

- `proto/interview.proto`
- `internal/interview/grpc/server.go`
- 相关文档文件

目标状态：

- proto 中保留的方法都具备真实可解释的服务端行为
- 不再出现“看起来是正式接口，实际上只是未落地 stub”的方法
- 文档与 proto、实现严格一致

### B. shared resume event contract

涉及层次：

- `internal/shared/events/`
- `resume-service` 发布事件的调用点
- `search-service` 消费事件与 payload 解析逻辑
- 相关设计文档

目标状态：

- action 与 payload 成为共享契约
- 发布端和消费端不再各自定义各自理解
- 未来修改搜索索引字段时，能够定位到明确契约入口

### C. dynamic discovery gateway

涉及层次：

- `cmd/gateway/main.go`
- `internal/shared/consul/`
- 相关设计与开发文档

目标状态：

- 路由仍按 `/resumes`、`/interviews`、`/portfolios`、`/search` 这种前缀决定服务名
- 具体目标地址不再来自硬编码 map，而来自 Consul 查询结果
- 若目标服务不可用，返回清晰的网关错误

## File Change Map

### Modify

- `ats-platform/proto/interview.proto`
- `ats-platform/internal/interview/grpc/server.go`
- `ats-platform/internal/shared/events/publisher.go`
- `ats-platform/internal/shared/events/consumer.go`
- `ats-platform/cmd/search-service/main.go`
- `ats-platform/internal/search/service/search_service.go`
- `ats-platform/cmd/gateway/main.go`
- `ats-platform/docs/interview-service-grpc.md`
- `docs/superpowers/specs/2026-03-20-ats-platform-design.md`
- `docs/SERVICE_DEVELOPMENT_GUIDE.md`

### Create or Split If Helpful

- 可新增共享事件契约文件到 `ats-platform/internal/shared/events/`
- 可新增 gateway 服务发现辅助文件到 `ats-platform/internal/shared/consul/` 或 `cmd/gateway/` 邻近位置

## Execution Strategy

采用一个总规划、三个独立子任务的方式执行：

1. `interview-service gRPC` 对齐
2. `search` 事件契约固化
3. `gateway` 动态发现改造

这样做的原因：

- 三项共享系统背景，但改动面相对独立
- 可以避免一个大 commit 混杂协议、事件和网关逻辑
- 每项都可以独立评审

## Risks

### Risk 1: gRPC 收缩会影响已有调用方

缓解方式：

- 优先收缩那些明确未完整实现的方法
- 文档与 proto 同步修改，避免静默破坏

### Risk 2: 事件契约收敛会引入发布端和消费端兼容问题

缓解方式：

- 保留当前 action 语义不大改名
- 优先把结构显式化，而不是重设计整套事件系统

### Risk 3: gateway 动态发现会增加运行时不稳定性

缓解方式：

- 保留现有路径前缀路由逻辑
- 仅把目标地址解析替换为 Consul 查询
- 为服务不可达、实例为空等情况提供明确错误响应

## Success Criteria

- `interview-service` 的 gRPC proto、实现、文档不再互相冲突
- `resume-service -> search-service` 的 action 与 payload 契约被共享定义并落地到代码
- `gateway` 能通过 Consul 动态解析目标服务实例并完成转发
- 本轮不引入超出 P0 范围的新平台能力
