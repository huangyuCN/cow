# gamestore — cow 独立接入示例

本目录是**独立 Go module**，演示业务方如何从零接入 cow（不 import 根包 `cow.Player`）。

## 前置

- Go 1.25+
- 在仓库根目录已启用 `go.work`（含本 module 与主模块）

## 快速开始

```bash
# 在仓库根目录
cd examples/gamestore

# 1. 代码生成（改 types.go 后必跑）
go generate ./...

# 2. 测试与演示
go test ./...
go run ./cmd/demo
```

## 本示例包含

| 能力                                       | 位置                            |
| ------------------------------------------ | ------------------------------- |
| 双根`+cow:undoproxy-gen=true`              | `types.go` — `Player`、`Guild` |
| 主要 FieldKind 代理写                      | `handler.go`                    |
| `TxContext` / `txPool` / Commit / Rollback | `service.go`、`demo.go`         |
| 生成物                                     | `zz_generated.undo_proxy.go`    |

## 静态守门（undocheck）

```bash
# 在仓库根目录
go install ./cmd/undocheck
cd examples/gamestore
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

示例源码无聚合根裸写；违例会触发 `cowbarewrite` 诊断。

## 存量裸写迁移

本示例不包含 `undorewrite` 执行步骤，见 [docs/guide/migration-undorewrite.md](../../docs/guide/migration-undorewrite.md)。

## 相关文档

- [integration-checklist.md](../../docs/guide/integration-checklist.md)
- [codegen-undoproxy.md](../../docs/guide/codegen-undoproxy.md)
- [proxy-api.md](../../docs/guide/proxy-api.md)
- [tx-context.md](../../docs/guide/tx-context.md)

