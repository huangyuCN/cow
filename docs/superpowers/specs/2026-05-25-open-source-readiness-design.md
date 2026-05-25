# 开源就绪设计说明

| 项 | 值 |
|---|---|
| 状态 | 已批准（brainstorming 2026-05-25，目标 **B**） |
| 模块 | `github.com/huangyuCN/cow` |
| 许可证 | Apache License 2.0（已有 `LICENSE`） |
| 开源目标 | 吸引维护者与 PR；保留完整设计史（specs + plans + benchmarks） |

## 1. 目标

在首次公开 GitHub 仓库前，完成「可被 `go get` 使用 + 可被贡献者理解与参与」的最小就绪集：

1. **集成方**仅凭 `README.md` + `docs/guide/` 即可接入，不必阅读 superpowers。
2. **贡献者**可通过 `CONTRIBUTING.md` + `docs/superpowers/` 理解设计决策、实现计划与性能基线。
3. **仓库卫生**：无草稿泄漏、无多余 `go.mod` 依赖、有 CI 门禁、有安全报告渠道。
4. **首版发布**：打 tag（建议 `v0.1.0`）并附 Release Notes。

## 2. 非目标

- 将 `docs/superpowers/` 迁出主仓、`.gitignore` 或改名为 `docs/design/`（首期保持路径与链接稳定）。
- 建设 mdbook/docsy 等文档站。
- 首期新增 `examples/` 目录（列为开源后 P2）。
- 为「瘦身」删除 `docs/superpowers/plans/` 历史。
- 全文英文化（仅 README 可选 2–3 行英文摘要）。

## 3. 方案选择

| 方案 | 结论 |
|---|---|
| A. 集成优先，superpowers 附录化 | 不采用（与目标 B 弱匹配） |
| **B. 文档双轨 + 全量 superpowers 公开** | **采用** |
| C. superpowers 迁第二仓库 | 不采用（提高 PR 门槛） |

## 4. 文档与仓库结构

### 4.1 `docs/superpowers/` 策略

- **全部提交 GitHub**：`specs/`、`plans/`、`benchmarks/`（及未来 `adr/`、`brainstorm/` 按 `AGENTS.md` 约定）。
- **目录名暂不改**：避免大规模链接变更；在 `docs/README.md` 增加说明：「superpowers = 设计 / 实现计划 / 性能档案；名称来自内部 Agent 工作流，与 Cursor 插件无运行时依赖。」
- **与既有文档设计关系**：延续 [2026-05-25-project-documentation-design.md](2026-05-25-project-documentation-design.md)——`guide/` 面向集成方，superpowers 面向维护者与设计史。

### 4.2 禁止进入公开仓库的路径

| 路径 | 原因 |
|------|------|
| `new.md`、`save_historey.md` | 未整理草稿；内容已沉淀于 spec |
| `.superpowers/` | 仅本地临时产物（`AGENTS.md` 禁止入库） |
| benchmark 临时原始 `*.txt` | 仅归档 Markdown 对比表（见 `benchmarks/README.md`） |

发布前执行：`git grep -E 'MVP_REQUIREMENTS|save_historey|new\.md'` 应无命中（superpowers 历史 spec 内引用须在 P2 清理为 guide 链接）。

### 4.3 `docs/README.md` 贡献者阅读顺序（实现项）

在「设计与 benchmark」节前增加 **贡献者路径**：

1. 根 [README.md](../../README.md) → [guide/overview.md](guide/overview.md)
2. [toolchain/README.md](toolchain/README.md) + 相关 [cmd/*/README.md](../cmd/)
3. 当前主题 [superpowers/specs/](superpowers/specs/) → 对应 [superpowers/plans/](superpowers/plans/)
4. 性能数据 [superpowers/benchmarks/](superpowers/benchmarks/)

### 4.4 根目录文件

| 文件 | 处理 |
|------|------|
| `AGENTS.md` | **保留**；`CONTRIBUTING.md` 中链接 |
| `LICENSE` | 已存在，不变 |
| `README.md` | 已有门户结构；可选增加英文一句话摘要 |

## 5. 发布前工程清单（P0）

| # | 任务 | 验收 |
|---|------|------|
| 1 | `go mod tidy` | `go mod why -m k8s.io/apimachinery` 等显示主模块不需要未使用间接依赖 |
| 2 | GitHub Actions：`go test ./...` | PR / push 主分支通过 |
| 3 | 可选 CI job：`go install ./cmd/undocheck` + `go vet -vettool=... ./...` | 本仓自检通过 |
| 4 | 确认 `go.mod` 模块路径与 GitHub 远程仓库一致 | `github.com/huangyuCN/cow` |
| 5 | 确认未跟踪草稿未 `git add` | 工作区无 `new.md`、`save_historey.md` |
| 6 | 首版 tag + GitHub Release | 含能力边界、文档地图、Go 版本要求 |

### 5.1 建议 CI 工作流骨架

```yaml
# .github/workflows/ci.yml
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25.x'
      - run: go test ./...
```

`undocheck` vet job 可作为同一 workflow 第二 job 或首期省略（实现计划中二选一）。

## 6. 社区与治理（P1）

| 文件 | 内容要点（中文） |
|------|------------------|
| `CONTRIBUTING.md` | Go 1.25；TDD；中文注释；新功能：spec → plan → PR；benchmark 归档约定链到 `docs/superpowers/benchmarks/README.md`；链到 `AGENTS.md` |
| `SECURITY.md` | 漏洞报告渠道（GitHub Security Advisory 或邮箱） |
| `.gitignore` | 增补：`.worktrees/`、`.DS_Store`、benchmark 临时 `*.txt`（可选 `*.test` 等按团队习惯） |

可选（不阻塞首 tag）：

- `.github/ISSUE_TEMPLATE/`：bug / feature（feature 提示先查现有 spec）

## 7. 开源后维护（P2）

| # | 任务 |
|---|------|
| 1 | 清理 superpowers 内对 `MVP_REQUIREMENTS.md`、`new.md`、`save_historey.md` 的引用 → 改为 `docs/guide/*` 链接 |
| 2 | 新增 `examples/` 最小可运行样例 |
| 3 | README 英文摘要（2–3 行） |

## 8. 与集成文档的边界（防混淆）

- 根 `README` 文档表：**默认推荐** `docs/guide/`；superpowers 标注为「维护者 / 设计档案」。
- `docs/guide/*` **不得**要求读者先读 plans；仅可链接 benchmarks **摘要**或结论句。
- 新贡献的功能：先更新或新增 `docs/superpowers/specs/`，再 `plans/`，最后代码；合并时同步 `docs/guide/` 若影响集成方行为。

## 9. 验收标准（首版公开）

1. 远程仓库可 clone；`go test ./...` 与 CI 一致通过。
2. `docs/superpowers/{specs,plans,benchmarks}` 均在默认分支可见。
3. 无 `new.md` / `save_historey.md` / `.superpowers/` 在版本库中。
4. 存在 `CONTRIBUTING.md`、`SECURITY.md`、CI workflow。
5. 存在至少一个 release tag 与 Release Notes。
6. `docs/README.md` 含贡献者阅读顺序（§4.3）。

## 10. 参考

- 文档拓扑：[2026-05-25-project-documentation-design.md](2026-05-25-project-documentation-design.md)
- Agent 约定：仓库根 [AGENTS.md](../../../AGENTS.md)
- Benchmark 归档：[../benchmarks/README.md](../benchmarks/README.md)
