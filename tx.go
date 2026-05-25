package cow

import "sync"

// TxContext 单次请求作用域的 Undo 日志（单协程，无锁）。
//
// +k8s:deepcopy-gen=false
type TxContext struct {
	undoLogs []func()
}

// AddUndo 注册一条逆操作。
func (ctx *TxContext) AddUndo(undo func()) {
	ctx.undoLogs = append(ctx.undoLogs, undo)
}

// Rollback 倒序执行所有逆操作。
func (ctx *TxContext) Rollback() {
	for i := len(ctx.undoLogs) - 1; i >= 0; i-- {
		ctx.undoLogs[i]()
	}
}

// Reset 清空日志并复用底层切片。
func (ctx *TxContext) Reset() {
	ctx.undoLogs = ctx.undoLogs[:0]
}

// txPool 复用 TxContext，降低高频路径分配。
var txPool = sync.Pool{
	New: func() any {
		return &TxContext{undoLogs: make([]func(), 0, 16)}
	},
}
