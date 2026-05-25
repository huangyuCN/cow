package cowfile

import (
	"go/ast"
	"path/filepath"
	"strings"
)

// SkipFile 是否跳过整文件（生成代码、夹具等）。
func SkipFile(filename string) bool {
	base := filepath.Base(filename)
	if strings.HasPrefix(base, "zz_generated") {
		return true
	}
	if strings.HasSuffix(base, "_fixture.go") || strings.HasSuffix(base, "_fixtures.go") {
		return true
	}
	switch base {
	case "deepcopy_generate.go", "undo_proxy_generate.go":
		return true
	}
	if strings.Contains(filepath.ToSlash(filename), "/cmd/undoproxy-gen/") {
		return true
	}
	return false
}

// AllowBareWrite 行级逃逸注释。
func AllowBareWrite(commentGroups ...*ast.CommentGroup) bool {
	for _, g := range commentGroups {
		if g == nil {
			continue
		}
		for _, c := range g.List {
			if strings.Contains(c.Text, "cow:allow-bare-write") {
				return true
			}
		}
	}
	return false
}
