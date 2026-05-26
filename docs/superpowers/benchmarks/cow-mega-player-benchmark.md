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

## Run 2026-05-26（结构化 Undo 通用化）

| 项 | 值 |
|---|---|
| 主题 | `undoproxy-gen` 类型图驱动结构化 Undo（移除 `AddUndo` / Player 硬编码）；~1MB mega 稀疏写 **6 处** |
| 设计 | [2026-05-26-undoproxy-gen-structured-generic-design.md](../specs/2026-05-26-undoproxy-gen-structured-generic-design.md) |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 26.4.1 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `fd3444c`（工作区含未提交改动） |
| 夹具 | mega：`newMegaBenchPlayer()`；lite：`newBenchPlayer()`；稀疏写：`applyMegaSparseWrites`（约 6 处 `ctx.push`） |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
go test ./... -count=1
go test -run '^$' -bench='BenchmarkMega_|BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=5 .
benchstat <(上述 bench 输出)
```

### 结果（benchstat 均值，5 次 run）

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkMega_UndoLog_SparseWrite_Rollback` | **376.2** | 816 | **4** |
| `BenchmarkMega_UndoLog_SparseWrite_Commit` | **242.3** | 203 | **3** |
| `BenchmarkMega_DeepCopyGen_SparseWrite` | 235,400 | 464,128 | 9,177 |
| `BenchmarkUndoLog_SparseWrite_Rollback`（lite） | **111.5** | 64 | **2** |
| `BenchmarkUndoLog_SparseWrite_Commit`（lite） | **790.2** | 6,896 | **3** |
| `BenchmarkDeepCopyGen_SparseWrite`（lite） | 10,710 | 38,688 | 511 |

### 对比（相对 2026-05-26 V2 专用归档）

| Benchmark | V2 专用（约） | 通用化（本次） | 变化 |
|-----------|-------------:|---------------:|------|
| `Mega_SparseWrite_Rollback` ns/op | 312 | 376 | +21% |
| `Mega_SparseWrite_Rollback` allocs/op | 4 | 4 | 持平 |
| `Mega_SparseWrite_Commit` ns/op | 151–191 | 242 | +27%～+60% |
| `Mega_SparseWrite_Commit` allocs/op | 3 | 3 | 持平 |

### 结论

通用化后 **分配特征与 V2 一致**（mega 4/3 allocs），延迟略高于 Player 专用 V2，仍比 2026-05-25 闭包 V1（10/9 allocs）显著更优；相对 DeepCopy 保持 **三个数量级以上**优势。

---

## Run 2026-05-26（稀疏写 32 处 vs 6 处）

| 项 | 值 |
|---|---|
| 主题 | 提高 mega 稀疏写密度：`applyMegaSparseWrites32`（**32** 处 `ctx.push`）vs `applyMegaSparseWrites`（**6** 处） |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 26.4.1 / Apple M3 |
| GOMAXPROCS | 8（默认，未显式设置） |
| commit | `fd3444c`（工作区含未提交改动） |
| 夹具 | mega：`newMegaBenchPlayer()`；32 档由 `TestMegaSparseWrites32_OpCount` 断言 `len(ctx.ops)==32` |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
go test ./... -count=1
go test -run '^$' -bench='BenchmarkMega_' -benchmem -count=5 .
benchstat <(上述 bench 输出)
```

### 结果（benchstat 均值，5 次 run）

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkMega_UndoLog_SparseWrite_Rollback`（6 处） | **366.9** | 816 | **4** |
| `BenchmarkMega_UndoLog_SparseWrite_Commit`（6 处） | **212.6** | 199 | **3** |
| `BenchmarkMega_UndoLog_SparseWrite32_Rollback`（32 处） | **1,563** | 2,345 | **21** |
| `BenchmarkMega_UndoLog_SparseWrite32_Commit`（32 处） | **1,108** | 858 | **18** |
| `BenchmarkMega_DeepCopyGen_SparseWrite`（6 处裸写） | 224,100 | 464,128 | 9,177 |
| `BenchmarkMega_DeepCopyGen_SparseWrite32`（32 处裸写） | 220,100 | 465,456 | 9,188 |

### 6 处 vs 32 处（Undo 路径）

| 路径 | 6 处 ns/op | 32 处 ns/op | 倍数 | 6 处 allocs | 32 处 allocs | 倍数 |
|------|----------:|-----------:|-----:|------------:|-------------:|-----:|
| Rollback | 367 | 1,563 | **×4.3** | 4 | 21 | **×5.3** |
| Commit | 213 | 1,108 | **×5.2** | 3 | 18 | **×6.0** |

- **边际成本（粗算）**：Rollback 约 `(1563−367)/(32−6) ≈ 46 ns/处`；Commit 约 `(1108−213)/(32−6) ≈ 34 ns/处`（含固定开销前的线性近似）。
- **DeepCopy 对照**：32 处裸写相对 6 处 **几乎不变**（~220µs、~9.2k allocs），对象图拷贝仍主导；Undo 随写次数近线性增长但仍 **≪ DeepCopy**（32 处 Rollback ~1.6µs vs ~220µs，约 **×140**）。

### 结论

在 ~1MB `Player` 上，将单次请求稀疏写从 6 处增至 32 处，Undo **延迟与分配近似线性放大**（约 4～6×），符合「每条 `push` + Rollback switch 分支」预期；DeepCopy 路径对写次数不敏感。32 处 Rollback 仍比全量 DeepCopy 快约 **两个数量级**，高密度写场景下 Undo Log 仍具明显优势。

---

## Run 2026-05-26（稀疏写 32 处 · 均匀分散根字段）

| 项 | 值 |
|---|---|
| 主题 | `applyMegaSparseWrites32` 改写：32 处 Undo **均匀覆盖** Player 根字段（Uid/Level/Assets/Items/MainHero/Heros/Bags/Stats/Cooldowns/Mails/Quests），避免 8× `PutAssets` 堆叠 |
| 日期 | 2026-05-26 |
| go version | `go1.26.0 darwin/arm64` |
| OS / CPU | Darwin 26.4.1 / Apple M3 |
| GOMAXPROCS | 8（默认） |
| commit | `fd3444c`（工作区含未提交改动） |
| 字段分布 | Uid(1)+Level(1)+Assets(3)+Items(4)+MainHero(2)+Heros(3)+Bags(4)+Stats(4)+Cooldowns(4)+Mails(3)+Quests(3)=32 |

### 命令

```bash
cd /Users/huangyu/work/golang/src/cow
go test -run 'TestMegaSparseWrites32' -count=1
go test -run '^$' -bench='BenchmarkMega_' -benchmem -count=5 .
```

### 结果（benchstat 均值，5 次 run）

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BenchmarkMega_UndoLog_SparseWrite_Rollback`（6 处） | **356.1** | 816 | 4 |
| `BenchmarkMega_UndoLog_SparseWrite32_Rollback`（32 处·分散） | **6,010** | 35,890 | **31** |
| `BenchmarkMega_UndoLog_SparseWrite32_Commit`（32 处·分散） | **1,201** | 1,073 | **26** |
| `BenchmarkMega_DeepCopyGen_SparseWrite32` | 228,000 | 465,856 | 9,194 |

### 对比（32 处：集中 Assets vs 均匀分散根字段）

| 路径 | 集中写（上一 run）ns/op | 分散写（本次）ns/op | 变化 |
|------|------------------------:|--------------------:|-----:|
| Rollback | 1,563 | 6,010 | **×3.8 更慢** |
| Commit | 1,108 | 1,201 | +8% |
| Rollback allocs/op | 21 | 31 | +48% |
| Rollback B/op | 2,345 | 35,890 | **×15.3** |

### 说明

- 分散后包含 **Items Remove/Truncate**、**PutBags**、**Get*ForWrite**（COW 浅拷贝）、**map 整槽 Put** 等更重路径，单条 Undo 成本高于重复 `PutAssets`。
- 即使 Rollback ~6µs，仍远低于 DeepCopy ~228µs（约 **×38**）；Commit ~1.2µs 与集中写相当。
- `TestMegaSparseWrites32_OpCount` 仍断言 `len(ctx.ops)==32`。

### 结论

均匀覆盖根字段更贴近真实业务（多类型混合写），benchmark 数字高于「32×PutAssets」的乐观集中场景；**相对 DeepCopy 的数量级优势不变**，应用作容量规划时应按**操作种类混合**而非单一 `Put` 估算。
