# ATS Platform 服务开发指南

本指南说明 ATS Platform 当前服务开发和本地联调的基础约定，供 `resume-service`、`interview-service`、`search-service` 与 `gateway` 参考。

## 适用范围

当前仓库中的服务状态：

- `resume-service`：HTTP + gRPC
- `interview-service`：HTTP + gRPC
- `search-service`：HTTP
- `gateway`：HTTP，负责首页、健康聚合、代理，以及主链路调试编排页

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

## 服务注册与发现

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

服务端优先使用：

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

## Gateway 路由约定

`gateway` 当前仍是轻量路径代理，按 `/api/v1/*` 下的路径前缀选择目标服务。

当前规则：

- `/resumes` 默认转发到 `resume-service`
- `/interviews` 转发到 `interview-service`
- `/portfolios` 转发到 `interview-service`
- `/search` 转发到 `search-service`
- `/resumes/:id/interviews` 与 `/resumes/:id/portfolios` 这类嵌套路由，优先转发到 `interview-service`

维护要求：

- route table 只能表达“逻辑服务名 + 协议”
- 不要在 gateway 运行逻辑里硬编码最终 Consul service name

## Gateway 前端约定

`gateway` 首页当前定位为 orchestration console，而不是简单导航页。

它负责：

- 通过真实 gateway 路由执行主链路调试
- 展示步骤状态、运行上下文、最近请求/响应
- 展示 gateway 自己的健康聚合和 upstream 发现结果

当前主闭环为：

1. 创建简历
2. 更新状态，并在可行时尝试解析
3. 创建面试
4. 搜索验证
5. 汇总 gateway 健康与链路结果

维护要求：

- 页面继续使用静态 HTML + 原生 JS
- 业务请求必须走 gateway 自己的 `/api/v1/*` 路由
- 不新增只服务页面 demo 的 orchestration backend API

## 本地开发约定

### 推荐启动方式

```bash
cd ats-platform
./scripts/run-services.sh --no-infra --gateway
```

`Makefile` 里的多服务命令应只作为脚本薄包装，不再复制编排逻辑。

### 默认本地拓扑

`run-services.sh` 当前默认按“Docker Consul + 宿主机服务”处理本地联调：

- 默认 `CONSUL_HOST=127.0.0.1`
- 默认数据库、Redis、MinIO、Elasticsearch 走 `127.0.0.1`
- 如果未显式设置 `SERVICE_ADDRESS`，脚本会默认导出 `host.docker.internal`

这使得：

- 宿主机运行的服务可以被 Docker 内 Consul 做健康检查
- gateway 在宿主机上能通过 Consul 解析并访问这些 HTTP 实例

### 重复启动保护

当前本地编排脚本和服务注册逻辑会主动防止开发环境越来越脏：

- `run-services.sh` 在启动服务前会检查关键端口是否空闲
- 若端口已被占用，脚本会直接失败，而不是继续启动并制造更多冲突
- 服务在注册 Consul 前，会先清理同一逻辑服务、同一地址、同一端口的旧实例

这意味着：

- 本地多次运行脚本时，不应再持续累积同地址同端口的历史 Consul 注册
- 如果脚本提示端口占用，应先停掉旧进程，再重新启动，而不是重复执行脚本

### `host.docker.internal` 兼容规则

在宿主机本地开发场景下：

- 服务注册到 Consul 时可使用 `SERVICE_ADDRESS=host.docker.internal`
- gateway 如果从 Consul 解析到 `host.docker.internal` 但当前环境不可解析，会回退到 `127.0.0.1`

## 文档分类规则

- 正式文档：放在 `ats-platform/docs/`
- 测试/实验文档：放在 `ats-platform/docs/tests/`
- 设计/计划文档：放在仓库根目录 `docs/superpowers/`

新增或更新文档时，应保证：

- 正式文档描述当前实现
- 测试文档描述验证步骤
- 设计/计划文档保留历史上下文，不替代正式说明

## 相关文档

- `ats-platform/docs/README.md`
- `ats-platform/docs/interview-service-grpc.md`
- `ats-platform/docs/tests/interview-service-api-test.md`
