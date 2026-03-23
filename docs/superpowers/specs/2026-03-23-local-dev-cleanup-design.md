# Local Dev Cleanup And Script Orchestration Design

## Goal

清理仓库中确认无用的遗留文件，并将本地开发的多服务启动方式统一收敛到脚本编排入口，减少分散命令和重复维护。

## Scope

本次设计只覆盖以下两类工作：

- 删除已确认不参与主工程构建或运行的遗留垃圾文件
- 统一本地开发启动入口到 `ats-platform/scripts/run-services.sh`

本次不包含：

- 业务接口变更
- 配置安全治理
- 网关能力扩展
- 测试补充

## Current State

当前仓库存在两个明显问题：

1. 顶层存在一个独立的 `internal/shared/logger/logger.go`，而真实工程代码位于 `ats-platform/internal/...` 下。这个文件不在 `ats-platform` Go module 内，容易误导维护者。
2. 本地开发启动入口分散。`ats-platform/scripts/run-services.sh` 已经具备统一编译和启动三个服务的能力，但 `ats-platform/Makefile` 里仍保留直接后台 `go run` 的 `run-all` 旧实现，入口没有收敛。

## Design Decisions

### 1. Script As The Standard Local Dev Entry

将 `ats-platform/scripts/run-services.sh` 定义为本地开发的标准编排入口。

原因：

- 现有脚本已经支持基础设施启动、统一 build、统一进程托管和退出清理
- 相比在 `Makefile` 中直接后台起多个 `go run`，脚本更适合维护复杂的启动逻辑
- 可以保留 `Makefile` 作为薄包装，兼顾使用习惯和实现收敛

### 2. Makefile Becomes A Thin Wrapper

`ats-platform/Makefile` 不再直接负责多服务并发编排，而是：

- 保留单服务运行命令，便于独立调试
- 将 `run-all` 改为调用 `scripts/run-services.sh`
- 视需要增加 `run-all-no-infra` 或 `build-services` 这类薄包装目标

这样可以让日常使用者仍通过 `make` 操作，但背后只有一套真实启动逻辑。

### 3. Remove Confirmed Dead File

删除顶层 `internal/shared/logger/logger.go`。

判断依据：

- 主工程 module 位于 `ats-platform/go.mod`
- 实际被引用的 logger 位于 `ats-platform/internal/shared/logger/logger.go`
- 顶层同名文件不参与 `ats-platform` 模块构建，属于高误导性的仓库残留物

## File Changes

### Delete

- `internal/shared/logger/logger.go`

### Modify

- `ats-platform/Makefile`
  - 将 `run-all` 改为调用统一脚本
  - 增加与脚本能力对应的薄包装目标
  - 删除或替换直接后台并发 `go run` 的旧实现

- `ats-platform/scripts/run-services.sh`
  - 如有必要，补足脚本帮助信息或参数命名
  - 保持它作为统一入口的定位更明确

- `docs/SERVICE_DEVELOPMENT_GUIDE.md`
  - 更新本地开发推荐启动方式

- 如仓库内存在运行说明文档引用旧方式，也同步修正

## Expected Developer Workflow

推荐的本地开发路径调整为：

1. 进入 `ats-platform/`
2. 使用统一脚本或等价的 `make` 薄包装命令启动依赖和服务
3. 如需单服务调试，继续使用 `make run-resume` / `make run-interview` / `make run-search`

结果是：

- 多服务联调只有一个标准入口
- 单服务调试仍然保留
- 仓库中不会同时维护两套并发启动逻辑

## Risks

### Risk: Existing Habits Still Depend On `make run-all`

处理方式：

- 不删除 `run-all` 目标
- 仅将其实现改为转发到脚本

### Risk: Script And Makefile Semantics Drift Again

处理方式：

- 让 `Makefile` 只保留薄包装，不再复制编排逻辑
- 所有多服务启动行为都收敛到脚本

### Risk: Misidentifying A File As Garbage

处理方式：

- 只删除已确认位于 module 外且无引用的顶层 logger 文件
- 不扩大清理范围到不确定用途的文档或脚本

## Success Criteria

- 顶层遗留 logger 文件被清理
- `run-all` 不再直接后台起多个 `go run`
- 多服务本地联调只有一套真实实现逻辑
- 文档中不再把旧方式写成推荐入口
