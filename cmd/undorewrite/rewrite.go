package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"os"

	"github.com/huangyuCN/cow/internal/cowfile"
	"github.com/huangyuCN/cow/internal/cowmon"
	"github.com/huangyuCN/cow/internal/cowproxy"
	"golang.org/x/tools/go/packages"
)

func Run(cfg Config, patterns []string) (*Result, error) {
	ws, err := loadWorkspace(cfg, patterns)
	if err != nil {
		return nil, err
	}
	res := &Result{}
	for _, pkg := range ws.Pkgs {
		env, ok := ws.envForPkgPath(pkg.PkgPath)
		if !ok {
			continue
		}
		for _, f := range pkg.Syntax {
			path := ws.Fset.File(f.Pos()).Name()
			if cowfile.SkipFile(path) {
				continue
			}
			fileRes := rewriteFile(ws, pkg, f, env, cfg)
			res.Errors = append(res.Errors, fileRes.errs...)
			res.SkippedFuncs += fileRes.skipped
			if fileRes.changed {
				res.FilesChanged++
				res.RewriteCount += fileRes.rewrites
				res.Diffs = append(res.Diffs, fileDiff{
					Path: path, Before: fileRes.before, After: fileRes.after,
				})
				if cfg.Write {
					if err := os.WriteFile(path, []byte(fileRes.after), 0o644); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return res, nil
}

type fileRewriteResult struct {
	changed  bool
	before   string
	after    string
	rewrites int
	skipped  int
	errs     []string
}

type catalogAdapter struct {
	*cowproxy.RewriteCatalog
}

func (c catalogAdapter) Lookup(structName, fieldName string) (fieldMethods, bool) {
	fm, ok := c.RewriteCatalog.Lookup(structName, fieldName)
	if !ok {
		return fieldMethods{}, false
	}
	return fieldMethods{
		Put: fm.Put, Append: fm.Append, SetAt: fm.SetAt,
		GetForWrite: fm.GetForWrite, GetAtForWrite: fm.GetAtForWrite,
		MapPutKeyCount: fm.MapPutKeyCount, TargetStruct: fm.TargetStruct,
	}, true
}

func rewriteFile(ws *workspace, pkg *packages.Package, f *ast.File, env *packageEnv, cfg Config) fileRewriteResult {
	info := pkg.TypesInfo
	cat := catalogAdapter{env.Catalog}
	out := fileRewriteResult{before: formatFile(ws.Fset, f)}
	cowLocal := ws.cowPkgName(pkg)

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || cowfile.AllowBareWrite(fn.Doc) {
			continue
		}
		ctxRes := resolveCtx(fn, info, env.TxPkgPath, cfg.CtxName, cfg.InjectCtx, cfg.PoolVar, ws.CowImport, cowLocal)
		if ctxRes.Skipped {
			out.skipped++
			out.errs = append(out.errs, fmt.Sprintf("%s: 缺少 TxContext", ws.Fset.Position(fn.Pos())))
			continue
		}
		if len(ctxRes.Prefix) > 0 {
			fn.Body.List = append(ctxRes.Prefix, fn.Body.List...)
		}
		var reps []stmtReplace
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				if nr, ok := rewriteAssign(s, info, env.Mon, cat, ctxRes.Expr); ok {
					reps = append(reps, stmtReplace{old: s, new: nr})
				}
			case *ast.IncDecStmt:
				if nr, ok := rewriteIncDec(s, info, env.Mon, cat, ctxRes.Expr); ok {
					reps = append(reps, stmtReplace{old: s, new: nr})
				}
			}
			return true
		})
		out.rewrites += applyStmtReplacements(fn.Body, reps)
	}
	if out.rewrites > 0 {
		out.changed = true
		out.after = formatFile(ws.Fset, f)
	}
	return out
}

type stmtReplace struct {
	old ast.Stmt
	new ast.Stmt
}

func applyStmtReplacements(body *ast.BlockStmt, reps []stmtReplace) int {
	return replaceStmtsInList(body.List, reps)
}

func replaceStmtsInList(list []ast.Stmt, reps []stmtReplace) int {
	n := 0
	for i, st := range list {
		for _, r := range reps {
			if st == r.old {
				list[i] = r.new
				n++
				break
			}
		}
		switch s := list[i].(type) {
		case *ast.BlockStmt:
			n += replaceStmtsInList(s.List, reps)
		case *ast.IfStmt:
			if s.Body != nil {
				n += replaceStmtsInList(s.Body.List, reps)
			}
			if s.Else != nil {
				switch el := s.Else.(type) {
				case *ast.BlockStmt:
					n += replaceStmtsInList(el.List, reps)
				case *ast.IfStmt:
					n += replaceStmtsInList([]ast.Stmt{el}, reps)
				}
			}
		}
	}
	return n
}

func rewriteAssign(s *ast.AssignStmt, info *types.Info, mon *cowmon.MonitoredSet, cat catalogAdapter, ctx ast.Expr) (ast.Stmt, bool) {
	if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
		return nil, false
	}
	lhs, rhs := s.Lhs[0], s.Rhs[0]
	if isProxyExpr(lhs, info) {
		return nil, false
	}
	if call, ok := rhs.(*ast.CallExpr); ok {
		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "append" {
			stmts := rewriteAppend(s, lhs, call, info, mon, cat, ctx)
			if len(stmts) == 1 {
				return stmts[0], true
			}
			if len(stmts) > 1 {
				return &ast.BlockStmt{List: stmts}, true
			}
			return nil, false
		}
	}
	wp, ok := parseWritePath(lhs, info, mon)
	if !ok {
		return nil, false
	}
	if ce := buildWriteCall(wp, rhs, cat, ctx); ce != nil {
		return &ast.ExprStmt{X: ce}, true
	}
	return nil, false
}

func rewriteIncDec(s *ast.IncDecStmt, info *types.Info, mon *cowmon.MonitoredSet, cat catalogAdapter, ctx ast.Expr) (ast.Stmt, bool) {
	wp, ok := parseWritePath(s.X, info, mon)
	if !ok {
		return nil, false
	}
	sel := cloneExpr(s.X)
	var rhs ast.Expr
	switch s.Tok {
	case token.INC:
		rhs = &ast.BinaryExpr{X: sel, Op: token.ADD, Y: &ast.BasicLit{Kind: token.INT, Value: "1"}}
	case token.DEC:
		rhs = &ast.BinaryExpr{X: sel, Op: token.SUB, Y: &ast.BasicLit{Kind: token.INT, Value: "1"}}
	default:
		return nil, false
	}
	if ce := buildWriteCall(wp, rhs, cat, ctx); ce != nil {
		return &ast.ExprStmt{X: ce}, true
	}
	return nil, false
}

func rewriteAppend(s *ast.AssignStmt, lhs ast.Expr, call *ast.CallExpr, info *types.Info, mon *cowmon.MonitoredSet, cat catalogAdapter, ctx ast.Expr) []ast.Stmt {
	wp, ok := parseWritePath(lhs, info, mon)
	if !ok || len(call.Args) < 2 {
		return nil
	}
	recv := receiverExpr(wp, cat, ctx)
	fm, _ := cat.Lookup(receiverStruct(wp.RootStruct, wp.Nav, cat), wp.LeafField)
	if fm.Append == "" {
		return nil
	}
	var stmts []ast.Stmt
	for _, arg := range call.Args[1:] {
		stmts = append(stmts, &ast.ExprStmt{X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: fm.Append}},
			Args: append([]ast.Expr{ctx}, arg),
		}})
	}
	return stmts
}

func buildWriteCall(wp *writePath, rhs ast.Expr, cat catalogAdapter, ctx ast.Expr) *ast.CallExpr {
	recv := receiverExpr(wp, cat, ctx)
	leafStruct := receiverStruct(wp.RootStruct, wp.Nav, cat)
	fm, ok := cat.Lookup(leafStruct, wp.LeafField)
	if !ok {
		return nil
	}
	args := []ast.Expr{ctx}
	switch wp.Kind {
	case writeMapMapPut:
		args = append(args, wp.MapKeys[0], wp.MapKeys[1], rhs)
		return &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: fm.Put}}, Args: args}
	case writeMapPut:
		args = append(args, wp.MapKeys[0], rhs)
		return &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: fm.Put}}, Args: args}
	case writeSetAt:
		args = append(args, wp.SliceIndex, rhs)
		return &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: fm.SetAt}}, Args: args}
	default:
		args = append(args, rhs)
		return &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: fm.Put}}, Args: args}
	}
}

func receiverExpr(wp *writePath, cat catalogAdapter, ctx ast.Expr) ast.Expr {
	cur := cloneExpr(wp.Root)
	st := wp.RootStruct
	for _, step := range wp.Nav {
		fm, ok := cat.Lookup(st, step.Field)
		if !ok {
			return cur
		}
		args := []ast.Expr{ctx}
		if step.MapKey != nil {
			args = append(args, cloneExpr(step.MapKey))
		}
		if fm.GetForWrite != "" {
			cur = &ast.CallExpr{
				Fun:  &ast.SelectorExpr{X: cur, Sel: &ast.Ident{Name: fm.GetForWrite}},
				Args: args,
			}
			if fm.TargetStruct != "" {
				st = fm.TargetStruct
			}
		}
	}
	if wp.SliceIndex != nil && len(wp.Nav) > 0 {
		last := wp.Nav[len(wp.Nav)-1]
		fm, ok := cat.Lookup(st, last.Field)
		if ok && fm.GetAtForWrite != "" {
			args := []ast.Expr{ctx, cloneExpr(last.MapKey), cloneExpr(wp.SliceIndex)}
			cur = &ast.CallExpr{
				Fun:  &ast.SelectorExpr{X: cur, Sel: &ast.Ident{Name: fm.GetAtForWrite}},
				Args: args,
			}
			if fm.TargetStruct != "" {
				st = fm.TargetStruct
			}
		}
	}
	return cur
}

func cloneExpr(e ast.Expr) ast.Expr {
	// 浅拷贝足够：仅复用同一 AST 中的子表达式引用
	return e
}

func isProxyExpr(e ast.Expr, info *types.Info) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	name := sel.Sel.Name
	return len(name) >= 3 && (name[:3] == "Put" || name[:3] == "Get" || len(name) > 6 && name[:6] == "Append")
}

func formatFile(fset *token.FileSet, f *ast.File) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return ""
	}
	return buf.String()
}
