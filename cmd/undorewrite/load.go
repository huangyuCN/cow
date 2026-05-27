package main

import (
	"fmt"
	"go/token"

	"github.com/huangyuCN/cow/internal/cowmon"
	"github.com/huangyuCN/cow/internal/cowproxy"
	"golang.org/x/tools/go/packages"
)

// packageEnv 保存单包改写所需的监控集与改写目录。
type packageEnv struct {
	Mon       *cowmon.MonitoredSet
	Catalog   *cowproxy.RewriteCatalog
	TxPkgPath string // *TxContext 所属包路径
}

type workspace struct {
	Fset      *token.FileSet
	Pkgs      []*packages.Package
	ByPath    map[string]*packageEnv
	CowImport string
	CowName   string // 默认 import 名 cow
}

func (ws *workspace) envForPkgPath(path string) (*packageEnv, bool) {
	env, ok := ws.ByPath[path]
	return env, ok
}

func resolvePackageEnv(pkg *packages.Package, cowImport string) (*packageEnv, error) {
	if pkg.Types == nil || len(pkg.Syntax) == 0 {
		return nil, fmt.Errorf("package %s missing types", pkg.PkgPath)
	}
	if mon, err := cowmon.BuildFromSyntax(pkg.Types, pkg.Syntax); err == nil {
		cat, err := cowproxy.NewCatalog(pkg.PkgPath)
		if err != nil {
			return nil, err
		}
		return &packageEnv{Mon: mon, Catalog: cat, TxPkgPath: pkg.PkgPath}, nil
	}
	if cowmon.Imports(pkg.Types, cowImport) {
		mon, err := cowmon.LoadMonitored(cowImport)
		if err != nil {
			return nil, err
		}
		cat, err := cowproxy.NewCatalog(cowImport)
		if err != nil {
			return nil, err
		}
		return &packageEnv{Mon: mon, Catalog: cat, TxPkgPath: cowImport}, nil
	}
	return nil, nil
}

func loadWorkspace(cfg Config, patterns []string) (*workspace, error) {
	mode := packages.NeedName | packages.NeedImports | packages.NeedSyntax |
		packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule
	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{Mode: mode, Fset: fset}, patterns...)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package load errors")
	}
	ws := &workspace{
		Fset:      fset,
		Pkgs:      pkgs,
		ByPath:    make(map[string]*packageEnv),
		CowImport: cfg.CowImport,
		CowName:   "cow",
	}
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || len(pkg.Errors) > 0 {
			continue
		}
		env, err := resolvePackageEnv(pkg, cfg.CowImport)
		if err != nil {
			return nil, err
		}
		if env != nil {
			ws.ByPath[pkg.PkgPath] = env
		}
	}
	return ws, nil
}

func (ws *workspace) cowPkgName(pkg *packages.Package) string {
	for path, imp := range pkg.Imports {
		if path == ws.CowImport && imp.Name != "" {
			return imp.Name
		}
	}
	return ws.CowName
}
