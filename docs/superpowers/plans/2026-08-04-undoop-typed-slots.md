# undoOp 槽位类型泛化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 使 `undoOp` 的 map key、内层 map 快照、标量旧值槽位与字段声明的精确 Go 类型一致，消除宿主大图三类赋值编译错误。

**Architecture:** `cowgen` 用 `TypeStr` 保留 key/标量声明类型；`undoBuilder` 按出现类型登记 `key_*` / `inner_*` 槽位（sanitize 命名）；结构化 emit 全路径改动态槽名；硬切删除 `keyI32` 五件套与 `innerMapOld map[string]int64`。

**Tech Stack:** Go 1.25、`go/types`、`internal/cowgen`、`cmd/undoproxy-gen`

**Spec:** [../specs/2026-08-04-undoop-typed-slots-design.md](../specs/2026-08-04-undoop-typed-slots-design.md)

## Global Constraints

- Go **1.25**；中文注释；TDD；单文件 ≤500 / 单函数 ≤50。
- **未经用户明确同意不得 `git commit` / `git push`**。
- 禁止 `.worktrees/`。
- 硬切字段名；examples/根包须 regenerate。
- 宿主冒烟非 Must；验收勿用仓库根 `./examples/...`（嵌套 module）。
- 完成后 `rg 'keyI32|innerMapOld|noteInnerMapSnap'` 在 `cmd/undoproxy-gen/*.go`（非测试字符串）应为零或仅历史注释。

---

## 文件结构（目标态）

| 文件 | 职责 |
|------|------|
| `internal/cowgen/classify.go` | 外层 key / 具名 LeafType → `TypeStr` |
| `internal/cowgen/classify_test.go` | 具名 key、具名标量断言 |
| `cmd/undoproxy-gen/testdata/types.go` | `map[int]map[string]*Node`、具名 `Cat`、双内层类型夹具 |
| `cmd/undoproxy-gen/emit_undo.go` | `keySlots`/`innerMapSlots`；动态 `writeUndoOpStruct` |
| `cmd/undoproxy-gen/emit_helpers.go` | 删除或瘦身 `mapKeyField`；sanitize 已有 |
| `cmd/undoproxy-gen/emit_structured.go` | 全路径动态 key/inner |
| `cmd/undoproxy-gen/emit_structured_write_ext.go` | Remove 等路径动态 key |
| `cmd/undoproxy-gen/emit_undo_slots_test.go` | 槽位集成测 |
| 生成物 / guide / design 状态 | regen + 文档 |

---

### Task 1: classify 精确类型（cowgen）

**Files:**
- Modify: `internal/cowgen/classify.go`
- Modify: `internal/cowgen/classify_test.go`
- Modify: `cmd/undoproxy-gen/testdata/types.go`（为本测提供类型）

**Interfaces:**
- Produces: 外层 `KeyLayer.KeyType == TypeStr(key)`；具名 basic 标量 `LeafType ==` 声明名（如 `Cat`），不再是 `int32`

- [ ] **Step 1: 扩展 testdata**

```go
type Cat int32

type Node struct {
	V int64
}

// +cow:undoproxy-gen=true
type TypedSlotsRoot struct {
	ByInt      map[int]map[string]*Node
	Score      Cat
	InnerA     map[int32]map[string]int64
	InnerB     map[int32]map[string]*Node
}
```

（`InnerA`/`InnerB` 供后续 Task 槽位测；本 Task 可用 BuildGraph 断言 KeyType/LeafType。）

- [ ] **Step 2: 写失败测试**

```go
func TestBuildGraph_exactKeyAndNamedScalar(t *testing.T) {
	pkg, err := cowmon.LoadPackage("github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata")
	// ...
	// TypedSlotsRoot.ByInt → Keys[0].KeyType == "int", Keys[1].KeyType == "string"
	// TypedSlotsRoot.Score → KindScalar, LeafType == "Cat"
}
```

- [ ] **Step 3: 运行确认失败**

```bash
go test ./internal/cowgen/ -run TestBuildGraph_exactKeyAndNamedScalar -count=1
```

期望：FAIL（KeyType 为 `int32` 或 LeafType 为 `int32`）。

- [ ] **Step 4: 改 classify**

1. `case *types.Map:` 中 `keyT := BasicTypeStr(...)` → `TypeStr(pkg, u.Key())`。
2. `case *types.Named:` 底层为 Basic 时：`plan.LeafType = TypeStr(pkg, t)`（含 len(keys)==0 的 KindScalar 与 MapScalar 分支），**不要** `BasicTypeStr`。

注意：同包 `Cat` 的 TypeStr 应为 `"Cat"`；跨包枚举带选择子（宿主场景）。

- [ ] **Step 5: 测试通过**

```bash
go test ./internal/cowgen/ -count=1
```

- [ ] **Step 6: 准备提交（等确认）**

建议说明：`fix(cowgen): keep declared TypeStr for map keys and named scalars`

---

### Task 2: undoBuilder 动态 key/inner 槽

**Files:**
- Modify: `cmd/undoproxy-gen/emit_undo.go`
- Modify: `cmd/undoproxy-gen/emit_helpers.go`（删除 `mapKeyField` 或改为禁止调用）
- Create/Modify: `cmd/undoproxy-gen/emit_undo_slots_test.go`（单元级登记与 writeRuntime）

**Interfaces:**
- Produces:
  - `func (ub *undoBuilder) keySlot(goType string) string`
  - `func (ub *undoBuilder) innerMapSlot(mapType string) string`
  - `writeUndoOpStruct` 仅发射已登记 key/inner/scalar/slice 字段

命名：

```go
func keySlotName(goType string) string {
	return "key_" + sanitizeIdent(goType)
}
func innerMapSlotName(mapType string) string {
	return "inner_" + sanitizeIdent(mapType)
}
```

冲突时数字后缀（抄 `scalarOldField`）。

- [ ] **Step 1: 写失败测试**

```go
func TestWriteRuntime_EmitsTypedKeyAndInnerSlots(t *testing.T) {
	ub := &undoBuilder{ /* 初始化 maps */ }
	ub.keySlot("int")
	ub.keySlot("string")
	ub.innerMapSlot("map[string]*Node")
	// entries 至少 1 个假 kind 以便写 const
	var buf bytes.Buffer
	if err := ub.writeRuntime(&buf); err != nil { t.Fatal(err) }
	s := buf.String()
	for _, need := range []string{
		"key_int int",
		"key_string string",
		"inner_map_string__Node map[string]*Node",
	} {
		if !strings.Contains(s, need) { t.Fatalf("missing %q", need) }
	}
	if strings.Contains(s, "keyI32") || strings.Contains(s, "innerMapOld") {
		t.Fatal("legacy slots must not appear")
	}
}
```

（sanitize 结果以本地 `sanitizeIdent("map[string]*Node")` 为准；写测前先印一次实际名，或断言 `ub.innerMapSlot(...)` 返回名出现在 struct 中。）

- [ ] **Step 2: RED**

```bash
go test ./cmd/undoproxy-gen/ -run TestWriteRuntime_EmitsTypedKeyAndInnerSlots -count=1
```

- [ ] **Step 3: 实现登记 + writeUndoOpStruct**

- 结构体增加 `keySlots`、`innerMapSlots map[string]string`；`newUndoBuilder` 初始化。
- 删除 `innerMapSnap` / `noteInnerMapSnap`。
- `writeUndoOpStruct`：删除五件套硬编码；按排序后的 `keySlots`/`innerMapSlots` 输出字段。
- 保留 `had`/`had2`、recv 指针、scalarOlds、sliceSnaps。

- [ ] **Step 4: GREEN + 更新依赖 noteInnerMapSnap 的编译**

若全包暂不编译因 emit 仍调 `noteInnerMapSnap`/`mapKeyField`，本 Task 可在 emit 侧做**临时兼容转发**：

```go
func (ub *undoBuilder) noteInnerMapSnap() { /* deprecated no-op — Task 3 删除调用 */ }
```

更干净做法：本 Task 只改 undo 写结构 + API；Task 3 立刻改调用方。若 `go test ./cmd/undoproxy-gen` 因未接线失败，以本测 + `emit_undo` 单测绿为准，并在 report 注明。

- [ ] **Step 5: 准备提交（等确认）**

建议说明：`feat(undoproxy-gen): dynamic typed key and inner-map undoOp slots`

---

### Task 3: 结构化 emit 全路径改动态槽

**Files:**
- Modify: `cmd/undoproxy-gen/emit_structured.go`
- Modify: `cmd/undoproxy-gen/emit_structured_write_ext.go`
- Modify: `cmd/undoproxy-gen/emit_helpers.go`（移除 `mapKeyField`）
- Create/extend: `cmd/undoproxy-gen/emit_undo_slots_test.go`（集成 Run 断言）

**Interfaces:**
- Consumes: `ub.keySlot(keyType)`、`ub.innerMapSlot(mapTypeString)`
- 所有 rollback 字符串与 `push(undoOp{...})` 使用返回的字段名

- [ ] **Step 1: 集成红测**

```go
func TestEmit_TypedSlotsForIntKeyAndInnerMaps(t *testing.T) {
	// Run(testdata) → 读生成源码
	// 含 key_int；含两个不同 inner_*（InnerA/InnerB）
	// 不含 innerMapOld map[string]int64
	// 对 Score/Cat：含类型 Cat 的 old 槽字段
}
```

- [ ] **Step 2: RED**

```bash
go test ./cmd/undoproxy-gen/ -run TestEmit_TypedSlotsForIntKeyAndInnerMaps -count=1
```

- [ ] **Step 3: 系统性替换**

策略：

1. `rg -n 'keyI32|keyString|innerMapOld|mapKeyField|noteInnerMapSnap' cmd/undoproxy-gen --glob '*.go'`
2. 单层 map：`k1` 槽 = `ub.keySlot(plan.Keys[0].KeyType)`；双层：`k1`/`k2` 分别登记两层 KeyType。
3. MapMap GetForWrite：`innerTy := "map["+Keys[1].KeyType+"]"+innerValueType(plan)`（或与 `plan.MapValue` 一致的 TypeStr），`slot := ub.innerMapSlot(innerTy)`；rollback 用 `op.<slot>`。
4. 删除对 `noteInnerMapSnap` 的调用。
5. `oldInt` 仍用于 slice 下标/长度（类型 `int`）——继续 `scalarOlds["int"]`；**不要**把 map key `int` 与 slice index 混用同一语义字段以外的特殊逻辑（二者都可以是 `key_int` vs `oldInt`：key 用 keySlot，index 用 oldInt）。

辅助建议：

```go
func (ub *undoBuilder) keySlotsFor(plan cowgen.FieldPlan) []string {
	out := make([]string, len(plan.Keys))
	for i, k := range plan.Keys {
		out[i] = ub.keySlot(k.KeyType)
	}
	return out
}
```

注意函数 ≤50 行：大 emit 函数可只替换字面量，暂不全量拆文件，除非触线上限。

- [ ] **Step 4: 全包 undoproxy-gen 测试**

```bash
go test ./cmd/undoproxy-gen/ -count=1
```

期望：PASS。

- [ ] **Step 5: 准备提交（等确认）**

建议说明：`fix(undoproxy-gen): wire emit paths to typed undoOp slots`

---

### Task 4: regenerate + 文档 + 收口

**Files:**
- Regenerate: `zz_generated.undo_proxy.go`、examples 生成物
- Modify: `docs/guide/codegen-undoproxy.md`
- Modify: `docs/README.md` / design 状态 → 已实现
- 修复因字段名变更而失败的根包/bench 测试（若有）

- [ ] **Step 1: rg 清扫**

```bash
rg -n 'keyI32|innerMapOld|noteInnerMapSnap|func mapKeyField' cmd/undoproxy-gen --glob '*.go'
```

允许仅出现在 `_test.go` 负向断言。

- [ ] **Step 2: regenerate**

```bash
go generate .
(cd examples/gamestore && go generate .)
(cd examples/gamestore-migrate/after && go generate .)
```

- [ ] **Step 3: 验收**

```bash
go test ./cmd/undoproxy-gen ./internal/... -count=1
go test . -count=1
(cd examples/gamestore && go test ./... -count=1)
(cd examples/gamestore-migrate/after && go test ./... -count=1)
```

- [ ] **Step 4: guide**

在边界节追加：`undoOp` 的 map key / 内层 map / 标量旧值槽按类型图实际出现的精确 Go 类型收集；字段名 `key_*` / `inner_*` / `old*`。

- [ ] **Step 5: design 状态改为已实现；README 同步**

- [ ] **Step 6: 准备提交（等确认）**

```text
fix(undoproxy-gen): type-accurate undoOp key and inner-map slots

Collect map key and inner map types during codegen instead of
hardcoding keyI32 and map[string]int64 snapshots.
```

---

## Spec 覆盖自检

| Spec 要求 | 任务 |
|-----------|------|
| KeyType/LeafType → TypeStr | Task 1 |
| keySlots / innerMapSlots + sanitize 命名 | Task 2 |
| 删除固定 innerMapOld / key 五件套 | Task 2–3 |
| emit 全路径动态槽 | Task 3 |
| 夹具 int key、双 inner、具名标量 | Task 1+3 |
| examples regen + guide | Task 4 |
| 宿主非 Must | 未列入硬门禁 |

## Placeholder 扫描

无 TBD；sanitize 示例名以 `sanitizeIdent` 实测为准，测试应断言「登记返回名 ∈ 生成源码」而非猜错字符串。
