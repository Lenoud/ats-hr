# Consul 服务发现集成文档

## 概述

ATS Platform 当前使用 HashiCorp Consul 做服务注册与发现。

现状已经完成：

- `resume-service` 注册 HTTP 与 gRPC endpoint
- `interview-service` 注册 HTTP 与 gRPC endpoint
- `search-service` 注册 HTTP endpoint
- `gateway` 通过 Consul 做 HTTP 服务发现与代理

本文档只描述当前仓库实现与本地开发约定，不再保留已经完成的 TODO 型描述。

## 当前注册约定

### 逻辑服务名

- `resume-service`
- `interview-service`
- `search-service`

### 协议

- `http`
- `grpc`

### 最终 Consul service name

最终注册名通过共享 helper 生成：

```go
consul.ServiceName(baseName, protocol)
```

当前对应关系：

| 服务 | HTTP | gRPC |
|------|------|------|
| resume-service | `resume-service-http` | `resume-service-grpc` |
| interview-service | `interview-service-http` | `interview-service-grpc` |
| search-service | `search-service-http` | - |

## 代码位置

| 文件 | 说明 |
|------|------|
| `ats-platform/internal/shared/consul/naming.go` | 逻辑服务名、协议、最终命名规则 |
| `ats-platform/internal/shared/consul/register.go` | 地址解析、注册、反注册 helper |
| `ats-platform/cmd/resume-service/main.go` | Resume 服务注册 |
| `ats-platform/cmd/interview-service/main.go` | Interview 服务注册 |
| `ats-platform/cmd/search-service/main.go` | Search 服务注册 |
| `ats-platform/cmd/gateway/discovery.go` | Gateway Consul 发现与本地地址兼容 |
| `ats-platform/cmd/gateway/main.go` | Gateway 路由目标与健康聚合 |

## 服务注册地址规则

服务注册地址遵循以下优先级：

1. 显式 `SERVICE_ADDRESS`
2. 自动探测出口 IP

对应 helper：

```go
addr, err := consul.ResolveServiceAddress(cfg.ServiceAddress)
```

这意味着：

- 如果显式设置 `SERVICE_ADDRESS`，注册时一律使用它
- 如果不设置，服务会回退到自动探测的出口 IP

## 本地开发拓扑约定

### 推荐场景

如果：

- Consul 跑在 Docker 容器
- 各服务进程跑在宿主机

则推荐：

```bash
SERVICE_ADDRESS=host.docker.internal
```

这样容器内 Consul 能正确访问宿主机上的服务端口做 TCP 健康检查。

### 启动脚本约定

`ats-platform/scripts/run-services.sh` 默认按“Docker Consul + 宿主机服务”运行模型处理：

- 若未显式设置 `SERVICE_ADDRESS`
- 且未传 `--host-consul`
- 则脚本默认导出：

```bash
SERVICE_ADDRESS=host.docker.internal
```

如果 Consul 本身运行在宿主机而非 Docker，可使用：

```bash
./scripts/run-services.sh --host-consul
```

或：

```bash
make run-all-host-consul
```

## Gateway 本地兼容行为

gateway 保持“逻辑服务名 + HTTP 协议”的发现方式不变，但增加了一个本地开发特例：

- 如果 Consul 返回的地址是 `host.docker.internal`
- 且 gateway 当前运行环境无法解析这个域名
- 则会回退到 `127.0.0.1`

该行为只用于本地开发兼容，不是通用地址重写机制。

## 健康检查与反注册

### 健康检查

当前注册时使用 TCP 检查：

```go
Check: &api.AgentServiceCheck{
    TCP:      fmt.Sprintf("%s:%d", ip, port),
    Interval: "5s",
    Timeout:  "3s",
}
```

这样可以同时覆盖：

- HTTP endpoint 的端口可达性
- gRPC endpoint 的端口可达性

### 反注册

当前反注册仍基于 `service ID` 执行：

```go
consulClient.Deregister(serviceID)
```

在本地开发场景下，注册、健康检查、gateway 发现已经通过 `SERVICE_ADDRESS` 约定达成一致；如果个别机器在“宿主机服务 + Docker Consul”组合下仍出现反注册边界问题，应优先按当前文档拓扑重新验证，再决定是否需要进一步调整 Consul agent 运行方式。

## 常用检查命令

### 查看已注册服务

```bash
curl -s http://127.0.0.1:8500/v1/catalog/services
```

### 查看指定服务健康状态

```bash
curl -s 'http://127.0.0.1:8500/v1/health/service/resume-service-http?passing=false'
```

### 查看 gateway 健康聚合

```bash
curl -s http://127.0.0.1:18080/health
```

## 相关文档

- `ats-platform/docs/README.md`
- `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md`
- `docs/superpowers/specs/2026-03-24-consul-topology-and-doc-normalization-design.md`
