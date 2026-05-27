# 开源就绪 Implementation Plan

> **状态：已实现**（截至 2026-05-27；本计划为历史执行记录，勿按未勾选步骤重复开发）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成首次 GitHub 公开前的工程与社区就绪项（CI、`go.mod` 清理、贡献/安全文档、`docs/README` 贡献者路径），满足 [开源就绪设计 spec](../specs/2026-05-25-open-source-readiness-design.md) 的 P0/P1 验收。

**Architecture:** 文档双轨不变——集成方 `docs/guide/`；全量 `docs/superpowers/` 公开。本计划只新增/修改仓库治理与门户文件，不改运行时逻辑。开发在**当前工作目录**进行（`AGENTS.md` 禁止 `.worktrees/`）。

**Tech Stack:** Go 1.25、GitHub Actions、`go mod tidy`、`go test` / `go vet` + `undocheck`。

**设计 spec：** [docs/superpowers/specs/2026-05-25-open-source-readiness-design.md](../specs/2026-05-25-open-source-readiness-design.md)

**范围外（P2，本计划不实施）：** 清理 superpowers 历史死链、`examples/`、README 英文摘要、Issue 模板。

---

## 文件清单

| 操作 | 路径 |
|------|------|
| 修改 | `go.mod`、`go.sum`（`go mod tidy`） |
| 修改 | `.gitignore` |
| 创建 | `.github/workflows/ci.yml` |
| 创建 | `CONTRIBUTING.md` |
| 创建 | `SECURITY.md` |
| 修改 | `docs/README.md` |
| 修改 | `README.md`（文档表一行说明） |
| 已存在 | `docs/superpowers/specs/2026-05-25-open-source-readiness-design.md` |

---

### Task 1: 清理 `go.mod` 间接依赖

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 运行 tidy**

```bash
cd /Users/huangyu/work/golang/src/cow
go mod tidy
```

- [ ] **Step 2: 确认未使用 k8s 模块已从 require 移除**

```bash
go mod why -m k8s.io/apimachinery 2>&1 | head -3
```

Expected: 含 `(main module does not need module k8s.io/apimachinery)` 或命令报错「module not in go.mod」。

- [ ] **Step 3: 全量测试**

```bash
go test ./...
```

Expected: 全部 `ok`。

- [ ] **Step 4: 提交（须经仓库所有者确认）**

```bash
git add go.mod go.sum
git commit -m "chore: go mod tidy，移除未使用间接依赖"
```

---

### Task 2: 补强 `.gitignore`

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: 将 `.gitignore` 设为以下内容**

```
.idea
.DS_Store
.worktrees/

# benchmark 临时对比输出（归档用 Markdown，见 docs/superpowers/benchmarks/README.md）
bench_*.txt
*.bench.txt
```

- [ ] **Step 2: 确认草稿仍不被跟踪**

```bash
git status --short new.md save_historey.md 2>/dev/null; test ! -f .git/index  # 可选
ls new.md save_historey.md 2>/dev/null && echo "WARN: 草稿仍在工作区，勿 git add" || echo "OK"
```

- [ ] **Step 3: 提交**

```bash
git add .gitignore
git commit -m "chore: 扩充 gitignore（worktree、benchmark 临时文件）"
```

---

### Task 3: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: 创建工作流文件**

```yaml
name: CI

on:
  push:
    branches: [main, master]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.x"
          cache: true
      - run: go test ./...

  vet-undocheck:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.x"
          cache: true
      - run: go install ./cmd/undocheck
      - run: go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

说明：默认分支若非 `main`/`master`，在 Step 1 中改为实际默认分支名。

- [ ] **Step 2: 本地复现 vet job**

```bash
go install ./cmd/undocheck
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

Expected: 无报错退出码 0。

- [ ] **Step 3: 提交**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: 添加 go test 与 undocheck go vet 工作流"
```

---

### Task 4: `CONTRIBUTING.md`

**Files:**
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: 创建文件（全文）**

```markdown
# 贡献指南

感谢考虑为 [cow](https://github.com/huangyuCN/cow) 做贡献。本仓库面向**集成方**与**维护者**两类读者，协同时请先确认你改的是哪一类文档或代码。

## 环境

- **Go 1.25**（与 `go.mod` 一致）
- 克隆后：`go test ./...`
- 裸写检查（修改业务写路径时）：
  ```bash
  go install ./cmd/undocheck
  go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
  ```

## 文档在哪里

| 读者 | 路径 |
|------|------|
| 集成方 | [docs/guide/](docs/guide/) |
| 工具链维护 | [docs/toolchain/](docs/toolchain/)、[cmd/*/README.md](cmd/) |
| 设计 / 计划 / 性能档案 | [docs/superpowers/](docs/superpowers/) |

**superpowers** 目录名来自内部 Agent 工作流，与 Cursor 插件无运行时依赖；存放已批准的 spec、实现 plan 与 [benchmark 归档](docs/superpowers/benchmarks/README.md)。

贡献者建议阅读顺序见 [docs/README.md#贡献者](docs/README.md)。

## 开发约定

完整约定见 [AGENTS.md](AGENTS.md)，摘要如下：

- **TDD**：先写/更新测试，再实现，再重构。
- **注释语言**：手写代码与 Go doc 注释使用**中文**（专有名词、生成代码除外）。
- **规模**：单文件 ≤500 行，单函数 ≤50 行；重复逻辑提取公共函数。
- **命名**：导出符号不得以包名 `cow` 为前缀。

## 新功能流程（目标：可审查的设计史）

1. 在 `docs/superpowers/specs/` 新增或更新设计 spec（brainstorming 批准后再写代码）。
2. 在 `docs/superpowers/plans/` 添加实现 plan（可勾选任务列表）。
3. 实现代码 + 测试；若影响集成方行为，同步 `docs/guide/`。
4. 复杂逻辑补充 benchmark；经确认后按 [benchmarks/README.md](docs/superpowers/benchmarks/README.md) 归档对比表。
5. 提 PR；CI 须通过（`go test` + `undocheck` vet）。

## 请勿提交

- 根目录草稿 `new.md`、`save_historey.md`
- 目录 `.superpowers/`（本地临时产物）
- 未整理的 benchmark 原始 `.txt`（仅提交 Markdown 归档）

## 许可证

贡献即表示你同意在 [Apache License 2.0](LICENSE) 下授权你的贡献。
```

- [ ] **Step 2: 提交**

```bash
git add CONTRIBUTING.md
git commit -m "docs: 添加 CONTRIBUTING.md"
```

---

### Task 5: `SECURITY.md`

**Files:**
- Create: `SECURITY.md`

- [ ] **Step 1: 创建文件**

将 `YOUR_EMAIL@example.com` 替换为实际联系邮箱，或删除邮箱行仅保留 GitHub Advisory 说明。

```markdown
# 安全策略

## 报告漏洞

请勿在公开 Issue 中讨论可利用的安全问题。

优先通过 GitHub **Security Advisories**（仓库 → Security → Report a vulnerability）私下报告。

若无法使用上述方式，请发邮件至：**YOUR_EMAIL@example.com**（请替换为本计划实施前的真实地址）。

## 响应

维护者会在合理时间内确认收到，并在修复可用后协调披露（含致谢，若你同意）。

## 支持版本

| 版本 | 支持 |
|------|------|
| 最新 tag | 是 |
| 更早版本 | 视情况修复或建议升级 |
```

- [ ] **Step 2: 提交**

```bash
git add SECURITY.md
git commit -m "docs: 添加 SECURITY.md"
```

---

### Task 6: 更新 `docs/README.md`（贡献者路径 + superpowers 说明）

**Files:**
- Modify: `docs/README.md`

- [ ] **Step 1: 在「## 维护者」之前插入以下章节**

```markdown
## 贡献者

阅读顺序（维护 / 提 PR 前建议通读）：

1. 根 [README.md](../README.md) → [guide/overview.md](guide/overview.md)
2. [toolchain/README.md](toolchain/README.md) 与 [cmd/](../cmd/) 下各子命令 README
3. 当前主题的 [superpowers/specs/](superpowers/specs/) → 对应 [superpowers/plans/](superpowers/plans/)
4. 性能基线 [superpowers/benchmarks/](superpowers/benchmarks/)

协同时程与约定见根目录 [CONTRIBUTING.md](../CONTRIBUTING.md)、[AGENTS.md](../AGENTS.md)。

```

- [ ] **Step 2: 将原「## 设计与 benchmark」小节替换为**

```markdown
## 设计与 benchmark

[superpowers/](superpowers/) — 设计 spec、实现 plan、经归档的 benchmark 日志。

**说明：** 目录名 superpowers 来自内部 Agent 工作流，与 Cursor 插件无运行时依赖；内容为设计决策与性能档案，**集成方不必阅读**。新功能请先查阅是否已有相关 spec。
```

- [ ] **Step 3: 提交**

```bash
git add docs/README.md
git commit -m "docs: docs/README 增加贡献者路径与 superpowers 说明"
```

---

### Task 7: 更新根 `README.md` 文档表

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 将「## 文档」表格替换为**

```markdown
## 文档

| 文档 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 文档总索引（含贡献者阅读顺序） |
| [docs/guide/](docs/guide/) | **集成方**功能手册（推荐从此入手） |
| [docs/toolchain/](docs/toolchain/) | **维护者**工具链说明 |
| [cmd/](cmd/) | 各子命令 README |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献流程与约定 |
| [docs/superpowers/](docs/superpowers/) | **维护者**设计 spec / plan / benchmark 档案 |
```

- [ ] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: README 文档表区分集成方与维护者"
```

---

### Task 8: 发布前验收（本地）

**Files:** 无新增

- [ ] **Step 1: 草稿与禁止路径检查**

```bash
git grep -E 'save_historey|new\.md' -- ':!docs/superpowers/plans/2026-05-25-open-source-readiness.md' ':!docs/superpowers/specs/2026-05-25-open-source-readiness-design.md' ':!docs/superpowers/specs/2026-05-25-project-documentation-design.md' ':!docs/superpowers/plans/2026-05-25-project-documentation.md' && exit 1 || echo "OK: 无生产代码/ guide 引用草稿文件名"
```

说明：允许设计/plan 文档自身提到这些文件名；`docs/guide/` 与 `*.go` 中不应出现。

- [ ] **Step 2: 确认 guide 无草稿引用**

```bash
git grep -E '需求草稿|save_historey|new\.md' -- docs/guide/ && exit 1 || echo "OK"
```

- [ ] **Step 3: 测试 + vet**

```bash
go test ./...
go install ./cmd/undocheck
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

- [ ] **Step 4: 核对 spec §9 清单**

人工勾选 [开源就绪设计 spec §9](../specs/2026-05-25-open-source-readiness-design.md) 第 1–4、6 条；第 5 条（tag + Release）在 Task 9 完成。

---

### Task 9: 首次公开发布（需仓库所有者操作）

**Files:** 无（GitHub 远程操作）

- [ ] **Step 1: 推送默认分支**

```bash
git push -u origin HEAD
```

- [ ] **Step 2: 打 tag**

```bash
git tag -a v0.1.0 -m "首版开源：Undo Log 写代理 + undoproxy-gen / undocheck / undorewrite"
git push origin v0.1.0
```

- [ ] **Step 3: 创建 GitHub Release**

标题：`v0.1.0`  
正文模板：

```markdown
## 简介

单协程聚合根 Undo Log 写代理：业务失败时 `Rollback`，成功路径不 DeepCopy 业务数据。

## 要求

- Go 1.25+
- 宿主对聚合根提供单 goroutine 串行写保证

## 文档

- 集成：[docs/guide/](https://github.com/huangyuCN/cow/tree/main/docs/guide)
- 贡献：[CONTRIBUTING.md](https://github.com/huangyuCN/cow/blob/main/CONTRIBUTING.md)
- 设计档案：[docs/superpowers/](https://github.com/huangyuCN/cow/tree/main/docs/superpowers)

## 工具

- `undoproxy-gen` — 生成写代理
- `undocheck` — 裸写静态分析（`go vet`）
- `undorewrite` — 历史裸写 AST 改写
```

将 `main` 换为实际默认分支名。

---

## Plan 自检（相对 spec）

| Spec 条款 | 对应 Task |
|-----------|-----------|
| §4.3 贡献者路径 | Task 6 |
| §4.1 全量 superpowers | 不删改，仅文档说明 |
| §5 P0 tidy + CI | Task 1、3 |
| §5 P0 无草稿入库 | Task 2、8 |
| §6 P1 CONTRIBUTING / SECURITY / gitignore | Task 4、5、2 |
| §6 可选 Issue 模板 | 范围外 |
| §7 P2 | 范围外 |
| §9 tag + Release | Task 9 |

无 TBD；CI 含 `undocheck`（spec §5.1 二选一，本计划选用「包含」）。

---

## 执行方式

计划已保存。可选：

1. **Subagent-Driven（推荐）** — 每 Task 派发子 agent，任务间人工/审查检查点  
2. **Inline Execution** — 本会话按 Task 1→9 连续实施，Task 8–9 前与你确认 push/tag  

你更倾向哪一种？若希望我现在开始实施，直接回复「inline」或「subagent」即可。
