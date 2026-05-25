package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	tagType = "+cow:undoproxy-gen=true"
)

// PackageInfo 已加载的目标包信息。
type PackageInfo struct {
	Name       string
	ImportPath string
	Pkg        *types.Package
	Roots      []*types.Named
	Structs    map[string]*types.Named // 同包具名 struct
}

func loadPackage(importPath string) (*PackageInfo, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %s", importPath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package load: %v", pkg.Errors[0])
	}
	info := &PackageInfo{
		Name:       pkg.Name,
		ImportPath: pkg.PkgPath,
		Pkg:        pkg.Types,
		Structs:    make(map[string]*types.Named),
	}
	scope := pkg.Types.Scope()
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
	for _, syn := range pkg.Syntax {
		for _, decl := range syn.Decls {
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
		return nil, fmt.Errorf("no type with %s in %s", tagType, importPath)
	}
	return info, nil
}

func hasUndoGenTag(docs ...*ast.CommentGroup) bool {
	for _, d := range docs {
		if d == nil {
			continue
		}
		for _, c := range d.List {
			if strings.Contains(c.Text, tagType) {
				return true
			}
		}
	}
	return false
}
