package main

import (
	"fmt"
	"go/token"

	"github.com/huangyuCN/cow/internal/cowmon"
	"github.com/huangyuCN/cow/internal/cowproxy"
	"golang.org/x/tools/go/packages"
)

type workspace struct {
	Fset      *token.FileSet
	Pkgs      []*packages.Package
	Mon       *cowmon.MonitoredSet
	Catalog   *cowproxy.RewriteCatalog
	CowImport string
	CowName   string // 本地 import 名，默认 cow
}

func loadWorkspace(cfg Config, patterns []string) (*workspace, error) {
	mon, err := cowmon.LoadMonitored(cfg.CowImport)
	if err != nil {
		return nil, err
	}
	cat, err := cowproxy.NewCatalog(cfg.CowImport)
	if err != nil {
		return nil, err
	}
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
		Mon:       mon,
		Catalog:   cat,
		CowImport: cfg.CowImport,
		CowName:   "cow",
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
