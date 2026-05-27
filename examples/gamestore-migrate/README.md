# gamestore-migrate — 从裸写到 cow 写模式的迁移对照示例

该目录是一个**独立 Go module**，用于给集成方提供“从存量裸写出发，经过 tag + generate + undorewrite + undocheck，最终进入 cow 写模式”的切换参考。

其中：
- `before/`：裸写起点（**无** `+cow:undoproxy-gen`），用于展示迁移前的业务写法
- `after/`：迁移完成态金标准（**含** tag、生成物、TxContext/txPool、代理写与回滚/提交测试）

> 说明：README 的 `undorewrite` 练习步骤建议在**临时工作目录**进行（不要提交对 `before/` 的改动）。仓库内 `after/` 作为期望输出对照。

## 快速自检

```bash
cd examples/gamestore-migrate
go test ./before/... ./after/... -count=1
```

## 迁移路线（C1，8 步）

以下步骤希望你最终得到的效果，等价于“把 `before/` 的裸写代码，通过工具迁移到 `after/` 的代理写形式”。

1. `diff` 对照理解模型差异

   ```bash
   diff -ru before/types.go after/types.go
   ```

2. 在临时工作拷贝中给根类型打 tag（例如把 `before/types.go` 复制出来并补齐）

   ```go
   // +cow:undoproxy-gen=true
   type Player struct { ... }
   ```

3. 在临时工作拷贝中补齐 `after/doc.go` 的包级 tag，并跑 generate

   ```bash
   # 进入临时包目录（等价于 after/ 生成工作目录）
   go generate ./...
   ```

4. 静态守门准备：安装并理解 `undocheck`

   ```bash
   go install ./cmd/undocheck
   ```

5. 迁移改写：运行 `undorewrite`（dry-run 审查 diff；再写回）

   ```bash
   # 干跑
   undorewrite ./... 
   # 确认后写回
   undorewrite -w ./...
   ```

6. 静态验收：再次跑 `undocheck`（无 `cowbarewrite` 诊断为通过）

   ```bash
   go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
   ```

7. 运行时接入：让业务作用域具备 `*TxContext`（对照 `after/service.go`）

8. 行为验收：保证回滚/提交语义与 `after/handler_test.go` 一致

## 与 `examples/gamestore` 的关系

- `examples/gamestore`：展示完整双根类型图与 `cmd/demo` 演示（能力覆盖更广）
- `examples/gamestore-migrate`：聚焦“存量裸写迁移流水线”，类型图略精简（A1），便于逐步跟跑

