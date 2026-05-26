# undoproxy-gen

**集成用法**：[docs/guide/codegen-undoproxy.md](../../docs/guide/codegen-undoproxy.md)、[proxy-api.md](../../docs/guide/proxy-api.md)

## 职责

为带 `+cow:undoproxy-gen` 标记的类型生成 Undo 写代理，输出 `zz_generated.undo_proxy.go`。基于 `go/packages` + `go/types` + 模板（与 k8s codegen 同族）。

## 能力边界

- 同包类型图；生成 `TxContext` / `undoOp` / `Rollback`（写入 `zz_generated.undo_proxy.go`）。
- 任意/多根 `+cow:undoproxy-gen=true`；仅结构化 `ctx.push(undoOp{...})`，不生成 `AddUndo`。
- 不支持跨包嵌套、`interface{}`/channel/func 容器元素。
- 不替代 `undocheck` / `undorewrite`。

## 安装与用法

```bash
go install ./cmd/undoproxy-gen

undoproxy-gen --output-file zz_generated.undo_proxy.go IMPORT_PATH
# 例：undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow
```

业务包内通常通过 `//go:generate` 调用（见根包 `undo_proxy_generate.go`）。

退出码：参数错误 `2`，生成失败 `1`。

## 典型 CI

CI 一般**不**跑 generate；要求提交已生成的 `zz_generated.undo_proxy.go`。PR 改 `types.go` 时维护者本地 `go generate` 并提交 diff。

## 源码地图

| 文件 | 职责 |
|------|------|
| `main.go` | CLI、`Run` 入口 |
| `loader.go` | `packages.Load` |
| `emit.go` | 入口：`emitFromGraph` |
| `emit_undo.go` | `undoBuilder`、运行时代码 |
| `emit_structured.go` | 按 `FieldPlan.Kind` 生成代理 |
| `emit_structured_graph.go` | 遍历类型图、写文件 |
| `emit_helpers.go` | 类型/旧值字段辅助 |
| `generate_golden_test.go` | 双根黄金断言（无 `AddUndo`） |

核心逻辑在 `internal/cowgen`（`graph.go`、`classify.go`、`naming.go`）。

## 相关链接

- 设计：[docs/superpowers/specs/2026-05-25-undoproxy-codegen-design.md](../../docs/superpowers/specs/2026-05-25-undoproxy-codegen-design.md)
- 类型图：[docs/toolchain/type-graph.md](../../docs/toolchain/type-graph.md)
