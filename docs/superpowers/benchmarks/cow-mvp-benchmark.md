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
