package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmit_DedupesCloneMapShallowBySignature(t *testing.T) {
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
	name := cloneMapShallowFuncName("string", "*Condition")
	def := "func " + name + "("
	if c := strings.Count(s, def); c != 1 {
		t.Fatalf("helper defs=%d want 1 for %s", c, name)
	}
	if strings.Count(s, name+"(") < 3 {
		// 1 次定义 + ≥2 次调用（Alpha/Beta Get*MapForWrite）
		t.Fatalf("expected definition + multiple call sites for %s", name)
	}
	if strings.Contains(s, "cloneConditionsMapShallow") {
		t.Fatal("old field-based helper name must not appear")
	}
}
