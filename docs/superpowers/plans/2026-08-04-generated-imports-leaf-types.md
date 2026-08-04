# 生成物自动 import 跨包叶子类型 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 生成 `zz_generated.undo_proxy.go` 时自动 import 跨包叶子类型（枚举/具名标量等），消除 `undefined: cfg`；不改变同包 struct 图边界。

**Architecture:** `cowgen.Qualifiers`（path→唯一别名）挂在 `Graph` 上；`TypeStr` 经其取选择子；`emitFromGraph` 从同一表写 `import`（含 `sync`）。冲突时 `Name()+N`，与类型字符串一致。

**Tech Stack:** Go 1.25、`go/types`、`internal/cowgen`、`cmd/undoproxy-gen`

**Spec:** [../specs/2026-08-04-generated-imports-leaf-types-design.md](../specs/2026-08-04-generated-imports-leaf-types-design.md)

## Global Constraints

- Go **1.25**；中文注释；TDD；单文件 ≤500 / 单函数 ≤50。
- **未经用户明确同意不得 `git commit` / `git push`**。
- 禁止 `.worktrees/`。
- 不扩展跨包 struct 入图；宿主冒烟非 Must。
- examples 按嵌套 module 分别 generate/test。

---

## 文件结构（目标态）

| 文件 | 职责 |
|------|------|
| `internal/cowgen/qualifiers.go` | `Qualifiers`、`Alias` |
| `internal/cowgen/qualifiers_test.go` | 冲突别名单测 |
| `internal/cowgen/graph.go` | `Graph.Qualifiers`；`TypeStr(..., q)`；`BuildGraph` 初始化 q |
| `internal/cowgen/classify.go` | `classifyField(..., q)` 全链路传 q |
| `internal/cowgen/kind.go` | 若 `Graph` 定义在此则加字段（按现状） |
| `cmd/undoproxy-gen/emit_structured_graph.go` | `writeImports(b, g)` |
| `cmd/undoproxy-gen/testdata/leafpkg/enum.go` | 外叶子夹具 |
| `cmd/undoproxy-gen/testdata/types.go` | 引用 leafpkg |
| `cmd/undoproxy-gen/emit_imports_test.go` | 生成物含 import |
| guide / design 状态 / README | 文档 |

---

### Task 1: Qualifiers + TypeStr(q)

**Files:**
- Create: `internal/cowgen/qualifiers.go`
- Create: `internal/cowgen/qualifiers_test.go`
- Modify: `internal/cowgen/graph.go`（TypeStr 签名、Graph、BuildGraph）
- Modify: `internal/cowgen/classify.go`（传 q）
- Modify: `BasicTypeStr` 若仍调 TypeStr：传 `nil` q 或增加 q 参数（仅回退路径）

**Interfaces:**
- Produces:
```go
type Qualifiers struct { /* byPath, usedAlias */ }
func NewQualifiers() *Qualifiers
func (q *Qualifiers) Alias(p *types.Package) string
func (q *Qualifiers) Imports() []struct{ Alias, Path string } // 稳定按 Path 排序
func TypeStr(pkg *types.Package, t types.Type, q *Qualifiers) string
```
- `type Graph struct { Structs []*StructPlan; Qualifiers *Qualifiers }`

- [ ] **Step 1: Qualifiers 单测（RED）**

```go
func TestQualifiers_ConflictGetsNumericSuffix(t *testing.T) {
	q := cowgen.NewQualifiers()
	// 用 types.NewPackage(path, name) 构造两包 Name 皆 "cfg"
	p1 := types.NewPackage("example.com/a/cfg", "cfg")
	p2 := types.NewPackage("example.com/b/cfg", "cfg")
	if q.Alias(p1) != "cfg" { t.Fatal(...) }
	if q.Alias(p2) != "cfg2" { t.Fatal(...) }
	if q.Alias(p1) != "cfg" { t.Fatal("stable") }
}
```

- [ ] **Step 2: RED**

```bash
go test ./internal/cowgen/ -run TestQualifiers_ConflictGetsNumericSuffix -count=1
```

- [ ] **Step 3: 实现 Qualifiers + 改 TypeStr/BuildGraph/classify**

1. `BuildGraph`: `q := NewQualifiers(); g.Qualifiers = q`；`classifyField(f.Type(), pkg.Pkg, q)`。
2. `classifyField` / `classifyType` / 辅助函数签名增加 `q *Qualifiers`；所有 `TypeStr(pkg,t)` → `TypeStr(pkg,t,q)`。
3. `TypeStr`：跨包 `return q.Alias(p)`；`q==nil` 时回退 `p.Name()`（便于孤立调用）。
4. `BasicTypeStr`：其内部 `TypeStr(pkg,t)` → `TypeStr(pkg,t,nil)` 或同样接收 q（若仍被 classify 使用则传 q）。

- [ ] **Step 4: 全量 cowgen 测试**

```bash
go test ./internal/cowgen/ ./internal/cowproxy/ ./cmd/undoproxy-gen/ -count=1
```

期望：PASS（尚无跨包 import 发射，但签名变更不得破坏）。

- [ ] **Step 5: 准备提交（等确认）**

建议说明：`feat(cowgen): Qualifiers for stable cross-package TypeStr aliases`

---

### Task 2: writeImports + leafpkg 夹具

**Files:**
- Modify: `cmd/undoproxy-gen/emit_structured_graph.go`
- Create: `cmd/undoproxy-gen/testdata/leafpkg/enum.go`
- Modify: `cmd/undoproxy-gen/testdata/types.go`
- Create: `cmd/undoproxy-gen/emit_imports_test.go`

**Interfaces:**
- Consumes: `g.Qualifiers.Imports()`
- Produces: 文件头 import 块

leafpkg：

```go
package leafpkg

type Enum int32

const EnumA Enum = 1
```

types.go 增加（可挂 TypedSlotsRoot 或新根）：

```go
// +cow:undoproxy-gen=true
type ImportLeafRoot struct {
	ByEnum map[leafpkg.Enum]int64
	Tag    leafpkg.Enum
}
```

（须 `import "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata/leafpkg"`。）

- [ ] **Step 1: 集成测（RED）**

```go
func TestEmit_ImportsCrossPackageLeaf(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	if err := Run(out, "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	s := string(b)
	if !strings.Contains(s, "testdata/leafpkg") {
		t.Fatal("missing leafpkg import path")
	}
	if !strings.Contains(s, "leafpkg.") {
		t.Fatal("missing leafpkg selector in generated types")
	}
	if !strings.Contains(s, `"sync"`) {
		t.Fatal("missing sync")
	}
}

func TestEmit_NoExtraImportsWithoutLeaf(t *testing.T) {
	// Run on github.com/huangyuCN/cow（或无跨包叶子的包）
	// 断言 import 块不含 leafpkg；允许仅 sync
}
```

- [ ] **Step 2: RED**

```bash
go test ./cmd/undoproxy-gen/ -run 'TestEmit_ImportsCrossPackageLeaf' -count=1
```

期望：FAIL（有 leafpkg. 选择子或 undo 槽但无 import path — 若尚未改 TypeStr 登记，选择子也可能缺失）。

- [ ] **Step 3: writeImports**

```go
func writeImports(b *bytes.Buffer, q *cowgen.Qualifiers) {
	b.WriteString("import (\n\t\"sync\"\n")
	if q != nil {
		imps := q.Imports()
		if len(imps) > 0 {
			b.WriteByte('\n')
			for _, im := range imps {
				fmt.Fprintf(b, "\t%s %q\n", im.Alias, im.Path)
			}
		}
	}
	b.WriteString(")\n\n")
}
```

`emitFromGraph`：`writeImports(&b, g.Qualifiers)` 替换硬编码行。

- [ ] **Step 4: GREEN**

```bash
go test ./cmd/undoproxy-gen/ -count=1
```

- [ ] **Step 5: 准备提交（等确认）**

建议说明：`fix(undoproxy-gen): emit imports for cross-package leaf types`

---

### Task 3: 文档 + 回归 regen + 收口

**Files:**
- Modify: `docs/guide/codegen-undoproxy.md`
- Modify: design 状态 → 已实现；`docs/README.md`
- Regenerate: 根包 / examples（import 块可能变为多行 `import ( "sync" )`，gofmt 可能仍简写单 import）

- [x] **Step 1: guide**

边界节追加：生成物会为类型图中出现的跨包叶子类型（map key/标量等）自动 `import`；跨包 struct 仍不入图。

- [x] **Step 2: regenerate + 验收**

```bash
go generate .
(cd examples/gamestore && go generate . && go test ./... -count=1)
(cd examples/gamestore-migrate/after && go generate . && go test ./... -count=1)
go test ./cmd/undoproxy-gen ./internal/... . -count=1
```

- [x] **Step 3: design / README 标已实现**

- [ ] **Step 4: 准备提交（等确认）**

```text
fix(undoproxy-gen): emit imports for cross-package leaf types

undoOp slots may reference cfg/luban enums used as map keys or
scalars; generated files must import those packages.
```

（可将 Task 2–3 合并为一次提交。）

---

## Spec 覆盖自检

| Spec 要求 | 任务 |
|-----------|------|
| Qualifiers + 冲突 Name()+N | Task 1 |
| TypeStr 共用 q | Task 1 |
| writeImports + sync | Task 2 |
| leafpkg 夹具 | Task 2 |
| 无跨包不多余 import | Task 2 |
| 同包原则不变 | 无 classify 边界改动 |
| guide + 状态 | Task 3 |
| 宿主非 Must | 未列入硬门禁 |

## Placeholder 扫描

无 TBD。`Graph` 字段落点以实现文件为准（`graph.go` 或 `kind.go`）。
