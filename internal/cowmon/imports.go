package cowmon

import "go/types"

// Imports 判断 pkg 是否直接 import 指定路径。
func Imports(pkg *types.Package, importPath string) bool {
	if pkg == nil {
		return false
	}
	for _, imp := range pkg.Imports() {
		if imp != nil && imp.Path() == importPath {
			return true
		}
	}
	return false
}
