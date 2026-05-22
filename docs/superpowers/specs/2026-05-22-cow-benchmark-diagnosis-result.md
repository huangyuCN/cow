# COW Benchmark 诊断结果

## 1. 执行环境

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值（基准输出为 `-8`）
- 命令：`go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndSession.*)$' -benchmem -count=1`

## 2. benchmark 分组结果

### 2.1 框架层

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkFrameworkBeginCommitRollback-8` | 218.6 | 760 | 13 | `Begin` + `Commit` 空事务 |
| `BenchmarkFrameworkEmptyClosure-8` | 0.2484 | 0 | 0 | 最小空路径对照 |

### 2.2 纯写路径层

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkCowWritePathOnSession-8` | 223.2 | 688 | 9 | 底层会话直测 |
| `BenchmarkCowWritePathInSessionLifecycle-8` | 382.7 | 1384 | 18 | 接近真实事务生命周期 |
| `BenchmarkDeepCopyWritePath-8` | 225.5 | 944 | 11 | 纯 `DeepCopy` 写路径 |

### 2.3 端到端层

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkEndToEndSessionWithCow-8` | 399.0 | 1384 | 18 | 整体显式会话 `COW` 事务 |
| `BenchmarkEndToEndSessionWithDeepCopy-8` | 244.1 | 944 | 11 | 整体显式会话 `DeepCopy` 路径 |

## 3. 诊断结论

### 3.1 框架层固定成本已经足够重

`BenchmarkFrameworkBeginCommitRollback` 显示，仅仅是：

- `Begin`
- `TxSession` 初始化
- `DirtySet` / `cloned` 初始化
- `Commit`

这些固定成本，就已经带来了：

- `218.6 ns/op`
- `760 B/op`
- `13 allocs/op`

这说明：当前 `COW` 端到端路径里，**框架层不是可忽略噪声，而是主要成本来源之一**。

### 3.2 纯写路径本身并不比 `DeepCopy` 更差

`BenchmarkCowWritePathOnSession` 与 `BenchmarkDeepCopyWritePath` 对比显示：

- `COW` 写路径：`223.2 ns/op`，`688 B/op`，`9 allocs/op`
- `DeepCopy` 写路径：`225.5 ns/op`，`944 B/op`，`11 allocs/op`

这说明：

- 在去掉外围事务框架成本后，当前 path-copy 写路径已经接近甚至略优于 `DeepCopy`；
- 当前主写路径的组件级 / 容器级复制方向没有跑偏；
- “path-copy 方向错了”不是当前 benchmark 支持的结论。

### 3.3 真正拖慢整体结果的是“框架层 + 写路径”叠加

`BenchmarkCowWritePathInSessionLifecycle` 与 `BenchmarkEndToEndSessionWithCow` 都落在：

- `1384 B/op`
- `18 allocs/op`

这与 `BenchmarkCowWritePathOnSession` 的 `688 B/op`、`9 allocs/op` 形成明显对比。

结论是：

- 一旦进入真实事务生命周期，`Begin/Commit/session` 的固定成本几乎把写路径本身的分配再抬高了一倍；
- 当前整体 benchmark 之所以比 `DeepCopy` 难看，主要不是 `mutableBag` / `mutableBagItems` 本身，而是外围事务框架成本过高。

## 4. 下一步建议

基于本轮 benchmark，下一轮性能优化应优先考虑**框架层固定成本**，而不是继续优先怀疑 path-copy 写路径。

优先排查方向：

1. `Begin` / `Commit` 每次初始化的对象数量是否可以减少。
2. `DirtySet` 与 `cloned` 是否可以降分配或延迟初始化。
3. 会话生命周期本身是否仍有可进一步压缩的固定开销。
4. `Store` 与事务初始化是否可以在 benchmark 或运行时中减少重复准备成本。

写路径层仍可继续优化，但它不再是第一嫌疑人。

## 5. 结论摘要

这轮 benchmark 诊断给出的最重要结论是：

- **path-copy 写路径本身已经基本站住了；**
- **当前整体性能问题主要集中在事务框架固定成本。**

因此，下一轮更合理的工作顺序应是：

1. 先为框架层固定成本做新的头脑风暴和设计收敛；
2. 确认要优化哪些固定分配和初始化；
3. 再进入新的实现计划。
