package cowmon

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// BuildFromSyntax 从已类型检查的包与语法树构建监控集合（供 analysis.Pass 使用）。
func BuildFromSyntax(pkg *types.Package, files []*ast.File) (*MonitoredSet, error) {
	info, err := packageInfoFromTypes(pkg, files)
	if err != nil {
		return nil, err
	}
	reachable, err := CollectReachable(info)
	if err != nil {
		return nil, err
	}
	set := &MonitoredSet{
		byObj:   make(map[*types.TypeName]struct{}, len(reachable)),
		pkgPath: info.ImportPath,
	}
	for _, n := range reachable {
		set.byObj[n.Obj()] = struct{}{}
	}
	return set, nil
}

func packageInfoFromTypes(pkg *types.Package, files []*ast.File) (*PackageInfo, error) {
	info := &PackageInfo{
		Name:       pkg.Name(),
		ImportPath: pkg.Path(),
		Pkg:        pkg,
		Structs:    make(map[string]*types.Named),
	}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); !ok {
			continue
		}
		info.Structs[name] = named
	}
	for _, f := range files {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !hasUndoGenTag(gen.Doc, ts.Doc, ts.Comment) {
					continue
				}
				named, ok := info.Structs[ts.Name.Name]
				if ok {
					info.Roots = append(info.Roots, named)
				}
			}
		}
	}
	if len(info.Roots) == 0 {
		return nil, fmt.Errorf("no type with %s in %s", TagUndoGen, pkg.Path())
	}
	return info, nil
}
