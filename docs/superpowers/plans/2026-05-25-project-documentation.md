# 项目文档体系 Implementation Plan

> **状态：已实现**（截至 2026-05-27；本计划为历史执行记录，勿按未勾选步骤重复开发）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立根 README、`docs/guide` 集成手册、`docs/toolchain` + `cmd/*/README` 维护文档；补充 `Example` 测试；删除临时草稿并修复 superpowers 中的死链引用。

**Architecture:** 枢纽 + 分册（见 [2026-05-25-project-documentation-design.md](../specs/2026-05-25-project-documentation-design.md)）。集成方读 `docs/guide/`；维护者读 `cmd/README` + `docs/toolchain/`；设计档案保留在 `docs/superpowers/`。示例策略 D：文档引用 `*_test.go` / `func Example*`。

**Tech Stack:** Markdown（中文）、Go `Example` 测试、`go test` / `go vet` 验收。

**设计 spec：** [docs/superpowers/specs/2026-05-25-project-documentation-design.md](../specs/2026-05-25-project-documentation-design.md)

---

## 文件清单（创建 / 修改 / 删除）

| 操作 | 路径 |
|------|------|
| 创建 | `README.md` |
| 创建 | `docs/README.md` |
| 创建 | `docs/guide/README.md` … `limitations.md`（共 9 篇，见 Task 4） |
| 创建 | `docs/toolchain/README.md`、`docs/toolchain/type-graph.md` |
| 创建 | `cmd/undoproxy-gen/README.md`、`cmd/undocheck/README.md`、`cmd/undorewrite/README.md` |
| 创建 | `doc_examples_test.go`（`func Example*`） |
| 修改 | `doc.go` |
| 修改 | 若干 `docs/superpowers/specs/*.md`（需求来源行，Task 8） |
| 删除 | `需求草稿.md（已删除）`、`new.md`、`save_historey.md` |

---

### Task 1: `Example` 测试（策略 D，先于 guide 撰写）

**Files:**
- Create: `doc_examples_test.go`
- 参考: `player_test.go`（`runScopedCommit`）、`bench_fixture_test.go`（`applySparseWrites`）

- [ ] **Step 1: 创建 `doc_examples_test.go`**

```go
package cow

import (
	"errors"
	"fmt"
)

// ExampleTxContext_rollback 演示失败路径下 Rollback 恢复聚合根状态。
func ExampleTxContext_rollback() {
	player := newBenchPlayer()
	beforeGold := player.Assets["gold"]

	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()

	applySparseWrites(player, ctx)
	_ = errors.New("business error") // 模拟中途失败

	if player.Assets["gold"] == beforeGold {
		fmt.Println("rolled back")
	}
	// Output: rolled back
}

// ExamplePlayer_sparseWrite 演示通过生成代理在 TxContext 下稀疏写。
func ExamplePlayer_sparseWrite() {
	player := newBenchPlayer()
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)

	player.PutAssets(ctx, "gold", 500)
	fmt.Println(player.Assets["gold"])
	// Output: 500
}
```

- [ ] **Step 2: 运行测试**

```bash
cd /Users/huangyu/work/golang/src/cow
go test -run='^Example' -v .
```

Expected: `PASS`，`ExampleTxContext_rollback` 与 `ExamplePlayer_sparseWrite` 均 OK。

- [ ] **Step 3: Commit（需用户明确同意后再执行）**

```bash
git add doc_examples_test.go
git commit -m "test: 为文档补充 ExampleTxContext 与 ExamplePlayer"
```

---

### Task 2: 根 `README.md`

**Files:**
- Create: `README.md`
- 参考提炼: `需求草稿.md（已删除）` §问题描述 / §PRD 背景（删除前阅读一次）

- [ ] **Step 1: 写入 `README.md`**

必备章节（中文，顺序固定）：

1. `# cow` + 一句话简介（Undo Log 写代理）
2. `## 解决的问题`（补偿脆弱、DeepCopy 成本 — 各 2–3 句）
3. `## 前提`（单 goroutine 串行写聚合根）
4. `## 能力边界`（bullets：无并发 TxContext、无运行期裸写检测、undoproxy-gen 同包图、不绑框架）
5. `## 快速开始`

```bash
go get github.com/huangyuCN/cow@latest   # 或 replace 本地路径

# 业务 types.go
# // +cow:undoproxy-gen=true
# type Player struct { ... }

go install ./cmd/undoproxy-gen
go generate ./...    # 需包内 //go:generate undoproxy-gen ...

go install ./cmd/undocheck
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

6. `TxContext` 最小模式（10 行内代码块，链 `docs/guide/tx-context.md`）
7. `## 文档`（链 `docs/README.md`、`docs/guide/`、`docs/toolchain/`、`docs/superpowers/benchmarks/`）
8. `## License`（MIT，见 `LICENSE`）

- [ ] **Step 2: 人工检查**

打开 `README.md`，确认无链接到 `需求草稿.md（已删除）` / `new.md` / `save_historey.md`。

- [ ] **Step 3: Commit（可选，用户同意后）**

```bash
git add README.md
git commit -m "docs: 添加根 README（问题、边界、快速开始）"
```

---

### Task 3: `docs/README.md` 总索引

**Files:**
- Create: `docs/README.md`

- [ ] **Step 1: 写入索引**

结构：

```markdown
# cow 文档

## 集成方
| 文档 | 说明 |
|------|------|
| [guide/README.md](guide/README.md) | 功能使用手册（含示例） |

## 维护者
| 文档 | 说明 |
|------|------|
| [toolchain/README.md](toolchain/README.md) | 三工具流水线 |
| [../cmd/undoproxy-gen/README.md](../cmd/undoproxy-gen/README.md) | 代码生成器 |
| [../cmd/undocheck/README.md](../cmd/undocheck/README.md) | 裸写分析器 |
| [../cmd/undorewrite/README.md](../cmd/undorewrite/README.md) | 存量改写 |

## 设计与 benchmark
[superpowers/](superpowers/) — 设计 spec、实现 plan、性能归档
```

- [ ] **Step 2: Commit（可选）**

```bash
git add docs/README.md
git commit -m "docs: 添加 docs 总索引"
```

---

### Task 4: `docs/guide/` 集成手册（9 篇）

**Files:**
- Create: `docs/guide/README.md`
- Create: `docs/guide/overview.md`
- Create: `docs/guide/tx-context.md`
- Create: `docs/guide/codegen-undoproxy.md`
- Create: `docs/guide/proxy-api.md`
- Create: `docs/guide/bare-write-guard.md`
- Create: `docs/guide/migration-undorewrite.md`
- Create: `docs/guide/integration-checklist.md`
- Create: `docs/guide/limitations.md`

每篇末尾固定一节 `## 相关链接`（指向 cmd README / toolchain / superpowers spec）。

- [ ] **Step 1: `docs/guide/README.md`**

表格列出 §6 全部 8 篇 + 一句话说明。

- [ ] **Step 2: `docs/guide/overview.md`**

- Undo Log 原理（不拷贝数据，只记录逆操作；成功 `Reset`，失败 `Rollback`）
- 与 DeepCopy 对比：嵌入 benchmark 摘要表（摘自 [cow-undo-log-benchmark.md](../superpowers/benchmarks/cow-undo-log-benchmark.md)：`Rollback` ~114 ns/op vs `DeepCopyGen` ~9961 ns/op，勿复制全文）
- 链 [mega benchmark](../superpowers/benchmarks/cow-mega-player-benchmark.md) 可选一句
- deepcopy-gen **仅 benchmark 基线**（运行路径不用请求级 DeepCopy）

- [ ] **Step 3: `docs/guide/tx-context.md`**

- API：`AddUndo`、`Rollback`、`Reset`、`txPool`
- **Commit 模式**：`runScopedCommit`（`player_test.go`）
- **Rollback 模式**：`runScopedWithRollback`（`player_test.go`）
- 示例：链 `doc_examples_test.go` 的 `ExampleTxContext_rollback`；粘贴 `runScopedCommit` 核心 15 行

- [ ] **Step 4: `docs/guide/codegen-undoproxy.md`**

- 根类型标记：`// +cow:undoproxy-gen=true`（`types.go` `Player`）
- 包级：`doc.go` 中 `// +cow:undoproxy-gen=package`
- `undo_proxy_generate.go` 的 `//go:generate` 行（原文照抄）
- 输出 `zz_generated.undo_proxy.go`，**勿手改**
- 修改类型后：`go generate` + 提交生成文件

- [ ] **Step 5: `docs/guide/proxy-api.md`**

按类别说明（各 1 段 + 1 行示例）：

| 模式 | 示例方法 |
|------|----------|
| 标量 Put | `PutAssets(ctx, "gold", v)` |
| map Put | `PutSkills(ctx, k, val)` |
| slice Append | `AppendItems(ctx, item)` |
| 指针 ForWrite | `GetMainHeroForWrite(ctx)` → 再 `PutLevel` |
| struct Clone | `CloneForWrite()` |

- 完整稀疏写：链 `bench_fixture_test.go` 的 `applySparseWrites`
- 链 `ExamplePlayer_sparseWrite`

- [ ] **Step 6: `docs/guide/bare-write-guard.md`**

```bash
go install ./cmd/undocheck
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
# 或（分析器已编入 undocheck 二进制时）:
go vet -cowbarewrite ./...
```

- 分析器名 `cowbarewrite`
- 行级逃逸：`//cow:allow-bare-write`
- 跳过文件规则摘要：`internal/cowfile/skip.go`（`zz_generated*`、`*_fixture.go`）
- 好坏例：`cmd/undocheck/testdata/src/barewrite/`

- [ ] **Step 7: `docs/guide/migration-undorewrite.md`**

```bash
go install ./cmd/undorewrite
undorewrite ./yourpkg/...              # dry-run
undorewrite -w ./yourpkg/...           # 写回
undorewrite -inject-ctx=pool -w ./...  # 注入 pool 模式（按需）
```

- flags：`-cow`、`-w`、`-ctx`、`-inject-ctx`、`-pool-var`（摘自 `cmd/undorewrite/main.go`）
- 改写后必须 `go vet` 通过
- 链 `cmd/undorewrite/README.md`

- [ ] **Step 8: `docs/guide/integration-checklist.md`**

Checkbox 列表：

- [ ] 根类型 `+cow:undoproxy-gen=true`
- [ ] `go generate` 已跑且 `zz_generated.undo_proxy.go` 已提交
- [ ] 业务写路径经 `Put*` / `Get*ForWrite`
- [ ] CI：`go vet -vettool=.../undocheck` 或 `-cowbarewrite`
- [ ] 存量：`undorewrite` dry-run 审查后 `-w`

- [ ] **Step 9: `docs/guide/limitations.md`**

集中 Non-goals（与根 README 一致并展开 Unmarshal 约定：反序列化后仍须代理写，静态分析看不见反射写）。

- [ ] **Step 10: 验证 guide 内链**

```bash
cd /Users/huangyu/work/golang/src/cow
for f in docs/guide/*.md; do grep -l '需求草稿\|new\.md\|save_historey' "$f" && exit 1; done
echo "guide links OK"
```

- [ ] **Step 11: Commit（可选）**

```bash
git add docs/guide/
git commit -m "docs: 添加集成方 guide 手册"
```

---

### Task 5: `docs/toolchain/`

**Files:**
- Create: `docs/toolchain/README.md`
- Create: `docs/toolchain/type-graph.md`
- 参考: `internal/cowgen/graph.go`、`internal/cowmon/load.go`、`cmd/undocheck/analyzer.go`

- [ ] **Step 1: `docs/toolchain/README.md`**

Mermaid 或 ASCII 流水线：

`types.go (+tag)` → `undoproxy-gen` → `zz_generated.undo_proxy.go` → 业务写 → `undocheck` → （可选）`undorewrite`

表格：

| 命令 | 输入 | 输出 | internal |
|------|------|------|----------|
| undoproxy-gen | import path | `zz_generated.undo_proxy.go` | cowgen |
| undocheck | packages | diagnostics | cowmon, cowfile |
| undorewrite | `./patterns` | 改写源文件 | cowmon, cowproxy |

- [ ] **Step 2: `docs/toolchain/type-graph.md`**

说明（中文）：

- 根：`// +cow:undoproxy-gen=true` 的 struct
- 可达：同包内嵌套 struct（与 `cowgen.BuildGraph` 一致）
- `undocheck` 的 `cowmon.BuildFromSyntax` / `LoadMonitored` 须与生成器同一图
- 不支持：跨包嵌套字段、`interface{}`/channel/func 作容器元素

- [ ] **Step 3: Commit（可选）**

```bash
git add docs/toolchain/
git commit -m "docs: 添加 toolchain 维护文档"
```

---

### Task 6: `cmd/*/README.md`

**Files:**
- Create: `cmd/undoproxy-gen/README.md`
- Create: `cmd/undocheck/README.md`
- Create: `cmd/undorewrite/README.md`

每篇顶部一行：**集成用法** → 对应 `docs/guide/*.md`。

- [ ] **Step 1: `cmd/undoproxy-gen/README.md`**

| 节 | 要点 |
|----|------|
| 职责 | `go/packages` 加载 → `cowgen.BuildGraph` → template 输出 |
| 用法 | `undoproxy-gen --output-file zz_generated.undo_proxy.go IMPORT_PATH` |
| 源码地图 | `main.go` Run、`loader.go` load、`emit.go` 写文件 |
| 边界 | 同包图；不生成 TxContext |
| 链接 | `docs/guide/codegen-undoproxy.md`；spec `2026-05-25-undoproxy-codegen-design.md` |

- [ ] **Step 2: `cmd/undocheck/README.md`**

| 节 | 要点 |
|----|------|
| 职责 | `go/analysis` 分析器 `cowbarewrite` |
| 用法 | `go install ./cmd/undocheck`；`go vet -vettool=...` |
| 源码地图 | `analyzer.go`、`inspect.go`、`whitelist.go`（skip 委托 `cowfile`） |
| 链接 | `docs/guide/bare-write-guard.md`；spec `2026-05-25-bare-write-guard-design.md` |

- [ ] **Step 3: `cmd/undorewrite/README.md`**

| 节 | 要点 |
|----|------|
| 职责 | AST 裸写 → 代理调用；默认 dry-run |
| 用法 | 全文 flags 表（`main.go`） |
| 源码地图 | `rewrite.go`、`load.go`、`config.go`、`ctx.go` |
| 链接 | `docs/guide/migration-undorewrite.md`；spec `2026-05-25-undorewrite-codemod-design.md` |

- [ ] **Step 4: Commit（可选）**

```bash
git add cmd/undoproxy-gen/README.md cmd/undocheck/README.md cmd/undorewrite/README.md
git commit -m "docs: 为 cmd 子命令添加 README"
```

---

### Task 7: 更新 `doc.go`

**Files:**
- Modify: `doc.go`

- [ ] **Step 1: 在包注释末尾追加一行**

```go
// 完整说明见仓库 README.md 与 docs/guide/。
```

保留现有 `go install` / `go vet` 两行。

- [ ] **Step 2: 验证 godoc**

```bash
go doc github.com/huangyuCN/cow 2>&1 | head -20
```

- [ ] **Step 3: Commit（可选）**

```bash
git add doc.go
git commit -m "docs: doc.go 指向 README 与 guide"
```

---

### Task 8: 删除临时文件并修复 superpowers 引用

**Files:**
- Delete: `需求草稿.md（已删除）`、`new.md`、`save_historey.md`
- Modify:
  - `docs/superpowers/specs/2026-05-25-undoproxy-codegen-design.md`
  - `docs/superpowers/specs/2026-05-25-cow-undo-log-design.md`
  - `docs/superpowers/specs/2026-05-25-undorewrite-codemod-design.md`
  - `docs/superpowers/specs/2026-05-25-bare-write-guard-design.md`
  - `docs/superpowers/specs/2026-05-25-mega-player-benchmark-design.md`
  - `docs/superpowers/plans/2026-05-25-undoproxy-codegen.md`（注释中的 `new.md`）
  - `docs/superpowers/plans/2026-05-25-cow-undo-log.md`（可选脚注任务删除）

- [ ] **Step 1: 批量替换「需求来源」行**

将表格中 `需求来源` 含 `需求草稿` / `new.md` / `save_historey` 的行改为：

```markdown
| 需求来源 | 已并入 [docs/guide/overview.md](../../guide/overview.md)（2026-05-25） |
```

`undorewrite` spec 原为 `save_historey.md` → 同上或写「与 undocheck 互补，见 guide/migration-undorewrite.md」。

- [ ] **Step 2: 清理正文中的 `new.md` 引用**

`undoproxy-codegen-design.md`：
- §1「符合 new.md §3–§4」→「符合项目 Undo 代理语义（见 guide/proxy-api.md）」
- §9 标题「与 new.md 对齐」→「生成语义」
- 文末 `- new.md` 列表项删除或改为 guide 链接

`undoproxy-codegen plan` 注释 `对齐 new.md` → `对齐嵌套 map 夹具`

- [ ] **Step 3: 删除三个临时文件**

```bash
rm 需求草稿.md（已删除） new.md save_historey.md
```

- [ ] **Step 4: 全库 grep 验收**

```bash
git grep -E '需求草稿|save_historey|new\.md' -- ':!docs/superpowers/specs/2026-05-25-project-documentation-design.md' || true
```

Expected: 仅剩 project-documentation-design.md 中**描述删除动作**的提及；其余为 0。

- [ ] **Step 5: Commit（可选）**

```bash
git add -A
git commit -m "docs: 删除临时草稿并修复 superpowers 需求来源链接"
```

---

### Task 9: 终验

- [ ] **Step 1: 测试全绿**

```bash
go test ./...
```

- [ ] **Step 2: 对照 spec §11 验收清单**

| # | 检查 |
|---|------|
| 1 | 根 `README.md` 含 7 节 |
| 2 | `docs/README.md` + `docs/guide/*`（9 文件）+ `docs/toolchain/*`（2 文件） |
| 3 | 三个 `cmd/README.md` |
| 4 | 临时三文件已删、`git grep` 无违规引用 |
| 5 | ≥2 个 `Example`，`go test -run Example` 通过 |
| 6 | `doc.go` 含 README/guide 指向 |

- [ ] **Step 3: 建议用户 commit spec + plan（若尚未提交）**

```bash
git add docs/superpowers/specs/2026-05-25-project-documentation-design.md \
        docs/superpowers/plans/2026-05-25-project-documentation.md
git commit -m "docs: 项目文档体系设计与实施计划"
```

---

## Plan 自检（对照 spec）

| Spec 章节 | 任务 |
|-----------|------|
| §4 拓扑 | Task 2–7 |
| §5 根 README | Task 2 |
| §6 guide 9 篇 | Task 4 |
| §6.1 Example D | Task 1 |
| §7 toolchain + cmd README | Task 5–6 |
| §8 临时文件 | Task 8 |
| §9 doc.go | Task 7 |
| §11 验收 | Task 9 |

无 TBD；无「稍后补充」步骤。

---

## 执行方式（完成后由负责人选择）

**Plan 已保存至：** `docs/superpowers/plans/2026-05-25-project-documentation.md`

**两种执行选项：**

1. **Subagent-Driven（推荐）** — 每 Task 派发子 agent，任务间你做 review  
2. **Inline Execution** — 本会话按 Task 1→9 连续实施，关键节点停顿确认  

回复 **1** 或 **2**（或「开始实施」+ 偏好）即可开工。未经你同意不会 `git commit` / `git push`。
