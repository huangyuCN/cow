# 生产就绪文档与措辞清理设计说明

| 项 | 值 |
|---|---|
| 状态 | 已批准（brainstorming 2026-05-27；范围 **A**） |
| 模块 | `github.com/huangyuCN/cow` 全仓库文档与包注释 |
| 目标 | 对外呈现稳定产品表述；去除阶段标签与「未完成」暗示；与当前 `undoOp` 实现对齐 |

## 1. 背景

核心能力（`undoproxy-gen`、结构化 Undo、`examples/gamestore`、CI）已实现并通过测试。残留「闭包 / AddUndo」等早期表述主要出现在包注释与未归档的 superpowers plans，易让集成方误判为试验品。

## 2. 范围

### 2.1 包含

- 根包与 `internal/cowgen` 相关中文注释措辞
- `README.md`、`docs/guide/*`、`docs/toolchain/type-graph.md`
- 已实现 plans 顶栏 **状态：已实现** 归档说明
- superpowers 内「需求来源」对已删草稿文件（`需求草稿.md（已删除）` 等）的引用清理

### 2.2 不包含（范围 A）

- Session / Savepoint / 跨包 codegen / 运行期裸写检测
- `docs/superpowers/benchmarks/cow-undo-log-benchmark.md` **重命名**
- README 英文摘要、发版 tag
- 将 plans 内全部 `- [ ]` 改为 `- [x]`

## 3. 方案

采用 **方案 2：集成面 + 计划归档**（不重命名 benchmark 文件）。

## 4. 措辞规范

| 原表述 | 替换为 |
|--------|--------|
| 试验/验证用语 | 删除；包注释描述能力本身 |
| 阶段标签（限制） | 「当前」或直接陈述限制 |
| guide 中闭包 / AddUndo 为现行 API | `undoOp` + `push`；mermaid 使用 `push(undoOp)` |

## 5. 必改文件

- `doc.go`、`types.go`、`internal/cowgen/naming.go`
- `README.md`
- `docs/guide/overview.md`、`tx-context.md`、`codegen-undoproxy.md`、`limitations.md`
- `docs/toolchain/type-graph.md`
- 九个已实现 plan（见 §6）

## 6. Plans 归档列表

在文首增加已实现状态块：

- `docs/superpowers/plans/2026-05-25-cow-undo-log.md`
- `docs/superpowers/plans/2026-05-25-undoproxy-codegen.md`
- `docs/superpowers/plans/2026-05-25-open-source-readiness.md`
- `docs/superpowers/plans/2026-05-25-project-documentation.md`
- `docs/superpowers/plans/2026-05-25-mega-player-benchmark.md`
- `docs/superpowers/plans/2026-05-25-bare-write-guard.md`
- `docs/superpowers/plans/2026-05-25-undorewrite-codemod.md`
- `docs/superpowers/plans/2026-05-26-undoproxy-gen-structured-generic.md`
- `docs/superpowers/plans/2026-05-26-examples-gamestore.md`

## 7. 验收

```bash
git grep -E 'AddUndo|逆操作闭包' -- README.md doc.go types.go docs/guide docs/toolchain internal/cowgen/naming.go
git grep -E 'AddUndo|逆操作闭包' -- docs/guide
go test ./...
cd examples/gamestore && go test ./...
```

- 上述 grep 对集成面无命中（benchmark 路径名除外）
- 测试全绿

## 8. 参考

- [2026-05-25-open-source-readiness-design.md](2026-05-25-open-source-readiness-design.md)
- [2026-05-25-cow-undo-log-design.md](2026-05-25-cow-undo-log-design.md)（历史阶段设计，保留文件名）
