package cow

import "fmt"

// ExampleTxContext_rollback 演示失败路径下 Rollback 恢复聚合根状态。
func ExampleTxContext_rollback() {
	player := newBenchPlayer()
	beforeGold := player.Assets["gold"]

	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)

	applySparseWrites(player, ctx)
	ctx.Rollback()

	if player.Assets["gold"] == beforeGold {
		fmt.Println("rolled back")
	}
	// Output: rolled back
}

// ExamplePlayer_sparseWrite 演示通过生成代理在 TxContext 下稀疏写。
func ExamplePlayer_sparseWrite() {
	player := newBenchPlayer()
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)

	player.PutAssets(ctx, "gold", 500)
	fmt.Println(player.Assets["gold"])
	// Output: 500
}
