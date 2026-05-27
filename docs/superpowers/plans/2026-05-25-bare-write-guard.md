# 裸写防护（undocheck）实现计划

> **状态：已实现**（截至 2026-05-27；本计划为历史执行记录，勿按未勾选步骤重复开发）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `cmd/undocheck` 分析器 `cowbarewrite`，在编译前禁止对 undoproxy 监控类型裸写；修复本仓库违规；CI 接入 `go vet`。

**Architecture:** 抽取 `internal/cowmon` 与 `undoproxy-gen` 共享「根标记 + 同包 struct 图」；`undocheck` 用 `go/analysis` 扫描赋值/IncDec/append/map&slice 下标写，按文件名白名单跳过；DeepCopy 对照裸写迁至 `*_fixture.go`。

**Tech Stack:** Go 1.25、`golang.org/x/tools/go/analysis`、`go/packages`、`go/types`

**工作目录:** `/Users/huangyu/work/golang/src/cow`（禁止 git worktree）

**设计说明:** `docs/superpowers/specs/2026-05-25-bare-write-guard-design.md`

---

## 文件一览

| 路径 | 操作 |
|------|------|
| `internal/cowmon/load.go` | 新建：加载包、解析 `+cow:undoproxy-gen`、收集同包 struct |
| `internal/cowmon/graph.go` | 新建：`MonitoredSet`（*types.Named 集合 + 方法名提示） |
| `internal/cowmon/load_test.go` | 新建 |
| `cmd/undoproxy-gen/loader.go` | 修改：改为调用 `cowmon` |
| `cmd/undocheck/main.go` | 新建：`singlechecker.Main` |
| `cmd/undocheck/analyzer.go` | 新建：`cowbarewrite` 分析器 |
| `cmd/undocheck/whitelist.go` | 新建：路径/注释白名单 |
| `cmd/undocheck/inspect.go` | 新建：AST 左值类型推断 + 报诊断 |
| `cmd/undocheck/analyzer_test.go` | 新建：`analysistest` |
| `cmd/undocheck/testdata/src/...` | 新建：正反例 |
| `bench_baseline_fixture.go` | 新建：`sparseWriteDirect` / `sparseWriteMegaDirect`（白名单） |
| `benchmark_test.go` | 修改：删除裸写函数，改调 fixture |
| `bench_mega_writes.go` | 修改：删除 `sparseWriteMegaDirect` |
| `benchmark_mega_test.go` | 修改：引用 fixture 函数 |
| `undoproxy_nested_test.go` | 修改：用 fixture 构造 `*Player`，禁止字面量写字段 |
| `bench_fixture.go` | 修改：新增 `newPlayerWithItems` 等测试用小构造 |
| `doc.go` 或 `AGENTS.md` | 修改：补充 `go vet` 说明 |
| `Makefile` 或 `.github/workflows`（若存在） | 修改：CI 增加 vet |

---

## 实施阶段概览

| 阶段 | 任务 | 可验证产出 |
|------|------|------------|
| 1 | Task 1–2 | `go test ./internal/cowmon/...` 绿 |
| 2 | Task 3–5 | `go test ./cmd/undocheck/...` 绿 |
| 3 | Task 6–7 | `go vet` 对本仓库 0 诊断 |
| 4 | Task 8 | 全量 `go test ./...` + 文档 |

---

### Task 1: `internal/cowmon` 抽取（与生成器同源）

**Files:**
- Create: `internal/cowmon/load.go`
- Create: `internal/cowmon/graph.go`
- Create: `internal/cowmon/load_test.go`
- Modify: `cmd/undoproxy-gen/loader.go`

- [ ] **Step 1: 写失败测试 `TestLoadMonitored_cow`**

`internal/cowmon/load_test.go`：

```go
func TestLoadMonitored_cow(t *testing.T) {
	set, err := LoadMonitored("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Player", "Hero", "Item", "Skill", "Mail", "Quest"} {
		if !set.ContainsName(name) {
			t.Fatalf("missing monitored type %s", name)
		}
	}
}
```

- [ ] **Step 2: 运行确认 FAIL**

```bash
cd /Users/huangyu/work/golang/src/cow
go test ./internal/cowmon/... -count=1 -run TestLoadMonitored_cow
```

Expected: compile error or test fail

- [ ] **Step 3: 实现 `LoadMonitored`**

从 `cmd/undoproxy-gen/loader.go` 抽出：

```go
package cowmon

// MonitoredSet 纳入 bare-write 检查的具名 struct（同包 + 从根 BFS 可达）。
type MonitoredSet struct {
	ByName map[string]*types.Named
}

func (s *MonitoredSet) ContainsName(name string) bool { ... }
func (s *MonitoredSet) Contains(t types.Type) bool { ... }

// LoadMonitored 加载 importPath 包并构建监控类型集合。
func LoadMonitored(importPath string) (*MonitoredSet, error) { ... }
```

规则（与 `undoproxy-gen` 一致）：
- 带 `// +cow:undoproxy-gen=true` 的 struct 为根；
- BFS 展开字段中的同包 `*Struct` / `Struct` / slice/map 元素中的同包 struct。

- [ ] **Step 4: 重构 `cmd/undoproxy-gen/loader.go` 调用 `cowmon`**

```go
import "github.com/huangyuCN/cow/internal/cowmon"

func loadPackage(importPath string) (*PackageInfo, error) {
	mon, err := cowmon.LoadMonitored(importPath)
	// 保留 Roots/Structs 供 graph 使用
}
```

- [ ] **Step 5: 验证**

```bash
go test ./internal/cowmon/... ./cmd/undoproxy-gen/... -count=1
go test ./... -count=1
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cowmon cmd/undoproxy-gen/loader.go
git commit -m "refactor: 抽取 cowmon 供 undoproxy/undocheck 共享类型图"
```

---

### Task 2: 白名单与建议方法名

**Files:**
- Create: `cmd/undocheck/whitelist.go`
- Create: `cmd/undocheck/suggest.go`

- [ ] **Step 1: 实现 `skipFile(path string) bool`**

```go
func skipFile(filename string) bool {
	base := filepath.Base(filename)
	if strings.HasPrefix(base, "zz_generated") {
		return true
	}
	if strings.HasSuffix(base, "_fixture.go") || strings.HasSuffix(base, "_fixtures.go") {
		return true
	}
	switch base {
	case "deepcopy_generate.go", "undo_proxy_generate.go":
		return true
	}
	return false
}
```

- [ ] **Step 2: 实现 `allowBareWrite(commentGroups []*ast.CommentGroup) bool`**

扫描上一行/同行是否含 `cow:allow-bare-write`。

- [ ] **Step 3: 实现 `suggestProxy(typeName, fieldName string, kind writeKind) string`**

| kind | 建议 |
|------|------|
| scalar assign | `Put{Field}(ctx, …)` |
| slice append | `Append{Field}(ctx, …)` |
| map index | `Put{Field}(ctx, key, …)` |

- [ ] **Step 4: 单元测试 `whitelist_test.go`**

```go
func TestSkipFile(t *testing.T) {
	if !skipFile("bench_fixture.go") { t.Fatal() }
	if skipFile("player_test.go") { t.Fatal("test must not skip") }
}
```

- [ ] **Step 5: Commit**

```bash
git add cmd/undocheck/whitelist.go cmd/undocheck/suggest.go cmd/undocheck/whitelist_test.go
git commit -m "feat(undocheck): 白名单与代理方法名建议"
```

---

### Task 3: `cowbarewrite` 分析器骨架

**Files:**
- Create: `cmd/undocheck/analyzer.go`
- Create: `cmd/undocheck/main.go`
- Modify: `go.mod`（若需显式 require，通常已满足）

- [ ] **Step 1: `analyzer.go` 注册分析器**

```go
var Analyzer = &analysis.Analyzer{
	Name: "cowbarewrite",
	Doc:  "disallow bare writes to +cow:undoproxy-gen monitored structs",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if skipFile(pass.Fset.File(pass.Files[0].Pos()).Name()) {
		return nil, nil
	}
	mon, err := cowmon.LoadMonitored(pass.Pkg.Path)
	if err != nil {
		return nil, err
	}
	for _, f := range pass.Files {
		if skipFile(pass.Fset.File(f.Pos()).Name()) {
			continue
		}
		inspectFile(pass, f, mon)
	}
	return nil, nil
}
```

注意：对**非定义监控类型的包**（如仅测试引用 `cow.Player`），`LoadMonitored` 应加载 **`github.com/huangyuCN/cow`** 而非 `pass.Pkg.Path`。实现方式：

```go
const cowImport = "github.com/huangyuCN/cow"

func monitoredForPass(pass *analysis.Pass) (*cowmon.MonitoredSet, error) {
	if pass.Pkg.Path == cowImport {
		return cowmon.LoadMonitored(cowImport)
	}
	// 仅当文件 import 了 cow 包且用到其类型时才检查
	if !importsCow(pass) {
		return nil, nil
	}
	return cowmon.LoadMonitored(cowImport)
}
```

- [ ] **Step 2: `main.go`**

```go
package main

import (
	"github.com/huangyuCN/cow/cmd/undocheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(undocheck.Analyzer)
}
```

包名：将 `analyzer.go` 设为 `package undocheck`，`main.go` 为 `package main` 且 `import` 路径一致（或全部 `package main` + 同目录 — 采用 **`package undocheck` 库 + `package main` 子目录** 更简单：全部 `cmd/undocheck` 下 `package main` 亦可）。

推荐结构：

```
cmd/undocheck/
  main.go      // package main, singlechecker.Main(Analyzer)
  analyzer.go  // package main, var Analyzer
```

- [ ] **Step 3: 编译**

```bash
go build -o /dev/null ./cmd/undocheck
```

- [ ] **Step 4: Commit**

```bash
git add cmd/undocheck/main.go cmd/undocheck/analyzer.go
git commit -m "feat(undocheck): cowbarewrite 分析器骨架"
```

---

### Task 4: AST 裸写检测（TDD testdata）

**Files:**
- Create: `cmd/undocheck/inspect.go`
- Create: `cmd/undocheck/testdata/src/barewrite/p.go`
- Create: `cmd/undocheck/testdata/src/barewrite/bad.go`
- Create: `cmd/undocheck/testdata/src/barewrite/good.go`
- Create: `cmd/undocheck/testdata/src/barewrite/bench_fixture.go`
- Create: `cmd/undocheck/analyzer_test.go`

- [ ] **Step 1: 准备 testdata 类型（简化版 Player）**

`testdata/src/barewrite/p.go`：

```go
// +cow:undoproxy-gen=true
package barewrite

type Player struct {
	Level int32
	Items []*Item
}
type Item struct{ Name string }
```

`bad.go`（同包，非白名单文件名）：

```go
func BadAssign(p *Player) {
	p.Level = 1
}
```

`good.go`：

```go
func GoodRead(p *Player) int32 {
	return p.Level
}
```

`bench_fixture.go`（应跳过）：

```go
func InitInFixture(p *Player) {
	p.Level = 0
}
```

- [ ] **Step 2: 写 `analyzer_test.go`**

```go
func TestAnalyzer(t *testing.T) {
	testdata.Run(t, Analyzer, "testdata/src/barewrite")
}
```

配置 `testdata/src/barewrite/go.mod` 指向模块，或用 `analysistest` 标准布局（`testdata/src/barewrite/barewrite_test.go` 为 module barewrite）。

采用官方布局：

```
cmd/undocheck/testdata/src/a/a.go
cmd/undocheck/testdata/src/a/bad/bad.go  // 分开 package 需在 go.mod 声明
```

更简单：**单包 `barewrite`**，文件 `bad_assign.go` / `good_read.go` / `fixture_init.go`（`fixture_init.go` 用 `skipFile` 需文件名 `x_fixture.go`）。

- [ ] **Step 3: 实现 `inspect.go` 核心**

对下列 AST 节点调用 `checkWrite(pass, lhs Expr)`：

| 节点 | 处理 |
|------|------|
| `*ast.AssignStmt` | `=`、`+=` 等检查 `lhs` |
| `*ast.IncDecStmt` | 检查 `X` |
| `assign` 右侧为 `append` 且左侧为 `p.Items` | 报 `AppendItems` |

`checkWrite` 逻辑概要：

1. 若 `allowBareWrite` 则 return
2. 沿 `SelectorExpr` / `IndexExpr` / `SliceExpr` 剥到 base
3. 用 `types.Info` 判断 base 类型是否为 `*MonitoredNamed`
4. 若仅为 **非监控** 的 `map[string]int64` 下标写（`inner["k"]=v`，`inner` 类型为 map），**不报**（v1 明确允许 `GetStatsMapForWrite` 返回的内层 map 直写；与 spec 中 `PutStats` 外键区分）

5. 诊断文本：

```text
cowbarewrite: 禁止对 *Player 裸写字段 Level，请使用 PutLevel(ctx, …)（见 docs/superpowers/specs/2026-05-25-bare-write-guard-design.md）
```

- [ ] **Step 4: 运行测试至 PASS**

```bash
go test ./cmd/undocheck/... -count=1 -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/undocheck/inspect.go cmd/undocheck/testdata cmd/undocheck/analyzer_test.go
git commit -m "feat(undocheck): 裸写 AST 检测与 analysistest"
```

---

### Task 5: 复合字面量与 `&Player{...}` 测试

**Files:**
- Modify: `cmd/undocheck/inspect.go`
- Modify: `cmd/undocheck/testdata/...`

- [ ] **Step 1: 增加 `bad_literal.go`**

```go
func BadLiteral() *Player {
	return &Player{Level: 1}
}
```

- [ ] **Step 2: 决策并实现**

**v1 策略：** `CompositeLit` 中**具名监控 struct** 的字段键视为裸写，**报 error**（与「测试也走 fixture」一致）。

- [ ] **Step 3: 测试期望诊断后 Commit**

```bash
git commit -am "feat(undocheck): 禁止监控类型复合字面量写字段"
```

---

### Task 6: 本仓库迁移（benchmark + 测试）

**Files:**
- Create: `bench_baseline_fixture.go`
- Modify: `benchmark_test.go`, `bench_mega_writes.go`, `benchmark_mega_test.go`
- Modify: `bench_fixture.go`, `undoproxy_nested_test.go`

- [ ] **Step 1: 新建 `bench_baseline_fixture.go`**

```go
package cow

// sparseWriteDirect DeepCopy 对照组：模拟无 Undo 的历史裸写（仅 fixture 允许）。
func sparseWriteDirect(p *Player) {
	p.Assets["gold"] = 500
	p.Items = append(p.Items, &Item{Id: 9999, Name: "Shield"})
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
}

func sparseWriteMegaDirect(p *Player) {
	// 从 bench_mega_writes.go 原样迁入
}
```

- [ ] **Step 2: 从 `benchmark_test.go` / `bench_mega_writes.go` 删除上述函数**

- [ ] **Step 3: `bench_fixture.go` 增加**

```go
func newPlayerWithItems(items []*Item) *Player {
	p := &Player{}
	p.Items = items
	return p
}
```

（放在 `bench_fixture.go` 白名单内。）

- [ ] **Step 4: 改 `undoproxy_nested_test.go`**

```go
p := newPlayerWithItems([]*Item{{Id: 1}, {Id: 2}})
```

- [ ] **Step 5: 运行 vet 扫仓库**

```bash
go install ./cmd/undocheck
go vet -cowbarewrite ./...
```

Expected: 0 issues；若有 `applyMegaProxyProbeFull` 内 `inner["probe_inner"]=1` 被误报，在 **该行** 加 `//cow:allow-bare-write: inner map lazy path` 或改为 `PutStats`（优先改代理调用以去掉 allow）。

- [ ] **Step 6: Commit**

```bash
git add bench_baseline_fixture.go benchmark_test.go bench_mega_writes.go bench_fixture.go undoproxy_nested_test.go
git commit -m "fix: 裸写迁至 baseline fixture，测试用 fixture 构造"
```

---

### Task 7: CI 与开发者文档

**Files:**
- Modify: `doc.go` 或 `AGENTS.md`
- Create/Modify: CI 配置（若仓库无 CI，在 `AGENTS.md` 写清命令）

- [ ] **Step 1: `doc.go` 增加段落**

```go
// 裸写检查：go install ./cmd/undocheck && go vet -cowbarewrite ./...
```

- [ ] **Step 2: 全量验证**

```bash
go test ./... -count=1
go vet -cowbarewrite ./...
```

- [ ] **Step 3: Commit**

```bash
git commit -am "docs: 记录 cowbarewrite vet 用法"
```

---

### Task 8: 验收对照 spec §9

- [ ] `cmd/undocheck/testdata` 覆盖：赋值、+=、++、append 赋回、map/slice 下标、嵌套 selector、复合字面量、白名单 fixture 不报错
- [ ] `*_test.go` 无文件级豁免且通过 vet
- [ ] `zz_generated.undo_proxy.go` 跳过
- [ ] 更新 `docs/superpowers/specs/2026-05-25-bare-write-guard-design.md` 状态为「已实现」

```bash
go test ./... -count=1
go vet -cowbarewrite ./...
```

- [ ] **Final commit**（若 spec 状态变更）

```bash
git commit -am "docs: bare-write-guard 标记已实现"
```

---

## Spec 自检（计划 vs spec）

| Spec 要求 | 对应任务 |
|-----------|----------|
| go/analysis + cowbarewrite | Task 3–4 |
| 与 undoproxy 同源类型图 | Task 1 |
| 裸写模式表 | Task 4 inspect |
| 文件白名单 | Task 2 |
| `*_test.go` 不豁免 | Task 4–6 |
| 一律 error | analysistest 默认 |
| 诊断含建议方法 | Task 2 suggest |
| DeepCopy 对照迁 fixture | Task 6 |
| CI go vet | Task 7 |
| 内层 `map[string]int64` 直写 v1 允许 | Task 4 注明 |

---

## 风险与说明

1. **`go vet` 插件加载**：Go 1.25 下使用 `go vet -cowbarewrite` 需已安装含该分析器的工具；文档写 `go install github.com/huangyuCN/cow/cmd/undocheck@...` 或本地 `go install ./cmd/undocheck`。
2. **跨模块消费方**：其他 repo import `cow.Player` 时，需在**其模块** CI 同样安装并 `go vet -cowbarewrite ./...`（分析器随其 packages 加载 `cow` 类型图）。
3. **复合字面量**：测试构造必须迁到 `*_fixture.go`，略增样板，与 spec 一致。
