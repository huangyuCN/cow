# Agent 约定（全局）

## Superpowers 文档根目录

- **默认文档根目录**：`docs/superpowers/`
- **禁止**：将文档放到 `.superpowers/`（该目录只允许本地临时产物）
- **其他 Agent/Skill 生成的设计文档**：除 superpowers 文档外，其他 Agent 或 Skill 生成的设计文档也必须放到 `docs/` 目录下（按项目约定选择子目录）。

如需新增：
- 计划写到 `docs/superpowers/plans/`
- 设计写到 `docs/superpowers/specs/`
- 架构决策（ADR）写到 `docs/superpowers/adr/`（约定见该目录 `README.md`）
- brainstorming/草图写到 `docs/superpowers/brainstorm/`
- **经确认的 benchmark 对比日志**写到 `docs/superpowers/benchmarks/`（约定见该目录 `README.md`）

## 开发规范（全局）

- **Go 版本**：使用 **Go 1.25** 开发；杜绝使用已过时/弃用（deprecated）的接口、函数与用法（以 `go doc` / 官方 release notes 为准）。
- **开发位置**：不要在 git worktree 中开发（例如 `.worktrees/`）；直接在当前分支/当前工作目录开发与提交。
- **Git 提交与推送**：未经用户**明确同意**，不得执行 `git commit` 或 `git push`。每次提交或推送前须说明拟纳入的变更摘要（或要点）与建议的提交说明，经用户确认后再执行。查看类操作（如 `git status`、`git diff`、`git log`）不受此限。
- **术语与缩写**：避免使用生涩难懂的术语；如必须使用术语或英文缩写，**首次出现**时需在括号中补充中文释义或英文全称，后续可仅使用缩写。
- **TDD**：采用 **TDD** 开发模式，**测试先行**（先写/更新测试，再实现业务代码，最后重构）。
- **性能与基准测试**：复杂逻辑需要补充 **benchmark**（`*_test.go` 中的 `BenchmarkXxx`）。实现后必须提醒开发者确认性能是否达标（包含关键指标与可复现实验方式）。
- **基准测试结果归档与对比（每次跑完 benchmark 后）**：
  1. 将本次输出与**上一次已归档**（或上一次提交的基线 commit）结果对比，使用 `benchstat old.txt new.txt` 或等价方式生成差异。
  2. 在回复或 PR 描述中用 **Markdown 表格**展示对比（至少含：基准名、前次 ns/op（或 B/op、allocs/op）、本次、相对变化或 `benchstat` 的 `vs base`）。
  3. **询问**使用者是否**保留本次结果**作为后续对比基线。
  4. 若使用者确认保留：将本次对比表及元数据（日期、`go version`、机器/OS、`GOMAXPROCS`、**commit**、完整 `go test -bench=...` 命令）**追加**写入 `docs/superpowers/benchmarks/` 下对应主题的日志文件（约定见该目录 `README.md`）；不得把仅用于临时对比的冗长原始 `.txt` 提交进仓库，除非团队明确要求。
- **规模与复用（新增/修改代码时遵守）**：
  - **单文件**：同一源文件行数**不超过 500 行**（含空行与注释）；若逼近上限，应拆分为多个文件或子包，并保证职责清晰。
  - **单函数**：同一函数**不超过 50 行**（含空行与注释）；超出则拆分为多个函数或提取步骤，避免单块过长。
  - **调用链追踪自检（跨模块/跨服务变更必须执行）**：修改跨模块、跨进程（如集群路由、消息投递、RPC 调用）的逻辑后，**必须**从入口点出发，跟踪完整的调用链并验证每一跳的关键假设（PID 格式、地址映射、消息编解码、端口/主题命名等）是否匹配。仅 `go build` 通过不足以证明链路正确。
  - **公共抽象**：多处重复或可被清晰命名的逻辑，**必须**提取为包内/跨包的**公共函数**（或小型类型与方法），避免复制粘贴；提取时保持命名与现有代码风格一致。
  - **命名不得以包名开头**：包内导出的函数名、类型名、变量名杜绝以包名作为前缀。调用侧在使用时本身带有包名限定（如 `lockstep.NewSession`），若函数名再以包名开头将形成冗余（`lockstep.LockstepNewSession`），且会触发 IDE 警告 "Name starts with the package name"。正确做法：`lockstep.NewSession`（而非 `lockstep.NewLockstepSession`）、`cells.ErrUnknownCID`（而非 `cells.CellsErrUnknownCID`）。
- **代码注释语言**：所有手写代码注释必须使用中文，包括 Go doc 注释、行内注释、复杂逻辑说明和测试意图说明。允许保留英文的情况仅限专有协议字段、外部标准名、错误码、指标名、trace attribute 名、第三方 API 原文，以及 protobuf/OpenAPI/工具生成文件中的生成注释。

