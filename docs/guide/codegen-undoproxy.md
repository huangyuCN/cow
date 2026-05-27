# undoproxy 代码生成

## 概述

`undoproxy-gen` 为带标记的类型生成 `Put*` / `Append*` / `Get*ForWrite` 等写代理，输出 **单个** `zz_generated.undo_proxy.go`。该文件同时包含：

- `TxContext`、`undoOp`、`Rollback`、`Reset`、`txPool`
- 所有根类型与可达嵌套类型的写代理

业务包**不需要**也不应再维护独立的 `tx.go` 或手写代理文件。

## 前置条件

- Go 1.25+
- 聚合根与嵌套 struct 在**同一包**内（当前版本限制）

## 步骤

### 1. 标记根类型

在聚合根 struct 上：

```go
// +cow:undoproxy-gen=true
type Player struct { ... }
```

包级（`doc.go` 已含）：

```go
// +cow:undoproxy-gen=package
```

### 2. 添加 go:generate

```go
//go:generate undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow
```

本仓库见 `undo_proxy_generate.go`。

### 3. 安装并生成

```bash
go install ./cmd/undoproxy-gen
go generate ./...
```

### 4. 提交生成文件

`zz_generated.undo_proxy.go` **须纳入 Git**；CI 默认不跑 generate。修改 `types.go` 或标记后本地重新 `go generate` 并提交 diff。

## deepcopy-gen（仅对照基线）

包级 `// +k8s:deepcopy-gen=package` 生成 `zz_generated.deepcopy.go`，用于 benchmark 中「每请求 DeepCopy」基线，**不是**业务运行时的回滚机制。更新方式：按 k8s code-generator 流程执行 deepcopy-gen（见 `deepcopy_generate.go`）。

## 边界

- 与 `TxContext`、`undoOp`、`Rollback` 一并写入 `zz_generated.undo_proxy.go`（按类型图裁剪字段）。
- 支持同包内**多个** `// +cow:undoproxy-gen=true` 根类型。
- 跨包嵌套字段、反射动态类型不在生成范围。

## 相关链接

- [proxy-api.md](proxy-api.md)
- [bare-write-guard.md](bare-write-guard.md)
- 维护：[../../cmd/undoproxy-gen/README.md](../../cmd/undoproxy-gen/README.md)
- 设计：[../superpowers/specs/2026-05-25-undoproxy-codegen-design.md](../superpowers/specs/2026-05-25-undoproxy-codegen-design.md)
