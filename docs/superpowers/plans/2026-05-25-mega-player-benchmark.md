# Mega Player 夹具与 Benchmark 实现计划

> **状态：已实现**（截至 2026-05-27；本计划为历史执行记录，勿按未勾选步骤重复开发）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 扩展游戏向 `Player`（堆约 1MB）、双档夹具、全覆盖探针与 mega Benchmark，验证 `undoproxy-gen` 在复杂嵌套下的正确性，并对比 `deepcopy-gen` 全量拷贝性能。

**Architecture:** 单 `Player` 类型 + `types_game.go` 子结构；`newBenchPlayer` 仅填 lite 字段；`newMegaBenchPlayer` 按可调常量填充；`go generate` 刷新代理与 DeepCopy；测试以 `DeepCopy` 快照 + `cmp` 断言回滚一致。

**Tech Stack:** Go 1.25、`undoproxy-gen`、`deepcopy-gen`、`github.com/google/go-cmp/cmp`

**工作目录:** `/Users/huangyu/work/golang/src/cow`

**设计说明:** `docs/superpowers/specs/2026-05-25-mega-player-benchmark-design.md`

---

## 文件一览

| 文件 | 操作 |
|------|------|
| `types_game.go` | 新建：`Skill`、`Mail`、`Quest`；扩展 `Hero` |
| `types.go` | 扩展 `Player` 字段 + `Level` 标量 |
| `zz_generated.deepcopy.go` | `go generate` 更新 |
| `zz_generated.undo_proxy.go` | `go generate` 更新 |
| `bench_mega_fixture.go` | 新建：常量、`newMegaBenchPlayer`、`approxPlayerHeapBytes` |
| `bench_fixture_test.go` | 确认 lite 不填 mega 字段 |
| `player_mega_test.go` | 新建：体积 + 探针 + 业务路径 |
| `benchmark_mega_test.go` | 新建：三组 mega Benchmark |
| `bench_fixture_test.go` | 扩展 `assertPlayerEqual` comparer |
| `docs/superpowers/benchmarks/cow-mega-player-benchmark.md` | 跑 bench 后追加（用户确认后） |

---

### Task 1: 游戏子类型与 Player 字段（TDD 先导）

**Files:**
- Create: `types_game.go`
- Modify: `types.go`

- [ ] **Step 1: 创建 `types_game.go`**

```go
package cow

// Skill 技能。
//
// +k8s:deepcopy-gen=true
type Skill struct {
	SkillId int32 `bson:"skill_id"`
	Level   int32 `bson:"level"`
}

// Mail 邮件（Body 用于拉高 mega 体积）。
//
// +k8s:deepcopy-gen=true
type Mail struct {
	Id      uint64 `bson:"id"`
	Subject string `bson:"subject"`
	Body    string `bson:"body"`
}

// Quest 任务。
//
// +k8s:deepcopy-gen=true
type Quest struct {
	Id         int32           `bson:"id"`
	State      int32           `bson:"state"`
	Objectives map[int32]int32 `bson:"objectives"`
}
```

- [ ] **Step 2: 扩展 `Hero` 与 `Item`（同文件或 `types.go`）**

在 `types.go` 将 `Hero` 改为含 `Skills`：

```go
type Hero struct {
	HeroId int32             `bson:"hero_id"`
	Level  int32             `bson:"level"`
	Skills map[int32]*Skill  `bson:"skills"`
}
```

`Item` 增加可选 `Extra string`（体积与区分用）。

- [ ] **Step 3: 扩展 `Player`**

```go
// +k8s:deepcopy-gen=true
// +cow:undoproxy-gen=true
type Player struct {
	Uid       int64                       `bson:"_id"`
	Level     int32                       `bson:"level"`
	Assets    map[string]int64            `bson:"assets"`
	Items     []*Item                     `bson:"items"`
	Hero      *Hero                       `bson:"hero"`
	Heros     map[int32]*Hero             `bson:"heros"`
	Bags      map[int32][]*Item           `bson:"bags"`
	Stats     map[int32]map[string]int64  `bson:"stats"`
	Cooldowns map[int32][]int32           `bson:"cooldowns"`
	Mails     map[uint64]*Mail            `bson:"mails"`
	Quests    map[int32]*Quest            `bson:"quests"`
}
```

- [ ] **Step 4: 编译检查（生成前会缺 DeepCopy 方法，仅语法）**

```bash
cd /Users/huangyu/work/golang/src/cow
go build -o /dev/null ./... 2>&1 | head -5
```

预期：可能报 `DeepCopy` 未定义，进入 Task 2。

---

### Task 2: 代码生成（deepcopy + undoproxy）

**Files:**
- Modify: `zz_generated.deepcopy.go`（生成）
- Modify: `zz_generated.undo_proxy.go`（生成）

- [ ] **Step 1: 重新生成**

```bash
go install ./cmd/undoproxy-gen
go generate ./...
```

- [ ] **Step 2: 确认生成物含 mega 字段代理**

```bash
grep -E 'func \(p \*Player\) (PutHeros|AppendBagsAt|PutStats|GetStatsMapForWrite|AppendCooldownsAt)' zz_generated.undo_proxy.go
```

预期：均存在。

- [ ] **Step 3: 全量编译**

```bash
go build ./...
```

Expected: success

---

### Task 3: Mega 夹具与体积测试（TDD）

**Files:**
- Create: `bench_mega_fixture.go`
- Create: `player_mega_test.go`（仅 `TestMegaFixtureSize` 先写）

- [ ] **Step 1: 写失败测试 `TestMegaFixtureSize`**

```go
func TestMegaFixtureSize(t *testing.T) {
	p := newMegaBenchPlayer()
	got := approxPlayerHeapBytes(p)
	const want = 1 << 20
	lo := want * 85 / 100
	hi := want * 115 / 100
	if got < lo || got > hi {
		t.Fatalf("heap approx %d not in [%d,%d]", got, lo, hi)
	}
}
```

- [ ] **Step 2: 运行确认 FAIL**

```bash
go test ./... -run TestMegaFixtureSize -count=1
```

- [ ] **Step 3: 实现 `bench_mega_fixture.go`**

顶部常量（与 spec §5.2 一致）：

```go
const (
	megaItemCount        = 2000
	megaItemNameLen      = 48
	megaHeroCount        = 120
	megaSkillsPerHero    = 40
	megaBagCount         = 30
	megaItemsPerBag      = 40
	megaMailCount        = 80
	megaMailBodyLen      = 4096
	megaQuestCount       = 100
	megaStatGroups       = 12
	megaStatKeysPerGroup = 50
	megaCooldownKeys     = 80
	megaCooldownListLen  = 24
	megaAssetCount       = 200
)
```

实现 `newMegaBenchPlayer()`：

- 填充 `Assets`、`Items`（`makeName(megaItemNameLen)`）、`Hero`（lite 兼容）
- `Heros`：1..megaHeroCount，各带 `Skills`
- `Bags`：每页 `megaItemsPerBag` 个 `*Item`（可复用指针或新建）
- `Mails`：`Body` 为 `strings.Repeat('x', megaMailBodyLen)`
- `Quests`、`Stats`、`Cooldowns` 按常量填充

实现 `approxPlayerHeapBytes`：递归估算 string/slice/map 贡献（见 spec §4.1）。

- [ ] **Step 4: 调参直至 PASS**

```bash
go test ./... -run TestMegaFixtureSize -count=1
```

若 `got > hi`：减小 `megaMailBodyLen` 或 `megaItemCount`；若 `got < lo`：增大 `megaMailBodyLen`。

- [ ] **Step 5: 确认 lite 构造仍轻量**

`newBenchPlayer()` 不初始化 `Heros`/`Bags`/`Mails` 等（nil）。

```bash
go test ./... -run 'TestRollback|TestCommit' -count=1
```

---

### Task 4: 扩展 `assertPlayerEqual`

**Files:**
- Modify: `bench_fixture_test.go`

- [ ] **Step 1: 添加 comparer**

```go
cmp.Comparer(func(a, b *Skill) bool { ... SkillId, Level ... }),
cmp.Comparer(func(a, b *Mail) bool { ... Id, Subject, len(Body) ... }),
cmp.Comparer(func(a, b *Quest) bool { ... Id, State, maps equal ... }),
```

`Hero` comparer 增加 `Skills` 浅比较（或 cmp 允许忽略未导出字段，按 Id 比每个 skill）。

- [ ] **Step 2: 跑 lite 测试**

```bash
go test ./... -run 'TestRollback|TestCommit' -count=1
```

---

### Task 5: `applyMegaSparseWrites` 与业务路径测试

**Files:**
- Modify: `bench_mega_fixture.go`（或 `benchmark_mega_test.go`）
- Modify: `player_mega_test.go`

- [ ] **Step 1: 实现 `applyMegaSparseWrites`**

使用 mega 夹具中**稳定存在的 key**（构造时写死 `heroId=1`、`bagId=1`、`statGroup=1`）：

```go
func applyMegaSparseWrites(p *Player, ctx *TxContext) {
	p.PutAssets(ctx, "gold", 500)
	h := p.GetHeroForWrite(ctx, 1)
	h.PutLevel(ctx, 99)
	p.AppendItems(ctx, &Item{Id: 9999, Name: "probe_item"})
	// PutHeros / GetHero + PutSkills 视生成名为准
	p.AppendBagsAt(ctx, 1, &Item{Id: 8888, Name: "bag_item"})
	p.PutStats(ctx, 1, "atk", 100)
}
```

- [ ] **Step 2: `TestMegaPlayer_BusinessPath_Rollback`**

```go
func TestMegaPlayer_BusinessPath_Rollback(t *testing.T) {
	p := newMegaBenchPlayer()
	want := clonePlayerSnapshot(p)
	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		applyMegaSparseWrites(p, ctx)
		return errors.New("fail")
	})
	// assert equal
}
```

- [ ] **Step 3: 运行 PASS**

```bash
go test ./... -run TestMegaPlayer_BusinessPath -count=1
```

---

### Task 6: 全覆盖探针测试

**Files:**
- Modify: `player_mega_test.go`

- [ ] **Step 1: 实现 `applyMegaProxyProbe`**

按 spec §6.1 表顺序调用（生成后对齐实际方法名），包含：

- `PutLevel`（Player 标量）
- `RemoveItemsAt` / `TruncateItems`
- `RemoveBagsAt`
- `GetStatsMapForWrite` 后 `inner["probe"]=1` 或 `PutStats`

- [ ] **Step 2: `TestMegaPlayer_ProxyProbe_Rollback`**

- [ ] **Step 3: 运行**

```bash
go test ./... -run TestMegaPlayer_ProxyProbe -count=1
```

失败时：检查是否漏生成代理或探针顺序导致不可逆（应用 `Rollback` 前应可逆）。

---

### Task 7: Mega Benchmark

**Files:**
- Create: `benchmark_mega_test.go`

- [ ] **Step 1: 实现 `sparseWriteMegaDirect`**

与 `applyMegaSparseWrites` 等价的三处**直接写**（无 ctx）。

- [ ] **Step 2: 三组 Benchmark**

```go
func BenchmarkMega_UndoLog_SparseWrite_Rollback(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		applyMegaSparseWrites(player, ctx)
		ctx.Rollback()
		txPool.Put(ctx)
	}
}

func BenchmarkMega_UndoLog_SparseWrite_Commit(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		applyMegaSparseWrites(player, ctx)
		ctx.Reset()
		txPool.Put(ctx)
	}
}

func BenchmarkMega_DeepCopyGen_SparseWrite(b *testing.B) {
	seed := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		work := seed.DeepCopy()
		sparseWriteMegaDirect(work)
	}
}
```

- [ ] **Step 3: 运行 Benchmark**

```bash
go test -bench='BenchmarkMega_' -benchmem -count=3 ./...
```

- [ ] **Step 4: 与 lite 对比（可选）**

```bash
go test -bench='BenchmarkMega_|BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=5 ./... | tee /tmp/cow-mega-bench.txt
```

- [ ] **Step 5: 向用户展示对比表并询问是否归档**

按 `AGENTS.md` 用 Markdown 表汇报；用户确认后写入 `docs/superpowers/benchmarks/cow-mega-player-benchmark.md`。

---

### Task 8: 验收与文档状态

- [ ] **Step 1: 全量测试**

```bash
go test ./... -count=1
```

- [ ] **Step 2: 更新 spec 状态**

将 `docs/superpowers/specs/2026-05-25-mega-player-benchmark-design.md` 状态改为「已实现」。

- [ ] **Step 3: spec §9 清单逐项勾选**

- [ ] **Step 4: Commit（仅用户明确要求时）**

```bash
git add types.go types_game.go bench_mega_fixture.go player_mega_test.go benchmark_mega_test.go
git add bench_fixture_test.go zz_generated.*.go docs/superpowers/
git commit -m "$(cat <<'EOF'
feat(cow): 添加 ~1MB mega Player 夹具与 Undo/DeepCopy 压测

EOF
)"
```

---

## Spec 覆盖自检

| Spec § | Task |
|--------|------|
| §4 体积 | Task 3 |
| §5 模型 | Task 1–2 |
| §5.3 lite | Task 3 Step 5 |
| §6 探针/业务 | Task 5–6 |
| §7 Benchmark | Task 7 |
| §8 文件 | 文件一览 |
| §9 验收 | Task 8 |

## 执行方式

计划已保存至 `docs/superpowers/plans/2026-05-25-mega-player-benchmark.md`。

1. **Subagent-Driven** — 按 Task 派发子 agent  
2. **Inline Execution** — 本会话直接实现  

回复 **1**、**2** 或 **「开始实现」** 即可开工。
