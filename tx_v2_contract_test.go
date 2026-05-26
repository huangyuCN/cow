package cow

import "testing"

// TestTxV2RuntimeContract 校验 V2 运行时由生成代码完整提供，
// 防止未来回退到手写文件依赖。
func TestTxV2RuntimeContract(t *testing.T) {
	var ctx TxContextV2
	ctx.Reset()
	ctx.Rollback()
	if txPoolV2.New == nil {
		t.Fatal("txPoolV2.New should not be nil")
	}
}

