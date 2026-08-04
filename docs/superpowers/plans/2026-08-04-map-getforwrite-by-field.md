# map GetForWrite 按字段名 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `map[K]*Struct` 的 GetForWrite 从元素类型名硬切为 `Get{Field}ForWrite`，消除同接收者多 map 撞名，并与 Put/Remove 命名轴一致。

**Architecture:** 单一真相源 `cowgen.MapKeyGetForWriteName(field)`；`emitStructuredMapPtrGet` 与 `cowproxy.methodsFromPlan` 均传入 `plan.FieldName`。不生成旧名别名；`ElemAtForWriteName` 本计划不改。

**Tech Stack:** Go 1.25、`internal/cowgen`、`cmd/undoproxy-gen`、`internal/cowproxy`、`cmd/undorewrite`

**Spec:** [../specs/2026-08-04-map-getforwrite-by-field-design.md](../specs/2026-08-04-map-getforwrite-by-field-design.md)

## Global Constraints

- Go **1.25**；禁止 deprecated API。
- TDD：命名与撞名夹具先行。
- 中文注释；导出名不以包名开头。
- 单文件 ≤500 行、单函数 ≤50 行。
- **未经用户明确同意不得 `git commit` / `git push`**；Commit 步骤改为「准备说明并等待确认」。
- 禁止 `.worktrees/` 开发。
- 硬切：无旧 `Get{Elem}ForWrite` 别名。
- examples 为嵌套 module：分别 `go generate` / `go test`，勿在仓库根跑 `./examples/...`。

### 破坏性重命名对照（实现时全文替换运行时代码）

| 字段 | 旧生成名 | 新生成名 |
|------|----------|----------|
| `Heros` | `GetHeroForWrite` | `GetHerosForWrite` |
| `Skills` | `GetSkillForWrite` | `GetSkillsForWrite` |
| `Mails` | `GetMailForWrite` | `GetMailsForWrite` |
| `Quests` | `GetQuestForWrite` | `GetQuestsForWrite` |

指针 `GetMainHeroForWrite` **不变**。

---

## 文件结构（目标态）

| 文件 | 职责 |
|------|------|
| `internal/cowgen/naming.go` | `MapKeyGetForWriteName(field)` |
| `internal/cowgen/naming_test.go` | 命名单测 |
| `cmd/undoproxy-gen/emit_structured.go` | `emitStructuredMapPtrGet` 用字段名 |
| `cmd/undoproxy-gen/testdata/types.go` | 双 map 同 Elem 撞名夹具 |
| `cmd/undoproxy-gen/emit_map_getforwrite_test.go` | 生成物撞名/命名集成测 |
| `internal/cowproxy/catalog.go` | `MapKeyGetForWriteName(plan.FieldName)` |
| `internal/cowproxy/catalog_test.go` | 断言 Heros → GetHerosForWrite |
| 根包 / examples 调用点与 `zz_generated.undo_proxy.go` | 新名 + regenerate |
| `docs/guide/proxy-api.md`、`docs/README.md` | 用户文档与索引 |
| `docs/superpowers/specs/2026-08-04-map-getforwrite-by-field-design.md` | 落地后改状态为已实现 |

不强制改历史 plan/spec 中的旧名（归档文字）；仅保证运行时代码与现行 guide 正确。

---

### Task 1: 命名函数硬切（cowgen）

**Files:**
- Modify: `internal/cowgen/naming.go`
- Modify: `internal/cowgen/naming_test.go`

**Interfaces:**
- Produces: `func MapKeyGetForWriteName(field string) string` → `"Get" + field + "ForWrite"`

- [ ] **Step 1: 写失败测试**

在 `naming_test.go` 追加：

```go
func TestMapKeyGetForWriteName_UsesField(t *testing.T) {
	cases := []struct{ field, want string }{
		{"Heros", "GetHerosForWrite"},
		{"VirtualStackable", "GetVirtualStackableForWrite"},
		{"Stackable", "GetStackableForWrite"},
		{"Skills", "GetSkillsForWrite"},
	}
	for _, tc := range cases {
		if got := cowgen.MapKeyGetForWriteName(tc.field); got != tc.want {
			t.Fatalf("MapKeyGetForWriteName(%q)=%q want %q", tc.field, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./internal/cowgen/ -run TestMapKeyGetForWriteName_UsesField -count=1
```

期望：FAIL（若仍按 Elem/Singular 语义，或注释/参数仍暗示 singular：`Heros` 若被别处 Singular 则不在本函数内——当前函数对任意入参只是前缀拼接，传入 `Heros` 已得 `GetHerosForWrite`；**若当前代码传入 ElemName 才撞名**。本测在改调用方前应已 PASS 于「字符串拼接」——故本任务的 RED 应体现为：先改测试期望并同步改函数注释；若函数体已是 `Get`+arg+`ForWrite`，则 Step 1 可能立刻绿）。

**标定 RED 方式（钉死）：** 先在测试中断言**错误旧行为**不可接受——即 `MapKeyGetForWriteName("Hero")` 虽等于 `GetHeroForWrite`，但契约改为字段名：文档注释必须写「按字段名」；并增加对 `VirtualStackable` 的断言。若函数体已是纯拼接，**本 Task 的实质工作是注释 + 契约测**；随后 Task 2 改调用方才是行为 GREEN。为严格 TDD 行为变更，将撞名生成测放 Task 2 作主红测；本 Task 完成契约测试 + 注释修订即可。

- [ ] **Step 3: 更新实现注释与参数名**

```go
// MapKeyGetForWriteName 按字段名生成 map[k]*Struct 的 GetForWrite（Heros → GetHerosForWrite）。
func MapKeyGetForWriteName(field string) string {
	return "Get" + field + "ForWrite"
}
```

- [ ] **Step 4: 测试通过**

```bash
go test ./internal/cowgen/ -run TestMapKeyGetForWriteName_UsesField -count=1
```

- [ ] **Step 5: 准备提交（等用户确认）**

建议说明：`test(cowgen): lock MapKeyGetForWriteName to field-based Get*ForWrite`

---

### Task 2: emit + catalog + 撞名夹具

**Files:**
- Modify: `cmd/undoproxy-gen/emit_structured.go`（`emitStructuredMapPtrGet`）
- Modify: `internal/cowproxy/catalog.go`
- Modify: `internal/cowproxy/catalog_test.go`
- Modify: `cmd/undoproxy-gen/testdata/types.go`
- Create: `cmd/undoproxy-gen/emit_map_getforwrite_test.go`

**Interfaces:**
- Consumes: `cowgen.MapKeyGetForWriteName(plan.FieldName)`
- Produces: 生成方法 `Get{Field}ForWrite`；catalog `GetForWrite` 同名

- [ ] **Step 1: 扩展 testdata**

在 `cmd/undoproxy-gen/testdata/types.go` 追加（或挂到已有根上）：

```go
type StackableItem struct {
	Qty int64
}

// +cow:undoproxy-gen=true
type Bag struct {
	Stackable        map[uint64]*StackableItem
	VirtualStackable map[uint64]*StackableItem
}
```

- [ ] **Step 2: 写失败集成测试**

```go
// emit_map_getforwrite_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmit_MapGetForWriteUsesFieldName(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	if err := Run(out, "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, name := range []string{"GetStackableForWrite", "GetVirtualStackableForWrite"} {
		def := "func (b *Bag) " + name + "("
		if c := strings.Count(s, def); c != 1 {
			t.Fatalf("%s defs=%d want 1", name, c)
		}
	}
	if strings.Contains(s, "GetStackableItemForWrite") {
		t.Fatal("must not emit elem-type-based GetStackableItemForWrite")
	}
}
```

```go
// catalog_test.go 追加
func TestCatalog_MapPtrGetForWriteUsesField(t *testing.T) {
	cat, err := cowproxy.NewCatalog("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	h, ok := cat.Lookup("Player", "Heros")
	if !ok || h.GetForWrite != "GetHerosForWrite" {
		t.Fatalf("Heros: %+v ok=%v", h, ok)
	}
}
```

（catalog 测在 Task 2 改 catalog 前对根包 `Heros` 会 FAIL：现为 `GetHeroForWrite`。）

- [ ] **Step 3: 运行确认失败**

```bash
go test ./cmd/undoproxy-gen/ -run TestEmit_MapGetForWriteUsesFieldName -count=1
go test ./internal/cowproxy/ -run TestCatalog_MapPtrGetForWriteUsesField -count=1
```

期望：FAIL（仍生成 `GetStackableItemForWrite` 两次，或 catalog 仍为 `GetHeroForWrite`）。

- [ ] **Step 4: 最小实现**

`emit_structured.go` 中 `emitStructuredMapPtrGet`：

```go
fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) %s {\n",
	r, structName, cowgen.MapKeyGetForWriteName(plan.FieldName), cowgen.KeyParams(plan.Keys), ret)
```

（替换原先 `Get%sForWrite` + `plan.ElemName`。）

`catalog.go`：

```go
fm.GetForWrite = cowgen.MapKeyGetForWriteName(plan.FieldName)
```

- [ ] **Step 5: 测试通过**

```bash
go test ./cmd/undoproxy-gen/ -run TestEmit_MapGetForWriteUsesFieldName -count=1
go test ./internal/cowproxy/ -run TestCatalog_MapPtrGetForWriteUsesField -count=1
go test ./internal/cowgen ./internal/cowproxy ./cmd/undoproxy-gen -count=1
```

期望：前两项 PASS；全包可能因根包尚未 regenerate / 调用点仍用旧名而失败——若失败仅因旧调用点，记录于 Task 3 处理，本 Task 以命名测 + catalog 测绿为准。

- [ ] **Step 6: 准备提交（等用户确认）**

建议说明：`fix(undoproxy-gen): emit map GetForWrite from field name`

---

### Task 3: 调用点硬切 + regenerate + 文档

**Files:**
- Modify: `bench_mega_writes.go`、`examples/gamestore/handler.go`、以及 `rg` 扫出的其它**运行时**引用
- Regenerate: `zz_generated.undo_proxy.go`、`examples/**/zz_generated.undo_proxy.go`
- Modify: `docs/guide/proxy-api.md`
- Modify: `docs/README.md`（待实现 → 已实现，若本任务收尾）
- Modify: `docs/superpowers/specs/2026-08-04-map-getforwrite-by-field-design.md` 状态 → 已实现
- Modify: `player_mega_test.go` 中错误信息字符串（若仍写旧方法名）

**Interfaces:**
- 消费新生成 API 名（见 Global 对照表）

- [ ] **Step 1: 全文检索运行时旧名**

```bash
rg -n 'GetHeroForWrite|GetSkillForWrite|GetMailForWrite|GetQuestForWrite|GetStackableItemForWrite' \
  --glob '*.go' --glob '*.md'
```

对**会编译的** `.go`（不含归档 docs/superpowers/plans|历史 specs 除非破坏阅读契约）按对照表替换。

至少：

| 文件 | 改动 |
|------|------|
| `bench_mega_writes.go` | `GetHerosForWrite` / `GetMailsForWrite` / `GetQuestsForWrite` |
| `examples/gamestore/handler.go` | `GetHerosForWrite` |
| `player_mega_test.go` | Fatal 文案中的方法名（若适用） |

- [ ] **Step 2: regenerate**

```bash
go generate .
(cd examples/gamestore && go generate .)
(cd examples/gamestore-migrate/after && go generate .)
```

确认生成物含 `GetHerosForWrite` / `GetSkillsForWrite`，**无** `func (p *Player) GetHeroForWrite`。

- [ ] **Step 3: 更新 proxy-api.md**

将类似表述：

> `map[K]*Struct` 在 map 元素类型上生成，例如 `GetHeroForWrite`

改为：

> `map[K]*Struct` 按**字段名**生成 `Get{Field}ForWrite`，例如 `player.GetHerosForWrite(ctx, heroID)`，再 `h.GetSkillsForWrite(ctx, skillID)`。  
> **破坏性变更：** 旧 `GetHeroForWrite`（字段 `Heros`）已更名为 `GetHerosForWrite`；同类字段不再用元素类型单数名。

- [ ] **Step 4: 全量验收**

```bash
go test ./cmd/undoproxy-gen ./internal/cowgen ./internal/cowproxy ./cmd/undorewrite -count=1
go test . -count=1   # 根包（含 bench 相关测试若默认跑）
(cd examples/gamestore && go test ./... -count=1)
(cd examples/gamestore-migrate/after && go test ./... -count=1)
```

期望：全绿。

- [ ] **Step 5: 标记 design 已实现；更新 docs/README**

- [ ] **Step 6: 准备提交（等用户确认）**

建议说明：

```text
fix(undoproxy-gen): name map GetForWrite after field, not elem type

Avoid duplicate GetStackableItemForWrite when Bag has both Stackable
and VirtualStackable maps of *StackableItem.
```

---

## Spec 覆盖自检

| Spec 要求 | 任务 |
|-----------|------|
| `MapKeyGetForWriteName(field)` | Task 1–2 |
| emit + catalog 接线 | Task 2 |
| 双 map 撞名夹具 | Task 2 |
| 硬切调用点 / regenerate / guide | Task 3 |
| 无旧名别名 | 全任务 |
| ElemAt follow-up 不做 | 未列入 |
| 宿主非 Must | 未列入硬验收 |

## Placeholder 扫描

无 TBD；对照表与验收命令钉死。
