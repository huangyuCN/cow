package cow

import "testing"

func TestTxContextReset_ClearsUndoOpReferences(t *testing.T) {
	var ctx TxContext
	p := &Player{}
	h := &Hero{}
	item := &Item{}
	ctx.push(undoOp{
		kind:      undoKindPlayerAssetsMapKeySet,
		player:    p,
		hero:      h,
		keyString: "k",
		snapItem:  []*Item{item},
		had:       true,
		had2:      true,
		oldI32:    1,
		oldI64:    2,
		oldInt:    3,
	})

	used := len(ctx.ops)
	if used == 0 {
		t.Fatal("expected undo ops before reset")
	}

	ctx.Reset()
	if len(ctx.ops) != 0 {
		t.Fatalf("len after reset got %d want 0", len(ctx.ops))
	}

	backing := ctx.ops[:cap(ctx.ops)]
	for i := 0; i < used; i++ {
		op := backing[i]
		if op.player != nil || op.hero != nil || op.keyString != "" {
			t.Fatalf("op slot %d keeps references after reset", i)
		}
		if op.snapItem != nil {
			t.Fatalf("op slot %d keeps slice snap after reset", i)
		}
		if op.had || op.had2 || op.oldI32 != 0 || op.oldI64 != 0 || op.oldInt != 0 {
			t.Fatalf("op slot %d should be zero value after reset", i)
		}
	}
}
