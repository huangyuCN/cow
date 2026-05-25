# COW MVP Benchmark Log

## 记录约定

- 在每次确认保留 benchmark 结果后追加一节。
- 使用 Markdown 表格记录 `ns/op`、`B/op`、`allocs/op`。
- 附带日期、`go version`、机器信息、`GOMAXPROCS`、commit、完整命令。

## 2026-05-22 首次基线

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值（基准输出为 `-8`）
- commit：`03874df53594914b0202e6baf9878edeb064e195`
- 命令：`go test ./... -bench 'BenchmarkRunWith(Cow|DeepCopy)$' -benchmem -run '^$' -count=1`

| 基准名 | 前次 ns/op | 本次 ns/op | 前次 B/op | 本次 B/op | 前次 allocs/op | 本次 allocs/op | 相对变化 |
|---|---:|---:|---:|---:|---:|---:|---|
| `BenchmarkRunWithCow-8` | N/A | 225.3 | N/A | 840 | N/A | 13 | 首次记录 |
| `BenchmarkRunWithDeepCopy-8` | N/A | 99.79 | N/A | 408 | N/A | 6 | 首次记录 |

说明：

- 当前实现仍以根快照克隆为主，尚未进入真正的 path-copy / COW 热路径优化。
- 因此本次结果仅作为正确性阶段的首个性能基线，不代表目标态性能。

## 2026-05-22 框架层固定分配优化

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值（基准输出为 `-8`）
- commit：`03874df53594914b0202e6baf9878edeb064e195`
- 命令：`go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndSession.*)$' -benchmem -count=1`
- 对比基线：`/tmp/cow-explicit-session-bench.txt`

| 基准名 | 前次 ns/op | 本次 ns/op | 相对变化 | 前次 B/op | 本次 B/op | 相对变化 | 前次 allocs/op | 本次 allocs/op | 相对变化 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `BenchmarkFrameworkBeginCommitRollback-8` | 201.7 | 140.0 | -30.6% | 712 | 504 | -29.2% | 12 | 8 | -33.3% |
| `BenchmarkFrameworkEmptyClosure-8` | 0.2757 | 0.2483 | -9.9% | 0 | 0 | 0.0% | 0 | 0 | 0.0% |
| `BenchmarkCowWritePathOnSession-8` | 208.3 | 302.9 | +45.4% | 688 | 1216 | +76.7% | 9 | 14 | +55.6% |
| `BenchmarkCowWritePathInSessionLifecycle-8` | 341.9 | 325.3 | -4.9% | 1336 | 1240 | -7.2% | 17 | 16 | -5.9% |
| `BenchmarkDeepCopyWritePath-8` | 219.8 | 224.9 | +2.3% | 944 | 944 | 0.0% | 11 | 11 | 0.0% |
| `BenchmarkEndToEndSessionWithCow-8` | 344.5 | 329.0 | -4.5% | 1336 | 1240 | -7.2% | 17 | 16 | -5.9% |
| `BenchmarkEndToEndSessionWithDeepCopy-8` | 241.8 | 224.3 | -7.2% | 944 | 944 | 0.0% | 11 | 11 | 0.0% |

说明：

- 本轮主要目标是降低事务框架固定成本，`BenchmarkFrameworkBeginCommitRollback-8` 的 `ns/op`、`B/op`、`allocs/op` 都出现明显下降，目标达成。
- `BenchmarkCowWritePathOnSession-8` 明显变差，原因是 `newBenchSession()` 已改为真实只读初始态，首次写升级成本现在被计入该基准。
- `BenchmarkCowWritePathInSessionLifecycle-8` 与 `BenchmarkEndToEndSessionWithCow-8` 仍有小幅改善，说明框架层固定成本下降已经传递到真实事务生命周期。

## 2026-05-22 大根稀疏写 benchmark

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值（基准输出为 `-8`）
- commit：`4edfea34c6ea540140ed83f6ba83a79162cd4876`
- 命令：`go test ./... -run '^$' -bench 'Benchmark(CowSparseWrite|DeepCopySparseWrite)' -benchmem -count=1`

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkCowSparseWrite16-8` | 16531 | 85280 | 94 | `COW` / 16 组件 |
| `BenchmarkDeepCopySparseWrite16-8` | 23366 | 159257 | 163 | `DeepCopy` / 16 组件 |
| `BenchmarkCowSparseWrite64-8` | 66511 | 324514 | 334 | `COW` / 64 组件 |
| `BenchmarkDeepCopySparseWrite64-8` | 95857 | 636958 | 643 | `DeepCopy` / 64 组件 |
| `BenchmarkCowSparseWrite256-8` | 279787 | 1281965 | 1294 | `COW` / 256 组件 |
| `BenchmarkDeepCopySparseWrite256-8` | 393652 | 2548270 | 2563 | `DeepCopy` / 256 组件 |

说明：

- 本轮属于“方向正确”。
- 在 `16 / 64 / 256` 三档规模下，`COW` 的 `ns/op`、`B/op`、`allocs/op` 都稳定优于整根 `DeepCopy`。
- `B/op` 与 `allocs/op` 大致保持在 `DeepCopy` 的一半量级，说明当根规模变大且每次只改一个组件时，当前 `COW` 路线已经体现出明确的结构性优势。
- 为排除单次波动，还额外做了第二次同命令采样，趋势一致：`COW` 在三个规模档都继续快于 `DeepCopy`。

## 2026-05-22 只读事务 benchmark

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值（基准输出为 `-8`）
- commit：`4edfea34c6ea540140ed83f6ba83a79162cd4876`
- 命令：`go test ./... -run '^$' -bench 'Benchmark(CowReadOnly|DeepCopyReadOnly)' -benchmem -count=1`

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkCowReadOnly16-8` | 16856 | 79648 | 83 | `COW` / 16 组件 |
| `BenchmarkDeepCopyReadOnly16-8` | 23476 | 159257 | 163 | eager clone / 16 组件 |
| `BenchmarkCowReadOnly64-8` | 64841 | 318499 | 323 | `COW` / 64 组件 |
| `BenchmarkDeepCopyReadOnly64-8` | 102400 | 636958 | 643 | eager clone / 64 组件 |
| `BenchmarkCowReadOnly256-8` | 267486 | 1274153 | 1283 | `COW` / 256 组件 |
| `BenchmarkDeepCopyReadOnly256-8` | 391338 | 2548272 | 2563 | eager clone / 256 组件 |

说明：

- 本轮属于“方向正确”。
- 在 `16 / 64 / 256` 三档规模下，`COW` 的 `ns/op`、`B/op`、`allocs/op` 都稳定优于 eager clone。
- 差距没有随规模增大而收敛，反而继续拉大，说明 lazy session 在完全不写事务上确实持续避免了整根 eager clone 的固定复制成本。
- 为排除单次波动，还额外做了第二次同命令采样，趋势一致：`COW` 在三个规模档都继续快于 eager clone。
