# Service Discovery Abstraction Design

## Goal

把当前分散在各服务 `main.go` 和 `gateway` 中的 Consul 注册/发现命名约定收敛成一套共享抽象，避免 HTTP / gRPC 服务名、注册 ID、gateway discover name 再次漂移。

本轮目标不是新增平台能力，而是把现有约定正式化、单点化。

## Scope

本轮只覆盖以下内容：

- 逻辑服务名定义
- 协议枚举定义（`http` / `grpc`）
- `逻辑服务名 + 协议 -> Consul discover name` 的统一生成
- 服务注册与注销的共享入口
- gateway 路由从“硬编码 discover name”改为“逻辑服务名 + 协议”

本轮不包含：

- 负载均衡策略抽象
- 动态路由配置中心
- Consul watch / 缓存 / 本地订阅
- 非 Consul 的服务发现实现
- 为 `search-service` 补 gRPC 能力

## Current Problems

当前仓库虽然已经统一到了 `*-http` / `*-grpc` 的命名方向，但约定仍散落在多个地方：

- `resume-service` / `interview-service` / `search-service` 的 `main.go` 里分别拼接 `-http` / `-grpc`
- `gateway` 路由表直接硬编码最终 discover name，例如 `resume-service-http`
- `internal/shared/consul/register.go` 只负责最低层注册，没有表达“逻辑服务名 + 协议”这一层语义
- 文档描述了命名约定，但代码没有一个统一的抽象边界去承载它

这导致三个风险：

1. 命名规则变更时，需要同时修改服务启动代码、gateway、文档
2. 新增服务时，容易重复复制旧逻辑并产生新漂移
3. gateway 路由表表达的是底层 discover name，而不是业务意图

## Design Decision

采用“共享服务标识 + 协议感知的注册/发现 helper”方案。

核心原则：

- gateway 依赖逻辑服务名，而不是最终 Consul service name
- 服务启动代码依赖共享 helper 生成 discover name 和 service ID
- HTTP / gRPC 命名规则只保留一份实现

## Proposed Model

### 1. 共享逻辑服务名

在 `internal/shared/consul/` 中定义逻辑服务名常量，例如：

- `resume-service`
- `interview-service`
- `search-service`

这些常量只表达“业务服务身份”，不表达协议。

### 2. 协议枚举

定义协议枚举：

- `ProtocolHTTP`
- `ProtocolGRPC`

该枚举只用于生成 discover name 和表达注册/发现意图，不承担传输实现逻辑。

### 3. discover name 生成规则

通过统一函数生成 Consul service name：

- `ServiceName("resume-service", ProtocolHTTP)` -> `resume-service-http`
- `ServiceName("resume-service", ProtocolGRPC)` -> `resume-service-grpc`
- `ServiceName("search-service", ProtocolHTTP)` -> `search-service-http`

这条规则成为唯一可信来源，服务代码和 gateway 都不得再手工拼接字符串。

### 4. 注册抽象

在 `internal/shared/consul/` 中提供更高层 helper：

- 按 `baseName + protocol + ip + port + instanceID` 注册单个端点
- 返回统一的 service ID
- 注销时使用同一套 endpoint 描述回收

对服务启动层来说，`main.go` 只需要声明：

- 自己的逻辑服务名
- 要暴露哪些协议端点
- 各端点对应端口

而不是再自己拼 `httpServiceName` / `grpcServiceName`。

### 5. gateway 发现抽象

gateway 的路由表改为只表达：

- 路径前缀
- 逻辑服务名
- 所需协议（当前均为 `http`）

gateway 在运行时通过共享 helper 将逻辑服务名解析为最终 discover name。

这样未来如果 discover name 规则调整，gateway 无需修改路由表。

## File Boundary Plan

### Modify

- `ats-platform/internal/shared/consul/register.go`
  - 增加逻辑服务名、协议枚举、discover name 生成和高层注册 helper
- `ats-platform/cmd/resume-service/main.go`
  - 改为通过共享 helper 注册 HTTP / gRPC 端点
- `ats-platform/cmd/interview-service/main.go`
  - 改为通过共享 helper 注册 HTTP / gRPC 端点
- `ats-platform/cmd/search-service/main.go`
  - 改为通过共享 helper 注册 HTTP 端点
- `ats-platform/cmd/gateway/main.go`
  - 路由表改为逻辑服务名 + 协议，而不是最终 discover name

### Optional Split

如果 `register.go` 体积增长过快，可以拆分为：

- `naming.go`：协议枚举、逻辑服务名、discover name 生成
- `register.go`：Consul register/deregister helper

本轮是否拆分，以清晰度为准，不强制。

## Gateway Example

当前：

- `/resumes` -> `resume-service-http`
- `/interviews` -> `interview-service-http`
- `/search` -> `search-service-http`

改造后：

- `/resumes` -> `resume-service` + `http`
- `/interviews` -> `interview-service` + `http`
- `/search` -> `search-service` + `http`

gateway 内部再通过共享 helper 转换为最终 discover name。

## Service Example

当前 `resume-service` 启动代码需要自己维护：

- `httpServiceName := cfg.ServiceName + "-http"`
- `grpcServiceName := cfg.ServiceName + "-grpc"`

改造后，启动代码只声明：

- `base service = resume-service`
- register `http` on `HTTPPort`
- register `grpc` on `GRPCPort`

这样服务代码只表达“我要暴露哪些协议”，不表达 discover name 拼接细节。

## Risks

### Risk 1: 抽象过重

如果把服务发现、路由、负载均衡一起收进来，会让一个简单问题变复杂。

缓解：

- 本轮只抽命名和注册/发现入口
- 不引入 watch、缓存、策略对象

### Risk 2: 共享层承担过多业务语义

如果把 gateway 路由逻辑也塞进共享层，会让边界混乱。

缓解：

- 共享层只负责“逻辑服务名 + 协议 -> discover name”
- 路由前缀与上游选择仍留在 gateway

### Risk 3: search-service 没有 gRPC，抽象后被误解为必须双协议

缓解：

- 协议抽象允许单协议服务存在
- `search-service` 本轮仍然只注册 HTTP
- 文档明确“统一命名约定 != 每个服务都必须同时实现两种协议”

## Success Criteria

- 代码中不再手工拼接 `-http` / `-grpc`
- gateway 路由表不再依赖最终 Consul discover name
- 服务注册与注销都使用同一套共享命名规则
- `resume-service` / `interview-service` / `search-service` 的 Consul 命名保持一致
- 现有 gateway 回归链路不被破坏
