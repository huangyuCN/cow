# TxContext 与请求作用域

## 概述

`TxContext` 管理单次请求/消息处理作用域内的 Undo 日志。实现见 `zz_generated.undo_proxy.go`（`undoOp` 结构化日志 + `Rollback` 按 `undoKind` 分发）：**单协程、无 Mutex**。

## 前置条件

- 同一 `TxContext` 实例仅在一个 goroutine 内使用。
- 所有对受监控聚合根的写须经生成代理，并向同一 `ctx` 注册 Undo。

## API

| 方法 | 作用 |
|------|------|
| `Rollback()` | 倒序执行全部逆操作（由生成代理经 `push(undoOp)` 注册） |
| `Reset()` | 清空日志，复用底层切片容量 |
| `txPool`（包内） | `sync.Pool` 复用 `TxContext` |

生成代理在写路径上调用包内 `push(undoOp{...})`；**无** `AddUndo` 公共 API。

## Commit 模式（成功提交）

业务函数返回 `nil` 时只 `Reset()`，不 `Rollback()`：

```go
func runScopedCommit(p *Player, fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)
	if err := fn(ctx); err != nil {
		ctx.Rollback()
		return err
	}
	ctx.Reset()
	return nil
}
```

完整实现：`player_test.go` 中的 `runScopedCommit`。

## Rollback 模式（失败回滚）

defer 中无论是否出错都 `Rollback()`（或仅在 `err != nil` 时 Rollback，按宿主习惯二选一；测试里采用「总是 Rollback」以验证可恢复性）：

```go
func runScopedWithRollback(p *Player, fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	return fn(ctx)
}
```

见 `player_test.go` 的 `runScopedWithRollback` 与 `TestRollback_RestoresInitialState`。

## 示例

可运行示例（`go test`）：

- `doc_examples_test.go` — `ExampleTxContext_rollback`
- `doc_examples_test.go` — `ExamplePlayer_sparseWrite`

稀疏写组合见 `bench_fixture_test.go` 的 `applySparseWrites`。

## 边界 / FAQ

- **Q：成功路径为何还要 `Reset()`？**  
  A：避免 `undoOp` 持有指针/slice/map 引用导致泄漏；并复用 `ops` 切片容量。
- **Q：能否多个聚合根共用一个 `TxContext`？**  
  A：MVP 按单聚合根设计；多根需自行划分作用域或扩展。

## 相关链接

- [proxy-api.md](proxy-api.md)
- [overview.md](overview.md)
- 维护：[../toolchain/README.md](../toolchain/README.md)
