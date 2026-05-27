package gamestore

// runScopedWithRollback 在 fn 返回后总是 Rollback（用于可恢复性测试/演示）。
func runScopedWithRollback(fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	return fn(ctx)
}

// runScopedCommit 成功时 Reset 提交；失败时 Rollback。
func runScopedCommit(fn func(ctx *TxContext) error) error {
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
