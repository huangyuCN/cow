# 工具链

**集成用法**见 [guide/](../guide/)；本文面向维护与扩展 `cmd/*`。

## 流水线

```text
types.go (+cow:undoproxy-gen)
        │
        ▼
  undoproxy-gen ──► zz_generated.undo_proxy.go
        │
        ▼
  业务代码（Put* / Get*ForWrite + TxContext）
        │
        ├── undocheck (cowbarewrite) ── CI 禁止新裸写
        └── undorewrite (可选) ── 批量改历史裸写
```

## 命令一览

| 命令 | 输入 | 输出 | internal 依赖 |
|------|------|------|----------------|
| [undoproxy-gen](../../cmd/undoproxy-gen/README.md) | Go import path | `zz_generated.undo_proxy.go` | `internal/cowgen` |
| [undocheck](../../cmd/undocheck/README.md) | packages 模式 `./...` | analysis 诊断 | `internal/cowmon`, `internal/cowfile` |
| [undorewrite](../../cmd/undorewrite/README.md) | 目录 glob | 改写后的 `.go` | `internal/cowmon`, `internal/cowproxy` |

## 类型图一致性

生成器与 `undocheck` **必须**对「哪些 struct 受监控」达成一致，规则见 [type-graph.md](type-graph.md)。

## 设计档案

| 主题 | spec |
|------|------|
| 代码生成 | [undoproxy-codegen-design.md](../superpowers/specs/2026-05-25-undoproxy-codegen-design.md) |
| 裸写检查 | [bare-write-guard-design.md](../superpowers/specs/2026-05-25-bare-write-guard-design.md) |
| 存量改写 | [undorewrite-codemod-design.md](../superpowers/specs/2026-05-25-undorewrite-codemod-design.md) |
