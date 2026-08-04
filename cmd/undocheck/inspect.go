package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/huangyuCN/cow/internal/cowmon"
	"golang.org/x/tools/go/analysis"
)

func allowBareWriteAtPos(fset *token.FileSet, f *ast.File, pos token.Pos) bool {
	if pos == token.NoPos {
		return false
	}
	line := fset.Position(pos).Line
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			cLine := fset.Position(c.Pos()).Line
			if cLine != line && cLine != line-1 {
				continue
			}
			if strings.Contains(c.Text, "cow:allow-bare-write") {
				return true
			}
		}
	}
	return false
}

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
				inspectStmt(pass, f, mon, n)
				return true
			})
		}
	}
}

func inspectStmt(pass *analysis.Pass, f *ast.File, mon *cowmon.MonitoredSet, n ast.Node) {
	switch s := n.(type) {
	case *ast.AssignStmt:
		if allowBareWriteAtPos(pass.Fset, f, s.Pos()) {
			return
		}
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
			checkExpr(pass, f, mon, lhs, kind, s.Pos())
		}
	case *ast.IncDecStmt:
		if allowBareWriteAtPos(pass.Fset, f, s.Pos()) {
			return
		}
		checkExpr(pass, f, mon, s.X, writeScalar, s.Pos())
	case *ast.CompositeLit:
		checkComposite(pass, f, mon, s)
	case *ast.CallExpr:
		if id, ok := s.Fun.(*ast.Ident); ok && id.Name == "delete" && len(s.Args) >= 1 {
			if allowBareWriteAtPos(pass.Fset, f, s.Pos()) {
				return
			}
			checkMapDelete(pass, f, mon, s.Args[0], s.Pos())
		}
	}
}

func checkMapDelete(pass *analysis.Pass, f *ast.File, mon *cowmon.MonitoredSet, mapExpr ast.Expr, pos token.Pos) {
	tv := pass.TypesInfo.Types[mapExpr]
	if tv.Type == nil {
		return
	}
	if _, isMap := tv.Type.Underlying().(*types.Map); !isMap {
		return
	}
	root := rootMonitoredStruct(pass, mapExpr)
	if root == nil || !mon.Contains(root) {
		return
	}
	field := fieldNameFromExpr(pass, mapExpr)
	if field == "" {
		return
	}
	reportBare(pass, f, root.Obj().Name(), field, writeMapDelete, pos)
}

func checkComposite(pass *analysis.Pass, f *ast.File, mon *cowmon.MonitoredSet, lit *ast.CompositeLit) {
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
			if allowBareWriteAtPos(pass.Fset, f, lit.Pos()) {
				continue
			}
			// 非键值形式的位置复合字面量，保守报整个字面量
			reportBare(pass, f, named.Obj().Name(), "composite literal", writeScalar, lit.Pos())
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok {
			if structHasField(st, key.Name) {
				if allowBareWriteAtPos(pass.Fset, f, kv.Pos()) {
					continue
				}
				reportBare(pass, f, named.Obj().Name(), key.Name, writeScalar, kv.Pos())
			}
		}
	}
}

func checkExpr(pass *analysis.Pass, f *ast.File, mon *cowmon.MonitoredSet, expr ast.Expr, kind writeKind, pos token.Pos) {
	field, root, ok := monitoredWriteTarget(pass, expr)
	if !ok || !mon.Contains(root) {
		return
	}
	reportBare(pass, f, root.Obj().Name(), field, kind, pos)
}

// monitoredWriteTarget 解析写左值：返回字段名与根 struct 类型。
func monitoredWriteTarget(pass *analysis.Pass, expr ast.Expr) (field string, root *types.Named, ok bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		tv := pass.TypesInfo.Types[e.X]
		if tv.Type == nil {
			return "", nil, false
		}
		named := namedStructType(tv.Type)
		if named == nil {
			return "", nil, false
		}
		sel := pass.TypesInfo.Selections[e]
		if sel == nil || !sel.Obj().Exported() {
			return "", nil, false
		}
		if _, ok := sel.Obj().(*types.Var); !ok {
			return "", nil, false
		}
		return sel.Obj().Name(), named, true
	case *ast.IndexExpr:
		tv := pass.TypesInfo.Types[e.X]
		if tv.Type == nil {
			return "", nil, false
		}
		// map/slice 下标写：根为监控 struct 字段则报
		if root := rootMonitoredStruct(pass, e.X); root != nil {
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
		return "", nil, false
	default:
		return "", nil, false
	}
}

func rootMonitoredStruct(pass *analysis.Pass, expr ast.Expr) *types.Named {
	for {
		switch e := expr.(type) {
		case *ast.SelectorExpr:
			tv := pass.TypesInfo.Types[e.X]
			if tv.Type == nil {
				return nil
			}
			if named := namedStructType(tv.Type); named != nil {
				return named
			}
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.Ident:
			tv := pass.TypesInfo.Types[e]
			if tv.Type == nil {
				return nil
			}
			if named := namedStructType(tv.Type); named != nil {
				return named
			}
			return nil
		default:
			return nil
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

func reportBare(pass *analysis.Pass, f *ast.File, typeName, field string, kind writeKind, pos token.Pos) {
	if allowBareWriteAtPos(pass.Fset, f, pos) {
		return
	}
	hint := suggestProxy(typeName, field, kind)
	pass.Reportf(pos, "cowbarewrite: 禁止对 *%s 裸写 %s，请使用 %s（见 %s）", typeName, field, hint, specDoc)
}
