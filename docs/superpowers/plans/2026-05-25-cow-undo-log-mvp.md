# COW Undo Log MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在根包 `cow` 实现 Undo Log 写代理与 `TxContext`，通过测试验证回滚/提交语义，并用 Benchmark 证明相对 `deepcopy-gen` 全量深拷贝在中等 `Player` 稀疏写下具备明显性能优势。

**Architecture:** 单协程下对常驻 `Player` 的每次写注册逆操作闭包；失败倒序 `Rollback`，成功仅 `Reset` 清空 log。对照组每次请求 `DeepCopy()` 整图后在副本上直接改字段。

**Tech Stack:** Go 1.25、`sync.Pool`、`k8s.io/code-generator/cmd/deepcopy-gen`、`k8s.io/apimachinery/pkg/runtime`、`github.com/google/go-cmp/cmp`

**工作目录:** 在当前仓库根目录开发（`AGENTS.md` 禁止 git worktree）。

**设计说明:** `docs/superpowers/specs/2026-05-25-cow-undo-log-mvp-design.md`

---

## 文件一览

| 文件 | 操作 |
|---|---|
| `go.mod` / `go.sum` | 添加依赖 |
| `doc.go` | 新建：包注释 + deepcopy 包级 tag |
| `types.go` | 新建：`Item`/`Hero`/`Player` + `Clone` |
| `deepcopy_generate.go` | 新建：`//go:generate` |
| `zz_generated.deepcopy.go` | 生成并**提交 Git** |
| `tx.go` | 新建：`TxContext` + Pool |
| `player_proxy.go` | 新建：三个写代理 |
| `player_test.go` | 新建：正确性测试 |
| `bench_fixture_test.go` | 新建：B 档夹具 + 快照 |
| `benchmark_test.go` | 新建：三组 Benchmark |

---

### Task 1: 模块依赖与类型骨架

**Files:**
- Modify: `go.mod`
- Create: `doc.go`
- Create: `types.go`

- [ ] **Step 1: 添加测试与 runtime 依赖**

```bash
cd /Users/huangyu/work/golang/src/cow
go get github.com/google/go-cmp/cmp@latest
go get k8s.io/apimachinery/pkg/runtime@v0.32.3
```

- [ ] **Step 2: 创建 `doc.go`**

```go
// Package cow 提供单协程聚合根 Undo Log 写代理（MVP 验证）。
//
// +k8s:deepcopy-gen=package
// +groupName=cow.huanghaiyu.cn
package cow
```

- [ ] **Step 3: 创建 `types.go`**

```go
package cow

// Item 模拟背包条目（标签兼容 PB/BSON，MVP 不序列化）。
type Item struct {
	Id   int64  `protobuf:"varint,1,opt,name=id" json:"id,omitempty" bson:"_id"`
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty" bson:"name"`
}

// Hero 模拟英雄子结构。
type Hero struct {
	HeroId int32 `protobuf:"varint,1,opt,name=hero_id" json:"hero_id,omitempty" bson:"hero_id"`
	Level  int32 `protobuf:"varint,2,opt,name=level" json:"level,omitempty" bson:"level"`
}

// Clone 单层拷贝，供 GetHeroForWrite 延迟局部深拷贝。
func (h *Hero) Clone() *Hero {
	if h == nil {
		return nil
	}
	return &Hero{HeroId: h.HeroId, Level: h.Level}
}

// Player 模拟聚合根。
type Player struct {
	Uid    int64            `protobuf:"varint,1,opt,name=uid" json:"uid,omitempty" bson:"_id"`
	Assets map[string]int64 `protobuf:"bytes,2,rep,name=assets" json:"assets,omitempty" bson:"assets"`
	Items  []*Item          `protobuf:"bytes,3,rep,name=items" json:"items,omitempty" bson:"items"`
	Hero   *Hero            `protobuf:"bytes,4,opt,name=hero" json:"hero,omitempty" bson:"hero"`
}
```

- [ ] **Step 4: 验证编译（尚无生成文件，预期可能失败）**

```bash
go build ./...
```

若因缺少 `DeepCopy` 失败，进入 Task 2。

- [ ] **Step 5: Commit（需用户确认后执行）**

```bash
git add go.mod go.sum doc.go types.go
git commit -m "$(cat <<'EOF'
chore(cow): 添加 Undo Log MVP 类型骨架与模块依赖

EOF
)"
```

---

### Task 2: deepcopy-gen 生成并提交

**Files:**
- Create: `deepcopy_generate.go`
- Create: `zz_generated.deepcopy.go`（生成）

- [ ] **Step 1: 安装 deepcopy-gen**

```bash
go install k8s.io/code-generator/cmd/deepcopy-gen@v0.32.3
```

确认：`which deepcopy-gen` 在 `$GOPATH/bin` 或 `$HOME/go/bin`。

- [ ] **Step 2: 创建 `deepcopy_generate.go`**

```go
package cow

//go:generate deepcopy-gen --output-file zz_generated.deepcopy.go github.com/huangyuCN/cow
```

- [ ] **Step 3: 运行 generate**

```bash
cd /Users/huangyu/work/golang/src/cow
go generate ./...
ls -la zz_generated.deepcopy.go
```

预期：生成 `zz_generated.deepcopy.go`，内含 `(*Player).DeepCopy()`、`DeepCopyInto` 等。

- [ ] **Step 4: 验证生成 API**

```bash
go build ./...
```

- [ ] **Step 5: 手写 smoke 测试（可选，确保 DeepCopy 可用）**

在 `types_deepcopy_test.go` 临时写：

```go
func TestPlayerDeepCopy_Isolated(t *testing.T) {
	src := &Player{Uid: 1, Assets: map[string]int64{"gold": 1}, Hero: &Hero{Level: 1}}
	dst := src.DeepCopy()
	src.Assets["gold"] = 999
	if dst.Assets["gold"] == 999 {
		t.Fatal("deep copy shares map")
	}
}
```

```bash
go test -run TestPlayerDeepCopy_Isolated -v ./...
```

Expected: PASS

- [ ] **Step 6: Commit 生成文件**

```bash
git add deepcopy_generate.go zz_generated.deepcopy.go types_deepcopy_test.go
git commit -m "$(cat <<'EOF'
chore(cow): 接入 deepcopy-gen 并提交生成深拷贝代码

EOF
)"
```

---

### Task 3: 测试夹具与快照辅助

**Files:**
- Create: `bench_fixture_test.go`

- [ ] **Step 1: 实现 B 档夹具与快照**

```go
package cow

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// clonePlayerSnapshot 用 deepcopy-gen 做测试间状态对比基线。
func clonePlayerSnapshot(p *Player) *Player {
	return p.DeepCopy()
}

func newBenchPlayer() *Player {
	assets := make(map[string]int64, 100)
	assets["gold"] = 1000
	assets["diamond"] = 100
	for i := 0; i < 98; i++ {
		assets[fmt.Sprintf("token_%d", i)] = int64(i + 1)
	}
	items := make([]*Item, 0, 500)
	for i := 0; i < 500; i++ {
		items = append(items, &Item{Id: int64(i + 1), Name: fmt.Sprintf("item_%d", i)})
	}
	return &Player{
		Uid:    10001,
		Assets: assets,
		Items:  items,
		Hero:   &Hero{HeroId: 99, Level: 1},
	}
}

// applySparseWrites 模拟一次请求的三处稀疏写。
func applySparseWrites(p *Player, ctx *TxContext) {
	p.PutAsset(ctx, "gold", 500)
	p.AppendItem(ctx, &Item{Id: 9999, Name: "Shield"})
	h := p.GetHeroForWrite(ctx)
	h.Level = 2
}

func assertPlayerEqual(t *testing.T, got, want *Player) {
	t.Helper()
	if diff := cmp.Diff(want, got, cmp.Comparer(func(a, b *Item) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Id == b.Id && a.Name == b.Name
	})); diff != "" {
		t.Fatalf("player mismatch (-want +got):\n%s", diff)
	}
}
```

- [ ] **Step 2: 运行（此时 `TxContext`/代理未实现，仅确保文件语法）**

```bash
go test -c ./... 2>&1 | head -20
```

预期：可能因未定义 `TxContext`/`PutAsset` 等编译失败——正常，Task 4–6 补齐。

- [ ] **Step 3: Commit**

```bash
git add bench_fixture_test.go
git commit -m "$(cat <<'EOF'
test(cow): 添加 B 档 Player 夹具与快照比较辅助

EOF
)"
```

---

### Task 4: 正确性测试（RED）

**Files:**
- Create: `player_test.go`

- [ ] **Step 1: 编写失败测试**

```go
package cow

import (
	"errors"
	"testing"
)

func runScopedWithRollback(p *Player, fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	if err := fn(ctx); err != nil {
		return err
	}
	ctx.Reset()
	return nil
}

func runScopedCommit(p *Player, fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)
	if err := fn(ctx); err != nil {
		ctx.Rollback()
		return err
	}
	ctx.Reset()
	return nil
}

func TestRollback_RestoresInitialState(t *testing.T) {
	player := newBenchPlayer()
	want := clonePlayerSnapshot(player)

	err := runScopedWithRollback(player, func(ctx *TxContext) error {
		applySparseWrites(player, ctx)
		return errors.New("business error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, player, want)
}

func TestCommit_KeepsMutations(t *testing.T) {
	player := newBenchPlayer()
	before := clonePlayerSnapshot(player)

	err := runScopedCommit(player, func(ctx *TxContext) error {
		applySparseWrites(player, ctx)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if player.Assets["gold"] == before.Assets["gold"] {
		t.Fatal("gold should change after commit")
	}
	if len(player.Items) != len(before.Items)+1 {
		t.Fatalf("items len got %d want %d", len(player.Items), len(before.Items)+1)
	}
	if player.Hero.Level != 2 {
		t.Fatalf("hero level got %d want 2", player.Hero.Level)
	}
}
```

- [ ] **Step 2: 运行确认 RED**

```bash
go test -run 'TestRollback|TestCommit' -v ./...
```

Expected: FAIL（`TxContext` 或方法未定义）

- [ ] **Step 3: Commit**

```bash
git add player_test.go
git commit -m "$(cat <<'EOF'
test(cow): 添加回滚与提交正确性测试（RED）

EOF
)"
```

---

### Task 5: TxContext 实现

**Files:**
- Create: `tx.go`

- [ ] **Step 1: 实现 `tx.go`**

```go
package cow

import "sync"

// TxContext 单次请求作用域的 Undo 日志（单协程，无锁）。
type TxContext struct {
	undoLogs []func()
}

// AddUndo 注册一条逆操作。
func (ctx *TxContext) AddUndo(undo func()) {
	ctx.undoLogs = append(ctx.undoLogs, undo)
}

// Rollback 倒序执行所有逆操作。
func (ctx *TxContext) Rollback() {
	for i := len(ctx.undoLogs) - 1; i >= 0; i-- {
		ctx.undoLogs[i]()
	}
}

// Reset 清空日志并复用底层切片。
func (ctx *TxContext) Reset() {
	ctx.undoLogs = ctx.undoLogs[:0]
}

// txPool 复用 TxContext，降低高频路径分配。
var txPool = sync.Pool{
	New: func() any {
		return &TxContext{undoLogs: make([]func(), 0, 16)}
	},
}
```

- [ ] **Step 2: 运行测试（仍可能 RED，缺代理）**

```bash
go test -run 'TestRollback|TestCommit' -v ./...
```

Expected: FAIL（`PutAsset` 等未定义）

- [ ] **Step 3: Commit**

```bash
git add tx.go
git commit -m "$(cat <<'EOF'
feat(cow): 实现 TxContext Undo 栈与 sync.Pool 复用

EOF
)"
```

---

### Task 6: 写代理实现（GREEN）

**Files:**
- Create: `player_proxy.go`

- [ ] **Step 1: 实现三个代理**

```go
package cow

// PutAsset 写入 Assets 并注册逆操作。
func (p *Player) PutAsset(ctx *TxContext, key string, val int64) {
	if p.Assets == nil {
		p.Assets = make(map[string]int64)
	}
	old, existed := p.Assets[key]
	p.Assets[key] = val
	if existed {
		ctx.AddUndo(func() { p.Assets[key] = old })
	} else {
		ctx.AddUndo(func() { delete(p.Assets, key) })
	}
}

// AppendItem 追加 Items 并注册截断回滚。
func (p *Player) AppendItem(ctx *TxContext, item *Item) {
	oldLen := len(p.Items)
	p.Items = append(p.Items, item)
	ctx.AddUndo(func() { p.Items = p.Items[:oldLen] })
}

// GetHeroForWrite 延迟局部拷贝 Hero，返回可写副本。
func (p *Player) GetHeroForWrite(ctx *TxContext) *Hero {
	old := p.Hero
	p.Hero = old.Clone()
	ctx.AddUndo(func() { p.Hero = old })
	return p.Hero
}
```

- [ ] **Step 2: 运行正确性测试**

```bash
go test -run 'TestRollback|TestCommit' -v ./...
```

Expected: PASS

- [ ] **Step 3: 全量测试**

```bash
go test ./... -count=1
```

Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add player_proxy.go
git commit -m "$(cat <<'EOF'
feat(cow): 实现 Player 写代理 PutAsset/AppendItem/GetHeroForWrite

EOF
)"
```

---

### Task 7: Benchmark 对比

**Files:**
- Create: `benchmark_test.go`

- [ ] **Step 1: 实现三组 Benchmark**

```go
package cow

import "testing"

func sparseWriteDirect(p *Player) {
	p.Assets["gold"] = 500
	p.Items = append(p.Items, &Item{Id: 9999, Name: "Shield"})
	p.Hero.Level = 2
}

func sparseWriteUndo(p *Player, ctx *TxContext) {
	applySparseWrites(p, ctx)
}

func BenchmarkUndoLog_SparseWrite_Commit(b *testing.B) {
	root := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		player := root
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		sparseWriteUndo(player, ctx)
		ctx.Reset()
		txPool.Put(ctx)
	}
}

func BenchmarkUndoLog_SparseWrite_Rollback(b *testing.B) {
	root := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		player := root
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		sparseWriteUndo(player, ctx)
		ctx.Rollback()
		txPool.Put(ctx)
	}
}

func BenchmarkDeepCopyGen_SparseWrite(b *testing.B) {
	root := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		work := root.DeepCopy()
		sparseWriteDirect(work)
	}
}
```

**注意：** `BenchmarkUndoLog_*` 在 `b.N` 循环内反复修改同一 `root`，Rollback 用例会恢复状态；Commit 用例每轮 Reset 后 gold/items/hero 会累积——实现时二选一：

- **推荐修复（写入计划执行时采用）：** 每轮 `clonePlayerSnapshot(root)` 还原，或每轮 `Rollback` 后再测 Commit 路径；Commit benchmark 应在循环内 `Rollback` 或重置 `root` 为 `newBenchPlayer()` 副本。

**修正后的 Commit benchmark：**

```go
func BenchmarkUndoLog_SparseWrite_Commit(b *testing.B) {
	seed := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		player := seed.DeepCopy() // 每轮干净对象，只测 undo 开销
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		sparseWriteUndo(player, ctx)
		ctx.Reset()
		txPool.Put(ctx)
	}
}
```

Rollback benchmark 同理用 `seed.DeepCopy()`，末步 `Rollback`，保证 `b.N` 可重复。

- [ ] **Step 2: 运行 Benchmark**

```bash
go test -bench='BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=5 ./... | tee /tmp/cow-bench-new.txt
```

- [ ] **Step 3: 用 benchstat 对比（若有旧基线）**

```bash
# 无旧文件时仅记录本次
go install golang.org/x/perf/cmd/benchstat@latest
benchstat /tmp/cow-bench-new.txt
```

在 PR/回复中用 Markdown 表格汇报 `ns/op`、`allocs/op`，并**询问用户**是否归档到 `docs/superpowers/benchmarks/cow-undo-log-mvp-benchmark.md`。

- [ ] **Step 4: Commit**

```bash
git add benchmark_test.go
git commit -m "$(cat <<'EOF'
test(cow): 添加 Undo Log 与 deepcopy-gen 全量拷贝 Benchmark

EOF
)"
```

---

### Task 8: 验收与文档收尾

**Files:**
- Modify: `docs/superpowers/benchmarks/`（用户确认后）

- [ ] **Step 1: 验收清单**

```bash
go test ./... -count=1
go test -bench='BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem ./...
```

确认：

- [ ] `TestRollback_RestoresInitialState` PASS
- [ ] `TestCommit_KeepsMutations` PASS
- [ ] `BenchmarkDeepCopyGen_SparseWrite` 的 `allocs/op` 显著高于 `BenchmarkUndoLog_SparseWrite_Commit`
- [ ] `zz_generated.deepcopy.go` 已跟踪
- [ ] 单文件 ≤500 行、单函数 ≤50 行

- [x] **Step 2: 需求已并入 `docs/guide/overview.md` 与根 README（2026-05-25）**

添加一行指向本 spec/plan（非必须，用户要求时再改）。

- [ ] **Step 3: 最终 Commit（用户确认后）**

```bash
git status
# 若有 benchmark 归档或未提交文件，按实际 add
```

---

## Spec 覆盖自检

| Spec 章节 | 任务 |
|---|---|
| Undo Log 架构 | Task 5–6 |
| TxContext Pool | Task 5 |
| 三写代理 | Task 6 |
| deepcopy-gen 提交 Git | Task 2 |
| B 档夹具 | Task 3 |
| 正确性测试 | Task 4, 6 |
| 三组 Benchmark | Task 7 |
| 无 main | 无 Task |
| 中文注释 | 各 .go 文件包/导出注释 |
| TDD 顺序 | Task 4 RED → 5–6 GREEN → 7 bench |

无 TBD；Benchmark Commit 循环内使用 `DeepCopy` 重置夹具已在 Task 7 修正说明。

---

## 执行方式

Plan 已保存至 `docs/superpowers/plans/2026-05-25-cow-undo-log-mvp.md`。

**两种执行方式：**

1. **Subagent-Driven（推荐）** — 按 Task 派发子代理，任务间你做审查  
2. **Inline Execution** — 本会话按 Task 1→8 逐步实现（使用 executing-plans，分批 checkpoint）

你更希望用哪种？回复 **「1」** 或 **「2」**（或直接说「开始实现」即走 Inline）。
