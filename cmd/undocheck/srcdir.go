package main

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// testdataSrcDir 若当前 Pass 位于 analysistest 的 testdata/src 下，返回该 src 根目录。
func testdataSrcDir(pass *analysis.Pass) (string, bool) {
	if len(pass.Files) == 0 {
		return "", false
	}
	name := filepath.ToSlash(pass.Fset.File(pass.Files[0].Pos()).Name())
	const marker = "/testdata/src/"
	i := strings.Index(name, marker)
	if i < 0 {
		return "", false
	}
	return filepath.FromSlash(name[:i+len(marker)-1]), true
}
