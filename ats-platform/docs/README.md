# ATS Platform Docs

本目录保存 ATS Platform 当前实现相关的文档，按用途分为三类。

## 正式文档

放在 `ats-platform/docs/` 根目录，描述当前实现和推荐开发/运行方式。

当前包括：

- `SERVICE_DEVELOPMENT_GUIDE.md`
- `interview-service-grpc.md`

## 测试与实验文档

放在 `ats-platform/docs/tests/`，用于保存接口验证步骤、`curl` 示例和实验性调用记录。

当前包括：

- `tests/interview-service-api-test.md`

## 设计与计划文档

设计与计划文档保留在仓库根目录 `docs/superpowers/`，用于记录阶段性方案与实施计划，不替代正式说明文档。

## 维护规则

- 正式文档只描述当前实现，不保留已经落地却仍写成 TODO 的内容。
- 测试/实验文档记录验证步骤，与正式说明分开维护。
- 设计与计划文档保留历史上下文，允许包含阶段性取舍和已完成事项。
