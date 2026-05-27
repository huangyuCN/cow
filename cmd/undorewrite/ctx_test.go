package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestIsTxContextType_localPackage(t *testing.T) {
	cfg := Config{CowImport: "github.com/huangyuCN/cow"}
	ws, err := loadWorkspace(cfg, []string{"./testdata/consumer"})
	if err != nil {
		t.Fatal(err)
	}
	env, ok := ws.envForPkgPath("github.com/huangyuCN/cow/cmd/undorewrite/testdata/consumer")
	if !ok {
		t.Fatal("consumer env")
	}
	var pkg *packages.Package
	for _, p := range ws.Pkgs {
		if p.PkgPath == env.TxPkgPath {
			pkg = p
			break
		}
	}
	if pkg == nil {
		t.Fatal("consumer package")
	}
	var ctxType types.Type
	for _, f := range pkg.Syntax {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Use" {
				continue
			}
			for _, field := range fn.Type.Params.List {
				for _, n := range field.Names {
					if n.Name == "ctx" {
						ctxType = pkg.TypesInfo.TypeOf(n)
					}
				}
			}
		}
	}
	if ctxType == nil {
		t.Fatal("ctx param type")
	}
	if !isTxContextType(ctxType, env.TxPkgPath) {
		t.Fatal("same package TxContext should match")
	}
	if isTxContextType(ctxType, ws.CowImport) {
		t.Fatal("cow import path should not match local TxContext")
	}
}

func TestInjectCtx_new_local(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", `package p; func F() {}`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := f.Decls[0].(*ast.FuncDecl)
	res := injectCtx(fn, "example.com/p", "github.com/huangyuCN/cow", "", "new", "")
	if res.Skipped || len(res.Prefix) != 1 {
		t.Fatalf("unexpected inject result: %+v", res)
	}
	assign, ok := res.Prefix[0].(*ast.AssignStmt)
	if !ok {
		t.Fatal("expected assign")
	}
	unary, ok := assign.Rhs[0].(*ast.UnaryExpr)
	if !ok {
		t.Fatal("expected &TxContext{}")
	}
	lit, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		t.Fatal("expected composite lit")
	}
	id, ok := lit.Type.(*ast.Ident)
	if !ok || id.Name != "TxContext" {
		t.Fatalf("want local TxContext ident, got %#v", lit.Type)
	}
}

func TestInjectCtx_new_cowImport(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", `package p; func F() {}`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := f.Decls[0].(*ast.FuncDecl)
	cowPath := "github.com/huangyuCN/cow"
	res := injectCtx(fn, cowPath, cowPath, "cow", "new", "")
	assign := res.Prefix[0].(*ast.AssignStmt)
	unary := assign.Rhs[0].(*ast.UnaryExpr)
	lit := unary.X.(*ast.CompositeLit)
	sel, ok := lit.Type.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "TxContext" {
		t.Fatalf("want cow.TxContext, got %#v", lit.Type)
	}
	if x, ok := sel.X.(*ast.Ident); !ok || x.Name != "cow" {
		t.Fatalf("want cow selector, got %#v", sel.X)
	}
}
