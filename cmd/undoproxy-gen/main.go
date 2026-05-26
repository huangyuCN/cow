// undoproxy-gen 为带 +cow:undoproxy-gen 标记的类型生成 Undo 写代理。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func main() {
	output := flag.String("output-file", "", "output Go file path")
	mode := flag.String("mode", "v1", "generate mode: v1 or v2")
	flag.Parse()
	if *output == "" || flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: undoproxy-gen --output-file FILE [--mode v1|v2] IMPORT_PATH\n")
		os.Exit(2)
	}
	if err := Run(*output, flag.Arg(0), *mode); err != nil {
		fmt.Fprintf(os.Stderr, "undoproxy-gen: %v\n", err)
		os.Exit(1)
	}
}

// Run 加载包、构建类型图并写入生成文件。
func Run(output, importPath, mode string) error {
	pkg, err := loadPackage(importPath)
	if err != nil {
		return err
	}
	graph, err := cowgen.BuildGraph(pkg)
	if err != nil {
		return err
	}
	return emit(output, pkg.Name, graph, mode)
}
