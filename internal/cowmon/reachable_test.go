package cowmon_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
	"golang.org/x/tools/go/packages"
)

func TestMonitoredSet_Contains_distinguishesSameNameAcrossPackages(t *testing.T) {
	set, err := cowmon.LoadMonitored("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	pkgA := types.NewPackage("a.example", "a")
	pkgB := types.NewPackage("b.example", "b")
	heroA := types.NewNamed(types.NewTypeName(0, pkgA, "Hero", nil), types.NewStruct(nil, nil), nil)
	heroB := types.NewNamed(types.NewTypeName(0, pkgB, "Hero", nil), types.NewStruct(nil, nil), nil)

	if !set.ContainsName("Hero") {
		t.Fatal("expected cow Hero in monitored set")
	}
	if set.Contains(heroB) {
		t.Fatal("unrelated Hero in another package must not match by name")
	}
	if set.Contains(heroA) {
		t.Fatal("synthetic Hero must not match cow monitored Hero")
	}
}

func TestMonitoredSet_Contains_crossPackagesLoadSession(t *testing.T) {
	set, err := cowmon.LoadMonitored("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	const legacy = "github.com/huangyuCN/cow/cmd/undorewrite/testdata/legacy"
	mode := packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps
	pkgs, err := packages.Load(&packages.Config{Mode: mode}, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("package errors")
	}
	var playerType types.Type
	for _, pkg := range pkgs {
		if pkg.PkgPath != legacy {
			continue
		}
		for _, f := range pkg.Syntax {
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != "Use" || fn.Type.Params == nil {
					continue
				}
				for _, field := range fn.Type.Params.List {
					for _, n := range field.Names {
						if n.Name == "p" {
							playerType = pkg.TypesInfo.TypeOf(n)
						}
					}
				}
			}
		}
	}
	if playerType == nil {
		t.Fatal("missing *cow.Player param type")
	}
	if !set.Contains(playerType) {
		t.Fatal("Player from another packages.Load session should match by import path and name")
	}
}
