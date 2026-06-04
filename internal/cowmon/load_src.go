package cowmon

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// LoadMonitoredFromSrcDir 在 GOPATH 式 src 根（如 analysistest 的 testdata/src）下加载监控集。
func LoadMonitoredFromSrcDir(importPath, srcDir string) (*MonitoredSet, error) {
	info, err := loadPackageFromSrcDir(importPath, srcDir)
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

func loadPackageFromSrcDir(importPath, srcDir string) (*PackageInfo, error) {
	// analysistest 布局：GOPATH=<testdata>，源码在 <testdata>/src/<import path>。
	gopath := filepath.Dir(srcDir)
	if filepath.Base(srcDir) != "src" {
		gopath = srcDir
	}
	gopath, err := filepath.Abs(gopath)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  gopath,
		Env:  append(os.Environ(), "GO111MODULE=off", "GOPATH="+gopath),
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %s under %s", importPath, srcDir)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package load: %w", pkg.Errors[0])
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
		return nil, fmt.Errorf("no type with %s in %s", TagUndoGen, importPath)
	}
	return info, nil
}
