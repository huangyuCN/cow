package cowgen_test

import (
	"go/types"
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func TestQualifiers_ConflictGetsNumericSuffix(t *testing.T) {
	q := cowgen.NewQualifiers()
	p1 := types.NewPackage("example.com/a/cfg", "cfg")
	p2 := types.NewPackage("example.com/b/cfg", "cfg")
	if q.Alias(p1) != "cfg" {
		t.Fatalf("first alias: got %q want cfg", q.Alias(p1))
	}
	if q.Alias(p2) != "cfg2" {
		t.Fatalf("conflict alias: got %q want cfg2", q.Alias(p2))
	}
	if q.Alias(p1) != "cfg" {
		t.Fatal("stable: first path must keep cfg")
	}
}
