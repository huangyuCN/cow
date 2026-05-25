package main

import "testing"

func TestClassifyViaLoad(t *testing.T) {
	pkg, err := loadPackage("github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata")
	if err != nil {
		t.Fatal(err)
	}
	g, err := buildGraph(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Structs) < 4 {
		t.Fatalf("structs=%d want >=4", len(g.Structs))
	}
}
