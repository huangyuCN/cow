# undocheck 检查范围补全（按 import 扩散）实现计划

> **状态：已实现**（2026-06-02）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans。步骤使用 `- [ ]` 勾选跟踪。

**Goal:** 补全 `cowbarewrite` 检查范围——业务包**直接 import** 带 `+cow:undoproxy-gen=true` 的定义包时，对该业务包源码中的裸写报 error。

**Architecture:** 在 `monitoredForPass` 中合并「本包 `BuildFromSyntax`」与「每个 direct import 的 `LoadMonitoredCached`」；`cowmon.Union` 合并 `MonitoredSet`；`inspect.go` 不变。移除仅认 `import cow` 的特殊分支。

**Tech Stack:** Go 1.25、`go/analysis`、`golang.org/x/tools/go/packages`、`internal/cowmon`

**设计说明:** [docs/superpowers/specs/2026-06-02-undocheck-import-scope-design.md](../specs/2026-06-02-undocheck-import-scope-design.md)

**工作目录:** 仓库根 `/Users/yuchen/mfhl/cow`（禁止 git worktree）

---

## 文件一览

| 路径 | 操作 |
|------|------|
| `internal/cowmon/union.go` | 新建：`Union` |
| `internal/cowmon/union_test.go` | 新建 |
| `internal/cowmon/cache.go` | 新建：`LoadMonitoredCached` + 无 tag 哨兵 |
| `internal/cowmon/cache_test.go` | 新建 |
| `cmd/undocheck/analyzer.go` | 修改：`monitoredForPass` |
| `cmd/undocheck/analyzer_test.go` | 修改：增加 `importscope` 用例 |
| `cmd/undocheck/testdata/src/importscope/defpkg/types.go` | 新建 |
| `cmd/undocheck/testdata/src/importscope/consumer/bad_assign.go` | 新建 |
| `cmd/undocheck/testdata/src/importscope/consumer/good_read.go` | 新建 |
| `docs/superpowers/specs/2026-05-25-bare-write-guard-design.md` | 修改：§6.3 交叉引用 |
| `cmd/undocheck/README.md` | 修改：检查范围 |
| `docs/guide/bare-write-guard.md` | 修改：消费方说明 |

---

## 实施阶段

| 阶段 | 任务 | 验证 |
|------|------|------|
| 1 | Task 1–2 | `go test ./internal/cowmon/...` |
| 2 | Task 3–4 | `go test ./cmd/undocheck/...` |
| 3 | Task 5 | 文档 + 全量 `go test ./...`（可选 `go vet`） |

---

### Task 1: `MonitoredSet.Union`

**Files:**
- Create: `internal/cowmon/union.go`
- Create: `internal/cowmon/union_test.go`

- [ ] **Step 1: 写失败测试**

`internal/cowmon/union_test.go`：

```go
func TestUnion_mergesDisjointPackages(t *testing.T) {
	setA, err := cowmon.LoadMonitored("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	// 用 LoadMonitored 加载带 tag 的 testdata 包（Task 3 创建后可改为 importscope/defpkg）
	// 首版可与 setA 做「同集 Union 幂等」测试：
	merged := cowmon.Union(setA, setA)
	if merged == nil || !merged.ContainsName("Player") {
		t.Fatal("expected Player in merged set")
	}
	empty := cowmon.Union()
	if empty != nil {
		t.Fatal("Union() with no args should be nil")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/cowmon/... -run TestUnion -count=1
```

Expected: `undefined: cowmon.Union`

- [ ] **Step 3: 实现 `Union`**

`internal/cowmon/union.go`：

```go
// Union 合并多个监控集；无有效输入时返回 nil。
func Union(sets ...*MonitoredSet) *MonitoredSet {
	var out *MonitoredSet
	for _, s := range sets {
		if s == nil || len(s.byObj) == 0 {
			continue
		}
		if out == nil {
			out = &MonitoredSet{byObj: make(map[*types.TypeName]struct{})}
		}
		for obj := range s.byObj {
			out.byObj[obj] = struct{}{}
		}
	}
	if out == nil {
		return nil
	}
	return out
}
```

（`byObj` 为包内未导出字段时，测试放在 `cowmon` 包或导出测试辅助 `UnionForTest`——优先在 `union_test` 使用 `cowmon_test` 包通过 `Contains` 测行为，避免导出 `byObj`。）

- [ ] **Step 4: 测试通过**

```bash
go test ./internal/cowmon/... -run TestUnion -count=1
```

---

### Task 2: `LoadMonitoredCached`

**Files:**
- Create: `internal/cowmon/cache.go`
- Create: `internal/cowmon/cache_test.go`

- [ ] **Step 1: 写失败测试**

```go
func TestLoadMonitoredCached_noTagPackage(t *testing.T) {
	// encoding/json 等标准库无 undoproxy tag
	_, err := cowmon.LoadMonitoredCached("encoding/json")
	if err == nil {
		t.Fatal("expected error for package without tag")
	}
	// 第二次应命中缓存（同样 error，不 panic）
	_, err2 := cowmon.LoadMonitoredCached("encoding/json")
	if err2 == nil {
		t.Fatal("expected cached error")
	}
}
```

- [ ] **Step 2: 实现缓存**

```go
var monitoredCache sync.Map // path string -> *monitoredCacheEntry

type monitoredCacheEntry struct {
	set *MonitoredSet
	err error
}

func LoadMonitoredCached(importPath string) (*MonitoredSet, error) {
	if v, ok := monitoredCache.Load(importPath); ok {
		e := v.(*monitoredCacheEntry)
		return e.set, e.err
	}
	set, err := LoadMonitored(importPath)
	monitoredCache.Store(importPath, &monitoredCacheEntry{set: set, err: err})
	return set, err
}
```

- [ ] **Step 3: 测试通过**

```bash
go test ./internal/cowmon/... -run TestLoadMonitoredCached -count=1
```

---

### Task 3: testdata `importscope`

**Files:**
- Create: `cmd/undocheck/testdata/src/importscope/defpkg/types.go`
- Create: `cmd/undocheck/testdata/src/importscope/consumer/bad_assign.go`
- Create: `cmd/undocheck/testdata/src/importscope/consumer/good_read.go`

- [ ] **Step 1: 定义包 `defpkg`**

`defpkg/types.go`：

```go
package defpkg

// Player 聚合根（测试）。
//
// +cow:undoproxy-gen=true
type Player struct {
	Level int32
}
```

- [ ] **Step 2: 消费方坏例 / 好例**

`consumer/bad_assign.go`：

```go
package consumer

import "importscope/defpkg"

func BadAssign(p *defpkg.Player) {
	p.Level = 1 // want `cowbarewrite:`
}
```

`consumer/good_read.go`：

```go
package consumer

import "importscope/defpkg"

func GoodRead(p *defpkg.Player) int32 {
	return p.Level
}
```

- [ ] **Step 3: 扩展 `analyzer_test.go`**

```go
analysistest.Run(t, analysistest.TestData(), Analyzer, "barewrite", "homonym/domain", "importscope/consumer")
```

- [ ] **Step 4: 运行测试（预期失败至 Task 4 完成）**

```bash
go test ./cmd/undocheck/... -count=1
```

---

### Task 4: 重写 `monitoredForPass`

**Files:**
- Modify: `cmd/undocheck/analyzer.go`

- [ ] **Step 1: 替换 `monitoredForPass`**

```go
func monitoredForPass(pass *analysis.Pass) (*cowmon.MonitoredSet, error) {
	var sets []*cowmon.MonitoredSet
	if set, err := cowmon.BuildFromSyntax(pass.Pkg, pass.Files); err == nil {
		sets = append(sets, set)
	}
	for _, imp := range pass.Pkg.Imports() {
		if imp == nil {
			continue
		}
		path := imp.Path()
		if path == "" || path == pass.Pkg.Path() {
			continue
		}
		set, err := cowmon.LoadMonitoredCached(path)
		if err != nil {
			continue // 无 tag 或不可加载：跳过该 import
		}
		sets = append(sets, set)
	}
	return cowmon.Union(sets...), nil
}
```

- 删除 `cowmon.Imports(pass.Pkg, cowImportPath)` 分支及 `inspect.go` 中未使用的 `cowImportPath`（若仅 analyzer 使用则移入 analyzer 或删除）。

- [ ] **Step 2: 测试通过**

```bash
go test ./cmd/undocheck/... -count=1
go test ./internal/cowmon/... -count=1
```

- [ ] **Step 3: 本仓库 vet 冒烟**

```bash
go install ./cmd/undocheck
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

Expected: 与变更前相同（0 裸写违规或仅已知存量）；若有新诊断，修代码或夹具，**不在此任务扩 scope**。

---

### Task 5: 文档

**Files:**
- Modify: `docs/superpowers/specs/2026-05-25-bare-write-guard-design.md`（§6.3 增加链接）
- Modify: `cmd/undocheck/README.md`
- Modify: `docs/guide/bare-write-guard.md`
- Modify: `docs/superpowers/specs/2026-06-02-undocheck-import-scope-design.md` 状态 → 已实现（实现完成后）

- [ ] **Step 1: 更新三处用户可见文档**（检查范围：直接 import 带 tag 包即检查消费方裸写）

- [ ] **Step 2: 全量测试**

```bash
go test ./...
```

---

## 自审清单（计划 vs spec）

| Spec § | 任务 |
|--------|------|
| §2 目标 import 扩散 | Task 3–4 |
| §5.2 Union + Cache | Task 1–2 |
| §6 测试 | Task 1–4 |
| §7 文档 | Task 5 |
| §3 非目标（传递 import） | 无任务（符合） |

## 执行门禁

**在用户书面确认本 spec 与 plan 之前，不得修改 `cmd/undocheck` / `internal/cowmon` 实现代码。**

确认后执行选项：

1. **Subagent-Driven** — 每 Task 独立子代理 + 审查  
2. **Inline** — 本会话按 Task 1→5 顺序实现  

---

## 建议提交说明（实现完成后）

```
feat(undocheck): 对直接 import 带 undoproxy tag 的包检查裸写

合并本包与依赖定义包的 MonitoredSet，修复仅 import 自有 model 时
业务包未参与 cowbarewrite 的问题。补充 importscope testdata。
```
