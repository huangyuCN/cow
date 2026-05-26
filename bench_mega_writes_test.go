package cow

import (
	"errors"
	"testing"
)

func TestMegaSparseWrites32_OpCount(t *testing.T) {
	p := newMegaBenchPlayer()
	ctx := &TxContext{ops: make([]undoOp, 0, megaSparseWriteCount)}
	applyMegaSparseWrites32(p, ctx)
	if got := len(ctx.ops); got != megaSparseWriteCount {
		t.Fatalf("undo ops got %d want %d", got, megaSparseWriteCount)
	}
}

func TestMegaPlayer_BusinessPath32_Rollback(t *testing.T) {
	p := newMegaBenchPlayer()
	want := clonePlayerSnapshot(p)
	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		applyMegaSparseWrites32(p, ctx)
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, p, want)
}
