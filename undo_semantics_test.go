package cow

import (
	"errors"
	"testing"
)

// TestRollback_MapMapNilInnerSlotRestored 验证外层 key 原值为 nil map 时，
// Rollback 应恢复该 nil 槽位而不是删除 key（外层 undo 记录必须携带 key 存在性）。
func TestRollback_MapMapNilInnerSlotRestored(t *testing.T) {
	p := &Player{Stats: map[int32]map[string]int64{5: nil}}

	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		p.PutStats(ctx, 5, "atk", 100)
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	inner, ok := p.Stats[5]
	if !ok {
		t.Fatal("rollback should restore the existing nil slot, not delete the key")
	}
	if inner != nil {
		t.Fatalf("inner map should stay nil after rollback, got %v", inner)
	}
}

// TestRollback_ForkRemoveAtKeepsOriginalRoot 验证 fork（CloneForWrite）后
// 在副本上 RemoveAt 再 Rollback，保留的原根不被移位损坏。
func TestRollback_ForkRemoveAtKeepsOriginalRoot(t *testing.T) {
	orig := newPlayerWithItems(newTestItemsByID(1, 2, 3))
	want := clonePlayerSnapshot(orig)
	fork := orig.CloneForWrite()

	err := runScopedWithRollback(fork, func(ctx *TxContext) error {
		fork.RemoveItemsAt(ctx, 0)
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, orig, want)
}
