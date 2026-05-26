package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_GeneratesStructuredUndoProxy(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	if err := Run(out, "github.com/huangyuCN/cow"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	required := []string{
		"type undoKind uint8",
		"type undoOp struct",
		"type TxContext struct",
		"func (ctx *TxContext) push",
		"func (ctx *TxContext) Reset()",
		"var txPool = sync.Pool{",
		"func (ctx *TxContext) Rollback()",
		"PutAssets(ctx *TxContext",
		"ctx.push(undoOp{",
	}
	for _, needle := range required {
		if !strings.Contains(s, needle) {
			t.Fatalf("generated file missing %q", needle)
		}
	}
	if strings.Contains(s, "V2") {
		t.Fatal("generated file should not contain V2 suffix")
	}
	if strings.Contains(s, "AddUndo") {
		t.Fatal("generated file should not contain AddUndo")
	}
}
