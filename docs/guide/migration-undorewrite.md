# 存量改写（undorewrite）

## 概述

`undorewrite` 将监控类型上的**裸写**批量改为 `undoproxy-gen` 代理调用。与 `undocheck` 互补：分析器防新增，改写器清历史。

## 前置条件

- 已生成 `zz_generated.undo_proxy.go`
- 目标函数内可解析到 `*TxContext` 变量（或通过 `-inject-ctx` 注入）

## 用法

```bash
go install ./cmd/undorewrite

# 默认 dry-run：仅打印 diff
undorewrite ./yourpkg/...

# 确认后写回
undorewrite -w ./yourpkg/...

# 可选：注入 ctx（new | pool | param:NAME）
undorewrite -inject-ctx=pool -pool-var=txPool -w ./yourpkg/...
```

### 常用 flags

| flag | 默认 | 说明 |
|------|------|------|
| `-cow` | `github.com/huangyuCN/cow` | cow 模块 import path |
| `-w` | false | 写回源文件 |
| `-ctx` | `ctx` | `TxContext` 变量名 |
| `-inject-ctx` | 空 | `new` / `pool` / `param:NAME` |
| `-pool-var` | `txPool` | `inject-ctx=pool` 时的 Pool 名 |

## 验收

改写后须通过：

```bash
go vet -vettool=$(go env GOPATH)/bin/undocheck ./yourpkg/...
```

## 边界（v1）

- 不改写 `zz_generated*`、`*_fixture.go` 等（与 `undocheck` 白名单一致）。
- 不处理 `json.Unmarshal` / 反射写。
- 复杂多赋值/泛型边界 case 可能需人工收尾。

## 相关链接

- [bare-write-guard.md](bare-write-guard.md)
- 维护：[../../cmd/undorewrite/README.md](../../cmd/undorewrite/README.md)
- 设计：[../superpowers/specs/2026-05-25-undorewrite-codemod-design.md](../superpowers/specs/2026-05-25-undorewrite-codemod-design.md)
