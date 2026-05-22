# COW 框架层固定分配优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `TxSession` 改造成“默认只读、首次写升级”的轻量事务模型，去掉 `Begin()` 的固定分配，并让只读 `Commit()` / `Rollback()` / `Savepoint()` 不再触发可写工作态构造。

**Architecture:** `Begin()` 只保留 `store/base/clone` 等最小状态，`work`、`dirty`、`cloned`、`checkpoints` 全部延迟到首次需要时再初始化。写路径通过统一的内部升级辅助函数进入可写态；`Savepoint` 通过 `writable` 标志区分只读检查点与可写检查点，`RollbackTo()` 可以恢复到真正的只读事务态。

**Tech Stack:** Go 1.25 兼容写法、标准库 `testing` / `maps` / `slices`

---

## 文件结构

### 计划修改文件

- `begin.go`
  - 让 `Begin()` 只返回最小只读会话，不再预分配 `work` / `dirty` / `cloned`。
- `session.go`
  - 增加内部“首次写升级”辅助逻辑，调整 `Commit()`、`Dirty()` 行为，保留显式会话 API。
- `dirty.go`
  - 让 `DirtySet.Clone()` 对 `nil` 集合保持安全、语义明确。
- `savepoint.go`
  - 让检查点支持只读态，显式恢复只读 `RollbackTo()` 语义。
- `example_types_test.go`
  - 让测试中的组件写入口走“首次写升级”与延迟 `dirty/cloned` 初始化。
- `run_test.go`
  - 新增只读会话初始态、只读 `Commit()`、只读 `Dirty()` 的行为测试。
- `savepoint_test.go`
  - 新增只读 `Savepoint()` 与回滚到只读检查点的行为测试。
- `bench_support_test.go`
  - 让 benchmark 辅助会话与真实只读初始态保持一致。
- `benchmark_test.go`
  - 保持基准名称不变，复用新会话模型验证固定成本下降。

### 本轮不改动文件

- `store.go`
  - `Store` 接口不在本轮范围内。
- `errors.go`
  - 现有错误类型足够，不新增错误值。
- `overlay_test.go`
  - 作为 path-copy 回归验证继续保留，除非实现细节迫使编译适配。

## 任务拆分

### Task 1: 先写只读会话生命周期的失败测试

**Files:**
- Modify: `run_test.go`

- [ ] **Step 1: 在 `run_test.go` 新增只读初始态与只读提交测试**

把 [run_test.go](/Users/huangyu/work/golang/src/cow/run_test.go) 追加为：

```go
func TestBeginStartsReadOnly(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if sess.work != nil {
		t.Fatal("work should stay nil before first write")
	}
	if sess.dirty != nil {
		t.Fatal("dirty should stay nil before first write")
	}
	if sess.cloned != nil {
		t.Fatal("cloned should stay nil before first write")
	}
	if sess.checkpoints != nil {
		t.Fatal("checkpoints should stay nil before first savepoint")
	}
}

func TestReadOnlyCommitSkipsWorkAllocation(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	base := store.Load()

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if store.Load() != base {
		t.Fatal("read-only commit should keep committed root pointer unchanged")
	}
	if sess.work != nil {
		t.Fatal("read-only commit should not materialize work root")
	}
	if !sess.finished {
		t.Fatal("session should be finished after commit")
	}
}

func TestDirtyOnReadOnlySessionReturnsEmpty(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	dirty := sess.Dirty()
	if len(dirty) != 0 {
		t.Fatalf("dirty = %v, want empty", dirty)
	}
	if sess.dirty != nil {
		t.Fatal("Dirty() should not allocate dirty set on read-only session")
	}
}
```

- [ ] **Step 2: 运行新增测试，确认当前实现失败**

Run: `go test ./... -run 'Test(BeginStartsReadOnly|ReadOnlyCommitSkipsWorkAllocation|DirtyOnReadOnlySessionReturnsEmpty)$' -count=1`

Expected: FAIL，因为当前 `Begin()` 仍会预分配 `work`、`dirty`、`cloned`。

- [ ] **Step 3: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入实现任务。

### Task 2: 实现只读初始态与首次写升级骨架

**Files:**
- Modify: `begin.go`
- Modify: `session.go`
- Modify: `dirty.go`
- Modify: `example_types_test.go`
- Test: `run_test.go`

- [ ] **Step 1: 让 `Begin()` 只创建最小只读会话**

把 [begin.go](/Users/huangyu/work/golang/src/cow/begin.go) 改成：

```go
package cow

func Begin[T any](store Store[T], clone func(*T) *T) (*TxSession[T], error) {
	return &TxSession[T]{
		store: store,
		base:  store.Load(),
		clone: clone,
	}, nil
}
```

- [ ] **Step 2: 在 `session.go` 增加首次写升级与延迟标记辅助**

把 [session.go](/Users/huangyu/work/golang/src/cow/session.go) 调整为：

```go
package cow

import "slices"

type TxSession[T any] struct {
	store       Store[T]
	base        *T
	work        *T
	clone       func(*T) *T
	checkpoints []checkpoint[T]
	nextID      SavepointID
	dirty       DirtySet
	cloned      DirtySet
	finished    bool
}

func Mutable[T any, C any](sess *TxSession[T], pick func(root *T) *C) *C {
	return pick(sess.work)
}

func (s *TxSession[T]) ensureWritable() *T {
	if s.work == nil {
		work := new(T)
		*work = *s.base
		s.work = work
	}
	return s.work
}

func (s *TxSession[T]) markDirty(name string) {
	if s.dirty == nil {
		s.dirty = make(DirtySet)
	}
	s.dirty.Mark(name)
}

func (s *TxSession[T]) markCloned(name string) {
	if s.cloned == nil {
		s.cloned = make(DirtySet)
	}
	s.cloned.Mark(name)
}

func (s *TxSession[T]) Commit() error {
	if s.finished {
		return ErrSessionClosed
	}
	if s.work != nil {
		s.store.Commit(s.work)
	}
	s.finished = true
	return nil
}

func (s *TxSession[T]) Rollback() {
	s.finished = true
}

func (s *TxSession[T]) Dirty() []string {
	names := make([]string, 0, len(s.dirty))
	for name := range s.dirty {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
```

- [ ] **Step 3: 让 `DirtySet.Clone()` 对 `nil` 集合保持安全**

把 [dirty.go](/Users/huangyu/work/golang/src/cow/dirty.go) 改成：

```go
package cow

type DirtySet map[string]struct{}

func (d DirtySet) Mark(name string) {
	d[name] = struct{}{}
}

func (d DirtySet) Clone() DirtySet {
	if d == nil {
		return nil
	}
	next := make(DirtySet, len(d))
	for name := range d {
		next[name] = struct{}{}
	}
	return next
}
```

- [ ] **Step 4: 让测试写入口负责升级会话和延迟脏标记**

把 [example_types_test.go](/Users/huangyu/work/golang/src/cow/example_types_test.go) 中的 `mutableBag` 与 `mutableBagItems` 改成：

```go
func mutableBag(sess *TxSession[testRoot]) *testBagComp {
	root := sess.ensureWritable()
	sess.markDirty("bag")
	if root.Bag == sess.base.Bag {
		baseBag := sess.base.Bag
		root.Bag = &testBagComp{
			Gold:  baseBag.Gold,
			Items: baseBag.Items,
		}
	}
	return Mutable(sess, func(root *testRoot) *testBagComp { return root.Bag })
}

func mutableBagItems(sess *TxSession[testRoot]) map[int]int {
	bag := mutableBag(sess)
	if _, ok := sess.cloned["bag.items"]; !ok {
		bag.Items = maps.Clone(sess.base.Bag.Items)
		sess.markCloned("bag.items")
	}
	return bag.Items
}
```

- [ ] **Step 5: 运行只读生命周期测试，确认通过**

Run: `go test ./... -run 'Test(BeginStartsReadOnly|ReadOnlyCommitSkipsWorkAllocation|DirtyOnReadOnlySessionReturnsEmpty)$' -count=1`

Expected: PASS

- [ ] **Step 6: 运行现有提交/回滚/overlay 回归测试**

Run: `go test ./... -run 'Test(CommitAppliesChanges|RollbackDiscardsChanges|CommitMarksDirtyComponents|Overlay.*)$' -count=1`

Expected: PASS

- [ ] **Step 7: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入下一组测试。

### Task 3: 先写只读 Savepoint 的失败测试

**Files:**
- Modify: `savepoint_test.go`

- [ ] **Step 1: 在 `savepoint_test.go` 新增只读检查点测试**

把 [savepoint_test.go](/Users/huangyu/work/golang/src/cow/savepoint_test.go) 追加为：

```go
func TestReadOnlySavepointDoesNotUpgradeSession(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	sp, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}
	if sp == 0 {
		t.Fatal("savepoint id should be assigned")
	}
	if sess.work != nil {
		t.Fatal("read-only savepoint should not materialize work root")
	}
	if len(sess.checkpoints) != 1 {
		t.Fatalf("checkpoint count = %d, want 1", len(sess.checkpoints))
	}
	if sess.checkpoints[0].writable {
		t.Fatal("read-only savepoint should record writable=false")
	}
}

func TestRollbackToReadOnlySavepointRestoresReadOnlyState(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	sp, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold = 99
	if sess.work == nil {
		t.Fatal("write should materialize work root")
	}

	if err := sess.RollbackTo(sp); err != nil {
		t.Fatalf("RollbackTo() error = %v", err)
	}
	if sess.work != nil {
		t.Fatal("rollback to read-only savepoint should clear work root")
	}
	if sess.dirty != nil {
		t.Fatal("rollback to read-only savepoint should clear dirty set")
	}
	if sess.cloned != nil {
		t.Fatal("rollback to read-only savepoint should clear cloned set")
	}

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got := store.Load().Bag.Gold; got != 10 {
		t.Fatalf("gold after commit = %d, want 10", got)
	}
}
```

- [ ] **Step 2: 运行新增 Savepoint 测试，确认当前实现失败**

Run: `go test ./... -run 'Test(ReadOnlySavepointDoesNotUpgradeSession|RollbackToReadOnlySavepointRestoresReadOnlyState)$' -count=1`

Expected: FAIL，因为当前 `Savepoint()` 仍假定 `s.work` 已存在，并且 `RollbackTo()` 还不会恢复只读态。

- [ ] **Step 3: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入实现任务。

### Task 4: 实现只读检查点恢复语义并校准 benchmark 辅助

**Files:**
- Modify: `savepoint.go`
- Modify: `bench_support_test.go`
- Modify: `benchmark_test.go`
- Test: `savepoint_test.go`

- [ ] **Step 1: 让检查点显式表示只读态**

把 [savepoint.go](/Users/huangyu/work/golang/src/cow/savepoint.go) 改成：

```go
package cow

type SavepointID uint64

type checkpoint[T any] struct {
	id       SavepointID
	writable bool
	root     *T
	dirty    DirtySet
}

func (s *TxSession[T]) Savepoint() (SavepointID, error) {
	if s.finished {
		return 0, ErrSessionClosed
	}
	s.nextID++
	cp := checkpoint[T]{
		id:       s.nextID,
		writable: s.work != nil,
	}
	if cp.writable {
		cp.root = s.clone(s.work)
		cp.dirty = s.dirty.Clone()
	}
	s.checkpoints = append(s.checkpoints, cp)
	return cp.id, nil
}

func (s *TxSession[T]) RollbackTo(id SavepointID) error {
	if s.finished {
		return ErrSessionClosed
	}
	n := len(s.checkpoints)
	if n == 0 || s.checkpoints[n-1].id != id {
		return ErrInvalidSavepoint
	}
	last := s.checkpoints[n-1]
	s.checkpoints = s.checkpoints[:n-1]
	if !last.writable {
		s.work = nil
		s.dirty = nil
		s.cloned = nil
		return nil
	}
	s.work = s.clone(last.root)
	s.dirty = last.dirty.Clone()
	s.cloned = nil
	return nil
}
```

- [ ] **Step 2: 让 benchmark 辅助会话符合新的只读初始态**

把 [bench_support_test.go](/Users/huangyu/work/golang/src/cow/bench_support_test.go) 改成：

```go
package cow

func newBenchSession() *TxSession[testRoot] {
	base := newTestRoot()
	return &TxSession[testRoot]{
		base:  base,
		clone: cloneTestRoot,
	}
}
```

- [ ] **Step 3: 保持 benchmark 名称稳定，只复用新会话模型**

检查 [benchmark_test.go](/Users/huangyu/work/golang/src/cow/benchmark_test.go)，确保以下函数名保持不变，且继续通过 `Begin()` / `newBenchSession()` 使用新 lazy session 语义：

```go
func BenchmarkFrameworkBeginCommitRollback(b *testing.B)
func BenchmarkFrameworkEmptyClosure(b *testing.B)
func BenchmarkCowWritePathOnSession(b *testing.B)
func BenchmarkCowWritePathInSessionLifecycle(b *testing.B)
func BenchmarkDeepCopyWritePath(b *testing.B)
func BenchmarkEndToEndSessionWithCow(b *testing.B)
func BenchmarkEndToEndSessionWithDeepCopy(b *testing.B)
```

这一步通常不需要改函数体；如果实现调整导致编译错误，只做最小修复，不重命名 benchmark。

- [ ] **Step 4: 运行只读 Savepoint 与既有 Savepoint 回归测试**

Run: `go test ./... -run 'Test(SavepointRollbackToLatest|SavepointRejectsOutOfOrderRollback|ReadOnlySavepointDoesNotUpgradeSession|RollbackToReadOnlySavepointRestoresReadOnlyState|RollbackRejectsConsumedSavepoint)$' -count=1`

Expected: PASS

- [ ] **Step 5: 运行完整测试**

Run: `go test ./... -count=1`

Expected: PASS

- [ ] **Step 6: 运行基准测试并生成对比**

Run: `go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndSession.*)$' -benchmem -count=1 > /tmp/cow-framework-fixed-allocation-bench.txt`

Expected: 输出包含以下 benchmark 名称：

```text
BenchmarkFrameworkBeginCommitRollback
BenchmarkFrameworkEmptyClosure
BenchmarkCowWritePathOnSession
BenchmarkCowWritePathInSessionLifecycle
BenchmarkDeepCopyWritePath
BenchmarkEndToEndSessionWithCow
BenchmarkEndToEndSessionWithDeepCopy
```

Run: `benchstat /tmp/cow-explicit-session-bench.txt /tmp/cow-framework-fixed-allocation-bench.txt`

Expected: `BenchmarkFrameworkBeginCommitRollback` 的 `B/op` 与 `allocs/op` 低于上一轮显式会话基线 `712 B/op`、`12 allocs/op`；`BenchmarkCowWritePathInSessionLifecycle` 与 `BenchmarkEndToEndSessionWithCow` 也出现同步下降趋势。

- [ ] **Step 7: 整理结果，等待用户确认是否归档 benchmark 或提交代码**

此处不要执行 `git commit`。把以下内容准备好反馈给用户：

- 通过的测试命令
- benchmark 对比结果
- 是否需要把本轮 benchmark 追加归档到 `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
- 建议的提交摘要与提交信息

## 自检清单

- spec 中的四个关键目标都有落点：
  - 默认只读 `Begin()`：Task 1、Task 2
  - 首次写升级：Task 2
  - 只读 `Commit()` / `Rollback()` / `Savepoint()` 不升级：Task 1、Task 3、Task 4
  - benchmark 聚焦框架固定成本下降：Task 4
- 计划中没有 `TODO` / `TBD` / `<fill after run>` 之类占位符。
- 函数名、测试名、benchmark 名都与当前仓库真实名字对齐。
