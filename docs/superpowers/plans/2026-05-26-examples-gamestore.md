# examples/gamestore 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增独立 module `examples/gamestore/`，演示集成方自有聚合根、`go generate`、双根类型图、TxContext Commit/Rollback 与 `undocheck` 用法。

**Architecture:** 示例子模块通过 `replace` 引用仓库根以运行 `cmd/undoproxy-gen`；`types.go` 定义 `Player`+`Guild` 类型图；`go generate` 产出 `zz_generated.undo_proxy.go`；`service` 管理 `txPool` 作用域；`handler` 集中代理写；测试用手写快照对比，不依赖 `deepcopy-gen`。

**Tech Stack:** Go 1.25、`undoproxy-gen`、`undocheck`（README/CI vet）

**Spec:** [../specs/2026-05-26-examples-gamestore-design.md](../specs/2026-05-26-examples-gamestore-design.md)

---

## 文件结构（目标态）

| 文件 | 职责 |
|------|------|
| `examples/gamestore/go.mod` | 独立 module + `replace ../..` |
| `examples/gamestore/doc.go` | 包注释、`+cow:undoproxy-gen=package` |
| `examples/gamestore/types.go` | 双根类型 + `newDemoPlayer` / `newDemoGuild` |
| `examples/gamestore/generate.go` | `//go:generate` |
| `examples/gamestore/zz_generated.undo_proxy.go` | 生成物（提交） |
| `examples/gamestore/service.go` | `runScopedCommit` / `runScopedWithRollback` |
| `examples/gamestore/snapshot.go` | 测试用手写深拷贝快照（无 deepcopy-gen） |
| `examples/gamestore/handler.go` | 业务写路径 |
| `examples/gamestore/main.go` | `go run .` 演示 |
| `examples/gamestore/handler_test.go` | Rollback / Commit / Guild / 生成物契约 |
| `examples/gamestore/README.md` | 集成步骤 |
| `.github/workflows/ci.yml` | `example-gamestore` job |
| `README.md` | 链到示例 |
| `docs/guide/README.md` | 一行索引 |

---

## Task 1: 模块骨架与类型图

**Files:**
- Create: `examples/gamestore/go.mod`
- Create: `examples/gamestore/doc.go`
- Create: `examples/gamestore/types.go`
- Create: `examples/gamestore/generate.go`

- [ ] **Step 1: 创建 `go.mod`**

```go
module github.com/huangyuCN/cow/examples/gamestore

go 1.25

require github.com/huangyuCN/cow v0.0.0

replace github.com/huangyuCN/cow => ../..
```

- [ ] **Step 2: 创建 `doc.go`**

```go
// Package gamestore 演示 cow 独立接入：自有聚合根、undoproxy-gen、TxContext。
//
// +cow:undoproxy-gen=package
package gamestore
```

- [ ] **Step 3: 创建 `types.go`（双根 + 夹具）**

```go
package gamestore

// +cow:undoproxy-gen=true
type Player struct {
	Gold     int64
	Wallet   map[string]int64
	Items    []*Item
	MainHero *Hero
	Heros    map[int32]*Hero
	Bags     map[int32][]*Item
	Stats    map[int32]map[string]int64
}

// +cow:undoproxy-gen=true
type Guild struct {
	Members map[int32]*Member
}

type Item struct {
	Id   int64
	Name string
}

type Hero struct {
	Level  int32
	Skills map[int32]*Skill
}

type Member struct {
	Name string
	Rank int32
}

type Skill struct {
	Level int32
}

func newDemoPlayer() *Player {
	p := &Player{
		Gold: 1000,
		Wallet: map[string]int64{
			"gold": 500,
			"gems": 10,
		},
		MainHero: &Hero{Level: 1, Skills: map[int32]*Skill{1: {Level: 1}}},
		Heros:    make(map[int32]*Hero),
		Bags:     make(map[int32][]*Item),
		Stats:    make(map[int32]map[string]int64),
	}
	for i := int32(1); i <= 5; i++ {
		p.Heros[i] = &Hero{Level: i, Skills: map[int32]*Skill{i: {Level: 1}}}
	}
	p.Items = []*Item{{Id: 1, Name: "starter"}}
	p.Bags[1] = []*Item{{Id: 101, Name: "bag1"}}
	p.Stats[1] = map[string]int64{"atk": 10, "def": 5}
	return p
}

func newDemoGuild() *Guild {
	return &Guild{
		Members: map[int32]*Member{
			1: {Name: "alice", Rank: 1},
		},
	}
}
```

- [ ] **Step 4: 创建 `generate.go`**

```go
package gamestore

//go:generate go run github.com/huangyuCN/cow/cmd/undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow/examples/gamestore
```

- [ ] **Step 5: 生成并确认包可加载**

```bash
cd /Users/huangyu/work/golang/src/cow/examples/gamestore
go generate ./...
go build -o /dev/null .
```

Expected: 生成 `zz_generated.undo_proxy.go`，`go build` 通过（尚无 handler 时仅类型+生成物即可）。

- [ ] **Step 6: Commit**

```bash
git add examples/gamestore/go.mod examples/gamestore/doc.go examples/gamestore/types.go examples/gamestore/generate.go examples/gamestore/zz_generated.undo_proxy.go
git commit -m "feat(examples): 新增 gamestore 模块骨架与类型图"
```

---

## Task 2: 作用域辅助与快照

**Files:**
- Create: `examples/gamestore/service.go`
- Create: `examples/gamestore/snapshot.go`
- Test: `examples/gamestore/service_test.go`

- [ ] **Step 1: 写失败测试 `TestRunScopedCommit_ResetsOnSuccess`**

```go
package gamestore

import "testing"

func TestRunScopedCommit_ResetsOnSuccess(t *testing.T) {
	p := newDemoPlayer()
	beforeGold := p.Gold
	err := runScopedCommit(func(ctx *TxContext) error {
		p.PutGold(ctx, beforeGold+100)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Gold != beforeGold+100 {
		t.Fatalf("gold got %d want %d", p.Gold, beforeGold+100)
	}
}
```

（生成后方法名为 `PutGold`；若生成器命名不同，以 `zz_generated.undo_proxy.go` 为准调整。）

- [ ] **Step 2: 运行确认失败**

```bash
cd examples/gamestore && go test -run TestRunScopedCommit -count=1
```

Expected: FAIL（`runScopedCommit` 未定义）

- [ ] **Step 3: 实现 `service.go`**

```go
package gamestore

// runScopedWithRollback 在 fn 返回后总是 Rollback（用于可恢复性测试/演示）。
func runScopedWithRollback(fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	return fn(ctx)
}

// runScopedCommit 成功时 Reset 提交；失败时 Rollback。
func runScopedCommit(fn func(ctx *TxContext) error) error {
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
```

- [ ] **Step 4: 实现 `snapshot.go`（手写拷贝，供测试对比）**

```go
package gamestore

// clonePlayer 深拷贝 Player 快照（仅示例测试用，不引入 deepcopy-gen）。
func clonePlayer(p *Player) *Player {
	if p == nil {
		return nil
	}
	c := &Player{
		Gold:     p.Gold,
		Wallet:   make(map[string]int64, len(p.Wallet)),
		Items:    make([]*Item, len(p.Items)),
		MainHero: cloneHero(p.MainHero),
		Heros:    make(map[int32]*Hero, len(p.Heros)),
		Bags:     make(map[int32][]*Item, len(p.Bags)),
		Stats:    make(map[int32]map[string]int64, len(p.Stats)),
	}
	for k, v := range p.Wallet {
		c.Wallet[k] = v
	}
	for i, it := range p.Items {
		if it != nil {
			c.Items[i] = &Item{Id: it.Id, Name: it.Name}
		}
	}
	for k, h := range p.Heros {
		c.Heros[k] = cloneHero(h)
	}
	for k, bag := range p.Bags {
		c.Bags[k] = cloneItems(bag)
	}
	for k, inner := range p.Stats {
		c.Stats[k] = make(map[string]int64, len(inner))
		for ik, iv := range inner {
			c.Stats[k][ik] = iv
		}
	}
	return c
}

func cloneHero(h *Hero) *Hero {
	if h == nil {
		return nil
	}
	c := &Hero{Level: h.Level, Skills: make(map[int32]*Skill, len(h.Skills))}
	for k, s := range h.Skills {
		if s != nil {
			c.Skills[k] = &Skill{Level: s.Level}
		}
	}
	return c
}

func cloneItems(s []*Item) []*Item {
	out := make([]*Item, len(s))
	for i, it := range s {
		if it != nil {
			out[i] = &Item{Id: it.Id, Name: it.Name}
		}
	}
	return out
}

func cloneGuild(g *Guild) *Guild {
	if g == nil {
		return nil
	}
	c := &Guild{Members: make(map[int32]*Member, len(g.Members))}
	for k, m := range g.Members {
		if m != nil {
			c.Members[k] = &Member{Name: m.Name, Rank: m.Rank}
		}
	}
	return c
}

func playersEqual(a, b *Player) bool {
	// 用 github.com/google/go-cmp 或关键字段比较；示例 module 可 require cmp
	...
}
```

实现 `playersEqual`：在 `go.mod` 增加 `require github.com/google/go-cmp v0.7.0`（与根模块同版本），`cmp.Equal(clonePlayer(a), b)` 或 `cmp.Diff`。

- [ ] **Step 5: 运行测试通过**

```bash
cd examples/gamestore && go test -run TestRunScopedCommit -count=1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add examples/gamestore/service.go examples/gamestore/snapshot.go examples/gamestore/service_test.go examples/gamestore/go.mod examples/gamestore/go.sum
git commit -m "feat(examples): gamestore 作用域辅助与测试快照"
```

---

## Task 3: 业务 handler 与测试

**Files:**
- Create: `examples/gamestore/handler.go`
- Create: `examples/gamestore/handler_test.go`

- [ ] **Step 1: 写失败测试 `TestHandlePurchaseFail_Rollback`**

```go
func TestHandlePurchaseFail_Rollback(t *testing.T) {
	p := newDemoPlayer()
	want := clonePlayer(p)
	err := runScopedWithRollback(func(ctx *TxContext) error {
		return HandlePurchaseFail(p, ctx)
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !playersEqual(p, want) {
		t.Fatal("player not restored after rollback")
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
cd examples/gamestore && go test -run TestHandlePurchaseFail -count=1
```

Expected: FAIL

- [ ] **Step 3: 实现 `handler.go`**

覆盖 spec §7.2（以生成方法名为准）：

```go
package gamestore

import "errors"

// HandlePurchaseSuccess 演示成功提交：多 Kind 代理写 + Guild 第二根。
func HandlePurchaseSuccess(p *Player, g *Guild, ctx *TxContext) error {
	p.PutGold(ctx, p.Gold-100)
	p.PutWallet(ctx, "gold", p.Wallet["gold"]+50)
	p.AppendItems(ctx, &Item{Id: 9001, Name: "loot"})
	if len(p.Items) > 0 {
		p.SetItemsAt(ctx, 0, &Item{Id: 1, Name: "upgraded"})
	}
	if mh := p.GetMainHeroForWrite(ctx); mh != nil {
		mh.PutLevel(ctx, mh.Level+1)
	}
	if h := p.GetHeroForWrite(ctx, 1); h != nil {
		h.PutLevel(ctx, h.Level+1)
	}
	p.PutStats(ctx, 1, "atk", 99)
	p.AppendBagsAt(ctx, 1, &Item{Id: 202, Name: "bag_drop"})
	if inner := p.GetStatsMapForWrite(ctx, 1); inner != nil {
		inner["bonus"] = 1
	}
	g.PutMembers(ctx, 2, &Member{Name: "bob", Rank: 2})
	return nil
}

// HandlePurchaseFail 中途失败，供 Rollback 演示。
func HandlePurchaseFail(p *Player, ctx *TxContext) error {
	p.PutGold(ctx, 99999)
	p.PutWallet(ctx, "gold", 0)
	return errors.New("payment declined")
}
```

生成后若方法名为 `PutStats(ctx,k1,k2,val)` / `GetStatsMapForWrite` 等与上不同，按 `zz_generated.undo_proxy.go` 调整。

- [ ] **Step 4: 实现 `TestHandlePurchaseSuccess_Commit`**

```go
func TestHandlePurchaseSuccess_Commit(t *testing.T) {
	p := newDemoPlayer()
	g := newDemoGuild()
	beforeGold := p.Gold
	err := runScopedCommit(func(ctx *TxContext) error {
		return HandlePurchaseSuccess(p, g, ctx)
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Gold != beforeGold-100 {
		t.Fatalf("gold got %d want %d", p.Gold, beforeGold-100)
	}
	if len(g.Members) < 2 {
		t.Fatal("guild member not added")
	}
}
```

- [ ] **Step 5: 实现 `TestGenerated_contract`**

```go
func TestGenerated_contract(t *testing.T) {
	// 只读检查：生成物在编译期已链接；此处断言包内符号存在即可
	var ctx TxContext
	ctx.Reset()
	_ = txPool
}
```

可选：读取 `zz_generated.undo_proxy.go` 用 `os.ReadFile` + `strings.Contains` 断言无 `AddUndo`（路径 `zz_generated.undo_proxy.go` 相对测试文件）。

- [ ] **Step 6: 全量示例测试**

```bash
cd examples/gamestore && go test ./... -count=1
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add examples/gamestore/handler.go examples/gamestore/handler_test.go
git commit -m "feat(examples): gamestore 业务 handler 与 Rollback/Commit 测试"
```

---

## Task 4: main 演示与 README

**Files:**
- Create: `examples/gamestore/main.go`
- Create: `examples/gamestore/README.md`

- [ ] **Step 1: 实现 `main.go`**

```go
package main

import (
	"fmt"

	"github.com/huangyuCN/cow/examples/gamestore"
)

func main() {
	fmt.Println("=== cow examples/gamestore ===")

	p1 := gamestore.NewDemoPlayerForMain() // 或导出 newDemoPlayer：在 types 增加 NewDemoPlayer()
	g1 := gamestore.NewDemoGuildForMain()

	// Rollback 演示
	pRollback := gamestore.ClonePlayerPublic(p1) // 或在 main 包无法访问时用 gamestore.DemoRollback()
	gamestore.RunDemoRollback(pRollback)

	// Commit 演示
	pCommit := gamestore.ClonePlayerPublic(p1)
	gamestore.RunDemoCommit(pCommit, gamestore.CloneGuildPublic(g1))

	fmt.Println("done")
}
```

**注意：** `newDemoPlayer` 为小写未导出。二选一：

- 在 `types.go` 增加 `func NewDemoPlayer() *Player { return newDemoPlayer() }` 供 `main` 使用；或
- `main.go` 使用 `package gamestore` 无 `main` —— 改为 `cmd/gamestore-demo/main.go` 子目录。

**推荐：** 保持 `package gamestore`，`main.go` 放在 `examples/gamestore/cmd/demo/main.go` **会增加复杂度**。更简单：**`main.go` 使用 `package main` 且将 `newDemoPlayer` 改为导出 `NewDemoPlayer` / `NewDemoGuild`**，并导出 `RunDemoRollback` / `RunDemoCommit` 包装函数在 `demo.go`。

在计划中锁定：**新增 `demo.go`（package gamestore）**：

```go
func NewDemoPlayer() *Player { return newDemoPlayer() }
func NewDemoGuild() *Guild { return newDemoGuild() }

func RunDemoRollback(p *Player) {
	before := p.Gold
	_ = runScopedWithRollback(func(ctx *TxContext) error {
		return HandlePurchaseFail(p, ctx)
	})
	fmt.Printf("rollback: gold %d -> %d (restored)\n", before, p.Gold)
}

func RunDemoCommit(p *Player, g *Guild) {
	before := p.Gold
	_ = runScopedCommit(func(ctx *TxContext) error {
		return HandlePurchaseSuccess(p, g, ctx)
	})
	fmt.Printf("commit: gold %d -> %d\n", before, p.Gold)
}
```

`main.go`：

```go
package main

import (
	"fmt"
	"github.com/huangyuCN/cow/examples/gamestore"
)

func main() {
	p := gamestore.NewDemoPlayer()
	g := gamestore.NewDemoGuild()
	p2 := gamestore.ClonePlayer(p) // 导出 ClonePlayer 或 Demo 内复制
	// 简化：Demo 内部分配两次 newDemoPlayer
	fmt.Println("rollback demo:")
	gamestore.RunDemoRollback(gamestore.NewDemoPlayer())
	fmt.Println("commit demo:")
	gamestore.RunDemoCommit(gamestore.NewDemoPlayer(), gamestore.NewDemoGuild())
}
```

- [ ] **Step 2: 编写 `README.md`**

须含：

1. 前提：Go 1.25、仓库根 `replace` 或发布后 `go get`
2. `go generate ./...`
3. `go test ./...` / `go run .`
4. `go install github.com/huangyuCN/cow/cmd/undocheck@...` + `go vet -vettool=... ./...`
5. 链到 `docs/guide/integration-checklist.md`、`proxy-api.md`
6. `undorewrite` 仅迁移用，链到 migration 文档

- [ ] **Step 3: 验证**

```bash
cd examples/gamestore && go run . && go test ./... -count=1
```

Expected: 打印 rollback/commit 说明；测试 PASS

- [ ] **Step 4: Commit**

```bash
git add examples/gamestore/main.go examples/gamestore/demo.go examples/gamestore/README.md
git commit -m "feat(examples): gamestore 可运行 demo 与 README"
```

---

## Task 5: CI 与根文档链接

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `docs/guide/README.md`

- [ ] **Step 1: 追加 CI job**

```yaml
  example-gamestore:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.x"
          cache: true
      - run: cd examples/gamestore && go test ./... -count=1 && go build -o /dev/null .
```

- [ ] **Step 2: 更新 `README.md`**

在「文档」表前或「快速开始」后增加：

```markdown
## 完整接入示例

独立 module 演示（自有聚合根、`go generate`、Rollback/Commit）：[examples/gamestore/README.md](examples/gamestore/README.md)。
```

- [ ] **Step 3: 更新 `docs/guide/README.md`**

在集成文档列表增加一行指向 `examples/gamestore/README.md`。

- [ ] **Step 4: 根包全量测试仍通过**

```bash
cd /Users/huangyu/work/golang/src/cow && go test ./... -count=1
```

Expected: PASS（不包含 examples，独立 module）

- [ ] **Step 5: 示例子模块 undocheck（本地验收）**

```bash
go install ./cmd/undocheck
cd examples/gamestore && go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

Expected: 无 `cowbarewrite` 诊断

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/ci.yml README.md docs/guide/README.md
git commit -m "ci(docs): 增加 gamestore 示例 job 与文档链接"
```

---

## Task 6: 终验

- [ ] **Step 1: 再生成无 diff**

```bash
cd examples/gamestore && go generate ./... && git diff --exit-code zz_generated.undo_proxy.go
```

Expected: 无差异

- [ ] **Step 2: Spec 验收清单**

| # | 项 | 命令 |
|---|-----|------|
| 1 | 测试+运行 | `cd examples/gamestore && go test ./... && go run .` |
| 2 | 双根+Kind | 人工核对 `types.go` 与 `handler.go` |
| 3 | 无裸写 | `go vet -vettool=undocheck ./...` |
| 4 | 无 AddUndo | `rg 'AddUndo' examples/gamestore` 无匹配 |

- [ ] **Step 3: 更新 spec 状态（可选）**

在 `docs/superpowers/specs/2026-05-26-examples-gamestore-design.md` 表头状态改为「已实现」。

---

## 计划自检（已完成）

| Spec 章节 | 对应 Task |
|-----------|-----------|
| §2 目标 1–5 | Task 1–5 |
| §6 类型图 | Task 1 `types.go` |
| §7 运行时 | Task 2–4 |
| §8 工具链 | Task 4 README, Task 5 vet |
| §9 测试 | Task 3 |
| §10 CI | Task 5 |
| §11 文档 | Task 5 |
| §12 验收 | Task 6 |

无 TBD；`main` 通过导出 `NewDemoPlayer` + `demo.go` 解决未导出符号问题。
