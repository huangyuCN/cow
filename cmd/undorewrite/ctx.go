package main

import (
	"go/ast"
	"go/token"
	"go/types"
)

type ctxResult struct {
	Expr    ast.Expr
	Prefix  []ast.Stmt
	Skipped bool
}

func resolveCtx(fn *ast.FuncDecl, info *types.Info, cowPkg, ctxName, inject string, poolVar string) ctxResult {
	if fn.Body == nil {
		return ctxResult{Skipped: true}
	}
	if inject != "" {
		return injectCtx(fn, cowPkg, inject, poolVar)
	}
	if id := findCtxInParams(fn, info, cowPkg, ctxName); id != nil {
		return ctxResult{Expr: id}
	}
	if id := findCtxInBody(fn.Body, info, cowPkg, ctxName); id != nil {
		return ctxResult{Expr: id}
	}
	return ctxResult{Skipped: true}
}

func findCtxInParams(fn *ast.FuncDecl, info *types.Info, cowPkg, prefer string) ast.Expr {
	if fn.Type.Params == nil {
		return nil
	}
	var fallback ast.Expr
	for _, f := range fn.Type.Params.List {
		for _, n := range f.Names {
			if !isTxContextType(info.TypeOf(n), cowPkg) {
				continue
			}
			if prefer != "" && n.Name == prefer {
				return n
			}
			if fallback == nil {
				fallback = n
			}
		}
	}
	return fallback
}

func findCtxInBody(body *ast.BlockStmt, info *types.Info, cowPkg, prefer string) ast.Expr {
	if prefer != "" {
		if id := findCtxName(body, info, cowPkg, prefer); id != nil {
			return id
		}
	}
	for _, name := range []string{"ctx", "tx", "txCtx"} {
		if id := findCtxName(body, info, cowPkg, name); id != nil {
			return id
		}
	}
	return nil
}

func findCtxName(body *ast.BlockStmt, info *types.Info, cowPkg, name string) ast.Expr {
	var found ast.Expr
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		id, ok := n.(*ast.Ident)
		if !ok || id.Name != name {
			return true
		}
		if isTxContextType(info.TypeOf(id), cowPkg) {
			found = id
			return false
		}
		return true
	})
	return found
}

func isTxContextType(t types.Type, cowPkg string) bool {
	if t == nil {
		return false
	}
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	return named.Obj().Name() == "TxContext" && named.Obj().Pkg().Path() == cowPkg
}

func injectCtx(fn *ast.FuncDecl, cowPkg, inject, poolVar string) ctxResult {
	ctxIdent := &ast.Ident{Name: "ctx"}
	var prefix []ast.Stmt
	switch inject {
	case "new":
		prefix = []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{ctxIdent},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.UnaryExpr{
						Op: token.AND,
						X: &ast.CompositeLit{
							Type: &ast.SelectorExpr{
								X:   &ast.Ident{Name: cowPkg},
								Sel: &ast.Ident{Name: "TxContext"},
							},
						},
					},
				},
			},
		}
	case "pool":
		prefix = []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{ctxIdent},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.TypeAssertExpr{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{Name: poolVar},
								Sel: &ast.Ident{Name: "Get"},
							},
						},
						Type: &ast.StarExpr{
							X: &ast.SelectorExpr{
								X:   &ast.Ident{Name: cowPkg},
								Sel: &ast.Ident{Name: "TxContext"},
							},
						},
					},
				},
			},
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: poolVar},
						Sel: &ast.Ident{Name: "Put"},
					},
					Args: []ast.Expr{ctxIdent},
				},
			},
		}
	default:
		if len(inject) > 6 && inject[:6] == "param:" {
			name := inject[6:]
			for _, f := range fn.Type.Params.List {
				for _, n := range f.Names {
					if n.Name == name {
						return ctxResult{Expr: n}
					}
				}
			}
			return ctxResult{Skipped: true}
		}
		return ctxResult{Skipped: true}
	}
	return ctxResult{Expr: ctxIdent, Prefix: prefix}
}
