package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_NoAddUndo_DualRoot(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	if err := Run(out, "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, bad := range []string{"AddUndo(", "V2", "undoKindClosure"} {
		if strings.Contains(s, bad) {
			t.Fatalf("generated contains %q", bad)
		}
	}
	for _, good := range []string{
		"type undoKind",
		"func (ctx *TxContext) push",
		"func (p *Player)",
		"func (r *Room)",
		"SetMainHero",
		"RemoveHeros",
		"PutEquips",
		"RemoveEquips",
	} {
		if !strings.Contains(s, good) {
			t.Fatalf("generated missing %q", good)
		}
	}
}
