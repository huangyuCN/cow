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

贡献者建议阅读顺序见 [docs/README.md](docs/README.md)。

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
