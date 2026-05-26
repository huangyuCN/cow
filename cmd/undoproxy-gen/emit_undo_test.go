package main

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func TestUndoBuilder_KindDedup(t *testing.T) {
	g := &cowgen.Graph{Structs: []*cowgen.StructPlan{{Name: "Player"}, {Name: "Account"}}}
	ub := newUndoBuilder(g)
	body := "op.player.Assets[op.keyString] = op.oldI64"
	k1 := ub.kind("Player", "Assets", "MapKeySet", body)
	k2 := ub.kind("Player", "Assets", "MapKeySet", body)
	if k1 != k2 {
		t.Fatalf("kind dedup: %s vs %s", k1, k2)
	}
	if len(ub.entries) != 1 {
		t.Fatalf("entries len got %d want 1", len(ub.entries))
	}
}
