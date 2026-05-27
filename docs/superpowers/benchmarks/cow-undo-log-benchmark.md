# Undo Log Benchmark 归档

集成方性能摘要见 [docs/guide/overview.md](../../guide/overview.md)。以下为维护者本机跑数记录。

## Run 2026-05-25（基线）

| 项 | 值 |
|---|---|
| 主题 | Undo Log 稀疏写 vs `deepcopy-gen` 全量 `Player.DeepCopy()` |
| 设计 | [2026-05-25-cow-undo-log-design.md](../specs/2026-05-25-cow-undo-log-design.md) |
| 计划 | [2026-05-25-cow-undo-log.md](../plans/2026-05-25-cow-undo-log.md) |
| 日期 | 2026-05-25 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `7d7d691` |
| 夹具 | B 档：`Assets`≈100、`Items`≈500；稀疏写：改 `gold`、append 1 `Item`、改 `MainHero.Level` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
go test -bench='BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=5 ./...
```

### 结果（5 次 run 聚合，`benchstat`）

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkUndoLog_SparseWrite_Rollback` | **114** | 128 | **5** |
| `BenchmarkUndoLog_SparseWrite_Commit` | **1,176** | 6,930 | 6 |
| `BenchmarkDeepCopyGen_SparseWrite`（基线） | **9,961** | 26,456 | **508** |

### 相对基线 `DeepCopyGen_SparseWrite`

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `UndoLog_SparseWrite_Rollback` | **~87× 更快** | ~0.5% | **~102× 更少** |
| `UndoLog_SparseWrite_Commit` | **~8.5× 更快** | ~26% | **~85× 更少** |

### 说明

- **Rollback**：在常驻 `Player` 上稀疏写后 `Rollback`；体现失败回滚成本。
- **Commit**：计时内含 Pool + 稀疏写 + `Reset`；部分历史跑法在计时外 `DeepCopy` 重置夹具，`B/op` 偏高，**不代表**生产成功路径（成功路径应接近 Rollback 量级、约 2–3 `allocs/op`，见下节）。
- **DeepCopyGen**：每轮整图 `DeepCopy` 后在副本上直接改字段，无 Undo。

---

## Run 2026-05-26（结构化 Undo + 代码生成，当前实现）

| 项 | 值 |
|---|---|
| 主题 | 结构化 `undoOp`、生成器产出 `TxContext` 与代理后的 lite 稀疏写 |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8 |
| commit | `2911196` 起多轮迭代；指标以当前 `BenchmarkUndoLog_*` 为准 |
| 夹具 | 同 B 档 |

### 命令（与仓库当前一致）

```bash
cd /Users/huangyu/work/golang/src/cow
go test -run '^$' -bench='BenchmarkUndoLog_SparseWrite_(Commit|Rollback)$' -benchmem -benchtime=1s ./...
```

### 结果（代表性，`benchstat`）

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkUndoLog_SparseWrite_Rollback` | **~87** | 64 | **2** |
| `BenchmarkUndoLog_SparseWrite_Commit` | **~670–790** | ~6,900 | **3** |

### 结论

相对 `deepcopy-gen` 全量拷贝，Rollback 路径在延迟与分配上保持数量级优势；成功路径以 `Reset` 清空 `undoOp` 为主，生产态分配应关注 **3 `allocs/op` 量级** 的 Commit 指标（勿与含计时外 DeepCopy 的旧跑法混淆）。

---

*新跑数在本文件末尾追加 `## Run YYYY-MM-DD`，并用 `benchstat` 对比上一节。*
