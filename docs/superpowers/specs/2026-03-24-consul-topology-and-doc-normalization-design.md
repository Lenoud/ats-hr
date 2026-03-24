# Consul 本地拓扑一致性与文档规范化设计

## 概述

本设计覆盖两个收尾项：

1. 统一 ATS 平台在本地开发场景下的 Consul 注册、健康检查、gateway 发现与反注册行为。
2. 将当前主仓库中未跟踪但有价值的实现文档、测试文档、实验文档整理入库，并建立清晰的文档分类规则。

本设计是收尾与规范化工作，不扩展到服务治理平台能力建设，不引入配置中心、服务 watch、负载均衡策略、Consul KV 等超出当前范围的内容。

## 背景与问题

### Consul 本地拓扑问题

当前代码已经完成 `service discovery abstraction`，但本地开发仍存在一个拓扑不一致问题：

- 服务进程运行在宿主机
- Consul 运行在 Docker 容器
- 宿主机与容器对同一服务地址的可达性假设不同

已知表现：

- 注册可成功
- 容器侧健康检查可能需要 `host.docker.internal`
- gateway 在宿主机运行时又可能无法直接解析 `host.docker.internal`
- 反注册在当前拓扑下仍可能出现 `404`

这说明当前实现已经具备命名抽象，但缺少“本地开发拓扑约定”的最后一层收口。

### 文档问题

主仓库中存在未跟踪文档，类型混杂：

- 正式开发指南
- 服务集成说明
- API 文档
- 测试/接口验证文档
- superpowers 计划文档

这些文档本身有价值，但没有完成分类入库，导致：

- 难以判断哪份是权威文档
- 实现文档与测试文档混放
- 实验性内容没有统一归档位置

## 目标

### Consul 目标

在本地开发场景下，为以下四个动作建立一致行为：

1. 服务注册
2. 健康检查
3. gateway 服务发现与代理
4. 服务退出反注册

### 文档目标

将现有未跟踪文档纳入规范化结构：

- 正式文档有稳定位置
- 测试/实验文档有单独目录
- superpowers 设计/计划文档保留在原有体系
- 补一个索引说明各类文档的用途和维护规则

## 非目标

本次不做以下事项：

- Consul watch / blocking query / cache
- 服务负载均衡策略扩展
- 动态路由配置系统
- Consul KV 配置中心
- 将所有历史文档统一重写
- 为文档系统引入站点生成器

## 方案选择

### 方案 1：最小收尾 + 三层文档结构

做法：

- Consul 保留现有命名抽象，只补本地拓扑一致性约定和最小实现
- 文档按正式文档、测试文档、superpowers 文档三层分类

优点：

- 与现有代码兼容
- 风险小
- 适合收尾
- 能快速消化当前未跟踪文档

缺点：

- 不解决更长期的服务治理问题

### 方案 2：继续扩展 discovery 抽象，顺手重做 Consul 运行模型

做法：

- 在当前抽象之上继续扩充 agent/client 语义、反注册策略、更多配置维度

优点：

- 未来空间更大

缺点：

- 范围明显扩大
- 容易把收尾做成新的平台工程

### 方案 3：只整理文档，不动 Consul 运行收尾

优点：

- 成本最低

缺点：

- 本地拓扑问题继续遗留
- 文档会描述一个仍不完全稳定的运行约定

### 推荐

推荐方案 1。

原因：

- 它与当前分支已完成的 `service discovery abstraction` 自然衔接
- 能在不推翻现有代码结构的前提下补齐本地运行一致性
- 能同时完成文档纳入与规范化，不把这轮收尾拖成大项目

## 设计

### 一、Consul 本地拓扑一致性

#### 1. 地址优先级

服务注册地址采用以下优先级：

1. 显式配置地址
2. 自动探测出口 IP

也就是：

- 若设置 `SERVICE_ADDRESS`，注册时一律使用该地址
- 若未设置，则回退到自动探测

#### 2. 本地开发约定

约定一个明确的本地开发模式：

- 若 Consul 在 Docker 中运行，而服务进程在宿主机运行，则本地启动服务时配置：
  - `SERVICE_ADDRESS=host.docker.internal`

该约定的意义：

- 容器内 Consul 可以对宿主机服务做健康检查
- 注册地址不再依赖宿主机外网/局域网出口 IP

#### 3. gateway 兼容

gateway 保持当前“逻辑服务名 + 协议”的发现方式不变，但对本地特例地址做最小兼容：

- 如果 Consul 返回 `host.docker.internal`
- 且 gateway 所在环境无法解析该域名
- 则回落到 `127.0.0.1`

该兼容仅用于本地开发，不引入新的发现语义。

#### 4. 反注册策略

本次目标不是重做 Consul agent 架构，而是把反注册行为收口到当前可解释状态：

- 保留现有基于 service ID 的反注册逻辑
- 明确记录当前本地开发拓扑对反注册的约束
- 若能用最小改动修正，则补齐
- 若不能稳定修正，则文档中说明边界并避免误导

是否进一步切换到更明确的 agent 拓扑，不在本次范围内展开。

### 二、文档规范化

#### 1. 目录结构

建议整理为：

```text
ats-platform/docs/
├── README.md                     # 文档索引
├── SERVICE_DEVELOPMENT_GUIDE.md  # 正式开发指南
├── consul-integration.md         # Consul 集成与运行说明
├── resume-service-api.md         # 正式 API 文档
├── tests/
│   └── interview-service-api-test.md
└── superpowers/
    ├── plans/
    └── specs/
```

#### 2. 分类规则

- 正式文档：
  - 面向开发者、维护者、使用者
  - 需与当前实现保持一致
  - 放 `ats-platform/docs/`

- 测试/实验文档：
  - 包含 curl 用例、接口验证步骤、实验性验证记录
  - 放 `ats-platform/docs/tests/`

- superpowers 文档：
  - 设计、计划、实现记录
  - 放 `ats-platform/docs/superpowers/`

#### 3. 具体归类

- `ats-platform/docs/SERVICE_DEVELOPMENT_GUIDE.md`
  - 正式开发指南
  - 入库到 `ats-platform/docs/`

- `ats-platform/docs/consul-integration.md`
  - 正式运行/集成说明
  - 需更新为当前实现状态后入库

- `ats-platform/docs/resume-service-api.md`
  - 正式 API 文档
  - 需按当前实现校正后入库

- `ats-platform/docs/interview-service-api-test.md`
  - 测试文档
  - 移动到 `ats-platform/docs/tests/`

- `ats-platform/docs/superpowers/plans/2026-03-23-search-service-implementation.md`
  - 保留为 superpowers 计划文档
  - 入库到现有 `ats-platform/docs/superpowers/plans/`

#### 4. 文档索引

新增一个简短索引，例如 `ats-platform/docs/README.md`，说明：

- 哪些是正式文档
- 哪些是测试/实验文档
- 哪些是设计/计划文档
- 文档维护原则

## 验证标准

### 代码验证

- `go build` 覆盖本次改动涉及的服务
- 本地 Consul 链路重新验证：
  - 注册
  - 健康检查
  - gateway 代理
  - 退出/反注册行为

### 文档验证

- 当前未跟踪文档全部被纳入版本库或被明确归类
- `ats-platform/docs/` 下形成稳定可解释结构
- 文档内容与当前实现不明显冲突

## 风险与控制

### 风险 1：反注册问题超出最小收尾范围

控制：

- 先做最小验证
- 若无法稳定修复，不盲目扩大为平台重构
- 在文档中准确记录边界

### 风险 2：文档存在大量过时内容

控制：

- 只校正与当前实现直接相关的关键段落
- 不做大规模重写
- 保证“结构正确、核心信息正确”优先于“全面重写”

### 风险 3：测试/实验文档与正式文档界限模糊

控制：

- 通过目录层级区分用途
- 在索引文档中明确说明使用场景

## 实施顺序建议

1. 先处理 Consul 本地拓扑一致性
2. 再整理文档分类与迁移
3. 最后做一轮统一验证与文档对齐

这样可以避免文档先写死，再因为运行方式调整而返工。
