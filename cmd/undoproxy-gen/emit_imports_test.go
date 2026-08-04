package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmit_ImportsCrossPackageLeaf(t *testing.T) {
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
	if !strings.Contains(s, "testdata/leafpkg") {
		t.Fatal("missing leafpkg import path")
	}
	if !strings.Contains(s, "leafpkg.") {
		t.Fatal("missing leafpkg selector in generated types")
	}
	if !strings.Contains(s, `"sync"`) {
		t.Fatal("missing sync")
	}
}

func TestEmit_NoExtraImportsWithoutLeaf(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	// 根包 cow 无跨包叶子，生成物 import 应仅含 sync
	if err := Run(out, "github.com/huangyuCN/cow"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "leafpkg") {
		t.Fatal("unexpected leafpkg import")
	}
	if !strings.Contains(s, `"sync"`) {
		t.Fatal("missing sync")
	}
}
