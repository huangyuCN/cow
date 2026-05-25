# COW 只读事务 Benchmark 设计

## 1. 背景

当前仓库中的 `COW`（Copy-On-Write，写时复制）运行时已经完成：

- 显式 `TxSession` 主模型切换；
- `Begin()` 默认只读、首次写升级；
- 组件级与容器级 `COW` 主写路径；
- 大根稀疏写 benchmark。

最近两轮 benchmark 已经说明：

- 框架层固定分配优化是有效的；
- 在“大根 + 稀疏写”场景下，当前 `COW` 路线已经体现出结构性优势。

但还有一个核心价值点尚未被单独钉死：

- 当事务**完全不写**时，lazy session（懒初始化会话）是否真的把 eager clone（事务开始即整根克隆）的成本几乎完整省掉。

当前已有 benchmark，例如：

- `BenchmarkFrameworkBeginCommitRollback`

虽然已经能部分反映只读事务收益，但它仍然使用小样例根结构，无法观察：

- 当根规模从中等增长到更大时，lazy session 与 eager clone 的差距是否会进一步拉大。

因此，下一轮 benchmark 设计需要专门围绕**完全只读事务**展开，并直接与 eager clone 模型对照。

## 2. 本次设计目标

本轮只覆盖以下目标：

1. 复用现有大根 benchmark 数据模型。
2. 设计一组“完全只读事务” benchmark。
3. 设计一组“事务开始即整根 clone，但最终不写”的 eager clone 对照 benchmark。
4. 让根规模仍按 `16 / 64 / 256` 三档递增。
5. 用结果判断：随着根规模增大，lazy session 相对 eager clone 的优势是否进一步拉大。

## 3. 本次设计不覆盖的范围

本轮不覆盖以下内容：

- 带读访问的只读事务 benchmark；
- 混合流量 benchmark（例如 90% 只读、10% 写）；
- 热点重复写 benchmark；
- `Savepoint` benchmark；
- 新 root 模型；
- 运行时实现重构。

也就是说：

- 本轮只回答一个核心问题：
  - 当事务完全不写时，当前 lazy session 设计是否真正避免了 eager clone 的整根复制成本。

## 4. 方案比较

### 方案 A：最小验证面的只读事务对照

- 复用现有 `benchSparseRoot`
- 根规模固定为 `16 / 64 / 256`
- `COW` 组只执行：
  - `Begin()`
  - 不写
  - `Commit()`
- 对照组执行：
  - 根构造
  - 事务开始即整根 clone
  - 不写
  - 结束

优点：

- 最纯粹地验证 lazy session 的核心收益；
- 与上一轮大根稀疏写 benchmark 连续性最好；
- 结果最容易解释。

缺点：

- 还不能说明读路径开销；
- 也不能说明真实混合流量。

### 方案 B：只读但带读访问

- 在方案 A 基础上，再加入字段读、容器读等只读动作。

优点：

- 更接近真实只读事务。

缺点：

- 第一轮会混入读路径成本；
- 不利于单独回答“无写事务是否还在 clone”。

### 方案 C：混合流量

- 例如 90% 只读事务 + 10% 写事务，观察整体吞吐。

优点：

- 更接近真实业务分布。

缺点：

- 不能纯粹衡量 lazy session 在只读事务上的核心价值；
- 第一轮不利于精确归因。

### 推荐方案

推荐采用**方案 A**。

原因：

- 现在最需要验证的是：当事务完全不写时，当前设计是否真的把 eager clone 的整根复制成本完整省掉；
- 这是 lazy session 设计最核心、也最值得先拿硬证据验证的一点。

## 5. 数据模型复用方案

### 5.1 复用大根 benchmark 模型

本轮建议完全复用上一轮大根稀疏写 benchmark 的数据模型：

```go
type benchSparseRoot struct {
    Comps []*benchSparseComp
}

type benchSparseComp struct {
    Gold  int
    Items map[int]int
}
```

原因：

- 保持与上一轮 benchmark 的数据形状一致；
- 避免引入新的 root 结构变量；
- 让“稀疏写”和“完全只读”两轮 benchmark 可以直接并排比较。

### 5.2 规模梯度

本轮继续固定三档根规模：

- `16`
- `64`
- `256`

原因：

- 与上一轮大根稀疏写 benchmark 连续；
- 更容易观察“只读事务优势是否随根规模扩大而拉大”。

## 6. Benchmark 命名方案

第一轮采用独立 benchmark 名称，不使用 `b.Run(...)` 子基准。

建议新增以下基准：

- `BenchmarkCowReadOnly16`
- `BenchmarkCowReadOnly64`
- `BenchmarkCowReadOnly256`
- `BenchmarkDeepCopyReadOnly16`
- `BenchmarkDeepCopyReadOnly64`
- `BenchmarkDeepCopyReadOnly256`

这样做的好处是：

- 与上一轮 `BenchmarkCowSparseWrite16/64/256` 的命名风格一致；
- 结果一眼就能区分“稀疏写事务”和“完全只读事务”；
- 后续日志归档与趋势对比都更直接。

## 7. Benchmark 流程设计

### 7.1 COW 只读组

每轮 benchmark 固定执行以下流程：

1. 构造 `benchSparseRoot(N)`；
2. `Begin(store, cloneBenchSparseRoot)`；
3. 不做任何写；
4. `Commit()`。

该流程下，按照当前设计，事务应保持只读态：

- `work` 不应被构造；
- `dirty` / `cloned` 不应被构造；
- `Commit()` 直接成功返回。

### 7.2 eager clone 对照组

每轮 benchmark 固定执行以下流程：

1. 构造 `benchSparseRoot(N)`；
2. 事务开始时立刻执行一次整根 `cloneBenchSparseRoot(root)`；
3. 不做任何写；
4. 结束。

该组不要求复刻完整事务 API，而是忠实表达一种旧式策略：

- 事务一开始就整根 clone，即使后面完全不写，也已经支付了复制成本。

### 7.3 对齐约束

两组 benchmark 必须严格对齐以下条件：

- 使用同一份 `benchSparseRoot` 数据模型；
- 使用相同的根规模梯度；
- 都不加入任何读访问；
- 都不加入任何写访问；
- 都不混入 `Savepoint`；
- 差异只保留在“是否一开始就整根 clone”。

这样 benchmark 的结论才能集中回答：

- lazy session 是否真的避免了无写事务上的整根复制浪费。

## 8. 结果判读标准

### 8.1 第一层：规模增长趋势

重点观察：

- 当根规模从 `16 -> 64 -> 256` 增大时；
- eager clone 对照组的 `ns/op`、`B/op`、`allocs/op` 应明显上升；
- `COW` 只读组也可能略有上升，因为根构造本身仍存在；
- 但 `COW` 的增长幅度应显著更小。

若出现这一趋势，说明 lazy session 的核心目的成立：

- 根越大，不写事务越能从“不做整根 clone”中获益。

### 8.2 第二层：差距是否被拉大

这一轮与大根稀疏写 benchmark 的关注点不同。

本轮理想预期是：

- `16` 组件时，`COW` 已经明显优于 eager clone；
- `64` 组件时，差距进一步扩大；
- `256` 组件时，差距继续扩大。

原因是：

- `COW` 组既不会发生整根 eager clone，也不会发生首次写升级、组件复制、容器复制；
- eager clone 组则仍然为完全没发生的写入付出了整根复制成本。

### 8.3 第三层：失败信号

若出现以下现象，则要认真怀疑当前 lazy session 路径或 benchmark 设计：

- `COW` 只读组随规模增大也明显接近 eager clone 的增长曲线；
- `B/op` 和 `allocs/op` 没有显著优势；
- 到 `256` 组件时差距没有拉大，反而接近或恶化。

这通常意味着以下某类问题存在：

- 事务开始时仍有隐藏的大分配；
- `Commit()` 的只读快路径仍然不够轻；
- benchmark 中混入了不该有的额外成本。

### 8.4 结果结论分级

本轮 benchmark 的验收结论建议分成三类：

1. `方向正确`
   - `COW` 明显优于 eager clone，且差距随规模扩大而拉大。
2. `方向可能正确，但只读快路径仍有残余固定成本`
   - 有优势，但扩大不明显。
3. `方向存疑`
   - 只读事务下仍看不到明显优势，或差距不随规模扩大。

## 9. 测试与归档要求

本轮实现后，至少需要运行：

```bash
go test ./... -run '^$' -bench 'Benchmark(CowReadOnly|DeepCopyReadOnly)' -benchmem -count=1
```

以及必要时补一轮重复采样，以确认趋势不是单次波动。

结果反馈时至少应包含：

- 各规模下 `COW` 与 eager clone 的 `ns/op`
- `B/op`
- `allocs/op`
- 差距是否随根规模扩大而拉大
- 对结论的三档分级

若用户确认保留结果：

- 将表格与元数据追加到 `docs/superpowers/benchmarks/cow-mvp-benchmark.md`。

## 10. 结论

下一轮不应一开始就做“只读但带读路径”或“混合流量” benchmark，而应先把“完全不写事务”的 lazy session 价值单独测清楚。

这轮 benchmark 的核心特征是：

- 复用现有大根 benchmark 数据模型；
- 继续沿用 `16 / 64 / 256` 根规模；
- `COW` 组只做 `Begin() -> Commit()`；
- 对照组只表达 eager clone 的固定复制成本。

只有先拿到这组结果，才能更有把握地回答：

- 当前 lazy session 设计在只读事务上的收益是否已经足够明确；
- 还是仍然存在隐藏固定成本，需要继续下钻优化。
