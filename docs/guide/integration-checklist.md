# 接入检查清单

在将 cow 接入业务模块前/后逐项确认。

## 模型与生成

- [ ] 聚合根已标记 `// +cow:undoproxy-gen=true`
- [ ] 已添加 `//go:generate undoproxy-gen ...` 并执行 `go generate ./...`
- [ ] `zz_generated.undo_proxy.go` 已提交版本库
- [ ] 嵌套 struct 与根类型在**同一包**（或确认未使用跨包嵌套字段）

## 运行时写路径

- [ ] 请求/消息作用域内从 `sync.Pool`（或等价）获取 `TxContext` 并 `Reset()`
- [ ] 失败路径调用 `Rollback()`；成功路径 `Reset()` 提交
- [ ] 业务写全部经 `Put*` / `Append*` / `Get*ForWrite`，无裸写

## 静态检查与 CI

- [ ] `go install ./cmd/undocheck`（或 pin 模块版本安装）
- [ ] CI 包含：`go vet -vettool=$(go env GOPATH)/bin/undocheck ./...`
- [ ] 已知合法裸写处已加 `//cow:allow-bare-write` 或位于 skip 文件（如 `*_fixture.go`）

## 存量迁移（可选）

- [ ] `undorewrite ./...` dry-run 审查 diff
- [ ] `undorewrite -w` 后再次 `go vet` 无 `cowbarewrite` 诊断
- [ ] `go test ./...` 通过

## 文档

- [ ] 团队阅读 [overview.md](overview.md)、[limitations.md](limitations.md)
