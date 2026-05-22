# COW 框架层固定分配优化设计

## 1. 背景

当前 `COW`（Copy-On-Write，写时复制）运行时已经完成两轮关键收敛：

- 主事务 API 已切换为显式 `TxSession`；
- 根事务主写路径已经进入组件级 + 容器级 `COW`；
- `Run` / `context` 主线已经被清理出主实现。

最近一轮 benchmark 诊断继续说明了一个核心事实：

- path-copy 主写路径本身已经不再是唯一主要问题；
- 事务框架层固定成本仍然偏高；
- 空事务生命周期 `Begin + Commit` 本身就带来了明显的 `B/op` 与 `allocs/op`。

当前实现中，`Begin()` 会立即分配：

- `work`
- `dirty`
- `cloned`

而 `Savepoint()` 还会沿用“默认已存在可写工作态”的结构假设。

这意味着即使事务只是：

- 只读访问；
- 开启后立刻 `Commit()`；
- 开启后只做只读 `Savepoint()`；

也会提前承担本不必要的分配成本。

因此，本轮优化的目标不是继续改 path-copy 算法，而是**削减事务框架层的固定分配**，把“未写事务”退化成真正的轻量只读会话。

## 2. 本次设计目标

本轮只覆盖以下目标：

1. 让 `TxSession` 在 `Begin()` 后默认处于只读态。
2. 让 `Begin()` 不再立即构造 `work`、`dirty`、`cloned`、`checkpoints`。
3. 让第一次真实写入时再升级为可写事务态。
4. 让只读事务的 `Commit()` / `Rollback()` / `Savepoint()` 不强制触发可写升级。
5. 让 benchmark 能反映框架层固定成本的下降，尤其是 `BenchmarkFrameworkBeginCommitRollback`。

## 3. 本次设计不覆盖的范围

本轮不覆盖以下内容：

- 再次重写组件级 / 容器级 path-copy 结构；
- 对象池；
- `Store` 接口改造；
- `Savepoint` 深度性能优化；
- 自动代码生成；
- 宿主集成或仓储层适配。

也就是说：

- 本轮解决的是“事务开始后为什么立刻分配这么多东西”；
- 不是再做一轮更大范围的运行时重写。

## 4. 方案比较

### 方案 A：保持当前预分配模型，仅做局部小修

- `Begin()` 继续立即构造 `work`、`dirty`、`cloned`；
- 仅尝试压缩个别辅助分配。

优点：

- 改动最小；
- 对现有代码路径干扰最少。

缺点：

- 主要固定成本仍然保留；
- 只读事务仍然为并不会发生的写入付费；
- 很难显著改善框架层 benchmark。

### 方案 B：读写分阶段，首次写入时升级

- `Begin()` 只保存只读事务最小状态；
- `work` 和附属结构都延迟到第一次真实写入时创建；
- 只读 `Commit()` / `Rollback()` / `Savepoint()` 保持只读语义，不触发升级。

优点：

- 直接命中当前 benchmark 主要问题；
- 不需要改动外部显式会话 API；
- 与后续继续做更细粒度优化完全兼容。

缺点：

- 需要补齐“只读态 / 可写态”状态转换语义；
- `Savepoint` 结构要能表示“只读检查点”。

### 方案 C：拆成只读会话类型和可写会话类型

- `Begin()` 返回只读会话；
- 第一次写入时切到另一套可写会话对象。

优点：

- 类型层面边界最强。

缺点：

- 需要更大 API 改造；
- 会把一次内部优化升级成外部模型变更；
- 当前阶段收益不值得额外复杂度。

### 推荐方案

推荐采用**方案 B**。

原因：

- 它直接针对 benchmark 暴露出的固定成本问题；
- 它保留当前显式会话 API，不会打断现有调用方式；
- 它只引入必要的内部状态机复杂度，范围可控。

## 5. 运行时状态模型

### 5.1 只读初始态

`Begin()` 返回的 `TxSession` 初始只保存：

- `store`
- `base`
- `clone`
- `finished`
- `nextID`

其余字段初始保持 `nil`：

- `work`
- `dirty`
- `cloned`
- `checkpoints`

也就是说：

- `base` 代表当前已提交根状态；
- `work == nil` 表示该事务尚未进入可写态；
- “是否已升级为可写态”可以直接通过 `work != nil` 判断。

### 5.2 可写升级后的状态

第一次真实写入发生后，会话升级为可写态：

- `work` 被初始化为根级浅拷贝；
- 后续组件级 / 容器级 `COW` 继续发生在 `work` 上；
- `dirty` 和 `cloned` 在首次需要时再初始化。

这样可以保证：

- 只读事务不再为工作副本付费；
- 已有 path-copy 主写路径仍然可以沿用现有核心思路。

## 6. 写入升级设计

### 6.1 升级触发点

升级必须由**真实写路径入口**触发，而不是由普通读操作触发。

推荐由现有组件写入口统一负责，例如：

```go
func mutableBag(sess *TxSession[testRoot]) *testBagComp
func mutableBagItems(sess *TxSession[testRoot]) map[int]*testItem
```

这些入口在返回可写组件前先执行一次“确保可写会话”：

```go
func ensureWritableSession[T any](sess *TxSession[T]) *T
```

该步骤负责：

1. 校验会话未关闭；
2. 若 `work == nil`，执行根级浅拷贝并写入 `work`；
3. 返回当前事务内可写根。

### 6.2 根级升级语义

根级升级只负责构造“本事务自己的根视图”，不在这里提前做所有辅助分配。

也就是说，第一次写入时：

- 必须创建 `work`；
- 不要求同时创建 `dirty`；
- 不要求同时创建 `cloned`；
- 不要求同时创建 `checkpoints`。

辅助结构应继续按需延迟。

### 6.3 辅助结构延迟初始化

建议约束如下：

- 首次 `markDirty(name)` 时，若 `dirty == nil`，先创建 `DirtySet`；
- 首次组件 / 容器克隆标记时，若 `cloned == nil`，先创建 `DirtySet`；
- 首次 `Savepoint()` 时，若需要保存检查点，再创建 `checkpoints` 切片。

这样可以避免出现“已经只读提交了，但仍然提前创建过 map”的情况。

## 7. 生命周期语义

### 7.1 Commit

`Commit()` 分为两种路径：

- 若 `finished == true`，返回 `ErrSessionClosed`；
- 若 `work == nil`，说明事务始终未写入，直接结束会话并返回成功；
- 若 `work != nil`，按现有语义提交 `work` 到 `Store`，然后结束会话。

只读 `Commit()` 的关键要求是：

- 不构造 `work`；
- 不构造 `dirty`；
- 不向 `Store` 提交一个新克隆根。

### 7.2 Rollback

`Rollback()` 保持无错风格。

当事务尚未结束时：

- 无论当前是只读态还是可写态，都直接把会话标记为 `finished`；
- 不触发任何补偿性分配。

### 7.3 Dirty

`Dirty()` 在只读事务上应返回空切片语义。

这意味着：

- `dirty == nil` 时也必须对外表现为“没有脏组件”；
- 不能因为调用 `Dirty()` 而初始化底层 `DirtySet`。

## 8. Savepoint 语义调整

### 8.1 只读 Savepoint 不触发升级

`Savepoint()` 在只读事务上必须允许成功，但不能因此触发可写升级。

因此需要支持“只读检查点”：

- 记录 `id`
- 记录该检查点是否处于可写态
- 若是可写态，再保存对应根快照与 `dirty` 快照

建议检查点结构演进为：

```go
type checkpoint[T any] struct {
    id       SavepointID
    writable bool
    root     *T
    dirty    DirtySet
}
```

其中：

- `writable == false` 表示该检查点创建时事务仍是只读态；
- 这时 `root` 与 `dirty` 可以保持 `nil`。

### 8.2 可写 Savepoint 继续沿用当前正确性优先策略

当事务已进入可写态时，`Savepoint()` 继续沿用当前策略：

- `root` 保存当前完整工作视图快照；
- `dirty` 保存当时的脏组件集合快照。

本轮不要求优化这一步的复制成本。

### 8.3 RollbackTo 的恢复语义

`RollbackTo(id)` 的恢复逻辑分为两类：

- 若目标检查点是只读检查点：
  - 弹出该检查点；
  - 将 `work` 置回 `nil`；
  - 将 `dirty`、`cloned` 置回 `nil`；
  - 恢复为只读事务态。
- 若目标检查点是可写检查点：
  - 弹出该检查点；
  - 用快照恢复 `work`；
  - `dirty` 恢复为快照副本；
  - `cloned` 重新置空，由后续写路径重新建立。

这样可以保证：

- “先只读 `Savepoint()`，后发生写入，再 `RollbackTo()`” 能回到真正的只读态；
- `RollbackTo()` 不会错误保留旧的可写副作用。

## 9. 对现有实现的结构影响

本轮建议的最小结构调整如下：

1. `Begin()` 不再预分配 `work` / `dirty` / `cloned`。
2. 引入统一的“确保可写会话”内部辅助函数。
3. `Mutable(...)` 或组件写入口改为在访问写路径前先确保会话已升级。
4. `checkpoint` 结构增加 `writable` 语义。
5. `Savepoint()` / `RollbackTo()` 显式处理只读检查点。

本轮不建议做的事：

- 不把 `Savepoint` 改造成增量日志；
- 不把 `dirty` / `cloned` 合并成更复杂的状态对象；
- 不引入对象池。

## 10. 测试设计

本轮新增或调整的测试至少应覆盖以下场景。

### 10.1 Begin 默认只读

- `Begin()` 后 `work == nil`
- `dirty == nil`
- `cloned == nil`
- `checkpoints == nil`

### 10.2 首次写触发升级

- 第一次组件写入前事务是只读态；
- 第一次组件写入后 `work != nil`；
- 未发生写入前调用只读辅助方法不会触发升级。

### 10.3 只读 Commit / Rollback

- 只读事务 `Commit()` 成功；
- 只读事务 `Commit()` 后状态关闭；
- 只读事务 `Rollback()` 不触发分配并关闭会话。

### 10.4 只读 Savepoint

- 只读事务 `Savepoint()` 成功；
- 执行只读 `Savepoint()` 后 `work` 仍为 `nil`；
- 只读 `Savepoint()` 后直接 `RollbackTo()` 可恢复为只读态；
- “只读 `Savepoint()` -> 写入 -> `RollbackTo()`” 后，事务恢复到只读态。

### 10.5 可写路径回归

- 现有组件级 / 容器级 `COW` 语义不退化；
- 已有 `Commit` / `Rollback` / `Savepoint` 行为测试继续通过；
- `ErrSessionClosed` 与 `ErrInvalidSavepoint` 行为不变。

## 11. Benchmark 验收

本轮 benchmark 的主要验收对象不是深写路径，而是框架层固定成本。

至少需要重新观察以下 benchmark：

- `BenchmarkFrameworkBeginCommitRollback`
- `BenchmarkCowWritePathInSessionLifecycle`
- `BenchmarkEndToEndSessionWithCow`

期望信号如下：

1. `BenchmarkFrameworkBeginCommitRollback` 的 `B/op` 和 `allocs/op` 明显下降。
2. `BenchmarkCowWritePathOnSession` 可能变化有限，因为它本来就绕开了部分生命周期固定成本。
3. `BenchmarkCowWritePathInSessionLifecycle` 与 `BenchmarkEndToEndSessionWithCow` 应随框架层成本下降而同步改善。
4. 只读事务相关新增 benchmark 若补充，应体现“无写事务不构造 `work`”。

本轮不强制要求：

- `COW` 端到端立即全面快于当前 `DeepCopy`；
- `Savepoint` benchmark 出现明显改善。

## 12. 风险与边界

本轮主要风险有三类：

### 12.1 写入口遗漏升级

若某条写路径未先执行“确保可写会话”，就可能在 `work == nil` 时直接访问可写视图。

对应要求是：

- 所有正式写入口都必须统一走会话升级辅助函数；
- 测试要覆盖首次写路径。

### 12.2 只读检查点恢复不彻底

若 `RollbackTo()` 回到只读检查点时没有清空 `work` / `dirty` / `cloned`，事务会残留错误写状态。

对应要求是：

- 明确把“恢复只读态”写成显式逻辑；
- 补充专门测试，而不是只依赖已有检查点测试。

### 12.3 只读 Commit 仍偷偷分配

若实现里把“生成提交根”藏进 `Commit()`，仍会破坏本轮目标。

对应要求是：

- 明确规定 `work == nil` 时 `Commit()` 直接成功返回；
- 用 benchmark 和测试双重约束。

## 13. 结论

下一阶段应把 `TxSession` 改造成“默认只读、首次写升级”的轻量事务模型。

具体落点是：

- `Begin()` 只保留最小状态；
- `work`、`dirty`、`cloned`、`checkpoints` 全部延迟初始化；
- 只读 `Commit()` / `Rollback()` / `Savepoint()` 不触发可写升级；
- `RollbackTo()` 必须能正确恢复到只读检查点。

这样做能直接针对当前 benchmark 中最明显的框架层固定分配问题下刀，同时不打断已经形成的显式会话 API 与 path-copy 主写路径方向。
