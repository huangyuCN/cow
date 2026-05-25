package main

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func TestClassifyViaLoad(t *testing.T) {
	pkg, err := loadPackage("github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata")
	if err != nil {
		t.Fatal(err)
	}
	g, err := cowgen.BuildGraph(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Structs) < 4 {
		t.Fatalf("structs=%d want >=4", len(g.Structs))
	}
}
