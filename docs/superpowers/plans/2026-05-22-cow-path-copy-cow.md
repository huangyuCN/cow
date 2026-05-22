# COW Path-Copy / Lazy-Clone Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前语义正确但整根克隆的 `COW MVP` 运行时，演进为“组件级 + 容器级”的真实 path-copy / lazy-clone 主写路径，并保留现有事务语义。

**Architecture:** 运行时从 `base + work` 整根副本模型改为 `base + overlay` 组件覆盖层模型。组件入口负责首次 materialize（实体化）组件，容器入口负责首次复制 `map` / `slice`，`Savepoint` 暂时继续通过较重的完整视图快照保持正确性。测试示例扩展为 `Bag + Quest` 双组件，用于证明“改 `Bag` 不触碰 `Quest`”。

**Tech Stack:** Go 1.26、标准库 `context` / `sync/atomic` / `maps` / `slices` / `testing`

---

## 文件结构

### 计划新增文件

- `overlay_test.go`
  - 覆盖双组件共享语义、组件级 materialize 和容器级复制行为。

### 计划修改文件

- `example_types_test.go`
  - 将示例根扩展为 `Bag + Quest`，并新增组件入口、容器入口、完整视图构造辅助函数。
- `session.go`
  - 将 `TxSession` 从 `work` 模型改为 `overlay` 模型，并保留 `Dirty()` 导出能力。
- `run.go`
  - 改为只绑定 `base + overlay`，提交时合成新根。
- `savepoint.go`
  - 从基于 `work` 的快照改为基于“完整视图”的快照恢复，但保持整根快照策略。
- `run_test.go`
  - 调整现有事务测试以匹配双组件示例和新组件入口。
- `savepoint_test.go`
  - 调整为走新主写路径，并补充回滚后结构恢复断言。
- `misuse_test.go`
  - 保持错误语义测试通过，必要时补充 `ErrSessionClosed` 在新结构下的覆盖。
- `benchmark_test.go`
  - 增加双组件场景下的 benchmark，观察 `COW` 路径分配下降趋势。
- `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
  - 追加本轮 benchmark 对比表和元数据，前提是用户确认保留结果。
- `MVP_REQUIREMENTS.md`
  - 默认不改；若实现过程中发现必须澄清的轻微用语冲突，再做最小修订。

## 任务拆分

### Task 1: 扩展示例根为双组件并固定共享语义失败测试

**Files:**
- Modify: `example_types_test.go`
- Create: `overlay_test.go`
- Test: `run_test.go`

- [ ] **Step 1: 扩展测试示例类型**

```go
package cow

import "maps"

type testRoot struct {
	Bag   *testBagComp
	Quest *testQuestComp
}

type testBagComp struct {
	Gold  int
	Items map[int]int
}

type testQuestComp struct {
	Stage int
	Flags map[string]bool
}

func newTestRoot() *testRoot {
	return &testRoot{
		Bag: &testBagComp{
			Gold:  10,
			Items: map[int]int{1001: 1},
		},
		Quest: &testQuestComp{
			Stage: 1,
			Flags: map[string]bool{"daily": true},
		},
	}
}

func cloneTestRoot(src *testRoot) *testRoot {
	next := &testRoot{}
	if src.Bag != nil {
		next.Bag = &testBagComp{
			Gold:  src.Bag.Gold,
			Items: maps.Clone(src.Bag.Items),
		}
	}
	if src.Quest != nil {
		next.Quest = &testQuestComp{
			Stage: src.Quest.Stage,
			Flags: maps.Clone(src.Quest.Flags),
		}
	}
	return next
}
```

- [ ] **Step 2: 写共享语义失败测试**

```go
package cow

import (
	"context"
	"testing"
)

func TestOverlayCommitKeepsUntouchedComponentShared(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	before := store.Load()

	err := Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
		sess, _ := FromContext[testRoot](ctx)
		bag := mutableBag(sess)
		bag.Gold += 5
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	after := store.Load()
	if after.Bag == before.Bag {
		t.Fatal("expected bag component to be replaced after write")
	}
	if after.Quest != before.Quest {
		t.Fatal("expected untouched quest component to remain shared")
	}
}

func TestOverlayDoesNotCloneQuestWhenBagChanges(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	err := Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
		sess, _ := FromContext[testRoot](ctx)
		bag := mutableBag(sess)
		bag.Gold = 20
		if got := sess.Dirty(); len(got) != 1 || got[0] != "bag" {
			t.Fatalf("dirty = %v, want [bag]", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./... -run 'TestOverlay(CommitKeepsUntouchedComponentShared|DoesNotCloneQuestWhenBagChanges)' -count=1`
Expected: FAIL，当前实现会整根克隆，导致 `Quest` 不能保持共享

- [ ] **Step 4: Commit**

```bash
git add example_types_test.go overlay_test.go
git commit -m "test: add overlay sharing semantics tests"
```

### Task 2: 为组件级 materialize 和容器级复制补失败测试

**Files:**
- Modify: `overlay_test.go`
- Modify: `run_test.go`
- Test: `example_types_test.go`

- [ ] **Step 1: 写容器级复制失败测试**

```go
func TestMutableBagItemsCloneMapOnFirstWrite(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	before := store.Load()

	err := Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
		sess, _ := FromContext[testRoot](ctx)
		items := mutableBagItems(sess)
		items[1001]++
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	after := store.Load()
	if after.Bag.Items[1001] != 2 {
		t.Fatalf("item count = %d, want 2", after.Bag.Items[1001])
	}
	if before.Bag.Items[1001] != 1 {
		t.Fatalf("base item count = %d, want 1", before.Bag.Items[1001])
	}
	if after.Bag.Items == before.Bag.Items {
		t.Fatal("expected bag items map to be cloned on write")
	}
}

func TestMutableBagGoldDoesNotCloneItemsMap(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	before := store.Load()

	err := Run(context.Background(), store, cloneTestRoot, func(ctx context.Context) error {
		sess, _ := FromContext[testRoot](ctx)
		bag := mutableBag(sess)
		bag.Gold++
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	after := store.Load()
	if after.Bag.Items != before.Bag.Items {
		t.Fatal("expected items map to stay shared when only gold changes")
	}
}
```

- [ ] **Step 2: 调整现有根事务测试走组件入口**

将 [run_test.go](/Users/huangyu/work/golang/src/cow/run_test.go) 中写路径统一改为：

```go
bag := mutableBag(sess)
```

需要修改的片段包括：

```go
bag := mutableBag(sess)
bag.Gold += 5
bag.Items[1001] = 2
```

和：

```go
bag := mutableBag(sess)
bag.Gold = 99
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./... -run 'Test(MutableBagItemsCloneMapOnFirstWrite|MutableBagGoldDoesNotCloneItemsMap|RunCommit|RunRollbackOnError|RunMarksDirtyComponents)' -count=1`
Expected: FAIL，当前实现没有 `mutableBagItems`，且当前整根克隆模型无法满足共享断言

- [ ] **Step 4: Commit**

```bash
git add overlay_test.go run_test.go
git commit -m "test: add component and container copy tests"
```

### Task 3: 将运行时从 `base + work` 改为 `base + overlay`

**Files:**
- Modify: `session.go`
- Modify: `run.go`
- Modify: `example_types_test.go`
- Test: `overlay_test.go`

- [ ] **Step 1: 调整 `TxSession` 结构**

```go
type TxSession[T any] struct {
	store       Store[T]
	base        *T
	overlay     *T
	cloneRoot   func(*T) *T
	checkpoints []checkpoint[T]
	nextID      SavepointID
	dirty       DirtySet
	finished    bool
}
```

并保留：

```go
func (s *TxSession[T]) Dirty() []string
```

- [ ] **Step 2: 在 `Run` 中改为延迟复制模型**

```go
func Run[T any](
	ctx context.Context,
	store Store[T],
	clone func(*T) *T,
	fn func(context.Context) error,
) (err error) {
	base := store.Load()
	session := &TxSession[T]{
		store:     store,
		base:      base,
		overlay:   new(T),
		cloneRoot: clone,
		dirty:     make(DirtySet),
	}
	txCtx := context.WithValue(ctx, sessionKey{}, session)
	defer func() {
		session.finished = true
		if recovered := recover(); recovered != nil {
			panic(recovered)
		}
	}()
	if err = fn(txCtx); err != nil {
		return err
	}
	store.Commit(commitRoot(session))
	return nil
}
```

- [ ] **Step 3: 在测试辅助中增加提交视图构造函数**

在 [example_types_test.go](/Users/huangyu/work/golang/src/cow/example_types_test.go) 中加入：

```go
func commitRoot(sess *TxSession[testRoot]) *testRoot {
	next := &testRoot{
		Bag:   sess.base.Bag,
		Quest: sess.base.Quest,
	}
	if sess.overlay.Bag != nil {
		next.Bag = sess.overlay.Bag
	}
	if sess.overlay.Quest != nil {
		next.Quest = sess.overlay.Quest
	}
	return next
}
```

- [ ] **Step 4: 运行共享语义测试确认通过**

Run: `go test ./... -run 'TestOverlay(CommitKeepsUntouchedComponentShared|DoesNotCloneQuestWhenBagChanges)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add session.go run.go example_types_test.go overlay_test.go
git commit -m "feat: switch runtime to base plus overlay model"
```

### Task 4: 实现 `Bag` 组件入口和 `map` 首次写复制

**Files:**
- Modify: `example_types_test.go`
- Modify: `session.go`
- Test: `overlay_test.go`
- Test: `run_test.go`

- [ ] **Step 1: 实现 `mutableBag` 的组件级 materialize**

```go
func mutableBag(sess *TxSession[testRoot]) *testBagComp {
	if sess.overlay.Bag == nil {
		baseBag := sess.base.Bag
		sess.overlay.Bag = &testBagComp{
			Gold:  baseBag.Gold,
			Items: baseBag.Items,
		}
	}
	sess.dirty.Mark("bag")
	return sess.overlay.Bag
}
```

- [ ] **Step 2: 增加容器入口 `mutableBagItems`**

```go
func mutableBagItems(sess *TxSession[testRoot]) map[int]int {
	bag := mutableBag(sess)
	if bag.Items == sess.base.Bag.Items {
		bag.Items = maps.Clone(sess.base.Bag.Items)
	}
	return bag.Items
}
```

- [ ] **Step 3: 保留 `Mutable(...)` 作为底层工具**

将 [session.go](/Users/huangyu/work/golang/src/cow/session.go) 中的 `Mutable` 保留为：

```go
func Mutable[T any, C any](sess *TxSession[T], pick func(root *T) *C) *C {
	root := sess.base
	if current := currentRoot(sess); current != nil {
		root = current
	}
	return pick(root)
}
```

并新增：

```go
func currentRoot(sess *TxSession[testRoot]) *testRoot {
	return &testRoot{
		Bag: pickBag(sess),
		Quest: pickQuest(sess),
	}
}
```

如果实现中发现为泛型 `Mutable` 保留这个形态会让范围膨胀，则允许把 `Mutable` 暂时改为“仅供测试未使用”，但必须保持函数存在，避免无关 API 抖动。

- [ ] **Step 4: 运行容器复制和根事务测试**

Run: `go test ./... -run 'Test(MutableBagItemsCloneMapOnFirstWrite|MutableBagGoldDoesNotCloneItemsMap|RunCommit|RunRollbackOnError|RunRollbackOnPanic|RunMarksDirtyComponents)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add example_types_test.go session.go run_test.go overlay_test.go
git commit -m "feat: add bag component materialize and map copy on write"
```

### Task 5: 让 `Savepoint` 适配 overlay 模型但保持整根快照

**Files:**
- Modify: `savepoint.go`
- Modify: `savepoint_test.go`
- Modify: `example_types_test.go`
- Test: `misuse_test.go`

- [ ] **Step 1: 为检查点增加完整视图快照函数**

在 [example_types_test.go](/Users/huangyu/work/golang/src/cow/example_types_test.go) 中加入：

```go
func snapshotRoot(sess *TxSession[testRoot]) *testRoot {
	return cloneTestRoot(commitRoot(sess))
}
```

- [ ] **Step 2: 调整 `Savepoint` 和 `RollbackTo`**

```go
type checkpoint[T any] struct {
	id    SavepointID
	root  *T
	dirty DirtySet
}

func Savepoint[T any](ctx context.Context) (SavepointID, error) {
	session, ok := FromContext[T](ctx)
	if !ok {
		return 0, ErrNoSession
	}
	if session.finished {
		return 0, ErrSessionClosed
	}
	session.nextID++
	cp := checkpoint[T]{
		id:    session.nextID,
		root:  session.cloneRoot(commitRoot(session)),
		dirty: session.dirty.Clone(),
	}
	session.checkpoints = append(session.checkpoints, cp)
	return cp.id, nil
}

func RollbackTo[T any](ctx context.Context, id SavepointID) error {
	session, ok := FromContext[T](ctx)
	if !ok {
		return ErrNoSession
	}
	if session.finished {
		return ErrSessionClosed
	}
	n := len(session.checkpoints)
	if n == 0 || session.checkpoints[n-1].id != id {
		return ErrInvalidSavepoint
	}
	last := session.checkpoints[n-1]
	session.checkpoints = session.checkpoints[:n-1]
	session.base = last.root
	session.overlay = new(T)
	session.dirty = last.dirty.Clone()
	return nil
}
```

- [ ] **Step 3: 调整 `Savepoint` 测试走新写路径**

将 [savepoint_test.go](/Users/huangyu/work/golang/src/cow/savepoint_test.go) 中的写操作替换为：

```go
bag := mutableBag(sess)
bag.Gold = 20
```

以及：

```go
items := mutableBagItems(sess)
items[1001]++
```

并新增一个断言：

```go
if store.Load().Quest.Stage != 1 {
	t.Fatalf("quest stage = %d, want 1", store.Load().Quest.Stage)
}
```

- [ ] **Step 4: 运行 `Savepoint` 与误用测试**

Run: `go test ./... -run 'Test(Savepoint|RollbackRejectsConsumedSavepoint|SavepointWithoutSession|RollbackWithoutSession|SavepointAfterRunReturnsSessionClosed)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add savepoint.go savepoint_test.go misuse_test.go example_types_test.go
git commit -m "feat: adapt savepoint snapshots to overlay runtime"
```

### Task 6: 更新 benchmark 并验证复制粒度下降

**Files:**
- Modify: `benchmark_test.go`
- Modify: `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
- Test: `overlay_test.go`

- [ ] **Step 1: 调整 benchmark 为双组件场景**

```go
func BenchmarkRunWithCow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
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

func BenchmarkRunWithDeepCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneTestRoot(newTestRoot())
		root.Bag.Gold++
		root.Bag.Items[1001]++
	}
}
```

- [ ] **Step 2: 运行全部测试**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 3: 跑 benchmark 并保存输出**

Run: `go test ./... -bench 'BenchmarkRunWith(Cow|DeepCopy)$' -benchmem -run '^$' -count=1 > /tmp/cow-path-copy-bench.txt`
Expected: `/tmp/cow-path-copy-bench.txt` 生成成功

- [ ] **Step 4: 与当前基线对比**

Run: `benchstat /tmp/cow-mvp-bench.txt /tmp/cow-path-copy-bench.txt`
Expected: 输出新旧对比；至少能看出 `BenchmarkRunWithCow` 的 `B/op` 与 `allocs/op` 下降趋势

- [ ] **Step 5: 若用户确认保留结果，再追加 benchmark 日志**

```markdown
## 2026-05-22 path-copy / lazy-clone 版本

| 基准名 | 前次 ns/op | 本次 ns/op | 前次 B/op | 本次 B/op | 前次 allocs/op | 本次 allocs/op | 相对变化 |
|---|---:|---:|---:|---:|---:|---:|---|
| `BenchmarkRunWithCow-8` | 225.3 | <fill after run> | 840 | <fill after run> | 13 | <fill after run> | `benchstat` 输出 |
| `BenchmarkRunWithDeepCopy-8` | 99.79 | <fill after run> | 408 | <fill after run> | 6 | <fill after run> | `benchstat` 输出 |
```

- [ ] **Step 6: Commit**

```bash
git add benchmark_test.go docs/superpowers/benchmarks/cow-mvp-benchmark.md
git commit -m "test: benchmark path copy cow runtime"
```

## 自检

### Spec Coverage

- 双组件示例和共享语义：Task 1
- 容器级复制：Task 2 和 Task 4
- `base + overlay` 结构：Task 3
- 组件入口成为主写路径、`Mutable(...)` 降级：Task 4
- `Savepoint` 保持较重快照但适配新结构：Task 5
- benchmark 以复制粒度下降为信号：Task 6

### Placeholder Scan

- 计划中的实现步骤都给出了明确代码形态与命令。
- 唯一待填写项仅限 Task 6 的 benchmark 数值回填，且它们依赖运行结果，属于执行后填表，不影响实现步骤。

### Type Consistency

- `testRoot`、`testBagComp`、`testQuestComp`、`TxSession`、`SavepointID`、`DirtySet`、`mutableBag`、`mutableBagItems` 命名保持一致。
- `Run`、`FromContext`、`Savepoint`、`RollbackTo` 继续沿用现有 API 命名，避免无关接口漂移。
