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
