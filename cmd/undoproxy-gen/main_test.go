package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_V2Mode_GeneratesV2Methods(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "v2.go")
	err := Run(out, "github.com/huangyuCN/cow", "v2")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "PutAssetsV2") {
		t.Fatalf("generated v2 file missing PutAssetsV2")
	}
	if !strings.Contains(s, "mode=v2") {
		t.Fatalf("generated v2 file missing mode marker")
	}
	required := []string{
		"AppendItemsV2",
		"RemoveItemsAtV2",
		"TruncateItemsV2",
		"AppendBagsAtV2",
		"PutStatsV2",
		"AppendCooldownsAtV2",
		"GetHeroForWriteV2",
	}
	for _, name := range required {
		if !strings.Contains(s, name) {
			t.Fatalf("generated v2 file missing %s", name)
		}
	}
}

func TestRun_InvalidMode(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "bad.go")
	err := Run(out, "github.com/huangyuCN/cow", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestRun_V2Mode_MapSliceAppendUndoStrategy(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "v2.go")
	if err := Run(out, "github.com/huangyuCN/cow", "v2"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	bagsUndo := "ctx.push(undoOpV2{kind: undoKindPlayerBagsAppendAtKeyV2, player: p, keyI32: k1, bagOld: old, oldInt: oldLen, had: existed})"
	if !strings.Contains(s, bagsUndo) {
		t.Fatalf("generated v2 file missing bags append undo strategy: %s", bagsUndo)
	}
	cooldownsUndo := "ctx.push(undoOpV2{kind: undoKindPlayerCooldownsSetAtKeyV2, player: p, keyI32: k1, cdOld: oldCopy, had: existed})"
	if !strings.Contains(s, cooldownsUndo) {
		t.Fatalf("generated v2 file missing cooldowns append undo strategy: %s", cooldownsUndo)
	}
}

func TestRun_V2Mode_GeneratesRuntime(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "v2.go")
	if err := Run(out, "github.com/huangyuCN/cow", "v2"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	required := []string{
		"type undoKindV2 uint8",
		"type undoOpV2 struct",
		"type TxContextV2 struct",
		"func (ctx *TxContextV2) Reset()",
		"var txPoolV2 = sync.Pool{",
		"func (ctx *TxContextV2) Rollback()",
		"func cloneStatsMapShallowV2(",
	}
	for _, needle := range required {
		if !strings.Contains(s, needle) {
			t.Fatalf("generated v2 file missing runtime symbol: %s", needle)
		}
	}
	if strings.Contains(s, "cloneStatsMapShallow(") {
		t.Fatal("generated v2 runtime should not depend on cloneStatsMapShallow")
	}
}
