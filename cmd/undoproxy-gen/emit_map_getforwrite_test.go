package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmit_MapGetForWriteUsesFieldName(t *testing.T) {
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
	for _, name := range []string{"GetStackableForWrite", "GetVirtualStackableForWrite"} {
		def := "func (b *Bag) " + name + "("
		if c := strings.Count(s, def); c != 1 {
			t.Fatalf("%s defs=%d want 1", name, c)
		}
	}
	if strings.Contains(s, "GetStackableItemForWrite") {
		t.Fatal("must not emit elem-type-based GetStackableItemForWrite")
	}
}

func TestEmit_MapMapPtrGetForWriteUsesFieldName(t *testing.T) {
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
	for _, name := range []string{"GetPrimaryForWrite", "GetSecondaryForWrite"} {
		def := "func (n *NestBag) " + name + "("
		if c := strings.Count(s, def); c != 1 {
			t.Fatalf("%s defs=%d want 1", name, c)
		}
	}
	if strings.Contains(s, "func (n *NestBag) GetConditionForWrite(") {
		t.Fatal("must not emit elem-type-based GetConditionForWrite on NestBag")
	}
}
