# ATS Platform 服务开发指南

本指南说明 ATS Platform 当前服务开发的基础约定，供 `resume-service`、`interview-service`、`search-service` 与 `gateway` 参考。

## 适用范围

当前仓库中的服务现状：

- `resume-service`：HTTP + gRPC
- `interview-service`：HTTP + gRPC
- `search-service`：HTTP
- `gateway`：HTTP，负责首页、健康聚合与代理

## 分层结构

平台服务遵循统一分层：

```text
main.go
  -> handler
  -> service
  -> repository
  -> shared infrastructure
```

### 各层职责

- `handler`
  - HTTP / gRPC 请求处理
  - 参数校验
  - 响应格式化
- `service`
  - 业务逻辑
  - 领域规则
  - 事件发布/消费协调
- `repository`
  - 数据持久化与查询封装
- `internal/shared`
  - 共享基础设施，如数据库、Consul、事件、日志、中间件

## 目录约定

```text
ats-platform/
├── cmd/
│   └── {service}/
├── internal/
│   ├── {domain}/
│   │   ├── handler/
│   │   ├── service/
│   │   ├── repository/
│   │   ├── model/
│   │   └── grpc/
│   └── shared/
│       ├── consul/
│       ├── database/
│       ├── events/
│       ├── logger/
│       ├── middleware/
│       ├── pb/
│       └── storage/
└── proto/
```

## 服务发现约定

服务发现逻辑统一收敛在：

- `ats-platform/internal/shared/consul/naming.go`
- `ats-platform/internal/shared/consul/register.go`

### 逻辑服务名

- `resume-service`
- `interview-service`
- `search-service`

### 协议

- `http`
- `grpc`

### 最终 discover name

通过以下 helper 生成：

```go
consul.ServiceName(baseName, protocol)
```

示例：

- `resume-service-http`
- `resume-service-grpc`
- `interview-service-http`
- `search-service-http`

### Endpoint 注册模型

服务端应优先使用：

```go
consul.Endpoint{
    BaseName: consul.ResumeServiceBaseName,
    Protocol: consul.ProtocolHTTP,
    IP:       ip,
    Port:     httpPort,
}
```

并通过：

```go
consulClient.RegisterEndpoint(endpoint, instanceID)
```

完成注册。

## 本地开发 Consul 约定

当：

- 服务运行在宿主机
- Consul 运行在 Docker

推荐使用：

```bash
SERVICE_ADDRESS=host.docker.internal
```

如果不设置 `SERVICE_ADDRESS`，服务会回退到自动探测出口 IP。

`gateway` 在本地开发场景下，如果从 Consul 发现到 `host.docker.internal` 但当前环境无法解析，会回落到 `127.0.0.1`。

## 运行方式

### 推荐本地启动

```bash
cd ats-platform
make run-all
```

该方式默认按“Docker Consul + 宿主机服务”运行模型处理。

如果 Consul 在宿主机运行：

```bash
cd ats-platform
make run-all-host-consul
```

### 单服务启动

```bash
cd ats-platform
go run ./cmd/resume-service
go run ./cmd/interview-service
go run ./cmd/search-service
go run ./cmd/gateway
```

## 文档分类规则

- 正式文档：放在 `ats-platform/docs/`
- 测试/实验文档：放在 `ats-platform/docs/tests/`
- 设计/计划文档：放在 `ats-platform/docs/superpowers/`

新增或更新文档时，应保证：

- 正式文档描述当前实现
- 测试文档描述验证步骤
- 设计/计划文档保留历史上下文，不替代正式说明

## 相关文档

- `ats-platform/docs/README.md`
- `ats-platform/docs/consul-integration.md`
- `ats-platform/docs/resume-service-api.md`
- `ats-platform/docs/tests/interview-service-api-test.md`
