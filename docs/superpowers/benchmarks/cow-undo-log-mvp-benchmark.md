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
