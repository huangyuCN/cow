# 集成指南

按功能查阅；每篇含步骤、示例与边界说明。

| 文档 | 内容 |
|------|------|
| [overview.md](overview.md) | Undo Log 架构、与 DeepCopy 对比 |
| [tx-context.md](tx-context.md) | `TxContext` 生命周期、commit / rollback |
| [codegen-undoproxy.md](codegen-undoproxy.md) | `+cow:undoproxy-gen`、`go generate` |
| [proxy-api.md](proxy-api.md) | `Put*` / `Append*` / `Get*ForWrite` 语义 |
| [bare-write-guard.md](bare-write-guard.md) | `undocheck`、`cowbarewrite` |
| [migration-undorewrite.md](migration-undorewrite.md) | 存量裸写批量改写 |
| [integration-checklist.md](integration-checklist.md) | 接入检查清单 |
| [limitations.md](limitations.md) | 非目标与 serde 约定 |

示例代码以仓库内 `*_test.go` 与 `func Example*` 为准（`go test` 校验）。
