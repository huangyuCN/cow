package cow

import "testing"

func TestTxContextReset_ClearsUndoClosures(t *testing.T) {
	var ctx TxContext
	ctx.AddUndo(func() {})
	ctx.AddUndo(func() {})

	used := len(ctx.undoLogs)
	if used == 0 {
		t.Fatal("expected undo logs before reset")
	}

	ctx.Reset()
	if len(ctx.undoLogs) != 0 {
		t.Fatalf("len after reset got %d want 0", len(ctx.undoLogs))
	}

	backing := ctx.undoLogs[:cap(ctx.undoLogs)]
	for i := 0; i < used; i++ {
		if backing[i] != nil {
			t.Fatalf("undo log slot %d should be nil after reset", i)
		}
	}
}

func TestTxContextV2Reset_ClearsUndoOpReferences(t *testing.T) {
	var ctx TxContextV2
	p := &Player{}
	h := &Hero{}
	item := &Item{}
	ctx.push(undoOpV2{
		player:    p,
		hero:      h,
		keyString: "k",
		tail:      []*Item{item},
		bagOld:    []*Item{item},
		statsOld:  map[string]int64{"atk": 1},
		cdOld:     []int32{1},
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
			t.Fatalf("op slot %d keeps pointer/string references after reset", i)
		}
		if op.tail != nil || op.bagOld != nil || op.statsOld != nil || op.cdOld != nil {
			t.Fatalf("op slot %d keeps slice/map references after reset", i)
		}
		if op.had || op.had2 || op.oldI32 != 0 || op.oldI64 != 0 || op.oldInt != 0 {
			t.Fatalf("op slot %d should be zero value after reset", i)
		}
	}
}
