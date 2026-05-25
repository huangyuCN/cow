package main

import (
	"github.com/huangyuCN/cow/internal/cowmon"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "cowbarewrite",
	Doc:      "disallow bare writes to +cow:undoproxy-gen monitored structs",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	_ = pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	mon, err := monitoredForPass(pass)
	if err != nil {
		return nil, err
	}
	if mon == nil {
		return nil, nil
	}
	for _, f := range pass.Files {
		path := pass.Fset.File(f.Pos()).Name()
		if skipFile(path) {
			continue
		}
		inspectFile(pass, f, mon)
	}
	return nil, nil
}

func monitoredForPass(pass *analysis.Pass) (*cowmon.MonitoredSet, error) {
	if set, err := cowmon.BuildFromSyntax(pass.Pkg, pass.Files); err == nil {
		return set, nil
	}
	if importsCow(pass) {
		return cowmon.LoadMonitored(cowImportPath)
	}
	return nil, nil
}

func importsCow(pass *analysis.Pass) bool {
	for _, imp := range pass.Pkg.Imports() {
		if imp.Path() == cowImportPath {
			return true
		}
	}
	return false
}
