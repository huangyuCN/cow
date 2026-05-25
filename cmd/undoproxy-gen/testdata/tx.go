package testdata

// TxContext 黄金测试用最小事务上下文。
type TxContext struct {
	undo []func()
}

// AddUndo 注册逆操作。
func (c *TxContext) AddUndo(fn func()) {
	c.undo = append(c.undo, fn)
}
