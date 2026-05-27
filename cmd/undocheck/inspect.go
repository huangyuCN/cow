package main

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/huangyuCN/cow/internal/cowmon"
	"golang.org/x/tools/go/analysis"
)

const cowImportPath = "github.com/huangyuCN/cow"

const specDoc = "docs/superpowers/specs/2026-05-25-bare-write-guard-design.md"

func inspectFile(pass *analysis.Pass, f *ast.File, mon *cowmon.MonitoredSet) {
	if allowBareWrite(f.Doc) {
		return
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if allowBareWrite(d.Doc) {
				continue
			}
			ast.Inspect(d.Body, func(n ast.Node) bool {
				inspectStmt(pass, mon, d.Doc, n)
				return true
			})
		}
	}
}

func inspectStmt(pass *analysis.Pass, mon *cowmon.MonitoredSet, fnDoc *ast.CommentGroup, n ast.Node) {
	switch s := n.(type) {
	case *ast.AssignStmt:
		for i, lhs := range s.Lhs {
			if i >= len(s.Rhs) {
				break
			}
			kind := writeScalar
			if call, ok := s.Rhs[i].(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "append" {
					kind = writeSliceAppend
				}
			}
			checkExpr(pass, mon, lhs, kind, s.Pos())
		}
	case *ast.IncDecStmt:
		checkExpr(pass, mon, s.X, writeScalar, s.Pos())
	case *ast.CompositeLit:
		checkComposite(pass, mon, s)
	case *ast.CallExpr:
		if id, ok := s.Fun.(*ast.Ident); ok && id.Name == "delete" && len(s.Args) >= 1 {
			checkMapDelete(pass, mon, s.Args[0], s.Pos())
		}
	}
}

func checkMapDelete(pass *analysis.Pass, mon *cowmon.MonitoredSet, mapExpr ast.Expr, pos token.Pos) {
	tv := pass.TypesInfo.Types[mapExpr]
	if tv.Type == nil {
		return
	}
	if _, isMap := tv.Type.Underlying().(*types.Map); !isMap {
		return
	}
	root := rootMonitoredStruct(pass, mapExpr)
	if root == "" || !mon.ContainsName(root) {
		return
	}
	field := fieldNameFromExpr(pass, mapExpr)
	if field == "" {
		return
	}
	reportBare(pass, root, field, writeMapDelete, pos)
}

func checkComposite(pass *analysis.Pass, mon *cowmon.MonitoredSet, lit *ast.CompositeLit) {
	tv := pass.TypesInfo.Types[lit]
	if tv.Type == nil {
		return
	}
	named := namedStructType(tv.Type)
	if named == nil || !mon.Contains(named) {
		return
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			// 非键值形式的位置复合字面量，保守报整个字面量
			reportBare(pass, named.Obj().Name(), "composite literal", writeScalar, lit.Pos())
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok {
			if structHasField(st, key.Name) {
				reportBare(pass, named.Obj().Name(), key.Name, writeScalar, kv.Pos())
			}
		}
	}
}

func checkExpr(pass *analysis.Pass, mon *cowmon.MonitoredSet, expr ast.Expr, kind writeKind, pos token.Pos) {
	field, typeName, ok := monitoredWriteTarget(pass, expr)
	if !ok || !mon.ContainsName(typeName) {
		return
	}
	reportBare(pass, typeName, field, kind, pos)
}

// monitoredWriteTarget 解析写左值：返回字段名、监控类型名。
func monitoredWriteTarget(pass *analysis.Pass, expr ast.Expr) (field, typeName string, ok bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		tv := pass.TypesInfo.Types[e.X]
		if tv.Type == nil {
			return "", "", false
		}
		named := namedStructType(tv.Type)
		if named == nil {
			return "", "", false
		}
		sel := pass.TypesInfo.Selections[e]
		if sel == nil || !sel.Obj().Exported() {
			return "", "", false
		}
		if _, ok := sel.Obj().(*types.Var); !ok {
			return "", "", false
		}
		return sel.Obj().Name(), named.Obj().Name(), true
	case *ast.IndexExpr:
		tv := pass.TypesInfo.Types[e.X]
		if tv.Type == nil {
			return "", "", false
		}
		// map/slice 下标写：根为监控 struct 字段则报
		if root := rootMonitoredStruct(pass, e.X); root != "" {
			if _, isMap := tv.Type.Underlying().(*types.Map); isMap {
				if field := fieldNameFromExpr(pass, e.X); field != "" {
					return field, root, true
				}
			}
			if _, isSlice := tv.Type.Underlying().(*types.Slice); isSlice {
				if field := fieldNameFromExpr(pass, e.X); field != "" {
					return field, root, true
				}
			}
		}
		// 内层 map[string]int64 等：根不是监控 struct 类型则不报
		return "", "", false
	default:
		return "", "", false
	}
}

func rootMonitoredStruct(pass *analysis.Pass, expr ast.Expr) string {
	for {
		switch e := expr.(type) {
		case *ast.SelectorExpr:
			tv := pass.TypesInfo.Types[e.X]
			if tv.Type == nil {
				return ""
			}
			if named := namedStructType(tv.Type); named != nil {
				return named.Obj().Name()
			}
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.Ident:
			tv := pass.TypesInfo.Types[e]
			if tv.Type == nil {
				return ""
			}
			if named := namedStructType(tv.Type); named != nil {
				return named.Obj().Name()
			}
			return ""
		default:
			return ""
		}
	}
}

func fieldNameFromExpr(pass *analysis.Pass, expr ast.Expr) string {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if s := pass.TypesInfo.Selections[sel]; s != nil {
			return s.Obj().Name()
		}
	}
	return ""
}

func namedStructType(t types.Type) *types.Named {
	switch u := t.(type) {
	case *types.Named:
		if _, ok := u.Underlying().(*types.Struct); ok {
			return u
		}
	case *types.Pointer:
		if n, ok := u.Elem().(*types.Named); ok {
			if _, ok := n.Underlying().(*types.Struct); ok {
				return n
			}
		}
	}
	return nil
}

func structHasField(st *types.Struct, name string) bool {
	for i := 0; i < st.NumFields(); i++ {
		if st.Field(i).Name() == name {
			return true
		}
	}
	return false
}

func reportBare(pass *analysis.Pass, typeName, field string, kind writeKind, pos token.Pos) {
	hint := suggestProxy(typeName, field, kind)
	pass.Reportf(pos, "cowbarewrite: 禁止对 *%s 裸写 %s，请使用 %s（见 %s）", typeName, field, hint, specDoc)
}
