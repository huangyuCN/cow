# COW Benchmark Diagnosis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 `COW` benchmark 口径，将“事务框架固定成本”和“纯写路径成本”拆开测量，并产出一份可指导下一轮性能优化的诊断结果。

**Architecture:** 保留现有端到端 benchmark 作为整体视角，新增“框架空跑”“底层会话直测”“接近真实事务体”三类 benchmark。先把归因结构测清楚，再基于数据回写诊断结论，明确下一步优先优化框架层还是写路径层。

**Tech Stack:** Go 1.26、标准库 `testing` / `context` / `maps`、`benchstat`

---

## 文件结构

### 计划新增文件

- `bench_support_test.go`
  - 放置 benchmark 专用的轻量辅助构造函数，避免把测试断言辅助和 benchmark 辅助混在一起。
- `docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md`
  - 记录本轮 benchmark 重构后的诊断结论。

### 计划修改文件

- `benchmark_test.go`
  - 重构 benchmark 结构，增加框架成本和纯写路径成本两层视角。
- `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
  - 在用户确认保留结果后追加本轮 benchmark 表格和元数据。
- `docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-design.md`
  - 默认不改；若跑数后发现术语或命名需要最小修订，再做最小更新。

## 任务拆分

### Task 1: 先补 benchmark 支撑函数与失败编译检查

**Files:**
- Create: `bench_support_test.go`
- Modify: `benchmark_test.go`

- [ ] **Step 1: 新建 benchmark 辅助函数文件**

```go
package cow

func newBenchSession() *TxSession[testRoot] {
	base := newTestRoot()
	work := new(testRoot)
	*work = *base
	return &TxSession[testRoot]{
		base:   base,
		work:   work,
		dirty:  make(DirtySet),
		cloned: make(DirtySet),
	}
}
```

- [ ] **Step 2: 将现有 benchmark 改名为端到端视角**

把 [benchmark_test.go](/Users/huangyu/work/golang/src/cow/benchmark_test.go) 改为：

```go
func BenchmarkEndToEndRunWithCow(b *testing.B) {
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

func BenchmarkEndToEndRunWithDeepCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneTestRoot(newTestRoot())
		root.Bag.Gold++
		root.Bag.Items[1001]++
	}
}
```

- [ ] **Step 3: 运行编译型 benchmark 检查**

Run: `go test ./... -run '^$' -bench 'BenchmarkEndToEndRunWith(Cow|DeepCopy)$' -count=1`
Expected: PASS，仅输出两条重命名后的 benchmark

- [ ] **Step 4: Commit**

```bash
git add bench_support_test.go benchmark_test.go
git commit -m "test: rename end to end cow benchmarks"
```

### Task 2: 先写框架成本 benchmark，再运行验证

**Files:**
- Modify: `benchmark_test.go`
- Test: `bench_support_test.go`

- [ ] **Step 1: 增加框架空跑 benchmark**

```go
func BenchmarkFrameworkRunOnly(b *testing.B) {
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

- [ ] **Step 2: 增加最小空路径对照**

```go
func BenchmarkFrameworkEmptyClosure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = func() error {
			return nil
		}()
	}
}
```

- [ ] **Step 3: 运行框架层 benchmark**

Run: `go test ./... -run '^$' -bench 'BenchmarkFramework(RunOnly|EmptyClosure)$' -benchmem -count=1`
Expected: PASS，并输出两条框架层 benchmark

- [ ] **Step 4: Commit**

```bash
git add benchmark_test.go
git commit -m "test: add framework overhead benchmarks"
```

### Task 3: 先写纯写路径 benchmark，再运行验证

**Files:**
- Modify: `benchmark_test.go`
- Modify: `bench_support_test.go`

- [ ] **Step 1: 增加底层会话直测 benchmark**

```go
func BenchmarkCowWritePathOnSession(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sess := newBenchSession()
		bag := mutableBag(sess)
		bag.Gold++
		items := mutableBagItems(sess)
		items[1001]++
	}
}
```

- [ ] **Step 2: 增加接近真实事务体的 benchmark**

```go
func BenchmarkCowWritePathInRunBody(b *testing.B) {
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
```

- [ ] **Step 3: 增加纯 `DeepCopy` 写路径对照**

```go
func BenchmarkDeepCopyWritePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneTestRoot(newTestRoot())
		root.Bag.Gold++
		root.Bag.Items[1001]++
	}
}
```

- [ ] **Step 4: 运行纯写路径 benchmark**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowWritePathOnSession|CowWritePathInRunBody|DeepCopyWritePath)$' -benchmem -count=1`
Expected: PASS，并输出三条写路径 benchmark

- [ ] **Step 5: Commit**

```bash
git add benchmark_test.go bench_support_test.go
git commit -m "test: add write path diagnosis benchmarks"
```

### Task 4: 跑完整诊断 benchmark 并写结论文档

**Files:**
- Create: `docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md`
- Modify: `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
- Test: `benchmark_test.go`

- [ ] **Step 1: 运行全部测试确认 benchmark 重构未破坏行为**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 2: 运行完整 benchmark 套件并保存输出**

Run: `go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndRunWith.*)$' -benchmem -count=1 > /tmp/cow-benchmark-diagnosis.txt`
Expected: `/tmp/cow-benchmark-diagnosis.txt` 生成成功

- [ ] **Step 3: 写诊断结果文档**

```markdown
# COW Benchmark 诊断结果

## 1. 执行环境

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- 命令：`go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndRunWith.*)$' -benchmem -count=1`

## 2. 结果归类

- 框架层：`BenchmarkFrameworkRunOnly`、`BenchmarkFrameworkEmptyClosure`
- 纯写路径层：`BenchmarkCowWritePathOnSession`、`BenchmarkCowWritePathInRunBody`、`BenchmarkDeepCopyWritePath`
- 端到端层：`BenchmarkEndToEndRunWithCow`、`BenchmarkEndToEndRunWithDeepCopy`

## 3. 诊断结论

- 若框架层成本已明显高于空闭包，对下一轮优先优化 `Run/context/session` 初始化。
- 若 `CowWritePathOnSession` 明显高于 `DeepCopyWritePath`，对下一轮优先优化组件/容器写路径。
- 若底层写路径相对可接受，但 `EndToEndRunWithCow` 仍明显偏重，则问题主要在外围事务框架成本。
```

- [ ] **Step 4: 如用户确认保留结果，追加 benchmark 日志**

```markdown
## 2026-05-22 benchmark 诊断轮次

- 命令：`go test ./... -run '^$' -bench 'Benchmark(Framework.*|CowWritePath.*|DeepCopyWritePath|EndToEndRunWith.*)$' -benchmem -count=1`

| 分组 | 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---|---:|---:|---:|---|
| 框架层 | `BenchmarkFrameworkRunOnly` | <fill after run> | <fill after run> | <fill after run> | `Run` + 空事务体 |
| 框架层 | `BenchmarkFrameworkEmptyClosure` | <fill after run> | <fill after run> | <fill after run> | 最小空路径对照 |
| 写路径层 | `BenchmarkCowWritePathOnSession` | <fill after run> | <fill after run> | <fill after run> | 底层会话直测 |
| 写路径层 | `BenchmarkCowWritePathInRunBody` | <fill after run> | <fill after run> | <fill after run> | 接近真实事务体 |
| 写路径层 | `BenchmarkDeepCopyWritePath` | <fill after run> | <fill after run> | <fill after run> | 纯 `DeepCopy` 写路径 |
| 端到端 | `BenchmarkEndToEndRunWithCow` | <fill after run> | <fill after run> | <fill after run> | 整体 `COW` 事务 |
| 端到端 | `BenchmarkEndToEndRunWithDeepCopy` | <fill after run> | <fill after run> | <fill after run> | 整体 `DeepCopy` 路径 |
```

- [ ] **Step 5: Commit**

```bash
git add benchmark_test.go bench_support_test.go docs/superpowers/specs/2026-05-22-cow-benchmark-diagnosis-result.md docs/superpowers/benchmarks/cow-mvp-benchmark.md
git commit -m "test: add benchmark diagnosis suite"
```

## 自检

### Spec Coverage

- benchmark 分组重构：Task 1 到 Task 3
- 框架层与写路径层拆分：Task 2 和 Task 3
- 保留端到端 benchmark：Task 1
- 跑完整数据并回写诊断结论：Task 4

### Placeholder Scan

- 所有实现步骤都有明确代码与命令。
- Task 4 中 benchmark 表格的 `<fill after run>` 仅用于执行后回填结果，不影响实现步骤本身。

### Type Consistency

- benchmark 命名与 spec 中建议命名保持一致。
- `newBenchSession`、`BenchmarkFrameworkRunOnly`、`BenchmarkCowWritePathOnSession`、`BenchmarkCowWritePathInRunBody`、`BenchmarkDeepCopyWritePath`、`BenchmarkEndToEndRunWithCow`、`BenchmarkEndToEndRunWithDeepCopy` 在全计划中名称一致。
