# COW Benchmark 诊断设计

## 1. 背景

当前仓库已经完成两轮与 `COW`（Copy-On-Write，写时复制）相关的实现与 benchmark：

- 第一轮：语义正确的 `MVP` 事务运行时；
- 第二轮：主写路径从整根深拷贝推进到“根浅拷贝 + 组件级 / 容器级按需复制”。

但第二轮 benchmark 没有得到预期中的性能改善，反而出现了更差的结果。当前数据说明：

- path-copy 方向的语义目标已经成立；
- 现有 benchmark 还不能清楚回答“慢在哪一层”。

问题不在于“没有 benchmark”，而在于**当前 benchmark 的归因粒度不够**。现有 benchmark 把以下成本混在了一起：

- `Run` 主入口固定成本；
- `context.Context` 绑定与读取成本；
- `Store` 构造与根对象准备成本；
- `DirtySet` 初始化与标记成本；
- 组件首次 materialize 成本；
- 容器首次复制成本；
- benchmark 自身示例根构造成本。

因此，在继续优化运行时代码之前，需要先把 benchmark 结构重构为更适合做性能诊断的形态。

## 2. 本次设计目标

本次设计只覆盖 benchmark 诊断结构，不直接实现新的性能优化。

目标如下：

1. 将 benchmark 拆成“事务框架成本”和“纯写路径成本”两大类。
2. 对“纯写路径成本”同时保留两个视角：
   - 底层会话直测；
   - 接近真实事务体的测法。
3. 保留现有端到端 benchmark，作为整体视角。
4. 让 benchmark 输出能够支持后续判断：下一步优先优化框架层还是写路径层。
5. 在结果出来后，再回写一份诊断结论，而不是先入为主修改运行时设计。

## 3. 本次设计不覆盖的范围

以下内容不在本轮范围内：

- 修改 `Run`、`TxSession`、`Savepoint` 等运行时代码；
- 新增对象池或其他性能优化机制；
- 调整 `MVP_REQUIREMENTS.md`；
- 讨论最终性能目标是否已经达成；
- 对 path-copy 结构再做一轮设计变更。

也就是说，这一轮的任务是**先把测量口径做对**。

## 4. 当前 benchmark 的问题

当前 benchmark 形态大致如下：

```go
func BenchmarkRunWithCow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
		_ = Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
			sess, _ := FromContext[testRoot](ctx)
			bag := mutableBag(sess)
			bag.Gold++
			bag.Items[1001]++
			return nil
		})
	}
}
```

这个 benchmark 的问题是：它把至少四类成本揉在了一起。

### 4.1 框架固定成本

- `newMemoryStore(newTestRoot())`
- `Run(...)`
- `context.WithValue(...)`
- `TxSession` 初始化
- `DirtySet` / 其他辅助结构初始化

### 4.2 事务体成本

- `FromContext(...)`
- `mutableBag(sess)`
- 写 `Bag.Gold`
- 写 `Bag.Items`

### 4.3 示例构造成本

- `newTestRoot()`
- 初始 `map` 分配

### 4.4 结构切换成本

- 提交时 `store.Commit(...)`

在这种口径下，即使“纯写路径”已经有收益，也可能被框架固定成本完全掩盖。

## 5. 推荐的 benchmark 结构

推荐将 benchmark 重构为三组，而不是继续只保留单一端到端测法。

### 5.1 组一：事务框架成本

目的：只测框架本身的固定成本，不混入真正的业务写路径。

建议形态：

```go
func BenchmarkRunFrameworkOnly(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
		if err := Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
			_, _ = FromContext[testRoot](ctx)
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
}
```

它用于回答：

- `Run` 自己有多重；
- 事务还没开始真正写数据时，已经付出了多少固定成本。

### 5.2 组二：纯写路径成本

目的：尽量把框架外围成本挪开，只看 path-copy 写路径本身。

这组需要保留两个视角。

#### 视角 A：底层会话直测

```go
func BenchmarkCowWritePathOnSession(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newTestRoot()
		sess := newTestSessionForBench(base)
		bag := mutableBag(sess)
		bag.Gold++
		items := mutableBagItems(sess)
		items[1001]++
	}
}
```

这个 benchmark 的目标是：

- 近距离观察组件 materialize 和容器复制成本；
- 不让 `Run`、`context`、`Store` 成本掩盖信号。

#### 视角 B：接近真实事务体

```go
func BenchmarkCowWritePathInRunBody(b *testing.B) {
	store := newMemoryStore(newTestRoot())
	for i := 0; i < b.N; i++ {
		if err := Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
			sess, _ := FromContext[testRoot](ctx)
			bag := mutableBag(sess)
			bag.Gold++
			items := mutableBagItems(sess)
			items[1001]++
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
}
```

这个 benchmark 的目标是：

- 保留接近真实事务用法的视角；
- 同时减少每轮重复构造 `Store` 的噪声。

### 5.3 组三：端到端整体成本

现有 benchmark 保留，但明确降级为“整体视角”，不再单独承担性能归因职责。

它用于回答：

- 从调用入口到提交完成，整体一轮事务有多重；
- 是否存在某一轮改动虽然改善了底层写路径，却让整体成本变差。

## 6. 对照组设计

为了让 benchmark 结果更有解释力，需要为每一组匹配合理的对照口径。

### 6.1 事务框架成本的对照

对于框架成本，建议对照以下两类实现：

- `Run` + 空事务体；
- 纯函数调用或最小空闭包。

这样可以大致看出框架层开销有多少来自事务基础设施本身。

### 6.2 纯写路径成本的对照

对于纯写路径，建议保留：

- `COW` 写路径；
- 全量 `DeepCopy` 后写路径。

但这里的 `DeepCopy` 应该也尽量剥离框架外围噪声，例如：

```go
func BenchmarkDeepCopyWritePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneTestRoot(newTestRoot())
		root.Bag.Gold++
		root.Bag.Items[1001]++
	}
}
```

这样比较的是“写路径本体”，而不是“事务框架 + 写路径”。

### 6.3 端到端整体成本的对照

整体视角可以继续保留：

- `BenchmarkRunWithCow`
- `BenchmarkRunWithDeepCopy`

但后续解释结果时，应明确说明：这组数据只用于看整体成本，不用于单独定位瓶颈。

## 7. benchmark 文件组织建议

推荐将 benchmark 分成三层命名，而不是继续只按 `Cow` / `DeepCopy` 平铺。

建议命名：

- `BenchmarkFrameworkRunOnly`
- `BenchmarkCowWritePathOnSession`
- `BenchmarkCowWritePathInRunBody`
- `BenchmarkDeepCopyWritePath`
- `BenchmarkEndToEndRunWithCow`
- `BenchmarkEndToEndRunWithDeepCopy`

这样做的好处是：

- 从 benchmark 名字就能看出它在测哪一层；
- 后续 benchmark 表格不会混成一团；
- 诊断结果更容易回写到文档里。

## 8. 结果解释规则

这一轮 benchmark 重构后，后续解释结果必须按层次进行，而不能只看一个总表。

### 8.1 如果框架成本高

若 `BenchmarkFrameworkRunOnly` 已经很重，说明下一步优先优化：

- `Run` 固定成本；
- `context` 绑定；
- `TxSession` 初始化；
- 小对象分配和辅助结构初始化。

### 8.2 如果纯写路径高

若 `BenchmarkCowWritePathOnSession` 本身已经很重，说明下一步优先优化：

- `mutableBag` 的组件 materialize；
- `mutableBagItems` 的容器复制；
- `DirtySet` / cloned 标记；
- 写路径上不必要的辅助调用。

### 8.3 如果底层轻、整体重

若底层写路径较轻，但端到端仍然重，说明问题主要不在 path-copy 本体，而在外围事务框架成本。

## 9. 实施建议

实现顺序建议如下：

1. 保留当前 benchmark 作为端到端视角；
2. 新增“框架空跑” benchmark；
3. 新增“底层会话直测” benchmark；
4. 新增“接近真实事务体” benchmark；
5. 对每一组添加对应的 `DeepCopy` 或空路径对照；
6. 跑一次完整 benchmark，和当前基线一起保存；
7. 基于结果再写一份诊断结论文档或补充 spec。

## 10. 结论

这轮工作的核心不是继续猜测“哪段代码该优化”，而是先把 benchmark 口径组织成能回答问题的结构。

本设计的输出应该帮助后续明确三件事：

1. 当前 path-copy 版本到底慢在框架层还是写路径层；
2. 下一轮性能优化应优先打哪一层；
3. 后续 benchmark 日志应该按什么结构归档，才不会混淆整体成本与局部成本。
