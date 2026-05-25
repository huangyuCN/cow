package main

import (
	"go/ast"

	"github.com/huangyuCN/cow/internal/cowfile"
)

func skipFile(filename string) bool {
	return cowfile.SkipFile(filename)
}

func allowBareWrite(commentGroups ...*ast.CommentGroup) bool {
	return cowfile.AllowBareWrite(commentGroups...)
}
