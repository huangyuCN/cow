# 文档与当前实现对齐设计说明

| 项 | 值 |
|---|---|
| 状态 | 已批准（2026-05-27） |
| 范围 | 仓库内全部 `*.md`（含 `docs/superpowers`） |
| 目标 | 文档仅描述当前实现；禁止阶段版本标签与试验性措辞 |

## 当前实现（文档基准）

- `undoproxy-gen` 生成 `zz_generated.undo_proxy.go`：含 `TxContext`、`undoOp`、`Rollback`、全部写代理。
- 写路径：`ctx.push(undoOp{kind:...})`；**无** `AddUndo`、**无** 独立 `tx.go` / `player_proxy.go`。
- Benchmark 名：`BenchmarkUndoLog_*`、`BenchmarkMega_UndoLog_*`（无版本后缀）。

## 文件重命名

| 原路径 | 新路径 |
|--------|--------|
| `benchmarks/cow-undo-log-benchmark.md` | `cow-undo-log-benchmark.md` |
| `specs/2026-05-25-cow-undo-log-design.md` | `2026-05-25-cow-undo-log-design.md` |
| `plans/2026-05-25-cow-undo-log.md` | `2026-05-25-cow-undo-log.md` |

## 术语

统一使用「当前实现」「结构化 Undo」；已移除闭包栈与 `AddUndo`。阶段版本代号、试验性标签不得出现在对外文档。

## 验收

全库 Markdown 不得含阶段版本代号（验收时用 `git grep` 检索常见遗留拼写，见实现 PR 说明）。
