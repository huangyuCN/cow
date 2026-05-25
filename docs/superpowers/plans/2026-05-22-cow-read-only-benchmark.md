# COW 只读事务 Benchmark Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `COW` 运行时补一组“16 / 64 / 256 根规模、完全不写事务”的只读 benchmark，并与 eager clone 对照，验证 lazy session 的收益是否会随根规模扩大而进一步拉大。

**Architecture:** 完全复用现有 `benchSparseRoot` 大根 benchmark 数据模型，不再新起 root。`COW` 组只执行 `Begin() -> Commit()`；对照组只表达“事务开始即整根 clone，但最终不写”的 eager clone 固定成本。结果通过两次采样和 Markdown 表格归档，重点判断差距是否随规模扩大而拉大。

**Tech Stack:** Go 1.25 兼容写法、标准库 `testing`

---

## 文件结构

### 计划修改文件

- `benchmark_test.go`
  - 新增 `BenchmarkCowReadOnly16/64/256` 与 `BenchmarkDeepCopyReadOnly16/64/256`。
- `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
  - 若用户确认保留结果，追加本轮只读事务 benchmark 对比表与结论。

### 本轮复用文件

- `bench_sparse_types_test.go`
  - 继续复用上一轮大根 benchmark 数据模型与构造辅助函数，不新增第二套 root。
- `begin.go`
  - 复用当前只读 `Begin()` 行为。
- `session.go`
  - 复用当前 `Commit()` 的只读快路径，不在本轮改实现。

### 本轮不改动文件

- `example_types_test.go`
  - 保持现有业务语义测试不变。
- `savepoint.go`
  - 本轮 benchmark 不涉及 `Savepoint`。
- `overlay_test.go`
  - 本轮不新增组件共享语义测试。

## 任务拆分

### Task 1: 先写只读 benchmark 壳子并确认命名与输出可编译

**Files:**
- Modify: `benchmark_test.go`

- [ ] **Step 1: 在 `benchmark_test.go` 追加 6 个只读 benchmark**

把 [benchmark_test.go](/Users/huangyu/work/golang/src/cow/benchmark_test.go) 追加为：

```go
func BenchmarkCowReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(16))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(64))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(256))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeepCopyReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloneBenchSparseRoot(newBenchSparseRoot(16))
	}
}

func BenchmarkDeepCopyReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloneBenchSparseRoot(newBenchSparseRoot(64))
	}
}

func BenchmarkDeepCopyReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloneBenchSparseRoot(newBenchSparseRoot(256))
	}
}
```

- [ ] **Step 2: 运行只读 benchmark，确认可以编译并输出 6 个名字**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowReadOnly|DeepCopyReadOnly)' -benchmem -count=1`

Expected: PASS，并输出以下 benchmark：

```text
BenchmarkCowReadOnly16
BenchmarkCowReadOnly64
BenchmarkCowReadOnly256
BenchmarkDeepCopyReadOnly16
BenchmarkDeepCopyReadOnly64
BenchmarkDeepCopyReadOnly256
```

- [ ] **Step 3: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入结果采样任务。

### Task 2: 跑两轮完整 benchmark 并整理只读趋势判读

**Files:**
- Modify: `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
- Test: `benchmark_test.go`

- [ ] **Step 1: 运行第一轮完整只读 benchmark 并保存原始结果**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowReadOnly|DeepCopyReadOnly)' -benchmem -count=1 > /tmp/cow-read-only-bench.txt`

Expected: `/tmp/cow-read-only-bench.txt` 包含 6 个 benchmark：

```text
BenchmarkCowReadOnly16
BenchmarkCowReadOnly64
BenchmarkCowReadOnly256
BenchmarkDeepCopyReadOnly16
BenchmarkDeepCopyReadOnly64
BenchmarkDeepCopyReadOnly256
```

- [ ] **Step 2: 运行第二轮同命令采样，确认趋势不是单次波动**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowReadOnly|DeepCopyReadOnly)' -benchmem -count=1 > /tmp/cow-read-only-bench-2.txt`

Expected: 第二轮结果与第一轮趋势一致；若有轻微数值波动，也不应改变“差距是否随规模拉大”的判断。

- [ ] **Step 3: 直接提取两轮 benchmark 原始结果**

Run: `cat /tmp/cow-read-only-bench.txt`

Expected: 能读取第一轮 `ns/op`、`B/op`、`allocs/op`。

Run: `cat /tmp/cow-read-only-bench-2.txt`

Expected: 能读取第二轮 `ns/op`、`B/op`、`allocs/op`。

- [ ] **Step 4: 按 spec 的三档结论标准整理判读**

在回复中至少给出以下结构化结论：

```text
方向正确
方向可能正确，但只读快路径仍有残余固定成本
方向存疑
```

判读规则必须直接对应 spec：

- 若 `COW` 明显优于 eager clone，且从 `16 -> 64 -> 256` 差距继续拉大，归为“方向正确”；
- 若有优势，但扩大不明显，归为“方向可能正确，但只读快路径仍有残余固定成本”；
- 若优势不明显或差距不随规模扩大，归为“方向存疑”。

- [ ] **Step 5: 若用户确认保留结果，再追加 benchmark 日志**

把 [cow-mvp-benchmark.md](/Users/huangyu/work/golang/src/cow/docs/superpowers/benchmarks/cow-mvp-benchmark.md) 追加一节，至少包含：

```md
## 2026-05-22 只读事务 benchmark

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值
- commit：执行时 `git rev-parse HEAD` 的实际输出
- 命令：`go test ./... -run '^$' -bench 'Benchmark(CowReadOnly|DeepCopyReadOnly)' -benchmem -count=1`

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkCowReadOnly16-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `COW` / 16 组件 |
| `BenchmarkDeepCopyReadOnly16-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | eager clone / 16 组件 |
| `BenchmarkCowReadOnly64-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `COW` / 64 组件 |
| `BenchmarkDeepCopyReadOnly64-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | eager clone / 64 组件 |
| `BenchmarkCowReadOnly256-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `COW` / 256 组件 |
| `BenchmarkDeepCopyReadOnly256-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | eager clone / 256 组件 |
```

并在该节后补一段文字结论，明确这轮属于三档中的哪一档。

注意：

- 只有在用户确认保留 benchmark 时，才执行这一步；
- 不要把原始 `/tmp/*.txt` 文件加入仓库。

- [ ] **Step 6: 准备提交摘要，但不执行提交**

把以下摘要整理给用户确认：

- 新增 6 个完全只读事务 benchmark；
- 复用上一轮大根 benchmark 数据模型；
- 给出 lazy session 相对 eager clone 的趋势结论；
- 若用户确认保留，则追加 benchmark 归档。

## 自检清单

- spec 中“复用 `benchSparseRoot`、规模仍为 `16 / 64 / 256`、`COW` 组只做 `Begin() -> Commit()`、对照组只表达 eager clone”都有明确任务落点。
- 计划没有引入读路径、混合流量、`Savepoint` 或新 root 模型，范围与 spec 一致。
- benchmark 归档步骤明确受“用户确认保留结果”控制，不会越过仓库约定。
