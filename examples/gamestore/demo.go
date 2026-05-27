package gamestore

import "fmt"

// RunDemoRollback 演示失败路径 Rollback 恢复 Gold。
func RunDemoRollback() {
	p := NewDemoPlayer()
	before := p.Gold
	_ = runScopedWithRollback(func(ctx *TxContext) error {
		return HandlePurchaseFail(p, ctx)
	})
	fmt.Printf("rollback: gold %d -> %d (expect restored to %d)\n", before, p.Gold, before)
}

// RunDemoCommit 演示成功路径 Commit 保留变更。
func RunDemoCommit() {
	p := NewDemoPlayer()
	g := NewDemoGuild()
	before := p.Gold
	_ = runScopedCommit(func(ctx *TxContext) error {
		return HandlePurchaseSuccess(p, g, ctx)
	})
	fmt.Printf("commit: gold %d -> %d\n", before, p.Gold)
}
