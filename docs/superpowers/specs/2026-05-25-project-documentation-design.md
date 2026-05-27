# 项目文档体系设计说明

| 项 | 值 |
|---|---|
| 状态 | 已批准（brainstorming 2026-05-25） |
| 模块 | `github.com/huangyuCN/cow` |
| 读者 | **C**：集成方（业务/游戏服）+ 维护者（本仓库）；分工见 §2 |
| 示例策略 | **D**：以 `*_test.go` / `func Example*` 为主（`go test` 校验）；稳定后可加 `examples/` |

## 1. 目标

建立可长期维护的正式文档体系，使开发者无需翻阅临时草稿或 superpowers 设计稿即可：

1. 理解 **cow 解决什么问题、能力边界、如何使用**（根 `README.md`）。
2. 按功能查阅 **集成指南**（`docs/guide/`，每功能有示例与边界说明）。
3. 理解 **工具链实现与维护**（`docs/toolchain/` + `cmd/*/README.md`）。
4. 清理 **`需求草稿.md（已删除）`、`new.md`、`save_historey.md`**，避免与正式文档并存或交叉引用。

## 2. 非目标

- 将 `docs/superpowers/` 迁出或改写为面向集成方的手册（superpowers 仍为设计/计划/benchmark 档案）。
- 第一期建设文档站（mdbook、docsy 等）。
- 第一期新增 `examples/` 端到端 demo 目录（列为二期可选）。
- 为 `internal/*` 每个子包单独写长篇 README（仅在 toolchain 与 cmd README 中摘要）。

## 3. 方案选择

| 方案 | 结论 |
|---|---|
| 1. README 集权（几乎无 `docs/guide`） | 不采用（根 README 过长、角色混杂） |
| **2. 枢纽 + 分册** | **采用** |
| 3. 文档站生成 | 不采用（当前阶段过重） |

## 4. 文档拓扑

```text
README.md                    ← 门户：问题、边界、5 分钟上手、文档地图
doc.go                       ← 短包注释 + 链接 README / docs/guide
docs/
  README.md                  ← 总索引（集成方 / 维护者 / 设计档案）
  guide/                     ← 集成方：按功能多文件
  toolchain/                 ← 维护者：工具链总览 + 类型图规则摘要
  superpowers/               ← 不变：spec / plan / benchmark
cmd/
  undoproxy-gen/README.md
  undocheck/README.md
  undorewrite/README.md
```

### 4.1 分工表

| 路径 | 读者 | 内容类型 |
|------|------|----------|
| `README.md` | 所有人 | 问题陈述、能力边界、最小上手、文档地图 |
| `docs/README.md` | 所有人 | 索引：guide / toolchain / superpowers |
| `docs/guide/*` | 集成方 | 用户可见功能：步骤、示例、FAQ、边界 |
| `docs/toolchain/*` | 维护者 | 三工具协作、与 `internal/cowgen`、`cowmon` 关系 |
| `cmd/*/README.md` | 维护者 | 单命令架构、flags、源码地图、链到 spec |
| `docs/superpowers/*` | 维护者/历史 | 设计决策与 benchmark；guide 仅链接结论 |

### 4.2 交叉链接规则

- 根 `README` → `docs/README.md` → 各 `guide` 页。
- 每个 `guide` 页底部：**维护细节** → 对应 `cmd/*/README` 或 `docs/toolchain/`。
- 每个 `cmd/README` 顶部：**集成用法** → 对应 `docs/guide/*.md`。
- Benchmark 数字：guide `overview.md` 引用 `docs/superpowers/benchmarks/*.md` 摘要表，不复制全文。

## 5. 根 `README.md` 纲要

必备章节（中文）：

1. **项目简介**：单协程聚合根 Undo Log 写代理；失败回滚、成功路径不 DeepCopy 业务数据。
2. **解决的问题**：长链路补偿脆弱；全量 DeepCopy CPU/alloc/GC 成本高。
3. **前提**：宿主对聚合根提供单 goroutine（或等价）串行写保证。
4. **能力边界（Non-goals）**：
   - 无 `TxContext` 并发安全。
   - 无运行期裸写检测（仅 `undocheck` 静态分析）。
   - `undoproxy-gen` 同包嵌套类型图；不支持 `interface{}`、channel、func 作为受监控容器元素类型。
   - 不捆绑具体 Actor/HTTP 框架。
5. **5 分钟上手**：模块路径、`+cow:undoproxy-gen`、`go generate`、`TxContext` 模式、`go install ./cmd/undocheck` + `go vet`。
6. **文档地图**：`docs/guide/`、`docs/toolchain/`、`cmd/`、`docs/superpowers/benchmarks/`。
7. **许可**（`LICENSE`）。

内容从 `需求草稿.md（已删除）` **提炼**；不保留 PRD 模板代码块。

## 6. `docs/guide/` 功能目录

每篇固定结构：**概述 → 前置条件 → 步骤 → 示例 → 边界/FAQ → 相关链接**。

| 文件 | 功能 |
|------|------|
| `README.md` | guide 索引 |
| `overview.md` | Undo Log 架构、与 DeepCopy 对比；链 benchmark 摘要 |
| `tx-context.md` | `TxContext`、`sync.Pool`、`Rollback`/`Reset`、commit vs rollback |
| `codegen-undoproxy.md` | `+cow:undoproxy-gen`、`go:generate`、生成文件约定 |
| `proxy-api.md` | `Put*` / `Append*` / `Get*ForWrite` / `CloneForWrite` 语义 |
| `bare-write-guard.md` | `undocheck`、`cowbarewrite`、白名单、`//cow:allow-bare-write` |
| `migration-undorewrite.md` | `undorewrite` dry-run、`-w`、`inject-ctx` |
| `integration-checklist.md` | 接入检查清单（generate → vet → CI） |
| `limitations.md` | 集中 Non-goals；Unmarshal 后仍须代理写的约定 |

### 6.1 示例策略（D）

- 文档内：**短片段** + 明确指向 `path/to/file.go` 或 `func ExampleXxx`。
- 实施阶段在根包补充 `func Example...`（当前仓库尚无 Example）；与 `player_test.go` 中 `runScopedCommit` / `runScopedWithRollback` 对齐。
- 片段须可通过 `go test ./...` 校验；不在 CI 解析 Markdown 抽代码。
- 二期可选：`examples/integration/` + guide 链接。

### 6.2 deepcopy 说明

不单独成篇；在 `overview.md` 注明 **k8s deepcopy-gen 仅作 benchmark 对照基线**，运行路径不依赖请求级 DeepCopy。生成与更新步骤写在 `codegen-undoproxy.md` 脚注或 `overview` 一小节。

## 7. `docs/toolchain/` 与 `cmd/*/README`

### 7.1 `docs/toolchain/README.md`

- 三工具流水线：`undoproxy-gen` → 业务使用生成 API → `undocheck` 守门 → `undorewrite` 迁移。
- 表：命令 | 输入 | 输出 | 依赖 internal 包。

### 7.2 `docs/toolchain/type-graph.md`（推荐）

- `cowgen.BuildGraph` 与 `cowmon` 监控集合的**同一规则**摘要（根标记、同包可达）。
- 避免生成器与分析器文档不一致。

### 7.3 各 `cmd/*/README.md` 模板

| 节 | 内容 |
|----|------|
| 职责 | 一句话 + 架构特点 |
| 能力边界 | 与该命令 design spec 非目标一致 |
| 安装与用法 | `go install`、flags、退出码 |
| 典型 CI | 可复制命令块 |
| 源码地图 | 主 `.go` 文件职责表 |
| 链接 | → `docs/superpowers/specs/...-design.md`；→ `docs/guide/...` |

| 命令 | 对应 guide |
|------|------------|
| `undoproxy-gen` | `codegen-undoproxy.md`、`proxy-api.md` |
| `undocheck` | `bare-write-guard.md` |
| `undorewrite` | `migration-undorewrite.md` |

## 8. 临时文件清理

| 文件 | 动作 |
|------|------|
| `需求草稿.md（已删除）` | 删除；痛点/前提迁入 `README.md`、`docs/guide/overview.md` |
| `new.md` | 删除；内容已由 superpowers spec 覆盖 |
| `save_historey.md` | 删除 |
| 全库引用 | 实施时 `grep` 三文件名并改为正式文档链接或删除「需求来源」行 |

**禁止**在正式文档中链接上述三文件。

### 8.1 已有 superpowers spec 的「需求来源」

将指向 `需求草稿.md（已删除）` / `new.md` 的行改为：

- `docs/guide/overview.md`，或
- 「需求已并入项目文档（2026-05-25）」

不在 spec 内复述 PRD 全文。

## 9. `doc.go` 更新

包注释保留最短安装提示，并增加：

```go
// 完整说明见仓库 README.md 与 docs/guide/。
```

避免 `doc.go` 重复长篇指南。

## 10. 语言与命名

- 用户文档：**中文**。
- 保留英文：命令名、`flag`、分析器名 `cowbarewrite`、go generate tag、`+cow:undoproxy-gen`。

## 11. 验收标准

1. 根目录存在 `README.md`，含 §5 全部章节。
2. `docs/README.md`、`docs/guide/`（§6 所列文件）、`docs/toolchain/README.md`（及 `type-graph.md`）存在且互相链接正确。
3. `cmd/undoproxy-gen`、`cmd/undocheck`、`cmd/undorewrite` 各有 `README.md`。
4. `需求草稿.md（已删除）`、`new.md`、`save_historey.md` 已删除；`git grep` 无引用。
5. 至少新增 2 个 `func Example`（`TxContext` 与 `Player` 代理各一），`go test ./...` 通过。
6. `doc.go` 指向正式文档路径。

## 12. 实施顺序建议（供 writing-plans 展开）

1. 撰写根 `README.md`、`docs/README.md`。
2. 撰写 `docs/guide/` 各篇（先索引与 `overview`、`tx-context`）。
3. 撰写 `docs/toolchain/` 与三个 `cmd/README.md`。
4. 补充 `Example_*` 测试；更新 `doc.go`。
5. 删除临时文件；修复 superpowers spec 中的需求来源链接。
6. 全库 `grep` 验证无死链。

---

*本 spec 批准后的下一步：由用户审阅文件 → 调用 writing-plans 生成实现计划。*
