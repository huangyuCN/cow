# 裸写防护（undocheck）

## 概述

分析器名 **`cowbarewrite`**：对 `+cow:undoproxy-gen` 类型图中类型的**裸写**报 error，引导改用生成代理。

## 前置条件

- 已 `go generate` 生成 `zz_generated.undo_proxy.go`
- 已安装 `cmd/undocheck`

## 用法

```bash
go install ./cmd/undocheck

# 推荐：显式指定 vet 工具二进制
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...

# 若分析器已编入该二进制，也可：
go vet -cowbarewrite ./...
```

跨模块：消费方仓库 import 你的 `Player` 时，须在其 CI 同样安装 `undocheck` 并对**自身** `./...` 跑 vet。

## 何为裸写

对受监控类型的字段赋值、复合字面量写字段、`++`/`--`、以及经 `Get*ForWrite` 返回指针后的字段直写等（详见设计 spec）。

## 逃逸与白名单

| 机制 | 用途 |
|------|------|
| `//cow:allow-bare-write` | 行级放行 |
| `internal/cowfile.SkipFile` | 跳过 `zz_generated*`、`*_fixture.go`、`cmd/undoproxy-gen/**` 等 |

好坏例源码：`cmd/undocheck/testdata/src/barewrite/`。

## 示例诊断

```
cowbarewrite: 禁止对 *Player 裸写 Level，请使用 PutLevel(ctx, …)
```

## 边界

- 仅编译期；运行期反射写不可见。
- 不强制「必须在带 `*TxContext` 参数的函数内」才允许调用代理（只禁裸写）。

## 相关链接

- [migration-undorewrite.md](migration-undorewrite.md)
- [integration-checklist.md](integration-checklist.md)
- 维护：[../../cmd/undocheck/README.md](../../cmd/undocheck/README.md)
- 设计：[../superpowers/specs/2026-05-25-bare-write-guard-design.md](../superpowers/specs/2026-05-25-bare-write-guard-design.md)
