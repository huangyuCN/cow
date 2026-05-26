package main

import (
	"github.com/huangyuCN/cow/internal/cowgen"
)

func emit(output, pkgName string, g *cowgen.Graph) error {
	return emitFromGraph(output, pkgName, g)
}
