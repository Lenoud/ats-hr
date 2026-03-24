# ATS Platform Docs

本目录用于保存 ATS 平台当前实现相关的文档，按用途分为三类。

## 正式文档

放在 `ats-platform/docs/` 根目录，面向开发者、维护者和使用者，要求尽量与当前实现保持一致。

当前包括：

- `SERVICE_DEVELOPMENT_GUIDE.md`
- `consul-integration.md`
- `resume-service-api.md`
- `interview-service-grpc.md`

## 测试与实验文档

放在 `ats-platform/docs/tests/`，保存接口验证步骤、curl 示例、实验性调用记录。

这些文档可以比正式文档更偏操作，但仍应基于当前实现维护。

## 设计与计划文档

放在 `ats-platform/docs/superpowers/`，保存实现计划、设计规格、阶段性记录。

- `specs/`：设计文档
- `plans/`：实现计划

## 维护规则

- 正式文档描述当前行为，不保留已经完成却仍写成 TODO 的内容。
- 测试文档记录可复现验证步骤，与正式说明文档分开维护。
- 设计与计划文档保留历史上下文，不替代正式文档。
