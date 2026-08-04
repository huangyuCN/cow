package main

import (
	"bytes"
	"fmt"
	"go/format"
	"math"
	"strings"
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

func TestCheckUndoKindCount_Overflow(t *testing.T) {
	if err := checkUndoKindCount(math.MaxUint16); err != nil {
		t.Fatalf("MaxUint16 kinds should be ok: %v", err)
	}
	if err := checkUndoKindCount(math.MaxUint16 + 1); err == nil {
		t.Fatal("want error when kind count exceeds uint16")
	}
}

func TestWriteRuntime_UsesUint16AndAllows256Kinds(t *testing.T) {
	ub := &undoBuilder{
		structs:      []string{"W"},
		kindIndex:    make(map[string]string),
		cloneHelpers: make(map[string]struct{}),
		sliceSnaps:   make(map[string]string),
		scalarOlds:   map[string]string{"int": "oldInt"},
	}
	for i := 0; i < 256; i++ {
		ub.entries = append(ub.entries, undoEntry{name: fmt.Sprintf("undoKindW%d", i), body: "break"})
	}
	var buf bytes.Buffer
	if err := ub.writeRuntime(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "type undoKind uint16") {
		t.Fatal("want uint16")
	}
	if strings.Contains(s, "type undoKind uint8") {
		t.Fatal("uint8 must not appear")
	}
	if _, err := format.Source(buf.Bytes()); err != nil {
		t.Fatalf("format: %v", err)
	}
}
