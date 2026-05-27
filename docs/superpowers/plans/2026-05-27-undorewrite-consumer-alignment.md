# undorewrite 独立接入方对齐 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 使 `undorewrite` 对含 `+cow:undoproxy-gen` 的**独立 Go 包**（不 import `cow` 业务类型）能正确改写裸写，与 `undocheck` 的 per-package 监控策略一致。

**Architecture:** `loadWorkspace` 为每个 `packages.Package` 解析 `MonitoredSet` + `RewriteCatalog`（先 `BuildFromSyntax` + `NewCatalog(pkgPath)`，无本地根时若 import `-cow` 则回退）。`rewriteFile` 使用**当前文件所属包**的目录；`ctx` 识别同包 `*TxContext`，注入 AST 使用本包类型名而非 `cow.TxContext`。

**Tech Stack:** Go 1.25、`golang.org/x/tools/go/packages`、`internal/cowmon`、`internal/cowproxy`

**设计说明:** `docs/superpowers/specs/2026-05-27-undorewrite-consumer-alignment-design.md`

**工作目录:** 仓库根目录（非 worktree）

---

## 文件一览

| 文件 | 操作 |
|------|------|
| `internal/cowmon/imports.go` | 新建：`Imports(pkg, path)` |
| `internal/cowmon/imports_test.go` | 新建 |
| `cmd/undorewrite/load.go` | 修改：`packageEnv`、per-pkg catalog |
| `cmd/undorewrite/ctx.go` | 修改：同包 `TxContext`、本包 inject |
| `cmd/undorewrite/rewrite.go` | 修改：`rewriteFile` 接收 `*packageEnv` |
| `cmd/undorewrite/ctx_test.go` | 新建 |
| `cmd/undorewrite/rewrite_test.go` | 修改：保留 legacy + 新增 consumer |
| `cmd/undorewrite/testdata/consumer/types.go` | 新建 |
| `cmd/undorewrite/testdata/consumer/use.go` | 新建 |
| `docs/guide/migration-undorewrite.md` | 修改 |
| `cmd/undorewrite/README.md` | 修改 |

---

### Task 1: `cowmon.Imports` 共用函数

**Files:**
- Create: `internal/cowmon/imports.go`
- Create: `internal/cowmon/imports_test.go`

- [ ] **Step 1: 写失败测试**

```go
package cowmon_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
	"golang.org/x/tools/go/packages"
)

func TestImports_cowRoot(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedImports | packages.NeedDeps,
	}, "github.com/huangyuCN/cow/cmd/undorewrite/testdata/legacy")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages")
	}
	if !cowmon.Imports(pkgs[0].Types, "github.com/huangyuCN/cow") {
		t.Fatal("legacy should import cow")
	}
}
```

- [ ] **Step 2: 运行确认 FAIL**

```bash
cd /Users/huangyu/work/golang/src/cow
go test ./internal/cowmon/ -run TestImports_cowRoot -count=1
```

Expected: `undefined: cowmon.Imports`

- [ ] **Step 3: 实现**

```go
package cowmon

import "go/types"

// Imports 判断 pkg 是否直接 import 指定路径。
func Imports(pkg *types.Package, importPath string) bool {
	if pkg == nil {
		return false
	}
	for _, imp := range pkg.Imports() {
		if imp != nil && imp.Path() == importPath {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 测试 PASS**

```bash
go test ./internal/cowmon/ -run TestImports -count=1
```

- [ ] **Step 5: 重构 `cmd/undocheck/analyzer.go` 使用 `cowmon.Imports`**

将 `importsCow(pass)` 改为调用 `cowmon.Imports(pass.Pkg, cowImportPath)` 并删除本地 `importsCow` 函数。

```bash
go test ./cmd/undocheck/... -count=1
```

---

### Task 2: per-package `packageEnv`

**Files:**
- Modify: `cmd/undorewrite/load.go`
- Create: `cmd/undorewrite/load_test.go`

- [ ] **Step 1: 写失败测试**

```go
package main

import (
	"testing"
)

func TestResolvePackageEnv_consumer(t *testing.T) {
	cfg := Config{CowImport: "github.com/huangyuCN/cow"}
	ws, err := loadWorkspace(cfg, []string{"./testdata/consumer"})
	if err != nil {
		t.Fatal(err)
	}
	env, ok := ws.envForPkgPath("github.com/huangyuCN/cow/cmd/undorewrite/testdata/consumer")
	if !ok {
		t.Fatal("missing consumer env")
	}
	if env.Mon == nil || env.Catalog == nil {
		t.Fatal("nil mon or catalog")
	}
	if !env.Mon.ContainsName("Player") {
		t.Fatal("Player not monitored")
	}
	if _, ok := env.Catalog.Lookup("Player", "Assets"); !ok {
		t.Fatal("Assets methods missing")
	}
}

func TestResolvePackageEnv_legacyFallback(t *testing.T) {
	cfg := Config{CowImport: "github.com/huangyuCN/cow"}
	ws, err := loadWorkspace(cfg, []string{"./testdata/legacy"})
	if err != nil {
		t.Fatal(err)
	}
	env, ok := ws.envForPkgPath("github.com/huangyuCN/cow/cmd/undorewrite/testdata/legacy")
	if !ok || env.Mon == nil {
		t.Fatal("legacy env")
	}
	if !env.Mon.ContainsName("Player") {
		t.Fatal("cow Player via fallback")
	}
}
```

- [ ] **Step 2: 运行确认 FAIL**

```bash
go test ./cmd/undorewrite/ -run TestResolvePackageEnv -count=1
```

- [ ] **Step 3: 修改 `load.go`**

将 `workspace` 改为：

```go
type packageEnv struct {
	Mon     *cowmon.MonitoredSet
	Catalog *cowproxy.RewriteCatalog
	// TxPkgPath 用于 ctx 类型判定（同包 *TxContext）
	TxPkgPath string
}

type workspace struct {
	Fset      *token.FileSet
	Pkgs      []*packages.Package
	ByPath    map[string]*packageEnv
	CowImport string
	CowName   string
}

func (ws *workspace) envForPkgPath(path string) (*packageEnv, bool) {
	env, ok := ws.ByPath[path]
	return env, ok
}

func resolvePackageEnv(pkg *packages.Package, cowImport string) (*packageEnv, error) {
	if pkg.Types == nil || len(pkg.Syntax) == 0 {
		return nil, fmt.Errorf("package %s missing types", pkg.PkgPath)
	}
	if mon, err := cowmon.BuildFromSyntax(pkg.Types, pkg.Syntax); err == nil {
		cat, err := cowproxy.NewCatalog(pkg.PkgPath)
		if err != nil {
			return nil, err
		}
		return &packageEnv{Mon: mon, Catalog: cat, TxPkgPath: pkg.PkgPath}, nil
	}
	if cowmon.Imports(pkg.Types, cowImport) {
		mon, err := cowmon.LoadMonitored(cowImport)
		if err != nil {
			return nil, err
		}
		cat, err := cowproxy.NewCatalog(cowImport)
		if err != nil {
			return nil, err
		}
		return &packageEnv{Mon: mon, Catalog: cat, TxPkgPath: cowImport}, nil
	}
	return nil, nil
}

func loadWorkspace(cfg Config, patterns []string) (*workspace, error) {
	// ... packages.Load 同现实现 ...
	ws := &workspace{
		Fset: fset, Pkgs: pkgs, ByPath: make(map[string]*packageEnv),
		CowImport: cfg.CowImport, CowName: "cow",
	}
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || len(pkg.Errors) > 0 {
			continue
		}
		env, err := resolvePackageEnv(pkg, cfg.CowImport)
		if err != nil {
			return nil, err
		}
		if env != nil {
			ws.ByPath[pkg.PkgPath] = env
		}
	}
	return ws, nil
}
```

删除 `loadWorkspace` 顶部对 `LoadMonitored`/`NewCatalog` 的全局单次调用。

- [ ] **Step 4: 测试 PASS**

```bash
go test ./cmd/undorewrite/ -run TestResolvePackageEnv -count=1
```

---

### Task 3: 同包 `TxContext` 与 inject

**Files:**
- Modify: `cmd/undorewrite/ctx.go`
- Create: `cmd/undorewrite/ctx_test.go`

- [ ] **Step 1: 写失败测试 `TestIsTxContextType_localPackage`**

在 `ctx_test.go` 用 `go/types` 构造同包 `*TxContext` 与异包类型，断言 `isTxContextType(t, pkgPath)` 仅对本包为 true。

- [ ] **Step 2: 写失败测试 `TestInjectCtx_new_local`**

对 `injectCtx(nil, "", "new", "")` 期望 `CompositeLit` 类型为 `&TxContext{}`（`*ast.Ident{Name:"TxContext"}`），**无** `SelectorExpr` 的 `cow` 前缀。

- [ ] **Step 3: 修改 `resolveCtx` 签名**

```go
func resolveCtx(fn *ast.FuncDecl, info *types.Info, txPkgPath, ctxName, inject, poolVar string) ctxResult
```

`findCtxInParams` / `findCtxInBody` / `findCtxName` 将 `cowPkg` 参数改为 `txPkgPath`，传给 `isTxContextType(t, txPkgPath)`。

- [ ] **Step 4: 修改 `injectCtx`**

`new` 分支：

```go
Type: &ast.StarExpr{X: &ast.Ident{Name: "TxContext"}},
```

`pool` 分支 `TypeAssert` 的 asserted type 同样用 `*TxContext` 本包 ident。

当 `txPkgPath != cowImport` 且 legacy 需要 `cow.TxContext` 时：若 `txPkgPath == cowImport`，可保留 `SelectorExpr{X: ident{Name: cowLocalName}, Sel: TxContext}`；更简单规则——**当 `txPkgPath == cfg.CowImport` 且包 import 名非空时用 Selector**，否则用本包 `TxContext`。实现：

```go
func txContextTypeExpr(txPkgPath, cowImport, cowLocalName string) ast.Expr {
	if txPkgPath == cowImport && cowLocalName != "" && cowLocalName != "." {
		return &ast.SelectorExpr{X: &ast.Ident{Name: cowLocalName}, Sel: &ast.Ident{Name: "TxContext"}}
	}
	return &ast.StarExpr{X: &ast.Ident{Name: "TxContext"}}
}
```

`injectCtx` 增加参数 `cowLocalName string`（legacy 传 `ws.cowPkgName(pkg)`，consumer 传 `""`）。

- [ ] **Step 5: 测试 PASS**

```bash
go test ./cmd/undorewrite/ -run 'TestIsTxContextType|TestInjectCtx' -count=1
```

---

### Task 4: `rewriteFile` 使用 per-package env

**Files:**
- Modify: `cmd/undorewrite/rewrite.go`
- Modify: `cmd/undorewrite/load.go`（`Run` 循环）

- [ ] **Step 1: 修改 `rewriteFile` 签名**

```go
func rewriteFile(ws *workspace, pkg *packages.Package, f *ast.File, env *packageEnv, cfg Config) fileRewriteResult
```

`catalogAdapter` 使用 `env.Catalog`；`mon` 使用 `env.Mon`；`resolveCtx(..., env.TxPkgPath, ..., ws.cowPkgName(pkg))`。

- [ ] **Step 2: `Run` 中按文件取 env**

```go
env, ok := ws.envForPkgPath(pkg.PkgPath)
if !ok {
	continue // 该包无监控类型，跳过
}
fileRes := rewriteFile(ws, pkg, f, env, cfg)
```

- [ ] **Step 3: 全量测试（legacy 仍绿）**

```bash
go test ./cmd/undorewrite/... -count=1
```

Expected: 可能 FAIL 至 Task 5 完成 consumer testdata

---

### Task 5: `testdata/consumer` 与集成测试

**Files:**
- Create: `cmd/undorewrite/testdata/consumer/types.go`
- Create: `cmd/undorewrite/testdata/consumer/use.go`
- Modify: `cmd/undorewrite/rewrite_test.go`

- [ ] **Step 1: 创建 `types.go`**

```go
package consumer

// TxContext 测试用最小上下文（真实接入由生成器产出）。
type TxContext struct{}

type Hero struct {
	Level int32
}

type Item struct {
	Id int64
}

// Player 带 undoproxy 标记的聚合根。
//
// +cow:undoproxy-gen=true
type Player struct {
	Level    int32
	Assets   map[string]int64
	MainHero *Hero
	Items    []*Item
}
```

- [ ] **Step 2: 创建 `use.go`**

```go
package consumer

func Use(p *Player, ctx *TxContext) {
	p.Level = 1
	p.Assets["gold"] = 100
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
	p.Items = append(p.Items, &Item{Id: 9})
}
```

- [ ] **Step 3: 扩展 `rewrite_test.go`**

```go
func TestRewriteConsumerDryRun(t *testing.T) {
	cfg := Config{CowImport: "github.com/huangyuCN/cow", CtxName: "ctx"}
	res, err := Run(cfg, []string{"./testdata/consumer"})
	if err != nil {
		t.Fatal(err)
	}
	if res.RewriteCount == 0 {
		t.Fatal("expected rewrites")
	}
	all := ""
	for _, d := range res.Diffs {
		all += d.After
	}
	for _, want := range []string{
		"PutLevel(ctx,", "PutAssets(ctx,", "GetMainHeroForWrite(ctx)",
		"AppendItems(ctx,",
	} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q:\n%s", want, all)
		}
	}
	if strings.Contains(all, "cow.") {
		t.Fatalf("should not reference cow package in output:\n%s", all)
	}
}
```

- [ ] **Step 4: 运行全部 undorewrite 测试**

```bash
go test ./cmd/undorewrite/... -count=1
```

Expected: PASS

- [ ] **Step 5: 全仓库回归**

```bash
go test ./...
```

---

### Task 6: 文档

**Files:**
- Modify: `docs/guide/migration-undorewrite.md`
- Modify: `cmd/undorewrite/README.md`

- [ ] **Step 1: `migration-undorewrite.md` 增加「独立 module」小节**

要点：

- 先 `go generate` 生成 `zz_generated.undo_proxy.go`
- 在**业务包目录**执行：`undorewrite ./...`（无需 import `github.com/huangyuCN/cow` 的类型）
- 函数须有 `*TxContext` 参数或 `-inject-ctx=pool`（使用本包 `txPool`）
- 改写后 `go vet -vettool=.../undocheck ./...`

- [ ] **Step 2: `cmd/undorewrite/README.md`**

- `-cow`：仅当目标包仍使用 cow 导出类型时的 catalog **回退**路径
- 默认按**每个包**的类型图生成改写目录

- [ ] **Step 3: 在 `2026-05-25-undorewrite-codemod-design.md` 文首加一行**

链接 `2026-05-27-undorewrite-consumer-alignment-design.md`（独立接入扩展）。

---

### Task 7: 验收清单（对照 spec §8）

- [ ] `go test ./cmd/undorewrite/...` 全绿
- [ ] `TestRewriteConsumerDryRun` 输出无 `cow.`
- [ ] `TestRewriteLegacyDryRun`（或 `TestRewriteLegacyDryRun` 现名）仍通过
- [ ] `go test ./internal/cowmon/... ./cmd/undocheck/...` 通过
- [ ] 文档与 gamestore 流程一致

---

## Spec 覆盖自检

| Spec 要求 | 任务 |
|-----------|------|
| per-package mon/cat | Task 2、4 |
| 同包 TxContext | Task 3 |
| cow 回退 | Task 2 legacy 测试 |
| testdata/consumer | Task 5 |
| 文档 | Task 6 |
| 不改 path/rewrite 规则 | — |

---

Plan complete and saved to `docs/superpowers/plans/2026-05-27-undorewrite-consumer-alignment.md`.

**两种执行方式：**

1. **Subagent-Driven（推荐）** — 每个 Task 单独子代理，任务间做审查，迭代快  
2. **本会话内联执行** — 按 `executing-plans` 分批实现并在检查点停下

你更倾向哪一种？确认后我可以开始写代码（未经你同意不会 `git commit`）。
