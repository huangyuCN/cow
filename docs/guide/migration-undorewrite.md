# 存量改写（undorewrite）

## 概述

`undorewrite` 将监控类型上的**裸写**批量改为 `undoproxy-gen` 代理调用。与 `undocheck` 互补：分析器防新增，改写器清历史。

## 前置条件

- 已生成 `zz_generated.undo_proxy.go`
- 目标函数内可解析到 `*TxContext` 变量（或通过 `-inject-ctx` 注入）

## 独立 module（不 import cow 业务类型）

典型接入（如 `examples/gamestore`）在业务包内生成 `TxContext` 与写代理，**无需** `import` cow 的 `Player` 等类型：

1. 在业务包目录执行 `go generate`，产出 `zz_generated.undo_proxy.go`
2. 在同一包路径运行改写（监控集与改写目录按**当前包**类型图构建）：

```bash
undorewrite ./...
undorewrite -w ./...
```

3. 函数参数须为**本包** `*TxContext`，或配合 `-inject-ctx=pool` 使用本包 `txPool`（注入代码引用本包 `TxContext`，非 `cow.TxContext`）
4. 改写后用 `undocheck` 验收（见下文）

`-cow` 仅在目标包仍通过 cow 导出类型做迁移时，作为 catalog **回退**路径（与 `undocheck` 行为一致）。

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
| `-cow` | `github.com/huangyuCN/cow` | 仍 import cow 类型时的 catalog 回退路径 |
| `-w` | false | 写回源文件 |
| `-ctx` | `ctx` | `TxContext` 变量名 |
| `-inject-ctx` | 空 | `new` / `pool` / `param:NAME` |
| `-pool-var` | `txPool` | `inject-ctx=pool` 时的 Pool 名 |

## 验收

改写后须通过：

```bash
go vet -vettool=$(go env GOPATH)/bin/undocheck ./yourpkg/...
```

## 能力边界

- 不改写 `zz_generated*`、`*_fixture.go` 等（与 `undocheck` 白名单一致）。
- 不处理 `json.Unmarshal` / 反射写。
- 复杂多赋值/泛型边界 case 可能需人工收尾。

## 相关链接

- [bare-write-guard.md](bare-write-guard.md)
- 维护：[../../cmd/undorewrite/README.md](../../cmd/undorewrite/README.md)
- 设计：[../superpowers/specs/2026-05-25-undorewrite-codemod-design.md](../superpowers/specs/2026-05-25-undorewrite-codemod-design.md)
- 独立接入扩展：[../superpowers/specs/2026-05-27-undorewrite-consumer-alignment-design.md](../superpowers/specs/2026-05-27-undorewrite-consumer-alignment-design.md)
