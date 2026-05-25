# COW 大根稀疏写 Benchmark Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `COW` 运行时补一组“16 / 64 / 256 组件规模、每次只改 1 个组件”的大根稀疏写 benchmark，用来判断随着根规模扩大，`COW` 相对整根 `DeepCopy` 的差距是否收敛。

**Architecture:** 新增一套 benchmark 专用的大根测试模型，不复用当前 `testRoot/Bag/Quest` 小样例。`COW` 与 `DeepCopy` 两组 benchmark 执行完全一致的业务动作，只让复制策略不同；结果通过 `benchstat` 和 Markdown 表格归档，重点判断趋势，而不是强求第一轮立刻反超。

**Tech Stack:** Go 1.25 兼容写法、标准库 `testing` / `maps`

---

## 文件结构

### 计划新增文件

- `bench_sparse_types_test.go`
  - 定义 benchmark 专用的大根数据模型、构造函数、深拷贝函数，以及 benchmark 专用的组件写入口。

### 计划修改文件

- `benchmark_test.go`
  - 新增 `BenchmarkCowSparseWrite16/64/256` 与 `BenchmarkDeepCopySparseWrite16/64/256`。
- `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
  - 若用户确认保留结果，追加本轮 benchmark 对比表与结论。

### 本轮不改动文件

- `begin.go`
  - 本轮 benchmark 不改事务运行时实现。
- `session.go`
  - 只复用当前 `Begin()` / `Commit()` / lazy session 语义。
- `example_types_test.go`
  - 保留现有小样例语义测试，不让 benchmark 专用模型污染它。
- `overlay_test.go`
  - 本轮不新增业务语义测试，重点是 benchmark。

## 任务拆分

### Task 1: 先写 benchmark 专用数据模型与失败校验

**Files:**
- Create: `bench_sparse_types_test.go`
- Modify: `benchmark_test.go`

- [ ] **Step 1: 新增 benchmark 专用类型与构造辅助**

创建 [bench_sparse_types_test.go](/Users/huangyu/work/golang/src/cow/bench_sparse_types_test.go)，先写完整的 benchmark 专用类型与辅助函数骨架：

```go
package cow

import "maps"

const benchSparseMapSize = 128

type benchSparseRoot struct {
	Comps []*benchSparseComp
}

type benchSparseComp struct {
	Gold  int
	Items map[int]int
}

func newBenchSparseRoot(compCount int) *benchSparseRoot {
	root := &benchSparseRoot{
		Comps: make([]*benchSparseComp, 0, compCount),
	}
	for i := 0; i < compCount; i++ {
		items := make(map[int]int, benchSparseMapSize)
		for key := 0; key < benchSparseMapSize; key++ {
			items[key] = key + i
		}
		root.Comps = append(root.Comps, &benchSparseComp{
			Gold:  i,
			Items: items,
		})
	}
	return root
}

func cloneBenchSparseRoot(src *benchSparseRoot) *benchSparseRoot {
	next := &benchSparseRoot{
		Comps: make([]*benchSparseComp, 0, len(src.Comps)),
	}
	for _, comp := range src.Comps {
		next.Comps = append(next.Comps, &benchSparseComp{
			Gold:  comp.Gold,
			Items: maps.Clone(comp.Items),
		})
	}
	return next
}

func mutableSparseComp(sess *TxSession[benchSparseRoot], idx int) *benchSparseComp {
	panic("not implemented")
}

func mutableSparseItems(sess *TxSession[benchSparseRoot], idx int) map[int]int {
	panic("not implemented")
}
```

- [ ] **Step 2: 先在 `benchmark_test.go` 新增 6 个 benchmark 壳子**

把 [benchmark_test.go](/Users/huangyu/work/golang/src/cow/benchmark_test.go) 追加为：

```go
func BenchmarkCowSparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(16))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		comp := mutableSparseComp(sess, 0)
		comp.Gold++
		items := mutableSparseItems(sess, 0)
		items[0]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowSparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(64))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		comp := mutableSparseComp(sess, 0)
		comp.Gold++
		items := mutableSparseItems(sess, 0)
		items[0]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowSparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(256))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		comp := mutableSparseComp(sess, 0)
		comp.Gold++
		items := mutableSparseItems(sess, 0)
		items[0]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeepCopySparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneBenchSparseRoot(newBenchSparseRoot(16))
		root.Comps[0].Gold++
		root.Comps[0].Items[0]++
	}
}

func BenchmarkDeepCopySparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneBenchSparseRoot(newBenchSparseRoot(64))
		root.Comps[0].Gold++
		root.Comps[0].Items[0]++
	}
}

func BenchmarkDeepCopySparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneBenchSparseRoot(newBenchSparseRoot(256))
		root.Comps[0].Gold++
		root.Comps[0].Items[0]++
	}
}
```

- [ ] **Step 3: 运行新 benchmark，确认当前实现失败**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowSparseWrite|DeepCopySparseWrite)' -benchmem -count=1`

Expected: FAIL，因为 `mutableSparseComp()` 与 `mutableSparseItems()` 仍是 `panic("not implemented")`。

- [ ] **Step 4: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入实现任务。

### Task 2: 实现 benchmark 专用写路径与基础可运行版本

**Files:**
- Modify: `bench_sparse_types_test.go`
- Test: `benchmark_test.go`

- [ ] **Step 1: 实现 benchmark 专用组件首次写入口**

把 [bench_sparse_types_test.go](/Users/huangyu/work/golang/src/cow/bench_sparse_types_test.go) 中两个 `panic` 函数替换为：

```go
func mutableSparseComp(sess *TxSession[benchSparseRoot], idx int) *benchSparseComp {
	root := sess.ensureWritable()
	name := "comp"
	sess.markDirty(name)

	if root.Comps[idx] == sess.base.Comps[idx] {
		baseComp := sess.base.Comps[idx]
		root.Comps[idx] = &benchSparseComp{
			Gold:  baseComp.Gold,
			Items: baseComp.Items,
		}
	}

	return root.Comps[idx]
}

func mutableSparseItems(sess *TxSession[benchSparseRoot], idx int) map[int]int {
	comp := mutableSparseComp(sess, idx)
	name := "comp.items"
	if _, ok := sess.cloned[name]; !ok {
		comp.Items = maps.Clone(sess.base.Comps[idx].Items)
		sess.markCloned(name)
	}
	return comp.Items
}
```

- [ ] **Step 2: 为基础正确性补一个轻量测试，防止 benchmark 专用路径写坏**

在 [bench_sparse_types_test.go](/Users/huangyu/work/golang/src/cow/bench_sparse_types_test.go) 追加：

```go
func TestBenchSparseWriteCommitKeepsUntouchedComponentsShared(t *testing.T) {
	store := newMemoryStore(newBenchSparseRoot(16))
	before := store.Load()

	sess, err := Begin(store, cloneBenchSparseRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	comp := mutableSparseComp(sess, 0)
	comp.Gold++
	items := mutableSparseItems(sess, 0)
	items[0]++

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	after := store.Load()
	if after.Comps[0] == before.Comps[0] {
		t.Fatal("expected written component to be replaced")
	}
	if after.Comps[1] != before.Comps[1] {
		t.Fatal("expected untouched component to remain shared")
	}
	if before.Comps[0].Items[0] == after.Comps[0].Items[0] {
		t.Fatal("expected written component items map to be detached")
	}
}
```

- [ ] **Step 3: 运行新测试与新 benchmark，确认通过**

Run: `go test ./... -run 'TestBenchSparseWriteCommitKeepsUntouchedComponentsShared$' -count=1`

Expected: PASS

Run: `go test ./... -run '^$' -bench 'Benchmark(CowSparseWrite|DeepCopySparseWrite)' -benchmem -count=1`

Expected: PASS，并输出以下基准名：

```text
BenchmarkCowSparseWrite16
BenchmarkCowSparseWrite64
BenchmarkCowSparseWrite256
BenchmarkDeepCopySparseWrite16
BenchmarkDeepCopySparseWrite64
BenchmarkDeepCopySparseWrite256
```

- [ ] **Step 4: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入结果整理任务。

### Task 3: 跑完整结果、生成对比并整理判读

**Files:**
- Modify: `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
- Test: `benchmark_test.go`

- [ ] **Step 1: 运行完整 benchmark 并保存原始结果**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowSparseWrite|DeepCopySparseWrite)' -benchmem -count=1 > /tmp/cow-large-root-sparse-write-bench.txt`

Expected: `/tmp/cow-large-root-sparse-write-bench.txt` 包含 6 个 benchmark：

```text
BenchmarkCowSparseWrite16
BenchmarkCowSparseWrite64
BenchmarkCowSparseWrite256
BenchmarkDeepCopySparseWrite16
BenchmarkDeepCopySparseWrite64
BenchmarkDeepCopySparseWrite256
```

- [ ] **Step 2: 若需要重复采样，再补一轮相同命令**

Run: `go test ./... -run '^$' -bench 'Benchmark(CowSparseWrite|DeepCopySparseWrite)' -benchmem -count=1 > /tmp/cow-large-root-sparse-write-bench-2.txt`

Expected: 第二份结果可用于人工确认波动；若本轮只保留单次样本，也要在反馈中明确说明“趋势判断基于单次采样”。

- [ ] **Step 3: 用 `benchstat` 对比同类 benchmark 的规模趋势**

Run: `benchstat -format csv /tmp/cow-large-root-sparse-write-bench.txt`

Expected: 能直接读取每个 benchmark 的 `ns/op`、`B/op`、`allocs/op`。

如果 `benchstat` 单文件输出不够直观，则补充：

Run: `sed -n '1,120p' /tmp/cow-large-root-sparse-write-bench.txt`

Expected: 直接提取 6 个 benchmark 的原始指标。

- [ ] **Step 4: 按 spec 的三档结论标准整理结论**

在回复或归档中至少给出以下结构化判断：

```text
方向正确
方向可能正确，但实现还有明显固定成本
方向存疑
```

判读规则必须直接对应 spec：

- 若 `COW` 与 `DeepCopy` 的差距在 `16 -> 64 -> 256` 之间明显收敛，归为“方向正确”；
- 若有收敛，但幅度有限，归为“方向可能正确，但实现还有明显固定成本”；
- 若几乎看不到收敛，甚至相对更差，归为“方向存疑”。

- [ ] **Step 5: 若用户确认保留结果，再追加 benchmark 日志**

把 [cow-mvp-benchmark.md](/Users/huangyu/work/golang/src/cow/docs/superpowers/benchmarks/cow-mvp-benchmark.md) 追加一节，至少包含：

```md
## 2026-05-22 大根稀疏写 benchmark

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值
- commit：执行时 `git rev-parse HEAD` 的实际输出
- 命令：`go test ./... -run '^$' -bench 'Benchmark(CowSparseWrite|DeepCopySparseWrite)' -benchmem -count=1`

| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
| `BenchmarkCowSparseWrite16-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `COW` / 16 组件 |
| `BenchmarkDeepCopySparseWrite16-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `DeepCopy` / 16 组件 |
| `BenchmarkCowSparseWrite64-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `COW` / 64 组件 |
| `BenchmarkDeepCopySparseWrite64-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `DeepCopy` / 64 组件 |
| `BenchmarkCowSparseWrite256-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `COW` / 256 组件 |
| `BenchmarkDeepCopySparseWrite256-8` | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | 直接抄录 Step 1 的实际输出 | `DeepCopy` / 256 组件 |
```

并在该节后补一段文字结论，明确这轮属于三档中的哪一档。

注意：

- 只有在用户确认保留 benchmark 时，才执行这一步；
- 不要把原始 `/tmp/*.txt` 文件加入仓库。

- [ ] **Step 6: 准备提交摘要，但不执行提交**

把以下摘要整理给用户确认：

- 新增 benchmark 专用大根模型；
- 新增 6 个大根稀疏写 benchmark；
- 新增 benchmark 专用轻量正确性测试；
- 若用户确认保留，则追加 benchmark 归档；
- 给出对 `COW` 方向的结论分级。

## 自检清单

- spec 中的数据模型、规模梯度、固定 `128` 项 `map`、只改 `Comps[0]`、独立 benchmark 命名，都有对应任务。
- 计划没有要求顺手修改运行时实现或扩展只读/热点写矩阵，范围与 spec 一致。
- benchmark 归档步骤明确受“用户确认保留结果”控制，不会越过仓库约定。
