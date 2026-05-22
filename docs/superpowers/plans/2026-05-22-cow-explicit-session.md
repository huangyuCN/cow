# COW Explicit Session API Migration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `COW` 事务主模型从 `Run/context` 切换为显式 `TxSession`，彻底移除 `Run` / `FromContext` / 包级 `Savepoint`，同时保持现有事务语义与 path-copy 写路径不退化。

**Architecture:** 核心 API 切换为 `Begin(store, clone)`、`sess.Commit()`、`sess.Rollback()`、`sess.Savepoint()`、`sess.RollbackTo(sp)`。内部继续沿用当前已跑通的 `base/work/dirty/cloned/checkpoints` 结构，不在这一轮顺手重写 path-copy 内部布局。测试和 benchmark 全部迁移到显式会话主线。

**Tech Stack:** Go 1.26、标准库 `testing` / `maps` / `slices`

---

## 文件结构

### 计划新增文件

- `begin.go`
  - 提供新的 `Begin` 主入口。

### 计划删除文件

- `run.go`
  - 删除旧 `Run` 事务入口。
- `context.go`
  - 删除旧 `FromContext` 上下文读取入口。

### 计划修改文件

- `session.go`
  - 为 `TxSession` 增加 `Commit()`、`Rollback()`、`Savepoint()`、`RollbackTo()` 方法，并收紧 closed 状态。
- `savepoint.go`
  - 将包级检查点函数迁移到会话方法，必要时保留类型定义。
- `run_test.go`
  - 全部迁移为显式 `Begin/Commit/Rollback` 风格。
- `savepoint_test.go`
  - 全部迁移为会话方法风格。
- `misuse_test.go`
  - 错误语义改为围绕显式会话验证。
- `overlay_test.go`
  - 共享语义测试改为显式会话风格。
- `benchmark_test.go`
  - benchmark 全部迁移为显式会话模型，不再出现 `Run/context`。
- `bench_support_test.go`
  - 调整 benchmark 辅助构造，适配新的显式会话主线。
- `example_types_test.go`
  - 保持组件级 / 容器级写路径辅助函数围绕显式会话使用。
- `docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md`
  - 如新 benchmark 结果改变结论表述，做最小修订。
- `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
  - 若用户确认保留新结果，追加显式会话轮次数据。

## 任务拆分

### Task 1: 先写显式会话 API 的失败测试

**Files:**
- Modify: `run_test.go`
- Modify: `savepoint_test.go`
- Modify: `misuse_test.go`

- [ ] **Step 1: 将根事务测试改为显式会话风格**

把 [run_test.go](/Users/huangyu/work/golang/src/cow/run_test.go) 改成：

```go
func TestCommitAppliesChanges(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold += 5
	items := mutableBagItems(sess)
	items[1001] = 2

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	committed := store.Load()
	if committed.Bag.Gold != 15 {
		t.Fatalf("gold = %d, want 15", committed.Bag.Gold)
	}
	if committed.Bag.Items[1001] != 2 {
		t.Fatalf("item count = %d, want 2", committed.Bag.Items[1001])
	}
}

func TestRollbackDiscardsChanges(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold = 99
	sess.Rollback()

	committed := store.Load()
	if committed.Bag.Gold != 10 {
		t.Fatalf("gold = %d, want 10", committed.Bag.Gold)
	}
}
```

- [ ] **Step 2: 将 `Savepoint` 测试改为方法式 API**

把 [savepoint_test.go](/Users/huangyu/work/golang/src/cow/savepoint_test.go) 中：

```go
sp1, err := Savepoint[testRoot](ctx)
```

统一改为：

```go
sp1, err := sess.Savepoint()
```

并将：

```go
if err := RollbackTo[testRoot](ctx, sp1); err != nil { ... }
```

改为：

```go
if err := sess.RollbackTo(sp1); err != nil { ... }
```

- [ ] **Step 3: 将误用测试改为显式会话 API**

在 [misuse_test.go](/Users/huangyu/work/golang/src/cow/misuse_test.go) 中新增：

```go
func TestCommitAfterRollbackReturnsSessionClosed(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	sess.Rollback()

	if err := sess.Commit(); err != ErrSessionClosed {
		t.Fatalf("error = %v, want %v", err, ErrSessionClosed)
	}
}
```

- [ ] **Step 4: 运行测试确认失败**

Run: `go test ./... -run 'Test(CommitAppliesChanges|RollbackDiscardsChanges|Savepoint|CommitAfterRollbackReturnsSessionClosed)' -count=1`
Expected: FAIL，当前尚无 `Begin`、方法式 `Commit` / `Savepoint`

- [ ] **Step 5: Commit**

```bash
git add run_test.go savepoint_test.go misuse_test.go
git commit -m "test: add explicit session api tests"
```

### Task 2: 删除旧入口并实现 `Begin/Commit/Rollback`

**Files:**
- Create: `begin.go`
- Delete: `run.go`
- Delete: `context.go`
- Modify: `session.go`
- Modify: `errors.go`

- [ ] **Step 1: 新增 `Begin` 入口**

```go
package cow

func Begin[T any](store Store[T], clone func(*T) *T) (*TxSession[T], error) {
	base := store.Load()
	work := new(T)
	*work = *base
	return &TxSession[T]{
		store:  store,
		base:   base,
		work:   work,
		clone:  clone,
		dirty:  make(DirtySet),
		cloned: make(DirtySet),
	}, nil
}
```

- [ ] **Step 2: 在 `TxSession` 上实现生命周期方法**

在 [session.go](/Users/huangyu/work/golang/src/cow/session.go) 中加入：

```go
func (s *TxSession[T]) Commit() error {
	if s.finished {
		return ErrSessionClosed
	}
	s.store.Commit(s.work)
	s.finished = true
	return nil
}

func (s *TxSession[T]) Rollback() {
	s.finished = true
}
```

- [ ] **Step 3: 删除旧入口文件**

删除：

- [run.go](/Users/huangyu/work/golang/src/cow/run.go)
- [context.go](/Users/huangyu/work/golang/src/cow/context.go)

并从测试和 benchmark 中移除所有 `Run` / `FromContext` 调用。

- [ ] **Step 4: 运行事务语义测试确认通过**

Run: `go test ./... -run 'Test(CommitAppliesChanges|RollbackDiscardsChanges|CommitAfterRollbackReturnsSessionClosed)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add begin.go session.go errors.go run_test.go misuse_test.go
git rm run.go context.go
git commit -m "feat: replace run context api with begin commit rollback"
```

### Task 3: 将 `Savepoint` 迁移为会话方法

**Files:**
- Modify: `savepoint.go`
- Modify: `savepoint_test.go`
- Modify: `misuse_test.go`

- [ ] **Step 1: 将包级 `Savepoint` / `RollbackTo` 迁移为方法**

把 [savepoint.go](/Users/huangyu/work/golang/src/cow/savepoint.go) 改为：

```go
func (s *TxSession[T]) Savepoint() (SavepointID, error) {
	if s.finished {
		return 0, ErrSessionClosed
	}
	s.nextID++
	cp := checkpoint[T]{
		id:    s.nextID,
		root:  s.clone(s.work),
		dirty: s.dirty.Clone(),
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
	s.work = s.clone(last.root)
	s.cloned = make(DirtySet)
	s.dirty = last.dirty.Clone()
	return nil
}
```

- [ ] **Step 2: 清理误用测试中的旧包级调用**

将 [misuse_test.go](/Users/huangyu/work/golang/src/cow/misuse_test.go) 中旧的：

```go
Savepoint[testRoot](...)
RollbackTo[testRoot](...)
```

全部改为基于 `sess` 的方法调用。

- [ ] **Step 3: 运行检查点测试确认通过**

Run: `go test ./... -run 'Test(Savepoint|RollbackRejectsConsumedSavepoint|SavepointAfterRunReturnsSessionClosed)' -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add savepoint.go savepoint_test.go misuse_test.go
git commit -m "feat: move savepoint api onto tx session"
```

### Task 4: 迁移共享语义测试与 benchmark 到显式会话模型

**Files:**
- Modify: `overlay_test.go`
- Modify: `benchmark_test.go`
- Modify: `bench_support_test.go`
- Modify: `docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md`

- [ ] **Step 1: 将共享语义测试改为显式会话**

把 [overlay_test.go](/Users/huangyu/work/golang/src/cow/overlay_test.go) 中：

```go
err := Run(...)
```

风格统一改成：

```go
sess, err := Begin(store, cloneTestRoot)
if err != nil {
	t.Fatalf("Begin() error = %v", err)
}
bag := mutableBag(sess)
bag.Gold += 5
if err := sess.Commit(); err != nil {
	t.Fatalf("Commit() error = %v", err)
}
```

- [ ] **Step 2: 将 benchmark 彻底切到显式会话模型**

把 [benchmark_test.go](/Users/huangyu/work/golang/src/cow/benchmark_test.go) 中：

```go
BenchmarkFrameworkRunOnly
```

改为：

```go
func BenchmarkFrameworkBeginCommitRollback(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
		sess, err := Begin(store, cloneTestRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}
```

并将：

- `BenchmarkCowWritePathInRunBody` 改为 `BenchmarkCowWritePathInSessionLifecycle`
- `BenchmarkEndToEndRunWithCow` 改为 `BenchmarkEndToEndSessionWithCow`
- `BenchmarkEndToEndRunWithDeepCopy` 改为 `BenchmarkEndToEndSessionWithDeepCopy`

全部去掉 `Run` / `FromContext` 依赖。

- [ ] **Step 3: 更新 benchmark 诊断结论文档中的名称与表述**

将 [2026-05-22-cow-benchmark-diagnosis-result.md](/Users/huangyu/work/golang/src/cow/docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md) 中涉及：

- `BenchmarkFrameworkRunOnly`
- `BenchmarkCowWritePathInRunBody`
- `BenchmarkEndToEndRunWithCow`
- `BenchmarkEndToEndRunWithDeepCopy`

的命名与文字，更新为显式会话版本。

- [ ] **Step 4: 运行全量测试**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add overlay_test.go benchmark_test.go bench_support_test.go docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md
git commit -m "test: migrate tests and benchmarks to explicit sessions"
```

### Task 5: 重跑 benchmark 并决定是否归档

**Files:**
- Modify: `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
- Test: `benchmark_test.go`

- [ ] **Step 1: 运行显式会话 benchmark 套件**

Run: `go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndSession.*)$' -benchmem -count=1 > /tmp/cow-explicit-session-bench.txt`
Expected: `/tmp/cow-explicit-session-bench.txt` 生成成功

- [ ] **Step 2: 与上一轮诊断 benchmark 对比**

Run: `benchstat /tmp/cow-benchmark-diagnosis.txt /tmp/cow-explicit-session-bench.txt`
Expected: 输出显式会话版本相对上一轮的变化

- [ ] **Step 3: 若用户确认保留结果，再追加 benchmark 日志**

```markdown
## 2026-05-22 显式会话 API 轮次

- 命令：`go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndSession.*)$' -benchmem -count=1`

| 分组 | 基准名 | 前次 ns/op | 本次 ns/op | 前次 B/op | 本次 B/op | 前次 allocs/op | 本次 allocs/op | 相对变化 |
|---|---|---:|---:|---:|---:|---:|---:|---|
| 框架层 | `BenchmarkFrameworkBeginCommitRollback` | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | `benchstat` 输出 |
| 写路径层 | `BenchmarkCowWritePathOnSession` | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | `benchstat` 输出 |
| 写路径层 | `BenchmarkCowWritePathInSessionLifecycle` | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | `benchstat` 输出 |
| 端到端 | `BenchmarkEndToEndSessionWithCow` | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | `benchstat` 输出 |
| 端到端 | `BenchmarkEndToEndSessionWithDeepCopy` | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | <fill after run> | `benchstat` 输出 |
```

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/benchmarks/cow-mvp-benchmark.md docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md
git commit -m "test: record explicit session benchmark results"
```

## 自检

### Spec Coverage

- 显式会话主 API：Task 1 到 Task 3
- 旧入口彻底清理：Task 2 和 Task 3
- 写路径与共享语义保持不退化：Task 4
- benchmark 全部迁移到显式会话模型：Task 4 和 Task 5

### Placeholder Scan

- 所有实现步骤都给出了明确代码与命令。
- Task 5 的 `<fill after run>` 仅用于执行后回填 benchmark 数值，不影响实现步骤本身。

### Type Consistency

- `Begin`、`Commit`、`Rollback`、`Savepoint`、`RollbackTo`、`TxSession` 在全计划中保持一致。
- benchmark 命名统一迁移到显式会话版本，避免旧 `Run` 术语残留。
