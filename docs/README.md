# cow 文档

## 贡献者

阅读顺序（维护 / 提 PR 前建议通读）：

1. 根 [README.md](../README.md) → [guide/overview.md](guide/overview.md)
2. [toolchain/README.md](toolchain/README.md) 与 [cmd/](../cmd/) 下各子命令 README
3. 当前主题的 [superpowers/specs/](superpowers/specs/) → 对应 [superpowers/plans/](superpowers/plans/)
4. 性能基线 [superpowers/benchmarks/](superpowers/benchmarks/)

协同时程与约定见根目录 [CONTRIBUTING.md](../CONTRIBUTING.md)、[AGENTS.md](../AGENTS.md)。

## 集成方

| 文档 | 说明 |
|------|------|
| [guide/README.md](guide/README.md) | 功能使用手册（步骤 + 示例 + 边界） |

## 维护者

| 文档 | 说明 |
|------|------|
| [toolchain/README.md](toolchain/README.md) | `undoproxy-gen` → `undocheck` → `undorewrite` 流水线 |
| [toolchain/type-graph.md](toolchain/type-graph.md) | 类型图规则（生成器与分析器共用） |
| [../cmd/undoproxy-gen/README.md](../cmd/undoproxy-gen/README.md) | 代码生成器 |
| [../cmd/undocheck/README.md](../cmd/undocheck/README.md) | 裸写分析器 `cowbarewrite` |
| [../cmd/undorewrite/README.md](../cmd/undorewrite/README.md) | 存量 AST 改写 |

## 设计与 benchmark

[superpowers/](superpowers/) — 设计 spec、实现 plan、经归档的 benchmark 日志。

**说明：** 目录名 superpowers 来自内部 Agent 工作流，与 Cursor 插件无运行时依赖；内容为设计决策与性能档案，**集成方不必阅读**。新功能请先查阅是否已有相关 spec。
