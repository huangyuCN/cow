# undoproxy-gen 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `cmd/undoproxy-gen`，对带 `+cow:undoproxy-gen=true` 的聚合根及同包可达嵌套类型生成 `zz_generated.undo_proxy.go`（Put/Append/Set/Remove/Truncate/Get*ForWrite/CloneForWrite），覆盖嵌套 map/slice，并替换手写 `player_proxy.go`。

**Architecture:** `go/packages` 加载目标包 → 构建 TypeGraph（递归展开 map/slice/ptr）→ 按字段分类选择模板 → 输出单文件 Go 源码。业务包通过 `go generate` 调用已安装的 `undoproxy-gen` 二进制。

**Tech Stack:** Go 1.25、`golang.org/x/tools/go/packages`、`go/types`、`text/template`、`//go:embed`

**工作目录:** 仓库根 `/Users/huangyu/work/golang/src/cow`（禁止 git worktree）

**设计说明:** `docs/superpowers/specs/2026-05-25-undoproxy-codegen-design.md`

---

## 文件一览

| 路径 | 操作 |
|------|------|
| `cmd/undoproxy-gen/main.go` | 新建：CLI |
| `cmd/undoproxy-gen/loader.go` | 新建：Load + 解析 tag |
| `cmd/undoproxy-gen/graph.go` | 新建：类型图、字段分类、递归路径 |
| `cmd/undoproxy-gen/naming.go` | 新建：方法名 / Singular |
| `cmd/undoproxy-gen/emit.go` | 新建：渲染与写文件 |
| `cmd/undoproxy-gen/templates/proxy.go.tmpl` | 新建：主模板（可拆多文件 embed） |
| `cmd/undoproxy-gen/*_test.go` | 新建：单元 + 黄金测试 |
| `cmd/undoproxy-gen/testdata/types.go` | 新建：生成器夹具类型 |
| `doc.go` | 修改：+cow:undoproxy-gen 包 tag |
| `types.go` | 修改：+tag、嵌套 Skill/Heros；删除手写 `Hero.Clone` |
| `undo_proxy_generate.go` | 新建：go:generate |
| `zz_generated.undo_proxy.go` | 生成并提交 |
| `player_proxy.go` | **删除** |
| `player_test.go` | 修改：仍测 MVP 三写；新增嵌套/ slice 删截断 |
| `undoproxy_nested_test.go` | 新建：map[k][]、map[k]map 回滚用例 |
| `go.mod` | 修改：添加 `golang.org/x/tools` |

---

## 实施阶段概览

| 阶段 | 任务 | 可验证产出 |
|------|------|------------|
| 1 | Task 1–2 | `go test ./cmd/undoproxy-gen/...` 绿；naming/graph 单测 |
| 2 | Task 3–5 | testdata 黄金文件生成一致 |
| 3 | Task 6–7 | `cow` 包 `go generate` + 删除手写 proxy |
| 4 | Task 8 | 全量 `go test ./...` + spec §13 验收 |

---

### Task 1: 生成器模块骨架与依赖

**Files:**
- Modify: `go.mod`
- Create: `cmd/undoproxy-gen/main.go`

- [ ] **Step 1: 添加 tools 依赖**

```bash
cd /Users/huangyu/work/golang/src/cow
go get golang.org/x/tools/go/packages@latest
```

- [ ] **Step 2: 创建 `main.go`（最小 CLI）**

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	output := flag.String("output-file", "", "output Go file path")
	flag.Parse()
	if *output == "" || flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: undoproxy-gen --output-file FILE IMPORT_PATH\n")
		os.Exit(2)
	}
	if err := run(*output, flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "undoproxy-gen: %v\n", err)
		os.Exit(1)
	}
}

func run(output, importPath string) error {
	// Task 3 填充
	return fmt.Errorf("not implemented")
}
```

- [ ] **Step 3: 验证可编译**

```bash
go build -o /dev/null ./cmd/undoproxy-gen
```

Expected: success

---

### Task 2: naming 与 graph 单元测试（TDD）

**Files:**
- Create: `cmd/undoproxy-gen/naming.go`
- Create: `cmd/undoproxy-gen/naming_test.go`
- Create: `cmd/undoproxy-gen/graph.go`
- Create: `cmd/undoproxy-gen/graph_test.go`

- [ ] **Step 1: 写 naming 失败测试**

`naming_test.go`:

```go
func TestSingular(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Heros", "Hero"},
		{"Items", "Item"},
		{"Assets", "Asset"},
	}
	for _, tc := range tests {
		if got := singular(tc.in); got != tc.want {
			t.Fatalf("singular(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestMethodNames_FieldSlice(t *testing.T) {
	m := sliceMethodNames("Items")
	if m.Append != "AppendItems" || m.SetAt != "SetItemsAt" ||
		m.RemoveAt != "RemoveItemsAt" || m.Truncate != "TruncateItems" {
		t.Fatalf("got %+v", m)
	}
}
```

- [ ] **Step 2: 运行确认 FAIL**

```bash
go test ./cmd/undoproxy-gen/ -run TestSingular -count=1
```

Expected: FAIL（`singular` 未定义）

- [ ] **Step 3: 实现 `naming.go`**

```go
func singular(field string) string {
	if len(field) > 1 && field[len(field)-1] == 's' {
		return field[:len(field)-1]
	}
	return field
}

type sliceMethods struct {
	Append, SetAt, RemoveAt, Truncate string
}

func sliceMethodNames(field string) sliceMethods {
	return sliceMethods{
		Append:    "Append" + field,
		SetAt:     "Set" + field + "At",
		RemoveAt:  "Remove" + field + "At",
		Truncate:  "Truncate" + field,
	}
}
```

（map 嵌套、MapForWrite 等函数同文件补充，测试随 Task 5 扩展。）

- [ ] **Step 4: 运行 naming 测试 PASS**

```bash
go test ./cmd/undoproxy-gen/ -run 'TestSingular|TestMethodNames' -count=1
```

- [ ] **Step 5: 写 graph 分类失败测试**

`graph_test.go` 使用内联 `types.Type` 构造或加载 `testdata`（Task 5 前可用最小 struct 字段列表 mock）。

初版断言（伪代码，实现时用 `go/types`）：

- `Assets map[string]int64` → `MapScalar`
- `Items []*Item` → `SlicePtr`
- `Hero *Hero` → `PtrStruct`
- `Loot map[int32][]int32` → `MapSliceValue`
- `Buffs map[int32]map[string]int64` → `MapMapScalar`

- [ ] **Step 6: 实现 `graph.go` 核心**

定义：

```go
type FieldKind int

const (
	Scalar FieldKind = iota
	PtrStruct
	MapScalar
	MapPtrStruct
	MapStruct
	SliceValue
	SlicePtr
	MapSliceValue
	MapSlicePtr
	MapMapScalar
	MapMapPtrStruct
	MapMapStruct
	// 更深嵌套：MapMapSlice* 等由 TypePath 链表示
)

type TypePath struct {
	Keys []types.Type   // map key 类型链
	// Leaf: 最终标量/slice/ptr/struct
}

type FieldSpec struct {
	Name     string
	Kind     FieldKind
	GoType   string // 打印用 go/types
	TypePath *TypePath
}
```

`classifyField(pkg *types.Package, f *types.Var)` 递归剥 map/slice 指针，遇外包包级 struct 返回 error。

- [ ] **Step 7: graph 测试 PASS**

```bash
go test ./cmd/undoproxy-gen/ -run TestClassify -count=1
```

---

### Task 3: loader 与 tag 解析

**Files:**
- Create: `cmd/undoproxy-gen/loader.go`
- Modify: `cmd/undoproxy-gen/main.go`

- [ ] **Step 1: 实现 `loader.go`**

```go
const (
	tagPackage = "// +cow:undoproxy-gen=package"
	tagType    = "// +cow:undoproxy-gen=true"
)

type PackageInfo struct {
	Name       string
	ImportPath string
	RootTypes  []*types.Named // 带 tagType 的 struct
	AllStructs []*types.Named // 同包可达
}

func loadPackage(importPath string) (*PackageInfo, error) {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo, Fset: token.NewFileSet()}
	pkgs, err := packages.Load(cfg, importPath)
	// 解析 doc.go / types.go 注释找 tag；收集 struct
}
```

- [ ] **Step 2: `main.run` 调用 loader**

无 root tag 时 `return fmt.Errorf("no type with +cow:undoproxy-gen=true")`

- [ ] **Step 3: 手动验证加载 cow 包**

```bash
go run ./cmd/undoproxy-gen --output-file /tmp/out.go github.com/huangyuCN/cow
```

Expected: 在 Task 6 前仍为 not implemented 或空输出；Task 3 后至少能 load 不 panic。

---

### Task 4: 模板与单层 emit（标量 / map / slice / ptr）

**Files:**
- Create: `cmd/undoproxy-gen/emit.go`
- Create: `cmd/undoproxy-gen/templates/proxy.go.tmpl`

- [ ] **Step 1: 定义 emit 数据模型**

```go
type GenFile struct {
	Package string
	Structs []GenStruct
}

type GenStruct struct {
	Name   string
	Clone  bool
	Fields []GenMethod
}

type GenMethod struct {
	Name    string
	Kind    string // 模板分支：scalar_put, map_put, slice_append, ...
	Params  []Param
	Body    string // 或由模板内联
}
```

- [ ] **Step 2: 编写模板片段 `scalar_put`**

生成示例：

```go
func (p *Player) PutGold(ctx *TxContext, amount int64) {
	oldGold := p.Gold
	ctx.AddUndo(func() { p.Gold = oldGold })
	p.Gold = amount
}
```

- [ ] **Step 3: 编写 `map_put` / `slice_append` / `ptr_get_for_write` / `clone_for_write` 模板**

语义严格照 spec §9.1–9.10、§9.17；`map_put` 含 nil map 与 delete 分支。

- [ ] **Step 4: `emit.go` 写文件**

```go
func emit(output string, gf *GenFile) error {
	// go:embed templates/*
	// execute 主模板；go/format.Source 格式化
}
```

- [ ] **Step 5: 单测：对内存 GenFile emit 后 `go/parser.ParseFile` 无语法错**

---

### Task 5: testdata 黄金测试（单层 + 嵌套夹具）

**Files:**
- Create: `cmd/undoproxy-gen/testdata/types.go`
- Create: `cmd/undoproxy-gen/golden_test.go`
- Modify: `cmd/undoproxy-gen/testdata/types.go`（完整夹具）

- [ ] **Step 1: 创建 testdata 包**

`cmd/undoproxy-gen/testdata/types.go`：

```go
// Package testdata 供 undoproxy-gen 黄金测试。
package testdata

// +cow:undoproxy-gen=true
type Player struct {
	Gold  int64
	Items []*Item
	Loot  map[int32][]int32
	Buffs map[int32]map[string]int64
}

type Item struct {
	Id int64
}

// +cow:undoproxy-gen=true
type Room struct {
	Heros map[int32]*Hero
}

type Hero struct {
	Skills map[int32]*Skill
}

type Skill struct {
	Level int32
}
```

- [ ] **Step 2: 写黄金测试**

```go
func TestGolden(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "zz_generated.go")
	if err := runGenerator(out, "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	want, _ := os.ReadFile("testdata/zz_generated.golden.go")
	if string(got) != string(want) {
		t.Fatal(cmp.Diff(string(want), string(got)))
	}
}
```

- [ ] **Step 3: 首次生成 golden 并人工审阅后提交**

```bash
go test ./cmd/undoproxy-gen/ -run TestGolden -update  # 若无 -update 则手动 cp
```

断言黄金文件含：`PutGold`、`AppendItems`、`RemoveItemsAt`、`AppendLootAt`、`GetBuffsMapForWrite`、`GetHeroForWrite`、`CloneForWrite`。

- [ ] **Step 4: TestGolden PASS**

```bash
go test ./cmd/undoproxy-gen/ -count=1
```

---

### Task 6: 嵌套 map/slice/map 递归 emit

**Files:**
- Modify: `cmd/undoproxy-gen/graph.go`
- Modify: `cmd/undoproxy-gen/templates/proxy.go.tmpl`

- [ ] **Step 1: 扩展 TypePath 生成多级参数**

`map[int32]map[string][]int32` → 方法 `AppendBuffsAt(ctx, k1 int32, k2 string, v int32)`。

- [ ] **Step 2: 模板 `map_slice_append_at`（§9.11）**

内联 ensure 外层 map + `prev, existed := m[k]` + 写回 `m[k]`。

- [ ] **Step 3: 模板 `map_map_get_for_write` + `clone_map_shallow` 辅助函数**

每种内层 map 类型生成：

```go
func cloneBuffsMapShallow(m map[string]int64) map[string]int64 {
	if m == nil { return nil }
	c := make(map[string]int64, len(m))
	for k, v := range m { c[k] = v }
	return c
}
```

- [ ] **Step 4: 更新 testdata golden 覆盖 §13 嵌套场景**

- [ ] **Step 5: `go test ./cmd/undoproxy-gen/ -count=1` PASS**

---

### Task 7: 接入 cow 根包

**Files:**
- Modify: `doc.go`
- Modify: `types.go`
- Create: `undo_proxy_generate.go`
- Create: `zz_generated.undo_proxy.go`（生成）
- Delete: `player_proxy.go`

- [ ] **Step 1: 修改 `doc.go`**

```go
// +cow:undoproxy-gen=package
// +groupName=cow.huanghaiyu.cn
```

- [ ] **Step 2: 扩展 `types.go` 并打 tag**

```go
// +cow:undoproxy-gen=true
type Player struct {
	// 保留 Uid/Assets/Items/Hero
	Heros map[int32]*Hero `bson:"heros"` // 嵌套 map 夹具
}

type Hero struct {
	HeroId int32
	Level  int32
	Skills map[int32]*Skill
}

type Skill struct {
	SkillId int32
	Level   int32
}
```

删除手写 `Hero.Clone()`（由 `CloneForWrite` 替代）。

- [ ] **Step 3: 创建 `undo_proxy_generate.go`**

```go
package cow

//go:generate undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow
```

- [ ] **Step 4: 安装并 generate**

```bash
go install ./cmd/undoproxy-gen
go generate ./...
go build ./...
```

Expected: 生成 `zz_generated.undo_proxy.go`；含 `PutAsset`、`AppendItem`、`GetHeroForWrite` 等与现手写等价方法。

- [ ] **Step 5: 删除 `player_proxy.go`**

- [ ] **Step 6: 确认 `applySparseWrites` 无需改签名**

`bench_fixture_test.go` 仍调用 `PutAsset`/`AppendItem`/`GetHeroForWrite`（生成器保持字段名方法）。

---

### Task 8: 根包集成测试（TDD）

**Files:**
- Modify: `player_test.go`
- Create: `undoproxy_nested_test.go`

- [ ] **Step 1: 跑现有测试确认仍绿**

```bash
go test ./... -count=1
```

- [ ] **Step 2: 新增 slice Remove/Truncate 测试**

`undoproxy_nested_test.go`:

```go
func TestRollback_SliceRemoveRestore(t *testing.T) {
	p := &Player{Items: []*Item{{Id: 1}, {Id: 2}}}
	want := clonePlayerSnapshot(p)
	_ = runScopedWithRollback(p, func(ctx *TxContext) error {
		p.RemoveItemsAt(ctx, 0)
		return errors.New("fail")
	})
	assertPlayerEqual(t, p, want)
}
```

（`TruncateItems` 同理。）

- [ ] **Step 3: 新增 `map[k][]` 测试**

```go
func TestRollback_MapSliceAppend(t *testing.T) {
	// Player 需有 map[int32][]*Item 或 test-only 字段；若用 Heros 下挂 slice 字段按 types 实际调整
}
```

若 `Player` 无 `map[k][]` 字段，在 `types.go` 增加 `Tags map[int32][]string` 仅用于测试或复用 `Loot` 类字段。

- [ ] **Step 4: 新增两层 map `Heros` → `Skills` 测试**

```go
_ = runScopedWithRollback(p, func(ctx *TxContext) error {
	h := p.GetHeroForWrite(ctx, 1)
	h.PutSkill(ctx, 1, &Skill{SkillId: 1, Level: 99}) // 生成名为 PutSkills(ctx, skillId, val *Skill)
	return errors.New("fail")
})
```

- [ ] **Step 5: 全量测试 PASS**

```bash
go test ./... -count=1
```

---

### Task 9: 非法类型与文档收尾

**Files:**
- Create: `cmd/undoproxy-gen/graph_error_test.go`
- Modify: `docs/superpowers/specs/2026-05-25-undoproxy-codegen-design.md`（状态改为「已实现」— 实现完成后）

- [ ] **Step 1: 测试外包包 struct 报错**

testdata 包 `bad` 含 `OtherPkgField other.Type` 时期望 `loadPackage`/`buildGraph` 返回 error。

- [ ] **Step 2: 测试 `map[string]interface{}` fail fast**

- [ ] **Step 3: README 片段写入 `cmd/undoproxy-gen/doc.go` 或根 `doc.go` 注释**

说明 install、generate、tag 用法。

- [ ] **Step 4: spec §13 验收清单逐项勾选**

```bash
go install ./cmd/undoproxy-gen
go generate ./...
go test ./... -count=1
```

- [ ] **Step 5: Commit（需用户明确同意后）**

```bash
git add cmd/undoproxy-gen doc.go types.go undo_proxy_generate.go zz_generated.undo_proxy.go
git add player_test.go undoproxy_nested_test.go go.mod go.sum
git rm player_proxy.go
git commit -m "$(cat <<'EOF'
feat(cow): 添加 undoproxy-gen 并生成 Undo 写代理

EOF
)"
```

---

## Spec 覆盖自检

| Spec 章节 | 计划任务 |
|-----------|----------|
| §6 tag / go:generate | Task 7 |
| §7 字段分类 | Task 2, 4 |
| §7.1 嵌套 map/slice | Task 5, 6 |
| §8 命名 | Task 2 |
| §9.1–9.17 语义 | Task 4, 6（模板） |
| §10 布局 | Task 1–4 |
| §12 测试 | Task 2, 5, 8, 9 |
| §13 验收 | Task 8, 9 |

## 实现提示（避免常见坑）

1. **闭包捕获**：Undo 闭包必须捕获 `old` 值/指针副本，禁止捕获循环变量。
2. **map[k] slice**：始终 `m[k] = append(prev, x)`，Undo 用 `prev[:oldLen]` 而非对 `m[k]` 截断后再读。
3. **生成文件行数**：单模板过大时拆 `templates/*.tmpl` 多个 `define` 块。
4. **TxContext 类型名**：生成文件在 `package cow`，使用已有 `*TxContext`，不要重复定义。
5. **Benchmark**：本计划不修改 `benchmark_test.go`；生成代理后跑 `go test -bench=. -benchmem` 确认仍优于 DeepCopy（实现后提醒用户是否归档 bench）。

---

## 执行方式

计划已保存至 `docs/superpowers/plans/2026-05-25-undoproxy-codegen.md`。

可选执行方式：

1. **Subagent-Driven（推荐）** — 每 Task 派发子 agent，任务间你做 review  
2. **Inline Execution** — 本会话按 Task 顺序实现，阶段检查点暂停

你更倾向哪一种？若直接开始实现，回复 **1** 或 **2**（或「开始写代码」）。
