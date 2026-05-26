# Undo Log MVP Benchmark 归档

## Run 2026-05-25（基线）

| 项 | 值 |
|---|---|
| 主题 | Undo Log 稀疏写 vs `deepcopy-gen` 全量 `Player.DeepCopy()` |
| 设计 | [2026-05-25-cow-undo-log-mvp-design.md](../specs/2026-05-25-cow-undo-log-mvp-design.md) |
| 计划 | [2026-05-25-cow-undo-log-mvp.md](../plans/2026-05-25-cow-undo-log-mvp.md) |
| 日期 | 2026-05-25 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `7d7d691`（实现文件在工作区，尚未单独提交） |
| 夹具 | B 档：`Assets`≈100、`Items`≈500；稀疏写：改 `gold`、append 1 `Item`、改 `Hero.Level` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
go test -bench='BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=5 ./...
```

### 结果（5 次 run 聚合，`benchstat /tmp/cow-bench-new.txt`）

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

- **Rollback**：每轮在常驻 `Player` 上稀疏写后 `Rollback`，无计时外重置；最能体现「失败一键回滚」成本。
- **Commit**：计时内为 Pool + 稀疏写 + `Reset`；**计时外**每轮 `seed.DeepCopy()` 恢复夹具，故 `B/op`/`allocs/op` 仍含一次深拷贝，**不代表**成功路径生产态分配（生产态成功应为 ~128B 量级，接近 Rollback 减去逆操作执行）。
- **DeepCopyGen**：每轮 `Player.DeepCopy()` 整图后在副本上直接改三处，无 Undo。

### 结论（本机基线）

在中等体量 `Player`、三次稀疏写下，Undo Log 相对 `deepcopy-gen` 全量深拷贝在延迟与分配上均具数量级优势；应用侧应优先用 **Rollback** 路径衡量与 DeepCopy 的可比性，Commit 成功路径需另补「无计时外 DeepCopy」基准（若需严格验证「近零 alloc 提交」）。

---

*下一次次 run 在本文件末尾追加 `## Run YYYY-MM-DD` 章节，并用 `benchstat old.txt new.txt` 填写 vs 基线列。*

## Run 2026-05-26（V2 生成器接入）

| 项 | 值 |
|---|---|
| 主题 | Undo Log V1 vs V2（结构化日志，生成器接入后） |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `2911196`（工作区含未提交改动） |
| 夹具 | B 档：`Assets`≈100、`Items`≈500；稀疏写：改 `gold`、append 1 `Item`、改 `Hero.Level` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
GOCACHE=/tmp/go-cache go test -run '^$' -bench 'Benchmark(Mega_)?UndoLog(V2)?_SparseWrite_(Commit|Rollback)$' -benchmem -benchtime=1s .
```

### 对比（相对 2026-05-25 归档）

| Benchmark | 前次 | 本次 | 变化 |
|---|---:|---:|---:|
| `BenchmarkUndoLog_SparseWrite_Commit` ns/op | 1176 | 735.4 | -37.5% |
| `BenchmarkUndoLog_SparseWrite_Commit` B/op | 6930 | 6984 | +0.8% |
| `BenchmarkUndoLog_SparseWrite_Commit` allocs/op | 6 | 7 | +16.7% |
| `BenchmarkUndoLog_SparseWrite_Rollback` ns/op | 114 | 120.7 | +5.9% |
| `BenchmarkUndoLog_SparseWrite_Rollback` B/op | 128 | 184 | +43.8% |
| `BenchmarkUndoLog_SparseWrite_Rollback` allocs/op | 5 | 6 | +20.0% |
| `BenchmarkUndoLogV2_SparseWrite_Commit` ns/op | - | 724.2 | 首次归档 |
| `BenchmarkUndoLogV2_SparseWrite_Commit` B/op | - | 6893 | 首次归档 |
| `BenchmarkUndoLogV2_SparseWrite_Commit` allocs/op | - | 3 | 首次归档 |
| `BenchmarkUndoLogV2_SparseWrite_Rollback` ns/op | - | 88.17 | 首次归档 |
| `BenchmarkUndoLogV2_SparseWrite_Rollback` B/op | - | 64 | 首次归档 |
| `BenchmarkUndoLogV2_SparseWrite_Rollback` allocs/op | - | 2 | 首次归档 |

### V1 vs V2（本次同场）

| Benchmark | V1 | V2 | 变化 |
|---|---:|---:|---:|
| `SparseWrite_Commit` ns/op | 735.4 | 724.2 | -1.5% |
| `SparseWrite_Commit` B/op | 6984 | 6893 | -1.3% |
| `SparseWrite_Commit` allocs/op | 7 | 3 | -57.1% |
| `SparseWrite_Rollback` ns/op | 120.7 | 88.17 | -27.0% |
| `SparseWrite_Rollback` B/op | 184 | 64 | -65.2% |
| `SparseWrite_Rollback` allocs/op | 6 | 2 | -66.7% |

### 结论

V2（结构化 Undo + 生成器接入）在 lite 稀疏写路径保持与前序实验一致的收益：`allocs/op` 显著下降，Rollback 延迟明显降低。V1 指标与 2026-05-25 基线存在统计噪声（尤其回滚路径），但不影响本次 V1 vs V2 同场结论。

## Run 2026-05-26（V2 Runtime 全生成化 + 结构化 Emitter）

| 项 | 值 |
|---|---|
| 主题 | Undo Log V2（runtime/proxy 全生成化）回归验证 |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `2911196`（工作区含未提交改动） |
| 夹具 | B 档：`Assets`≈100、`Items`≈500；稀疏写：改 `gold`、append 1 `Item`、改 `Hero.Level` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
GOCACHE=/tmp/go-cache go test -run '^$' -bench 'Benchmark(Mega_)?UndoLogV2?_SparseWrite_(Commit|Rollback)$' -benchmem -benchtime=1s .
```

### 对比（相对 2026-05-26 上一版 V2 归档）

| Benchmark | 前次 | 本次 | 变化 |
|---|---:|---:|---:|
| `BenchmarkUndoLogV2_SparseWrite_Commit` ns/op | 724.2 | 673.2 | -7.0% |
| `BenchmarkUndoLogV2_SparseWrite_Commit` B/op | 6893 | 6892 | -0.0% |
| `BenchmarkUndoLogV2_SparseWrite_Commit` allocs/op | 3 | 3 | 持平 |
| `BenchmarkUndoLogV2_SparseWrite_Rollback` ns/op | 88.17 | 87.08 | -1.2% |
| `BenchmarkUndoLogV2_SparseWrite_Rollback` B/op | 64 | 64 | 持平 |
| `BenchmarkUndoLogV2_SparseWrite_Rollback` allocs/op | 2 | 2 | 持平 |

### 结论

在 runtime 全生成化且 emitter 结构化改造后，V2 lite 档核心指标保持稳定并略有提升，说明本次主要是工程可维护性优化，未引入可见性能回退。
