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

func resolveCtx(fn *ast.FuncDecl, info *types.Info, txPkgPath, ctxName, inject, poolVar, cowImport, cowLocalName string) ctxResult {
	if fn.Body == nil {
		return ctxResult{Skipped: true}
	}
	if inject != "" {
		return injectCtx(fn, txPkgPath, cowImport, cowLocalName, inject, poolVar)
	}
	if id := findCtxInParams(fn, info, txPkgPath, ctxName); id != nil {
		return ctxResult{Expr: id}
	}
	if id := findCtxInBody(fn.Body, info, txPkgPath, ctxName); id != nil {
		return ctxResult{Expr: id}
	}
	return ctxResult{Skipped: true}
}

func findCtxInParams(fn *ast.FuncDecl, info *types.Info, txPkgPath, prefer string) ast.Expr {
	if fn.Type.Params == nil {
		return nil
	}
	var fallback ast.Expr
	for _, f := range fn.Type.Params.List {
		for _, n := range f.Names {
			if !isTxContextType(info.TypeOf(n), txPkgPath) {
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

func findCtxInBody(body *ast.BlockStmt, info *types.Info, txPkgPath, prefer string) ast.Expr {
	if prefer != "" {
		if id := findCtxName(body, info, txPkgPath, prefer); id != nil {
			return id
		}
	}
	for _, name := range []string{"ctx", "tx", "txCtx"} {
		if id := findCtxName(body, info, txPkgPath, name); id != nil {
			return id
		}
	}
	return nil
}

func findCtxName(body *ast.BlockStmt, info *types.Info, txPkgPath, name string) ast.Expr {
	var found ast.Expr
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		id, ok := n.(*ast.Ident)
		if !ok || id.Name != name {
			return true
		}
		if isTxContextType(info.TypeOf(id), txPkgPath) {
			found = id
			return false
		}
		return true
	})
	return found
}

func isTxContextType(t types.Type, txPkgPath string) bool {
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
	return named.Obj().Name() == "TxContext" && named.Obj().Pkg().Path() == txPkgPath
}

// txContextStructTypeExpr 返回 TxContext 结构体类型 AST（无指针）。
func txContextStructTypeExpr(txPkgPath, cowImport, cowLocalName string) ast.Expr {
	if txPkgPath == cowImport && cowLocalName != "" {
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: cowLocalName},
			Sel: &ast.Ident{Name: "TxContext"},
		}
	}
	return &ast.Ident{Name: "TxContext"}
}

func txContextPtrTypeExpr(txPkgPath, cowImport, cowLocalName string) ast.Expr {
	return &ast.StarExpr{X: txContextStructTypeExpr(txPkgPath, cowImport, cowLocalName)}
}

func injectCtx(fn *ast.FuncDecl, txPkgPath, cowImport, cowLocalName, inject, poolVar string) ctxResult {
	ctxIdent := &ast.Ident{Name: "ctx"}
	txStruct := txContextStructTypeExpr(txPkgPath, cowImport, cowLocalName)
	txPtr := txContextPtrTypeExpr(txPkgPath, cowImport, cowLocalName)
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
						X:  &ast.CompositeLit{Type: txStruct},
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
						Type: txPtr,
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
