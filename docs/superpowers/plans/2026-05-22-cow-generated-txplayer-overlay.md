# COW 代码生成 TxPlayer Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不推翻现有 `COW` 原型的前提下，先做一个 `Player -> TxPlayer` 的字段级 overlay 事务视图原型，用测试和 benchmark 验证“读穿透 base、写时字段级 copy、提交时顶层 merge”这条新模型是否成立。

**Architecture:** 第一版不直接做通用代码生成器，而是在测试文件中放一个“手写的生成结果等价物”。`TxPlayer` 只处理 `Player` 的直接字段，支持标量、`map`、`slice`，全部通过方法访问；不提供 `Rollback()`，`Commit()` 只重建一个新的顶层 `Player`。benchmark 重点验证三类负载：只读、稀疏写、热点重复写。

**Tech Stack:** Go 1.25 兼容写法、标准库 `testing` / `maps`

---

## 文件结构

### 计划新增文件

- `txplayer_proto_types_test.go`
  - 定义 `Player` 原型类型、测试数据构造函数、eager clone 对照函数。
- `txplayer_proto_generated_test.go`
  - 放手写的 `TxPlayer`“生成结果等价物”，包括 `BeginPlayer`、字段方法、`Commit()`。
- `txplayer_proto_test.go`
  - 语义测试：读穿透、首次写 copy、不污染 `base`、提交顶层 merge。
- `txplayer_proto_bench_test.go`
  - 只读 / 稀疏写 / 热点重复写 benchmark。

### 计划修改文件

- `docs/superpowers/benchmarks/cow-mvp-benchmark.md`
  - 若用户确认保留 benchmark 结果，再追加本轮 `TxPlayer` overlay 原型的对比表和结论。

### 本轮不改动文件

- `begin.go`
  - 本轮不把 `TxPlayer` 原型硬塞进现有 `TxSession`。
- `session.go`
  - 本轮不改现有 path-copy runtime。
- `benchmark_test.go`
  - 保留现有 `TxSession` 路线 benchmark，不在同一文件混入新模型。
- `bench_sparse_types_test.go`
  - 保留上一轮大根 benchmark 数据模型，避免两类实验互相污染。

## 任务拆分

### Task 1: 先写 `Player` 原型类型和失败语义测试

**Files:**
- Create: `txplayer_proto_types_test.go`
- Create: `txplayer_proto_test.go`

- [ ] **Step 1: 新增 `Player` 原型类型与测试构造函数**

创建 [txplayer_proto_types_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_types_test.go)，先写完整内容：

```go
package cow

import "maps"

type Player struct {
	Name   string
	Level  int
	Items  map[int]int
	Skills []int
}

func newPlayer() *Player {
	return &Player{
		Name:   "hero",
		Level:  10,
		Items:  map[int]int{1001: 1, 1002: 2},
		Skills: []int{11, 22, 33},
	}
}

func newBenchPlayer(size int) *Player {
	items := make(map[int]int, size)
	skills := make([]int, size)
	for i := 0; i < size; i++ {
		items[i] = i
		skills[i] = i
	}
	return &Player{
		Name:   "hero",
		Level:  10,
		Items:  items,
		Skills: skills,
	}
}

func clonePlayer(src *Player) *Player {
	return &Player{
		Name:   src.Name,
		Level:  src.Level,
		Items:  maps.Clone(src.Items),
		Skills: append([]int(nil), src.Skills...),
	}
}
```

- [ ] **Step 2: 新增第一批失败语义测试**

创建 [txplayer_proto_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_test.go)，先写完整内容：

```go
package cow

import "testing"

func TestTxPlayerReadsFallbackToBase(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)

	if got := tx.Name(); got != "hero" {
		t.Fatalf("Name() = %q, want hero", got)
	}
	if got := tx.Level(); got != 10 {
		t.Fatalf("Level() = %d, want 10", got)
	}
	if got, ok := tx.Item(1001); !ok || got != 1 {
		t.Fatalf("Item(1001) = (%d, %v), want (1, true)", got, ok)
	}
	if got, ok := tx.Skill(1); !ok || got != 22 {
		t.Fatalf("Skill(1) = (%d, %v), want (22, true)", got, ok)
	}
}

func TestTxPlayerSetScalarDoesNotMutateBase(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)
	tx.SetName("mage")
	tx.SetLevel(20)

	if base.Name != "hero" {
		t.Fatalf("base.Name = %q, want hero", base.Name)
	}
	if base.Level != 10 {
		t.Fatalf("base.Level = %d, want 10", base.Level)
	}
}

func TestTxPlayerMapAndSliceWritesDoNotMutateBase(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)
	tx.SetItem(1001, 9)
	tx.DeleteItem(1002)
	tx.SetSkill(1, 77)
	tx.AppendSkill(99)

	if got := base.Items[1001]; got != 1 {
		t.Fatalf("base.Items[1001] = %d, want 1", got)
	}
	if got := base.Items[1002]; got != 2 {
		t.Fatalf("base.Items[1002] = %d, want 2", got)
	}
	if got := base.Skills[1]; got != 22 {
		t.Fatalf("base.Skills[1] = %d, want 22", got)
	}
	if got := len(base.Skills); got != 3 {
		t.Fatalf("len(base.Skills) = %d, want 3", got)
	}
}

func TestTxPlayerCommitRebuildsTopLevelAndReusesUntouchedFields(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)
	tx.SetName("mage")
	tx.SetItem(1001, 9)

	out := tx.Commit()

	if out == base {
		t.Fatal("Commit() should return a new top-level Player")
	}
	if out.Name != "mage" {
		t.Fatalf("out.Name = %q, want mage", out.Name)
	}
	if out.Level != base.Level {
		t.Fatalf("out.Level = %d, want %d", out.Level, base.Level)
	}
	if out.Items[1001] != 9 {
		t.Fatalf("out.Items[1001] = %d, want 9", out.Items[1001])
	}
	if out.Skills == nil || len(out.Skills) != len(base.Skills) {
		t.Fatalf("out.Skills should reuse base slice")
	}
	if &out.Skills[0] != &base.Skills[0] {
		t.Fatal("untouched slice field should reuse base backing data")
	}
}
```

- [ ] **Step 3: 运行测试，确认当前实现失败**

Run: `go test ./... -run 'TestTxPlayer' -count=1`

Expected: FAIL，当前还不存在 `BeginPlayer`、`TxPlayer` 及其字段方法。

- [ ] **Step 4: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入实现任务。

### Task 2: 实现手写的 `TxPlayer` 生成结果等价物

**Files:**
- Create: `txplayer_proto_generated_test.go`
- Test: `txplayer_proto_test.go`

- [ ] **Step 1: 新增 `TxPlayer` 结构与 `BeginPlayer()`**

创建 [txplayer_proto_generated_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_generated_test.go)，先写完整内容：

```go
package cow

import "maps"

type TxPlayer struct {
	base *Player

	name    string
	hasName bool

	level    int
	hasLevel bool

	items    map[int]int
	hasItems bool

	skills    []int
	hasSkills bool
}

func BeginPlayer(base *Player) *TxPlayer {
	return &TxPlayer{base: base}
}
```

- [ ] **Step 2: 实现标量字段方法**

在 [txplayer_proto_generated_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_generated_test.go) 追加：

```go
func (tx *TxPlayer) Name() string {
	if tx.hasName {
		return tx.name
	}
	return tx.base.Name
}

func (tx *TxPlayer) SetName(v string) {
	tx.name = v
	tx.hasName = true
}

func (tx *TxPlayer) Level() int {
	if tx.hasLevel {
		return tx.level
	}
	return tx.base.Level
}

func (tx *TxPlayer) SetLevel(v int) {
	tx.level = v
	tx.hasLevel = true
}
```

- [ ] **Step 3: 实现 `map` 字段方法**

继续在 [txplayer_proto_generated_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_generated_test.go) 追加：

```go
func (tx *TxPlayer) ensureItems() map[int]int {
	if !tx.hasItems {
		tx.items = maps.Clone(tx.base.Items)
		tx.hasItems = true
	}
	return tx.items
}

func (tx *TxPlayer) Item(id int) (int, bool) {
	if tx.hasItems {
		v, ok := tx.items[id]
		return v, ok
	}
	v, ok := tx.base.Items[id]
	return v, ok
}

func (tx *TxPlayer) SetItem(id int, v int) {
	items := tx.ensureItems()
	items[id] = v
}

func (tx *TxPlayer) DeleteItem(id int) {
	items := tx.ensureItems()
	delete(items, id)
}
```

- [ ] **Step 4: 实现 `slice` 字段方法**

继续在 [txplayer_proto_generated_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_generated_test.go) 追加：

```go
func (tx *TxPlayer) ensureSkills() []int {
	if !tx.hasSkills {
		tx.skills = append([]int(nil), tx.base.Skills...)
		tx.hasSkills = true
	}
	return tx.skills
}

func (tx *TxPlayer) Skill(i int) (int, bool) {
	var skills []int
	if tx.hasSkills {
		skills = tx.skills
	} else {
		skills = tx.base.Skills
	}
	if i < 0 || i >= len(skills) {
		return 0, false
	}
	return skills[i], true
}

func (tx *TxPlayer) SkillCount() int {
	if tx.hasSkills {
		return len(tx.skills)
	}
	return len(tx.base.Skills)
}

func (tx *TxPlayer) SetSkill(i int, v int) {
	skills := tx.ensureSkills()
	skills[i] = v
}

func (tx *TxPlayer) AppendSkill(v int) {
	skills := tx.ensureSkills()
	tx.skills = append(skills, v)
}
```

- [ ] **Step 5: 实现 `Commit()` 顶层 merge**

继续在 [txplayer_proto_generated_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_generated_test.go) 追加：

```go
func (tx *TxPlayer) Commit() *Player {
	base := tx.base
	out := &Player{}

	if tx.hasName {
		out.Name = tx.name
	} else {
		out.Name = base.Name
	}

	if tx.hasLevel {
		out.Level = tx.level
	} else {
		out.Level = base.Level
	}

	if tx.hasItems {
		out.Items = tx.items
	} else {
		out.Items = base.Items
	}

	if tx.hasSkills {
		out.Skills = tx.skills
	} else {
		out.Skills = base.Skills
	}

	return out
}
```

- [ ] **Step 6: 运行语义测试，确认通过**

Run: `go test ./... -run 'TestTxPlayer' -count=1`

Expected: PASS

- [ ] **Step 7: 再跑完整测试，确认没有污染现有原型**

Run: `go test ./... -count=1`

Expected: PASS

- [ ] **Step 8: 记录本任务完成后的预期 diff，不执行提交**

本仓库要求未经用户明确同意不得 `git commit`。此处只保留工作区改动，进入 benchmark 任务。

### Task 3: 为 `TxPlayer` 原型补三类 benchmark

**Files:**
- Create: `txplayer_proto_bench_test.go`
- Test: `txplayer_proto_bench_test.go`

- [ ] **Step 1: 新增只读 benchmark**

创建 [txplayer_proto_bench_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_bench_test.go)，先写只读 benchmark：

```go
package cow

import "testing"

func BenchmarkTxPlayerReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		tx := BeginPlayer(base)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		_ = clonePlayer(base)
	}
}

func BenchmarkTxPlayerReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		tx := BeginPlayer(base)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		_ = clonePlayer(base)
	}
}

func BenchmarkTxPlayerReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		tx := BeginPlayer(base)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		_ = clonePlayer(base)
	}
}
```

- [ ] **Step 2: 追加稀疏写 benchmark**

继续在 [txplayer_proto_bench_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_bench_test.go) 追加：

```go
func BenchmarkTxPlayerSparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		tx := BeginPlayer(base)
		tx.SetLevel(20)
		tx.SetItem(0, 99)
		tx.SetSkill(0, 77)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneSparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(16))
		out.Level = 20
		out.Items[0] = 99
		out.Skills[0] = 77
	}
}

func BenchmarkTxPlayerSparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		tx := BeginPlayer(base)
		tx.SetLevel(20)
		tx.SetItem(0, 99)
		tx.SetSkill(0, 77)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneSparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(64))
		out.Level = 20
		out.Items[0] = 99
		out.Skills[0] = 77
	}
}

func BenchmarkTxPlayerSparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		tx := BeginPlayer(base)
		tx.SetLevel(20)
		tx.SetItem(0, 99)
		tx.SetSkill(0, 77)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneSparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(256))
		out.Level = 20
		out.Items[0] = 99
		out.Skills[0] = 77
	}
}
```

- [ ] **Step 3: 追加热点重复写 benchmark**

继续在 [txplayer_proto_bench_test.go](/Users/huangyu/work/golang/src/cow/txplayer_proto_bench_test.go) 追加：

```go
func BenchmarkTxPlayerHotWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		tx := BeginPlayer(base)
		for j := 0; j < 8; j++ {
			tx.SetItem(0, j)
			tx.SetSkill(0, j)
		}
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneHotWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(16))
		for j := 0; j < 8; j++ {
			out.Items[0] = j
			out.Skills[0] = j
		}
	}
}

func BenchmarkTxPlayerHotWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		tx := BeginPlayer(base)
		for j := 0; j < 8; j++ {
			tx.SetItem(0, j)
			tx.SetSkill(0, j)
		}
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneHotWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(64))
		for j := 0; j < 8; j++ {
			out.Items[0] = j
			out.Skills[0] = j
		}
	}
}

func BenchmarkTxPlayerHotWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		tx := BeginPlayer(base)
		for j := 0; j < 8; j++ {
			tx.SetItem(0, j)
			tx.SetSkill(0, j)
		}
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneHotWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(256))
		for j := 0; j < 8; j++ {
			out.Items[0] = j
			out.Skills[0] = j
		}
	}
}
```

- [ ] **Step 4: 运行 prototype benchmark，确认全部可跑**

Run: `go test ./... -run '^$' -bench 'Benchmark(TxPlayer|EagerClone)' -benchmem -count=1`

Expected: PASS，并输出：

```text
BenchmarkTxPlayerReadOnly16
BenchmarkEagerCloneReadOnly16
BenchmarkTxPlayerSparseWrite16
BenchmarkEagerCloneSparseWrite16
BenchmarkTxPlayerHotWrite16
BenchmarkEagerCloneHotWrite16
```

以及对应 `64 / 256` 三档。

- [ ] **Step 5: 再跑一轮完整采样保存到 `/tmp`**

Run: `go test ./... -run '^$' -bench 'Benchmark(TxPlayer|EagerClone)' -benchmem -count=1 > /tmp/txplayer-overlay-bench.txt`

Expected: `/tmp/txplayer-overlay-bench.txt` 包含 18 个 benchmark。

- [ ] **Step 6: 整理三类结论，但不先归档**

至少输出以下三类比较：

- 完全只读：`TxPlayer` vs eager clone
- 稀疏写：`TxPlayer` vs eager clone
- 热点重复写：`TxPlayer` vs eager clone

并按以下结论模板组织：

```text
只读路径是否更轻
稀疏写是否更轻
热点重复写是否更轻
相对当前 TxSession 路线是否值得继续演进
```

此步只整理结论，不改 benchmark 日志文件。

### Task 4: 若用户确认保留，再追加 benchmark 归档

**Files:**
- Modify: `docs/superpowers/benchmarks/cow-mvp-benchmark.md`

- [ ] **Step 1: 仅在用户确认保留后，追加日志**

把 [cow-mvp-benchmark.md](/Users/huangyu/work/golang/src/cow/docs/superpowers/benchmarks/cow-mvp-benchmark.md) 追加一节，至少包含：

```md
## 2026-05-22 TxPlayer overlay prototype benchmark

- 日期：2026-05-22
- `go version`：`go version go1.26.0 darwin/arm64`
- 机器 / OS：`Apple M3` / `Darwin 25.4.0`
- `GOOS` / `GOARCH`：`darwin` / `arm64`
- `GOMAXPROCS`：默认值
- commit：执行时 `git rev-parse HEAD` 的实际输出
- 命令：`go test ./... -run '^$' -bench 'Benchmark(TxPlayer|EagerClone)' -benchmem -count=1`
```

并至少给出三组表：

- ReadOnly 16 / 64 / 256
- SparseWrite 16 / 64 / 256
- HotWrite 16 / 64 / 256

每组表至少含：

```md
| 基准名 | ns/op | B/op | allocs/op | 说明 |
|---|---:|---:|---:|---|
```

- [ ] **Step 2: 不纳入 `/tmp` 原始结果文件**

不要把：

- `/tmp/txplayer-overlay-bench.txt`

加入仓库。

## 自检清单

- spec 中“只做 `Player -> TxPlayer`、不提供 `Rollback()`、直接字段、标量 + `map` + `slice`、顶层 merge”都有明确任务落点。
- 计划没有把第一版错误扩展成通用生成器、递归对象图或运行时反射框架。
- benchmark 归档步骤明确受“用户确认保留结果”控制，不会越过仓库约定。
