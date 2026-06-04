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
	var sets []*cowmon.MonitoredSet
	if set, err := cowmon.BuildFromSyntax(pass.Pkg, pass.Files); err == nil {
		sets = append(sets, set)
	}
	srcDir, _ := testdataSrcDir(pass)
	for _, imp := range pass.Pkg.Imports() {
		if imp == nil {
			continue
		}
		path := imp.Path()
		if path == "" || path == pass.Pkg.Path() {
			continue
		}
		set, err := cowmon.LoadMonitoredCached(path, srcDir)
		if err != nil {
			continue
		}
		sets = append(sets, set)
	}
	return cowmon.Union(sets...), nil
}
