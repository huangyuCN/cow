# undoproxy-gen 结构化 Undo 通用化实现计划

> **状态：已实现**（截至 2026-05-27；本计划为历史执行记录，勿按未勾选步骤重复开发）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `undoproxy-gen` 改为仅依赖 `cowgen.FieldPlan.Kind` 的结构化 Undo 生成，支持任意/多根 `+cow:undoproxy-gen`，一次移除 `Player` 硬编码与 `AddUndo` 闭包双轨。

**Architecture:** `emitFromGraph` 遍历类型图 → `emitStructuredMethods(ub, struct, plan)` 按 Kind 生成 `ctx.push(undoOp{...})` 并注册 `undoKind` + Rollback case → `undoBuilder.writeRuntime` 输出 `TxContext`/`undoOp`/switch。删除 `emit_structured_player.go` 与静态 `emit_structured_runtime.go`。

**Tech Stack:** Go 1.25、`go/packages`、`internal/cowgen`、`go/format`

**Spec:** [../specs/2026-05-26-undoproxy-gen-structured-generic-design.md](../specs/2026-05-26-undoproxy-gen-structured-generic-design.md)

---

## 文件结构（目标态）

| 文件 | 职责 | 行数预算 |
|------|------|----------|
| `cmd/undoproxy-gen/emit_undo.go` | `undoBuilder`：kind 注册、运行时、`undoOp` 字段裁剪 | ≤500 |
| `cmd/undoproxy-gen/emit_structured.go` | `emitStructuredMethods` + 各 Kind 的 `emitStructured*` | ≤500（超出则拆 `emit_structured_map.go`） |
| `cmd/undoproxy-gen/emit_structured_graph.go` | `emitFromGraph`、`emitClone` | ≤120 |
| `cmd/undoproxy-gen/emit_helpers.go` | `mapTypeString`、`valueTypeForMap` 等（从 `emit.go` 迁入） | ≤80 |
| `cmd/undoproxy-gen/emit.go` | 仅 `emit()` → `emitFromGraph` | ≤20 |
| **删除** | `emit_structured_player.go`、`emit_structured_runtime.go` | — |
| `cmd/undoproxy-gen/testdata/types.go` | 双根夹具（已有 Player+Room，补 `Account` 可选） | — |
| `cmd/undoproxy-gen/generate_golden_test.go` | 双根生成断言（新建） | ≤120 |
| `zz_generated.undo_proxy.go` | 再生成 | — |

---

## Task 1: `undoBuilder` 基础

**Files:**
- Create: `cmd/undoproxy-gen/emit_undo.go`
- Test: `cmd/undoproxy-gen/emit_undo_test.go`

- [ ] **Step 1: 写失败测试 `TestUndoBuilder_KindDedup`**

```go
func TestUndoBuilder_KindDedup(t *testing.T) {
	g := &cowgen.Graph{Structs: []*cowgen.StructPlan{{Name: "Player"}, {Name: "Account"}}}
	ub := newUndoBuilder(g)
	k1 := ub.kind("Player", "Assets", "MapKeySet", "if op.had { op.player.Assets[op.keyString] = op.oldI64 } else { delete(op.player.Assets, op.keyString) }")
	k2 := ub.kind("Player", "Assets", "MapKeySet", "if op.had { op.player.Assets[op.keyString] = op.oldI64 } else { delete(op.player.Assets, op.keyString) }")
	if k1 != k2 {
		t.Fatalf("kind dedup: %s vs %s", k1, k2)
	}
	if len(ub.entries) != 1 {
		t.Fatalf("entries len got %d want 1", len(ub.entries))
	}
}
```

- [ ] **Step 2: 运行失败测试**

```bash
cd /Users/huangyu/work/golang/src/cow
go test ./cmd/undoproxy-gen/ -run TestUndoBuilder_KindDedup -count=1
```

Expected: FAIL（`newUndoBuilder` 未定义）

- [ ] **Step 3: 实现 `emit_undo.go` 核心**

要点：

```go
type undoBuilder struct {
	structs   []string
	entries   []undoEntry
	kindIndex map[string]string // key -> const name
	sliceElems map[string]bool  // 图中 slice 元素类型，用于 undoOp 字段
	mapValueTypes map[string]bool // statsOld 等
}

func kindName(structName, field, op string) string {
	return "undoKind" + structName + field + op
}

func recvLower(structName string) string {
	if structName == "" { return "x" }
	return strings.ToLower(structName[:1]) + structName[1:]
}

func (ub *undoBuilder) kind(structName, field, op, rollbackBody string) string {
	key := structName + "." + field + "." + op
	if n, ok := ub.kindIndex[key]; ok {
		return n
	}
	n := kindName(structName, field, op)
	ub.kindIndex[key] = n
	ub.entries = append(ub.entries, undoEntry{name: n, body: rollbackBody})
	return n
}

func (ub *undoBuilder) recvField(structName string) string {
	return recvLower(structName)
}

func (ub *undoBuilder) writeRuntime(b *bytes.Buffer) { /* 见 spec §6；无 undoKindClosure */ }
```

`writeRuntime` 生成：`undoKind` 常量块、`undoOp`（每个 `ub.structs` 一行 `player *Player`）、`TxContext`、`push`（小写）、`Reset`、`txPool`、`Rollback` switch。

- [ ] **Step 4: 运行测试通过**

```bash
go test ./cmd/undoproxy-gen/ -run TestUndoBuilder -count=1
```

Expected: PASS

---

## Task 2: `emitStructuredMethods` — 标量 / 指针 / map 标量

**Files:**
- Create: `cmd/undoproxy-gen/emit_structured.go`
- Modify: `cmd/undoproxy-gen/emit_helpers.go`（从 `emit.go` 剪切 `mapTypeString` 等）

- [ ] **Step 1: 从 `emit.go` 迁出 helper 到 `emit_helpers.go`**

保留：`mapTypeString`、`valueTypeForMap`、`mapTypeFromPlan`、`innerValueType`。

- [ ] **Step 2: 实现 `emitStructuredMethods` 分发**

```go
func emitStructuredMethods(b *bytes.Buffer, ub *undoBuilder, structName string, plan cowgen.FieldPlan) {
	r := cowgen.RecvIdent(structName)
	acc := r + "." + plan.FieldName
	switch plan.Kind {
	case cowgen.KindScalar:
		emitStructuredScalarPut(b, ub, structName, r, acc, plan)
	case cowgen.KindPtrStruct:
		emitStructuredPtrGetForWrite(b, ub, structName, r, acc, plan.FieldName, plan.LeafType)
	case cowgen.KindMapScalar, cowgen.KindMapStruct:
		emitStructuredMapPut(b, ub, structName, r, acc, plan, false)
	case cowgen.KindMapPtrStruct:
		emitStructuredMapPtrGet(b, ub, structName, r, acc, plan)
		emitStructuredMapPut(b, ub, structName, r, acc, plan, true)
	// Task 3 补全其余 Kind
	default:
		panic(fmt.Sprintf("unsupported kind %v for %s.%s", plan.Kind, structName, plan.FieldName))
	}
}
```

- [ ] **Step 3: `emitStructuredScalarPut`（对照现 `emitScalarPut`）**

生成示例（`int32` 字段）：

```go
kind := ub.kind(structName, plan.FieldName, "ScalarSet",
	fmt.Sprintf("op.%s.%s = op.%s", recvLower(structName), plan.FieldName, oldField(plan.LeafType)))
// push: ctx.push(undoOp{kind: kind, player: p, oldI32: old})
```

`oldField`：`int32`→`oldI32`，`int64`→`oldI64`，`string`→`oldString`。

- [ ] **Step 4: `emitStructuredPtrGetForWrite`**

Rollback：`op.{recv}.{field} = op.{leafSlot}`；`push` 的 `{leafSlot}` 为 `strings.TrimPrefix(typ,"*")` 的小写槽位（`hero` 存 `*Hero`）。

- [ ] **Step 5: `emitStructuredMapPut` + `emitMapEnsure`**

- `MapEnsureNil`：`op.{recv}.{field} = nil`
- `MapKeySet`：与现 `emitMapPut` 闭包语义相同的 if existed / delete 分支

- [ ] **Step 6: 临时接线的图生成测试（仅 KindScalar）**

在 `emit_structured_graph_test.go` 或扩展现有 test：对 testdata 仅启用 scalar 前，先不跑全量。

---

## Task 3: Slice / MapSlice / MapMap Kind

**Files:**
- Modify: `cmd/undoproxy-gen/emit_structured.go`（或 `emit_structured_slice.go` 若超 500 行）

- [ ] **Step 1: 从 `emit_structured_player.go` 提炼通用 `emitStructuredMapSlice`**

参数：`ub, structName, r, acc, plan, mapSliceCfg`（`ensureKind` 改为 `ub.kind(...)` 动态注册）。

覆盖：`AppendAt`、`SetAt`、`RemoveAt`、`Truncate`、`Put{Field}`（map 整体赋值）。

- [ ] **Step 2: `emitStructuredSliceOps`（字段级 slice）**

对照 `emit.go` `emitSliceOps`：`SliceTruncate`（`oldInt` + 截断恢复）、`SliceRestore`（tail 快照，元素类型记入 `ub.sliceElems`）。

- [ ] **Step 3: `emitStructuredMapMap*`**

对照 `emitMapMapPut`、`emitMapMapGetForWrite`、`emitMapMapSliceOps`；生成 `clone{Field}MapShallow` 辅助函数（无 Undo）。

- [ ] **Step 4: 补全 `emitStructuredMethods` switch 全部分支**

确保 `panic` 分支不可达（`cowgen` 已分类字段均应覆盖）。

---

## Task 4: 重写 `emitFromGraph` 并删除旧文件

**Files:**
- Modify: `cmd/undoproxy-gen/emit_structured_graph.go`
- Modify: `cmd/undoproxy-gen/emit.go`
- Delete: `cmd/undoproxy-gen/emit_structured_player.go`
- Delete: `cmd/undoproxy-gen/emit_structured_runtime.go`

- [ ] **Step 1: 重写 `emitFromGraph`**

```go
func emitFromGraph(output, pkgName string, g *cowgen.Graph) error {
	ub := newUndoBuilder(g)
	var methods bytes.Buffer
	for _, sp := range g.Structs {
		emitClone(&methods, sp)
		for _, plan := range sp.Plans {
			emitStructuredMethods(&methods, ub, sp.Name, plan)
		}
	}
	var b bytes.Buffer
	b.WriteString("// Code generated by undoproxy-gen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", pkgName)
	b.WriteString("import \"sync\"\n\n")
	ub.writeRuntime(&b)
	b.Write(methods.Bytes())
	formatted, err := format.Source(b.Bytes())
	// ...
}
```

删除：`playerStructuredFields`、`emitPlayerMethods`、`emitHeroMethods`、`emitRemainingPlans`。

- [ ] **Step 2: `emit.go` 仅保留**

```go
func emit(output, pkgName string, g *cowgen.Graph) error {
	return emitFromGraph(output, pkgName, g)
}
```

- [ ] **Step 3: 删除 `emit_structured_player.go`、`emit_structured_runtime.go`**

- [ ] **Step 4: 编译生成器**

```bash
go build -o /dev/null ./cmd/undoproxy-gen
```

Expected: 成功

---

## Task 5: 生成器测试（双根、无 AddUndo）

**Files:**
- Modify: `cmd/undoproxy-gen/testdata/types.go`
- Create: `cmd/undoproxy-gen/generate_golden_test.go`
- Modify: `cmd/undoproxy-gen/main_test.go`

- [ ] **Step 1: 确认 testdata 双根**

`testdata/types.go` 已有 `Player` 与 `Room` 两个 `+cow:undoproxy-gen=true`。可选新增：

```go
// +cow:undoproxy-gen=true
type Account struct {
	Balance int64
	Flags   map[string]bool
}
```

- [ ] **Step 2: `TestGenerate_NoAddUndo_DualRoot`**

```go
func TestGenerate_NoAddUndo_DualRoot(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	if err := Run(out, "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata"); err != nil {
		t.Fatal(err)
	}
	s := readFile(t, out)
	for _, bad := range []string{"AddUndo(", "undoKindClosure"} {
		if strings.Contains(s, bad) {
			t.Fatalf("generated contains %q", bad)
		}
	}
	for _, good := range []string{"type undoKind", "func (ctx *TxContext) push", "func (p *Player)", "func (r *Room)"} {
		if !strings.Contains(s, good) {
			t.Fatalf("generated missing %q", good)
		}
	}
}
```

- [ ] **Step 3: 更新 `main_test.go`**

移除对 `AddUndo`、`undoKindClosure` 的期望；保留 `PutAssets` + `ctx.push` 片段断言（针对 cow 主包 `Run`）。

- [ ] **Step 4: 运行**

```bash
go test ./cmd/undoproxy-gen/... -count=1
```

Expected: PASS

---

## Task 6: 再生成 `cow` 主包并修测试

**Files:**
- Modify: `zz_generated.undo_proxy.go`（`go generate`）
- Modify: `tx_reset_test.go`
- Modify: `cmd/undoproxy-gen/testdata/tx.go`（若生成器 testdata 需要 bootstrap）

- [ ] **Step 1: Bootstrap 再生成**

若 `go/packages` 加载失败，临时添加最小 `tx.go`（空 `TxContext` stub），再：

```bash
go run ./cmd/undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow
rm -f tx.go   # 生成物已含 TxContext
```

- [ ] **Step 2: 更新 `tx_reset_test.go`**

删除 `ctx.AddUndo`；仅 `push` 填满字段的 op，断言 `Reset` 后零值（含 `fn` 字段删除——`undoOp` 无 `fn`）。

- [ ] **Step 3: 全量测试**

```bash
go test ./... -count=1
```

Expected: PASS（`player_mega_test`、`undoproxy_nested_test`、`doc_examples_test` 等无需改调用方签名）

- [ ] **Step 4: 调用链自检**

确认：`applySparseWrites` / `applyMegaProxyProbeFull` 仍使用 `PutAssets`、`AppendItems` 等**无后缀** API；`undocheck` 仍识别 `TxContext`。

---

## Task 7: Benchmark 与文档

**Files:**
- Modify: `docs/guide/tx-context.md`
- Modify: `docs/guide/codegen-undoproxy.md`
- Modify: `cmd/undoproxy-gen/README.md`

- [ ] **Step 1: Lite + Mega benchmark**

```bash
GOCACHE=/tmp/go-cache go test -run '^$' \
  -bench 'Benchmark(Mega_)?UndoLog_SparseWrite_(Commit|Rollback)$' \
  -benchmem -benchtime=1s . 2>&1 | tee /tmp/bench-structured-generic.txt
```

与 `docs/superpowers/benchmarks/cow-undo-log-benchmark.md` 最近结构化条目对比；若任一项回退 >5%，优化热点 Kind 模板后再测。

- [ ] **Step 2: 更新文档**

- `tx-context.md`：实现位于 `zz_generated.undo_proxy.go`；**无** `AddUndo` 公共 API；任意 tag 根。
- `codegen-undoproxy.md`：删除「Player 热点 / 闭包兜底」描述。
- `undoproxy-gen/README.md`：生成器产出含 `TxContext`；单文件。

- [ ] **Step 3: 仓库内 grep 验收**

```bash
rg 'AddUndo|emit_structured_player|undoKindPlayerAssets' --glob '*.go' 
```

Expected：仅 `emit.go` 历史删除后无命中；benchmark 文档可有历史文字。

---

## Task 8: 最终验收清单（对照 spec §8）

- [ ] `go test ./...` 绿
- [ ] 生成文件无 `AddUndo`、无版本后缀
- [ ] 双根 testdata 生成含 `Room`/`Player`（及可选 `Account`）方法
- [ ] `zz_generated.undo_proxy.go` 仅一份 `TxContext`/`txPool`
- [ ] 文档已更新
- [ ] Benchmark 无显著回退（或记录原因与后续 issue）

---

## Spec 覆盖自检

| Spec § | 任务 |
|--------|------|
| §2 目标 1–4 | Task 3–6 |
| §5 删除 Player 专用 | Task 4 |
| §6 运行时 | Task 1, 4 |
| §7 全 Kind | Task 2–3 |
| §8 测试 | Task 5–8 |
| §9 B/C 行为 | Task 5–6（双根）+ 文档 Task 7 |

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| `emit_structured.go` 超 500 行 | 拆 `emit_structured_map.go` / `emit_structured_slice.go` |
| 生成 `Rollback` switch 过大 | 接受；按 kind 去重；后续可考虑 op 编码（非本次） |
| `go generate` 需 bootstrap | Task 6  documented |
| 语义回归 | 依赖现有 `player_mega_test` + mega 探针；Kind 模板逐段对照原 `emit.go` 闭包 |

---

**Plan complete.** 实现时按 Task 1→8 顺序执行；未经用户明确要求不要 `git commit`。
