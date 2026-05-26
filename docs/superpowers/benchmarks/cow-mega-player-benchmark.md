# Mega Player Benchmark 归档

## Run 2026-05-25

| 项 | 值 |
|---|---|
| 主题 | ~1MB `Player` 稀疏写：Undo Log vs `deepcopy-gen` 全量 `DeepCopy()` |
| 设计 | [2026-05-25-mega-player-benchmark-design.md](../specs/2026-05-25-mega-player-benchmark-design.md) |
| 日期 | 2026-05-25 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `7d7d691`（benchmark 跑数时；正确性测试后续增补） |
| 夹具 | mega：`newMegaBenchPlayer()` 堆估算 ≈1MiB±15%；lite：`newBenchPlayer()` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
go test -bench='BenchmarkMega_|BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=3 ./...
```

### 结果（3 次 run，本机）

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkMega_UndoLog_SparseWrite_Rollback` | **354–451** | ~1056 | **10** |
| `BenchmarkMega_UndoLog_SparseWrite_Commit` | **288–312** | ~443–448 | **9** |
| `BenchmarkMega_DeepCopyGen_SparseWrite` | **222k–330k** | ~464k | **~9178** |
| `BenchmarkUndoLog_SparseWrite_Rollback`（lite） | **129–157** | 184 | **6** |
| `BenchmarkUndoLog_SparseWrite_Commit`（lite） | **839–1038** | 6984 | **7** |
| `BenchmarkDeepCopyGen_SparseWrite`（lite） | **10.6k–12.1k** | 38688 | **511** |

### 相对 mega 档 `BenchmarkMega_DeepCopyGen_SparseWrite`

| Benchmark | ns/op（约） | allocs/op（约） |
|-----------|------------|----------------|
| `Mega_UndoLog_Rollback` | **500–900× 更快** | **~900× 更少** |
| `Mega_UndoLog_Commit` | **700–1100× 更快** | **~1000× 更少** |

### 说明

- **Mega Undo**：对象图约 1MB，但单次请求仅稀疏写 6–8 处，Undo 成本仍接近常数级（与 lite 同量级 ns/op）。
- **Mega DeepCopy**：`B/op` ~464KB、`allocs/op` ~9k，随夹具体积显著高于 lite（~39KB / 511 allocs）。
- **Commit（mega）**：计时内无 `DeepCopy` 重置，更能代表成功提交路径分配。
- **正确性**：`player_mega_test.go` — `ProxyProbe`/`BusinessPath` 的 **Rollback + Commit**；`CommitPersistsAfterLaterRollback` 验证提交不可被后续 Rollback 撤销。

### 结论（本机）

~1MB 常驻 `Player` 下，Undo Log 相对全量 DeepCopy 在延迟与分配上仍具数量级优势；夹具体积主要惩罚 DeepCopy 路径，不惩罚稀疏 Undo。

---

*下一次次 run 在本文件末尾追加 `## Run YYYY-MM-DD` 章节。*

## Run 2026-05-26（V2 生成器接入）

| 项 | 值 |
|---|---|
| 主题 | ~1MB `Player` 稀疏写：Undo Log V1 vs V2（结构化日志，生成器接入后） |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `2911196`（工作区含未提交改动） |
| 夹具 | mega：`newMegaBenchPlayer()` 堆估算 ≈1MiB±15%；lite：`newBenchPlayer()` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
GOCACHE=/tmp/go-cache go test -run '^$' -bench 'Benchmark(Mega_)?UndoLog(V2)?_SparseWrite_(Commit|Rollback)$' -benchmem -benchtime=1s .
```

### 对比（相对 2026-05-25 归档）

| Benchmark | 前次 | 本次 | 变化 |
|---|---:|---:|---:|
| `BenchmarkMega_UndoLog_SparseWrite_Rollback` ns/op | 354–451 | 338.8 | 低于前次区间下界 |
| `BenchmarkMega_UndoLog_SparseWrite_Rollback` B/op | ~1056 | 1056 | 持平 |
| `BenchmarkMega_UndoLog_SparseWrite_Rollback` allocs/op | 10 | 10 | 持平 |
| `BenchmarkMega_UndoLog_SparseWrite_Commit` ns/op | 288–312 | 318.3 | 略高于前次区间上界 |
| `BenchmarkMega_UndoLog_SparseWrite_Commit` B/op | ~443–448 | 440 | 持平 |
| `BenchmarkMega_UndoLog_SparseWrite_Commit` allocs/op | 9 | 9 | 持平 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Rollback` ns/op | - | 316.1 | 首次归档 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Rollback` B/op | - | 816 | 首次归档 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Rollback` allocs/op | - | 4 | 首次归档 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Commit` ns/op | - | 190.9 | 首次归档 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Commit` B/op | - | 196 | 首次归档 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Commit` allocs/op | - | 3 | 首次归档 |

### V1 vs V2（本次同场）

| Benchmark | V1 | V2 | 变化 |
|---|---:|---:|---:|
| `Mega_SparseWrite_Commit` ns/op | 318.3 | 190.9 | -40.0% |
| `Mega_SparseWrite_Commit` B/op | 440 | 196 | -55.5% |
| `Mega_SparseWrite_Commit` allocs/op | 9 | 3 | -66.7% |
| `Mega_SparseWrite_Rollback` ns/op | 338.8 | 316.1 | -6.7% |
| `Mega_SparseWrite_Rollback` B/op | 1056 | 816 | -22.7% |
| `Mega_SparseWrite_Rollback` allocs/op | 10 | 4 | -60.0% |

### 结论

V2（结构化 Undo + 生成器接入）在 mega 档也保持稳定收益，尤其 `Commit` 路径改善显著（ns/op、B/op、allocs/op 同时下降）。这说明 V2 并非只在 lite 夹具有效，在 ~1MB 对象图下同样成立。

## Run 2026-05-26（V2 Runtime 全生成化 + 结构化 Emitter）

| 项 | 值 |
|---|---|
| 主题 | ~1MB `Player` 稀疏写：Undo Log V2（runtime/proxy 全生成化）回归验证 |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 25.4.0 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `2911196`（工作区含未提交改动） |
| 夹具 | mega：`newMegaBenchPlayer()`；lite：`newBenchPlayer()` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
GOCACHE=/tmp/go-cache go test -run '^$' -bench 'Benchmark(Mega_)?UndoLogV2?_SparseWrite_(Commit|Rollback)$' -benchmem -benchtime=1s .
```

### 对比（相对 2026-05-26 上一版 V2 归档）

| Benchmark | 前次 | 本次 | 变化 |
|---|---:|---:|---:|
| `BenchmarkMega_UndoLogV2_SparseWrite_Commit` ns/op | 190.9 | 150.9 | -20.9% |
| `BenchmarkMega_UndoLogV2_SparseWrite_Commit` B/op | 196 | 208 | +6.1% |
| `BenchmarkMega_UndoLogV2_SparseWrite_Commit` allocs/op | 3 | 3 | 持平 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Rollback` ns/op | 316.1 | 311.8 | -1.4% |
| `BenchmarkMega_UndoLogV2_SparseWrite_Rollback` B/op | 816 | 816 | 持平 |
| `BenchmarkMega_UndoLogV2_SparseWrite_Rollback` allocs/op | 4 | 4 | 持平 |

### 结论

V2 在 mega 档仍保持稳定优势。此次改造后，`Commit` 延迟继续改善，`Rollback` 指标稳定，整体可视为“工程化收敛且性能不回退”。
