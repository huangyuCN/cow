# undoproxy-gen 大规模类型图可编译性 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复大类型图生成物两类编译失败：`undoKind` 升为 `uint16`（超限报错），内层 map 浅拷贝 helper 按类型签名全局去重。

**Architecture:** 在 `undoBuilder`（单次生成趟状态）上挂 `cloneHelpers` 登记表与 kind 上限校验；`emitCloneMapShallow` 改为 `cloneMapShallow_{KeySan}_{ElemSan}`；`writeRuntime` 输出 `uint16` 并在 `len(entries) > math.MaxUint16` 时失败。开发直接在当前工作区进行（**禁止** `.worktrees/`）。

**Tech Stack:** Go 1.25、`cmd/undoproxy-gen`、`internal/cowgen`、`go test` / `go generate`

**Spec:** [../specs/2026-08-04-undoproxy-gen-scalable-codegen-design.md](../specs/2026-08-04-undoproxy-gen-scalable-codegen-design.md)

## Global Constraints

- Go **1.25**；禁止 deprecated API。
- TDD：每个功能任务先红测再实现。
- 中文注释；导出名不以包名开头。
- 单文件 ≤500 行、单函数 ≤50 行；逼近则拆分。
- **未经用户明确同意不得 `git commit` / `git push`**；计划中的 Commit 步骤改为「准备提交说明并等待确认」后再执行。
- 验收 Must 仅绑定 cow：`go test ./cmd/undoproxy-gen ./internal/...`、`go generate ./examples/...`、`go test ./examples/...`。

---

## 文件结构（目标态）

| 文件 | 职责 |
|------|------|
| `cmd/undoproxy-gen/emit_helpers.go` | 新增 `sanitizeIdent`、`cloneMapShallowFuncName` |
| `cmd/undoproxy-gen/emit_helpers_test.go` | sanitize / 命名单测 |
| `cmd/undoproxy-gen/emit_undo.go` | `undoBuilder.cloneHelpers`；`writeRuntime` → `uint16` + 上限检查（返回 `error`） |
| `cmd/undoproxy-gen/emit_undo_test.go` | ≥256 kind、超限报错单测 |
| `cmd/undoproxy-gen/emit_structured.go` | 调用点与 `emitCloneMapShallow` 去重 |
| `cmd/undoproxy-gen/emit_structured_graph.go` | `writeRuntime` 错误上抛 |
| `cmd/undoproxy-gen/testdata/types.go` | 双根同名 `Conditions`（MapMap）供去重集成测 |
| `cmd/undoproxy-gen/generate_golden_test.go` / `main_test.go` | 断言 `uint16`；去重计数 |
| `examples/*/zz_generated.undo_proxy.go` | regenerate |
| `docs/guide/codegen-undoproxy.md` | 位宽与 helper 去重一行说明 |
| `zz_generated.undo_proxy.go`（根包，若 `go generate` 触及） | regenerate |

---

### Task 1: sanitize 与 helper 命名（纯函数）

**Files:**
- Modify: `cmd/undoproxy-gen/emit_helpers.go`
- Modify: `cmd/undoproxy-gen/emit_helpers_test.go`

**Interfaces:**
- Produces: `sanitizeIdent(s string) string`；`cloneMapShallowFuncName(keyType, elemType string) string` → `"cloneMapShallow_"+sanitizeIdent(keyType)+"_"+sanitizeIdent(elemType)`

- [ ] **Step 1: 写失败测试**

在 `emit_helpers_test.go` 追加：

```go
func TestSanitizeIdent(t *testing.T) {
	cases := []struct{ in, want string }{
		{"string", "string"},
		{"*Condition", "_Condition"},
		{"", "T"},
		{"[]byte", "__byte"},
		{"1bad", "T1bad"},
	}
	for _, tc := range cases {
		if got := sanitizeIdent(tc.in); got != tc.want {
			t.Fatalf("sanitizeIdent(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCloneMapShallowFuncName(t *testing.T) {
	got := cloneMapShallowFuncName("string", "*Condition")
	want := "cloneMapShallow_string__Condition"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./cmd/undoproxy-gen/ -run 'TestSanitizeIdent|TestCloneMapShallowFuncName' -count=1
```

期望：FAIL（函数未定义）。

- [ ] **Step 3: 实现**

在 `emit_helpers.go` 追加：

```go
func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || (out[0] >= '0' && out[0] <= '9') {
		return "T" + out
	}
	return out
}

func cloneMapShallowFuncName(keyType, elemType string) string {
	return "cloneMapShallow_" + sanitizeIdent(keyType) + "_" + sanitizeIdent(elemType)
}
```

（`emit_helpers.go` 已 import `strings`。）

- [ ] **Step 4: 运行确认通过**

```bash
go test ./cmd/undoproxy-gen/ -run 'TestSanitizeIdent|TestCloneMapShallowFuncName' -count=1
```

期望：PASS。

- [ ] **Step 5: 准备提交（等用户确认）**

建议说明：`test(undoproxy-gen): add sanitizeIdent and clone helper naming`

---

### Task 2: clone helper 按签名去重

**Files:**
- Modify: `cmd/undoproxy-gen/emit_undo.go`（`undoBuilder` 增加 `cloneHelpers map[string]struct{}`；`newUndoBuilder` 初始化）
- Modify: `cmd/undoproxy-gen/emit_structured.go`（`emitCloneMapShallow` 签名改为接收 `ub`；调用点改名）
- Modify: `cmd/undoproxy-gen/testdata/types.go`
- Modify: `cmd/undoproxy-gen/generate_golden_test.go`（或新建 `emit_clone_dedupe_test.go`）

**Interfaces:**
- Consumes: `cloneMapShallowFuncName`
- Produces: `emitCloneMapShallow(b *bytes.Buffer, ub *undoBuilder, plan cowgen.FieldPlan)` — 若 `ub.cloneHelpers[name]` 已存在则只保证调用点可用、不再写定义

- [ ] **Step 1: 扩展 testdata（双根同名 Conditions）**

在 `testdata/types.go` 追加：

```go
type Condition struct {
	Val int64
}

// +cow:undoproxy-gen=true
type Alpha struct {
	Conditions map[int32]map[string]*Condition
}

// +cow:undoproxy-gen=true
type Beta struct {
	Conditions map[int32]map[string]*Condition
}
```

- [ ] **Step 2: 写失败的去重集成测试**

新建 `cmd/undoproxy-gen/emit_clone_dedupe_test.go`：

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmit_DedupesCloneMapShallowBySignature(t *testing.T) {
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
	name := cloneMapShallowFuncName("string", "*Condition")
	def := "func " + name + "("
	if c := strings.Count(s, def); c != 1 {
		t.Fatalf("helper defs=%d want 1 for %s", c, name)
	}
	if strings.Count(s, name+"(") < 3 {
		// 1 次定义 + ≥2 次调用（Alpha/Beta Get*MapForWrite）
		t.Fatalf("expected definition + multiple call sites for %s", name)
	}
	if strings.Contains(s, "cloneConditionsMapShallow") {
		t.Fatal("old field-based helper name must not appear")
	}
}
```

- [ ] **Step 3: 运行确认失败**

```bash
go test ./cmd/undoproxy-gen/ -run TestEmit_DedupesCloneMapShallowBySignature -count=1
```

期望：FAIL（defs=2 或仍见旧名）。

- [ ] **Step 4: 实现去重**

1. `undoBuilder` 增加字段并在 `newUndoBuilder` 中：`cloneHelpers: make(map[string]struct{})`。
2. 改写 `emitCloneMapShallow`：

```go
func emitCloneMapShallow(b *bytes.Buffer, ub *undoBuilder, plan cowgen.FieldPlan) {
	keyType := plan.Keys[1].KeyType
	elemType := innerValueType(plan)
	name := cloneMapShallowFuncName(keyType, elemType)
	if _, ok := ub.cloneHelpers[name]; ok {
		return
	}
	ub.cloneHelpers[name] = struct{}{}
	fmt.Fprintf(b, "func %s(m map[%s]%s) map[%s]%s {\n",
		name, keyType, elemType, keyType, elemType)
	b.WriteString("\tif m == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tc := make(map[%s]%s, len(m))\n", keyType, elemType)
	b.WriteString("\tfor k, v := range m {\n\t\tc[k] = v\n\t}\n")
	b.WriteString("\treturn c\n}\n\n")
}
```

3. 在 `emitStructuredMapMapGetForWrite` 中：

```go
name := cloneMapShallowFuncName(plan.Keys[1].KeyType, innerValueType(plan))
fmt.Fprintf(b, "\tdirty := %s(oldInner)\n", name)
// ...
emitCloneMapShallow(b, ub, plan)
```

（若有其它调用 `emitCloneMapShallow` 的站点，一并改传 `ub`。）

- [ ] **Step 5: 运行确认通过**

```bash
go test ./cmd/undoproxy-gen/ -run TestEmit_DedupesCloneMapShallowBySignature -count=1
go test ./cmd/undoproxy-gen/ -count=1
```

期望：PASS（后者允许因 `uint8` 断言暂红，见 Task 3；若仅去重相关绿即可先继续）。

- [ ] **Step 6: 准备提交（等用户确认）**

建议说明：`fix(undoproxy-gen): dedupe cloneMapShallow helpers by map signature`

---

### Task 3: `undoKind` → `uint16` + 超限报错

**Files:**
- Modify: `cmd/undoproxy-gen/emit_undo.go`
- Modify: `cmd/undoproxy-gen/emit_structured_graph.go`
- Modify: `cmd/undoproxy-gen/main_test.go`
- Modify: `cmd/undoproxy-gen/emit_undo_test.go`（新建或扩展）

**Interfaces:**
- Produces: `func checkUndoKindCount(n int) error`；`func (ub *undoBuilder) writeRuntime(b *bytes.Buffer) error`

- [ ] **Step 1: 写失败测试**

在 `emit_undo_test.go`：

```go
func TestCheckUndoKindCount_Overflow(t *testing.T) {
	if err := checkUndoKindCount(math.MaxUint16); err != nil {
		t.Fatalf("MaxUint16 kinds should be ok: %v", err)
	}
	if err := checkUndoKindCount(math.MaxUint16 + 1); err == nil {
		t.Fatal("want error when kind count exceeds uint16")
	}
}

func TestWriteRuntime_UsesUint16AndAllows256Kinds(t *testing.T) {
	ub := &undoBuilder{
		structs:   []string{"W"},
		kindIndex: make(map[string]string),
		cloneHelpers: make(map[string]struct{}),
		sliceSnaps: make(map[string]string),
		scalarOlds: map[string]string{"int": "oldInt"},
	}
	for i := 0; i < 256; i++ {
		ub.entries = append(ub.entries, undoEntry{name: fmt.Sprintf("undoKindW%d", i), body: "break"})
	}
	var buf bytes.Buffer
	if err := ub.writeRuntime(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "type undoKind uint16") {
		t.Fatal("want uint16")
	}
	if strings.Contains(s, "type undoKind uint8") {
		t.Fatal("uint8 must not appear")
	}
	if _, err := format.Source(buf.Bytes()); err != nil {
		t.Fatalf("format: %v", err)
	}
}
```

将 `main_test.go` 中 `"type undoKind uint8"` 改为 `"type undoKind uint16"`。

- [ ] **Step 2: 运行确认失败**

```bash
go test ./cmd/undoproxy-gen/ -run 'TestCheckUndoKindCount_Overflow|TestWriteRuntime_UsesUint16|TestRun_GeneratesStructuredUndoProxy' -count=1
```

期望：FAIL（仍为 uint8 / 无 check 函数）。

- [ ] **Step 3: 实现**

```go
func checkUndoKindCount(n int) error {
	if n > math.MaxUint16 {
		return fmt.Errorf("undoKind count %d exceeds uint16 max %d", n, math.MaxUint16)
	}
	return nil
}

func (ub *undoBuilder) writeRuntime(b *bytes.Buffer) error {
	if err := checkUndoKindCount(len(ub.entries)); err != nil {
		return err
	}
	b.WriteString("type undoKind uint16\n\nconst (\n")
	// ... 其余与现逻辑相同 ...
	return nil
}
```

`emitFromGraph`：

```go
if err := ub.writeRuntime(&b); err != nil {
	return err
}
```

注意：`emit_undo.go` 增加 `math` import；原 `writeRuntime` 末尾若无 return，补齐 `return nil`。函数体若超 50 行则拆分写 const / 写 undoOp 的私有方法。

- [ ] **Step 4: 运行确认通过**

```bash
go test ./cmd/undoproxy-gen/ -count=1
```

期望：PASS。

- [ ] **Step 5: 准备提交（等用户确认）**

建议说明：`fix(undoproxy-gen): widen undoKind to uint16 with overflow check`

---

### Task 4: 再生成 examples / 根包 + 文档

**Files:**
- Modify: `docs/guide/codegen-undoproxy.md`
- Regenerate: `examples/**/zz_generated.undo_proxy.go`、（若适用）根包 `zz_generated.undo_proxy.go`
- Modify: golden 测试中其它硬编码 `uint8`（`rg 'undoKind uint8'`）

- [ ] **Step 1: 全文检索旧断言**

```bash
rg -n 'undoKind uint8|clone\w+MapShallow' --glob '*.go' --glob '*.md'
```

将测试/文档中的过时假设改掉。

- [ ] **Step 2: 更新指南**

在 `docs/guide/codegen-undoproxy.md`「边界」节追加要点：

- 生成运行时 `undoKind` 为 `uint16`；超过 65535 种操作时生成失败。
- 内层 map 浅拷贝 helper 名为 `cloneMapShallow_{Key}_{Elem}`，按类型签名在单文件内只定义一次。

- [ ] **Step 3: regenerate**

```bash
go generate ./examples/...
# 若仓库根有 go:generate 指向 undoproxy-gen：
go generate .
go test ./cmd/undoproxy-gen ./internal/... ./examples/... -count=1
```

期望：全绿；生成物含 `type undoKind uint16`，无 `type undoKind uint8`。

- [ ] **Step 4: Spec 交叉勾选**

对照 [design §7.2](../specs/2026-08-04-undoproxy-gen-scalable-codegen-design.md)：去重测、≥256 kind、超限测、文档均已覆盖。

- [ ] **Step 5: 准备提交（等用户确认）**

建议说明：

```text
fix(undoproxy-gen): use uint16 undoKind and dedupe map clone helpers

Large graphs exceed 255 undo ops; emit clone helpers once per map
signature instead of per field emit site.
```

（可将 Task 2–4 合并为一条提交，或分条；以用户当时确认为准。）

---

## Spec 覆盖自检

| Spec 要求 | 任务 |
|-----------|------|
| `undoKind uint16` | Task 3 |
| kind >65535 生成失败 | Task 3 `checkUndoKindCount` |
| 方案 A 命名 + 全局去重 | Task 1–2 |
| 包级 helper 审计约定（设计文档已写；实现仅改 clone） | Task 2 + 不引入新裸字段名 helper |
| 双 struct Conditions 定义恰 1 次 | Task 2 |
| ≥256 kind 夹具 | Task 3 `TestWriteRuntime_UsesUint16AndAllows256Kinds` |
| examples regenerate + guide | Task 4 |
| 宿主非 Must | 未列入验收命令 |

## Placeholder 扫描

无 TBD/TODO；命名与函数签名在各 Task Interfaces 钉死。
