# undorewrite 历史裸写改写实现计划

> **状态：已实现**（截至 2026-05-27；本计划为历史执行记录，勿按未勾选步骤重复开发）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `cmd/undorewrite` 与 `internal/cowproxy`，将监控类型裸写批量改为 `undoproxy-gen` 代理调用；默认 dry-run，改写后通过 `cowbarewrite` vet。

**Architecture:** 抽取 `undoproxy-gen` 字段分类至 `internal/cowgen`；`cowproxy` 构建 RewriteCatalog；`undorewrite` 用 `go/packages` 加载目标、`go/types` 分解 LHS 路径并生成链式 `CallExpr`；白名单与 `undocheck` 一致。

**Tech Stack:** Go 1.25、`go/packages`、`go/ast`、`go/types`、`go/format`、`internal/cowmon`

**工作目录:** `/Users/huangyu/work/golang/src/cow`

**设计说明:** `docs/superpowers/specs/2026-05-25-undorewrite-codemod-design.md`

---

## 文件一览

| 路径 | 操作 |
|------|------|
| `internal/cowgen/kind.go` | 新建：`FieldKind`、`FieldPlan`、`KeyLayer` |
| `internal/cowgen/naming.go` | 新建：从 `cmd/undoproxy-gen/naming.go` 迁入 |
| `internal/cowgen/classify.go` | 新建：从 `cmd/undoproxy-gen/graph.go` 迁入 `classifyField` 等 |
| `internal/cowgen/graph.go` | 新建：`BuildGraph(*cowmon.PackageInfo)` |
| `internal/cowgen/graph_test.go` | 新建 |
| `internal/cowproxy/catalog.go` | 新建：`RewriteCatalog`、路径段→方法 |
| `internal/cowproxy/catalog_test.go` | 新建 |
| `internal/cowfile/skip.go` | 新建：`skipFile`、`allowBareWrite`（`undocheck`/`undorewrite` 共用） |
| `cmd/undoproxy-gen/graph.go` | 修改：改用 `cowgen` |
| `cmd/undoproxy-gen/naming.go` | 删除或薄包装 |
| `cmd/undoproxy-gen/emit.go` | 修改：类型引用 `cowgen.FieldPlan` |
| `cmd/undocheck/whitelist.go` | 修改：改用 `cowfile` |
| `cmd/undorewrite/main.go` | 新建：CLI |
| `cmd/undorewrite/load.go` | 新建：`packages.Load` |
| `cmd/undorewrite/ctx.go` | 新建：ctx 解析与注入 |
| `cmd/undorewrite/path.go` | 新建：LHS → `WritePath` |
| `cmd/undorewrite/rewrite.go` | 新建：AST 替换 |
| `cmd/undorewrite/diff.go` | 新建：dry-run 输出 |
| `cmd/undorewrite/rewrite_test.go` | 新建 |
| `cmd/undorewrite/testdata/legacy/legacy.go` | 新建：改写前 |
| `cmd/undorewrite/testdata/legacy/legacy_golden.go` | 新建：期望输出（或 `-w` 后对比） |
| `doc.go` | 修改：补充 `undorewrite` 用法 |

---

## 阶段概览

| 阶段 | 任务 | 验证 |
|------|------|------|
| 1 | Task 1–2 | `go test ./internal/cowgen/... ./internal/cowproxy/...` |
| 2 | Task 3–4 | `undoproxy-gen` 仍生成一致；`go test ./cmd/undoproxy-gen/...` |
| 3 | Task 5–8 | `go test ./cmd/undorewrite/...` |
| 4 | Task 9 | 集成 vet + 文档 |

---

### Task 1: 抽取 `internal/cowgen`

**Files:** `internal/cowgen/*.go`，修改 `cmd/undoproxy-gen/graph.go`、`naming.go`、`emit.go`

- [ ] **Step 1: 写 `TestBuildGraph_Player` 失败测试**

`internal/cowgen/graph_test.go`：

```go
func TestBuildGraph_Player(t *testing.T) {
	pkg, err := cowmon.LoadPackage("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	g, err := BuildGraph(pkg)
	if err != nil {
		t.Fatal(err)
	}
	var player *StructPlan
	for _, sp := range g.Structs {
		if sp.Name == "Player" {
			player = sp
			break
		}
	}
	if player == nil {
		t.Fatal("no Player plan")
	}
	// 至少含 Items(Append)、Assets(Put)、MainHero(Get+Put)、Stats(Put 双层键)
}
```

- [ ] **Step 2: 迁入 `kind.go`、`naming.go`、`classify.go`、`graph.go`**

从 `cmd/undoproxy-gen/graph.go` 剪切 `FieldKind`、`classifyField`、`buildGraph`（改名为 `BuildGraph`），包名 `cowgen`。

- [ ] **Step 3: 改 `undoproxy-gen` 引用**

```go
import "github.com/huangyuCN/cow/internal/cowgen"

func buildGraph(pkg *PackageInfo) (*cowgen.Graph, error) {
	return cowgen.BuildGraph(pkg)
}
```

`emit.go` 中 `StructPlan`/`FieldPlan` 改为 `cowgen.StructPlan`。

- [ ] **Step 4: 验证**

```bash
go test ./internal/cowgen/... ./cmd/undoproxy-gen/... -count=1
go test ./... -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/cowgen cmd/undoproxy-gen
git commit -m "refactor: 抽取 cowgen 供 undoproxy-gen/undorewrite 共用"
```

---

### Task 2: `internal/cowproxy.RewriteCatalog`

**Files:** `internal/cowproxy/catalog.go`、`catalog_test.go`

- [ ] **Step 1: 定义路径与写种类**

```go
type WriteKind int
const (
	WritePut WriteKind = iota
	WriteAppend
	WriteSetAt
	WriteMapPut    // map[k]=v
	WriteMapMapPut // Stats[k1][k2]=v
)

// PathStep 路径一步：字段名或下标层。
type PathStep struct {
	Field string
	Index bool // true 表示 map/slice 下标（key 表达式由调用方保留）
}

// FieldMethods 某 struct 某字段的代理方法名。
type FieldMethods struct {
	Put       string
	Append    string
	SetAt     string
	GetForWrite string // 指针 / map[*T] / 进入子 struct
	GetAtForWrite string // map[k][] 元素 *T
	MapForWrite string // 内层 map（v1 仅用于推断，裸写统一 Put 双层）
	MapPutKeys  int    // Put 所需额外 key 个数（0/1/2）
}

type RewriteCatalog struct {
	ByStruct map[string]map[string]FieldMethods // structName -> fieldName
}
```

- [ ] **Step 2: `NewCatalog(pkg *cowmon.PackageInfo) (*RewriteCatalog, error)`**

内部 `cowgen.BuildGraph(pkg)`，对每个 `FieldPlan` 填 `FieldMethods`（规则与 `emit.go` 一致，见 spec §6）。

- [ ] **Step 3: 测试 `TestCatalog_PlayerMainHero`**

```go
m := cat.ByStruct["Player"]["MainHero"]
if m.GetForWrite != "GetMainHeroForWrite" { t.Fatal() }
m2 := cat.ByStruct["Player"]["Stats"]
if m.MapPutKeys != 2 { t.Fatal() }
```

- [ ] **Step 4: Commit**

```bash
git add internal/cowproxy
git commit -m "feat(cowproxy): RewriteCatalog 字段→代理方法名"
```

---

### Task 3: 共用白名单 `internal/cowfile`

**Files:** `internal/cowfile/skip.go`；改 `cmd/undocheck/whitelist.go`

- [ ] **Step 1: 迁入 `SkipFile`、`AllowBareWrite`**

- [ ] **Step 2: `undocheck` 改为 `cowfile.SkipFile`**

- [ ] **Step 3: `go test ./cmd/undocheck/...` 仍绿**

- [ ] **Step 4: Commit**

```bash
git commit -am "refactor: cowfile 共用 undocheck/undorewrite 白名单"
```

---

### Task 4: `undorewrite` CLI 骨架

**Files:** `cmd/undorewrite/main.go`、`load.go`

- [ ] **Step 1: `main.go` flags**

```go
var (
	flagCow      = flag.String("cow", "github.com/huangyuCN/cow", "cow module import path")
	flagWrite    = flag.Bool("w", false, "write changes to source files")
	flagCtx      = flag.String("ctx", "ctx", "TxContext variable name")
	flagInject   = flag.String("inject-ctx", "", "new|pool|param:NAME")
	flagPoolVar  = flag.String("pool-var", "txPool", "pool identifier for inject-ctx=pool")
)
```

- [ ] **Step 2: `load.go` — `loadWorkspace(patterns, cowImport)`**

`packages.Load` 模式 `NeedSyntax|NeedTypes|NeedTypesInfo|NeedImports`；过滤 `cowfile.SkipFile`。

- [ ] **Step 3: `go build ./cmd/undorewrite`**

- [ ] **Step 4: Commit**

```bash
git add cmd/undorewrite/main.go cmd/undorewrite/load.go
git commit -m "feat(undorewrite): CLI 与 packages 加载"
```

---

### Task 5: ctx 解析（TDD）

**Files:** `cmd/undorewrite/ctx.go`、`ctx_test.go`

- [ ] **Step 1: 测试 `TestFindCtx_param` / `TestFindCtx_missing`**

- [ ] **Step 2: 实现 `resolveCtx(fn *ast.FuncDecl, info *types.Info, opts ctxOpts) (ast.Expr, []ast.Stmt, error)`**

返回：ctx 标识符表达式、可选注入的前置语句、`error`。

| inject | 前置语句 |
|--------|----------|
| 无 | 在 `fn.Body` 内找 `*cow.TxContext` 赋值/短变量/参数 |
| `new` | `ctx := &cow.TxContext{}`（`cow` 用 import 名） |
| `pool` | `ctx := txPool.Get().(*cow.TxContext)` + `defer txPool.Put(ctx)` |
| `param:ctx` | 签名必须有 `ctx *cow.TxContext` |

- [ ] **Step 3: Commit**

```bash
git commit -am "feat(undorewrite): TxContext 解析与可选注入"
```

---

### Task 6: LHS 路径分解

**Files:** `cmd/undorewrite/path.go`、`path_test.go`

- [ ] **Step 1: `WritePath` 结构**

```go
type WritePath struct {
	Root     ast.Expr   // 根表达式（*Player 变量）
	RootType string     // "Player"
	Steps    []PathStep // 字段与下标
	Leaf     PathLeaf   // 最后一跳：字段写 / map索引 / slice索引
}

type PathLeaf struct {
	Kind     WriteKind
	Field    string
	KeyExprs []ast.Expr // 各层 Index 的 key
}
```

- [ ] **Step 2: `pathsFromExpr(pass, lhs ast.Expr) (*WritePath, bool)`**

用 `types.Info` 展开 `SelectorExpr`/`IndexExpr`；根类型须在 `MonitoredSet` 内。

- [ ] **Step 3: 测试用 snippet 覆盖**

`p.Level`、`p.Assets[k]`、`p.MainHero.Level`、`p.Heros[k].Level`、`p.Bags[k][i].Name`、`p.Stats[k1][k2]`。

- [ ] **Step 4: Commit**

```bash
git commit -am "feat(undorewrite): LHS 路径分解"
```

---

### Task 7: AST 重写引擎

**Files:** `cmd/undorewrite/rewrite.go`、`rewrite_test.go`

- [ ] **Step 1: `buildReceiver(path *WritePath, ctx ast.Expr, cat *cowproxy.RewriteCatalog) ast.Expr`**

从 `path.Root` 起，对 `Steps[0:n-1]` 链式插入 `Get*ForWrite(ctx, keys...)`。

- [ ] **Step 2: `rewriteAssign`、`rewriteIncDec`、`rewriteAppendAssign`**

示例 — `p.MainHero.Level = v`：

```go
&ast.ExprStmt{
	X: &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.CallExpr{ /* GetMainHeroForWrite */ },
			Sel: &ast.Ident{Name: "PutLevel"},
		},
		Args: []ast.Expr{ctxIdent, rhs},
	},
}
```

`append` 多参：循环生成多条 `AppendItems(ctx, arg)`。

`+=` / `++`：生成 `PutLevel(ctx, binExpr)`。

- [ ] **Step 3: `rewriteFile(pass, cat, opts) (*RewriteResult, error)`**

收集待替换节点；**从后往前**替换（避免偏移）；跳过已是 `Put`/`Get*ForWrite` 的调用。

- [ ] **Step 4: 黄金测试目录**

`testdata/legacy/legacy.go`：

```go
func use(p *cow.Player, ctx *cow.TxContext) {
	p.Level = 1
	p.Assets["gold"] = 100
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
}
```

运行 `rewrite` 后与 `legacy_golden.go` `cmp` 或 `go test` 内嵌期望源码。

- [ ] **Step 5: Commit**

```bash
git add cmd/undorewrite/rewrite.go cmd/undorewrite/testdata
git commit -m "feat(undorewrite): AST 重写与 testdata 黄金用例"
```

---

### Task 8: dry-run diff 与 `-w` 写回

**Files:** `cmd/undorewrite/diff.go`、`main.go` 串联

- [ ] **Step 1: `formatFile(fset, f *ast.File) ([]byte, error)`**

- [ ] **Step 2: dry-run 打印 unified diff**（`github.com/pmezard/go-difflib` 或简易 line diff；优先标准库逐行对比 before/after 字符串）

- [ ] **Step 3: `-w` 原子写回**（`os.WriteFile` 至 `path.tmp` 再 `Rename`）

- [ ] **Step 4: stderr 汇总**

```text
undorewrite: 3 files, 12 rewrites, 1 function skipped (no ctx)
```

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(undorewrite): dry-run diff 与 -w 写回"
```

---

### Task 9: 集成验收与文档

- [ ] **Step 1: `doc.go` 增加**

```go
// 历史裸写批量改写：
//
//	undorewrite ./mypkg/...
//	undorewrite -w -ctx=tx ./mypkg/...
```

- [ ] **Step 2: 对 testdata 跑通**

```bash
go install ./cmd/undorewrite
undorewrite -cow=github.com/huangyuCN/cow -w ./cmd/undorewrite/testdata/legacy/...
go vet -vettool=$(go env GOPATH)/bin/undocheck ./cmd/undorewrite/testdata/...
```

- [ ] **Step 3: 全量回归**

```bash
go test ./... -count=1
```

- [ ] **Step 4: 更新 spec 状态为「已实现」**

- [ ] **Step 5: Commit**

```bash
git commit -am "docs: undorewrite 用法与 spec 已实现"
```

---

## Spec 覆盖自检

| Spec § | 任务 |
|--------|------|
| cowproxy 与 undoproxy 同源 | Task 1–2 |
| 通用 CLI `-cow` | Task 4 |
| 根→叶重写 | Task 6–7 |
| ctx D | Task 5 |
| dry-run A | Task 8 |
| append 拆分 | Task 7 |
| 白名单 | Task 3 |
| vet 验收 | Task 9 |
| 退出码非 0 有 skip | Task 8 main |

---

## 风险

1. **抽取 cowgen** 可能触动 `emit.go` 大量 import — 每步跑 `go generate` 对比 `zz_generated.undo_proxy.go` 可选（diff 应为空）。
2. **消费方包名**：`-cow` 的 import 在目标文件中可能别名，重写时用 `types.Info` 的 `Uses` 取本地标识符，勿写死 `cow.`。
3. **复合字面量**：v1 可仅改写 `AssignStmt`/`IncDecStmt`，`CompositeLit` 可选 Phase 2（与 undocheck 一致时再补）。
