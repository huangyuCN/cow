# COW Path-Copy / Lazy-Clone 设计

## 1. 背景

当前仓库中的 `COW MVP` 已经具备以下能力：

- `Run(ctx, store, fn)` 作为纯库级事务入口；
- 根事务提交、错误回滚、`panic` 回滚；
- 多层栈式 `Savepoint / RollbackTo`；
- 简版 `DirtySet`；
- 基础 benchmark 与 benchmark 归档。

但当前实现仍然不是“真正的 `COW`（Copy-On-Write，写时复制）”：

- `Run` 开始时直接对整根做一次 `clone(base)`；
- `Savepoint` 仍依赖整根快照；
- 示例写路径还没有明确区分“组件首次写复制”和“容器首次写复制”；
- benchmark 已证明当前 `RunWithCow` 仍慢于全量 `DeepCopy`。

因此，下一阶段的目标不是扩功能，而是把当前“语义正确但复制粒度太粗”的实现推进到真正的 path-copy / lazy-clone（懒克隆）主写路径。

## 2. 本次设计目标

本次设计只覆盖以下目标：

1. 在现有最小运行时和测试骨架上演进。
2. 将根事务主写路径改为真正的**组件级 + 容器级** `COW`。
3. 保留 `Mutable(...)` 作为底层工具，但测试、文档和主写路径转向明确的组件入口。
4. 用新增测试证明“修改 `Bag` 时未修改组件保持共享”。
5. 用 benchmark 证明复制粒度和分配路径下降。

## 3. 本次设计不覆盖的范围

以下内容不在本轮范围内：

- `Savepoint` 的性能优化；
- 元素级 `EnsureWritable`，例如 `map[int]*Item` 中指针元素的逐元素可写保护；
- 代码生成器；
- 宿主适配层；
- 仓储层持久化策略；
- 多组件通用自动化 path-copy 框架。

这意味着：

- `Savepoint` 继续允许使用较重的整根快照方案；
- 真正优化的重点只放在**平时最常走的根事务主写路径**。

## 4. 方案比较

### 方案 A：组件覆盖层（overlay）+ 组件入口按需 materialize

- `TxSession` 不再在 `Run` 开始时整根克隆；
- 会话持有 `base root + overlay root`；
- 组件写入口在首次写时把目标组件从 `base` 浅拷贝到 `overlay`；
- 容器如 `map`、`slice` 在首次写时再各自复制；
- 未修改组件继续共享 `base`。

优点：

- 改动集中，最符合当前 `MVP` 演进路径；
- 容易通过测试证明“改 `Bag` 不触碰 `Quest`”；
- 不需要先重写底层泛化工具。

缺点：

- 组件入口会比当前更显式；
- `Savepoint` 暂时会与主写路径采用不同复制策略。

### 方案 B：扩展通用 `Mutable(...)` 成为自动 path-copy 框架

- 试图让底层工具统一理解组件复制、容器复制、dirty 标记和写路径切换。

优点：

- 长期看更通用。

缺点：

- 当前抽象没有足够信息描述“这是哪个组件、何时该复制容器、何时该记 dirty”；
- 很容易把这一步做成半个框架重写，范围膨胀明显。

### 方案 C：仅为 `Bag` 做专用优化通道

- 不先演进运行时结构，只针对 `Bag` 写一条专用 path-copy 路径。

优点：

- 最快能看到单示例效果。

缺点：

- 容易把示例代码和运行时抽象割裂开；
- 后续还需要再回填通用结构。

### 推荐方案

推荐采用**方案 A**。

原因：

- 它最符合本轮的范围控制；
- 它能直接把主写路径从“假 `COW`”推进到“真 `COW`”；
- 它允许 `Savepoint` 暂时保持较重实现，而不影响根事务主线收敛。

## 5. 运行时结构调整

### 5.1 现状问题

当前 `TxSession` 直接保存：

- `base`：事务开始时根状态；
- `work`：通过 `clone(base)` 得到的完整工作副本。

这个结构的问题是：即使事务只修改一个组件，也已经付出了整根复制成本。

### 5.2 目标结构

下一阶段建议把 `TxSession` 改为：

```go
type TxSession[T any] struct {
    store       Store[T]
    base        *T
    overlay     *T
    cloneRoot   func(*T) *T
    checkpoints []checkpoint[T]
    dirty       DirtySet
    finished    bool
}
```

语义如下：

- `base`：事务开始时的已提交根；
- `overlay`：只保存已经被 materialize（实体化）的组件；
- `cloneRoot`：仅供 `Savepoint` 快照与必要的完整视图构造；
- `dirty`：当前事务已被写过的组件集合。

### 5.3 overlay 语义

`overlay` 的关键语义是：

- 某个组件指针为 `nil`，表示当前事务仍共享 `base` 的该组件；
- 某个组件指针非 `nil`，表示该组件已经进入当前事务的可写视图；
- 事务提交时，以 `overlay` 中已实体化组件覆盖 `base`，构造新的根状态。

因此，“未修改组件保持共享”在结构上会自然成立，而不是靠调用方自觉不去复制。

## 6. 组件入口与容器级复制

### 6.1 组件入口成为主写路径

下一阶段的主写路径不再以通用 `Mutable(...)` 为核心，而是转向明确的组件入口，例如：

```go
func mutableBag(sess *TxSession[testRoot]) *testBagComp
```

该入口负责：

1. 检查 `overlay.Bag` 是否已经 materialize；
2. 若未 materialize，则从 `base.Bag` 做组件浅拷贝写入 `overlay.Bag`；
3. 标记 `dirty["bag"]`；
4. 返回当前事务内可写的 `Bag` 组件。

### 6.2 `Mutable(...)` 的角色

`Mutable(...)` 仍可保留，但降级为底层工具，不再作为文档和测试主推入口。

原因是：

- 真正的 path-copy/COW 需要知道“当前正在写哪个组件”；
- 还需要控制 dirty 标记和首次容器复制时机；
- 这些语义单靠泛化 `Mutable(...)` 很难表达清楚。

### 6.3 容器级复制

组件已经 materialize，并不代表组件内的 `map` 或 `slice` 已经可以安全共享写入。

因此，下一阶段还需要做到：

- 第一次写 `Bag.Items` 时执行 `maps.Clone`；
- 若后续在同一事务中继续写 `Bag.Items`，则直接写已复制的容器；
- 若组件存在 `slice`，则首次写时复制底层数组；
- 本轮不处理容器内指针元素的逐元素 `EnsureWritable`。

也就是说：

- **组件级 `COW`** 解决“这个组件是否需要复制”；
- **容器级 `COW`** 解决“这个组件里的容器是否需要复制”。

## 7. Savepoint 在本轮的挂接方式

### 7.1 明确边界

本轮不优化 `Savepoint` 性能。

### 7.2 保持较重但正确的实现

推荐继续使用较重的检查点策略：

- 创建 `Savepoint` 时，将“当前事务完整视图”整根快照保存；
- 快照中同时保存当时的 `DirtySet`；
- `RollbackTo` 时直接恢复该根快照和对应的 `DirtySet`。

这样做的代价是：

- 检查点仍然偏重；
- `Savepoint` 性能不会与主写路径一起改善。

但收益是：

- 语义清晰；
- 范围稳定；
- 不会把本轮工作扩大成“双线优化”。

## 8. 测试设计

为了证明真正的 path-copy/COW 已经成立，示例根需要从单组件扩展为至少两个组件，例如：

```go
type testRoot struct {
    Bag   *testBagComp
    Quest *testQuestComp
}
```

### 8.1 共享语义测试

新增测试至少覆盖：

- 事务中只修改 `Bag` 时，`Quest` 不被 materialize；
- 提交后，新的根状态中 `Quest` 仍可复用旧组件对象；
- 只修改 `Bag.Items` 时，不应触发 `Quest` 的 dirty 标记。

### 8.2 容器复制测试

新增测试至少覆盖：

- 写 `Bag.Gold` 不应强制复制 `Bag.Items`；
- 第一次写 `Bag.Items` 时触发容器复制；
- 回滚后，已提交态中的 `Bag.Items` 保持原值。

### 8.3 语义回归测试

现有测试必须继续通过：

- 根事务提交/错误回滚/`panic` 回滚；
- 多层栈式 `Savepoint`；
- 无事务上下文和已消费检查点的错误语义；
- `ErrSessionClosed` 行为。

## 9. 性能验收

本轮性能验收不要求立刻优于当前 `DeepCopy` benchmark。

本轮真正关心的是以下信号：

1. 相对当前版本，`RunWithCow` 的 `B/op` 和 `allocs/op` 下降；
2. 在新增第二组件 `Quest` 后，`DeepCopy` 路径会随着整根变大而继续受影响；
3. 新的 `COW` 路径在只修改 `Bag` 时，不应为未修改组件支付同等复制成本。

因此，本轮 benchmark 的目标是证明**复制粒度下降**，而不是立即追求绝对胜负。

## 10. 实施建议

实现顺序建议如下：

1. 扩展示例根，加入第二组件 `Quest`；
2. 先写共享语义和容器复制的失败测试；
3. 将 `TxSession` 从 `base + work` 改为 `base + overlay`；
4. 将 `mutableBag(sess)` 改为真正的组件级 materialize 入口；
5. 为 `Bag.Items` 增加首次写容器复制；
6. 保持 `Savepoint` 的较重快照实现，仅适配新结构；
7. 重跑事务测试与 benchmark，对比当前基线。

## 11. 结论

本轮设计的本质不是继续增加功能，而是把当前语义正确的 `MVP` 运行时推进到更符合目标的复制模型：

- 主写路径从整根克隆改为组件级 + 容器级 `COW`；
- 未修改组件保持共享；
- `Savepoint` 暂时接受较重快照；
- benchmark 以复制粒度下降为验收信号。

当这一轮完成后，后续才值得继续讨论：

- `Savepoint` 的性能优化；
- 元素级 `EnsureWritable`；
- 更通用的组件访问框架；
- 代码生成器。
