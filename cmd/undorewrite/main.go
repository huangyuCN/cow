// undorewrite 将监控类型的裸写批量改为 undoproxy 代理调用。
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cowImport := flag.String("cow", "github.com/huangyuCN/cow", "cow 模块 import path")
	write := flag.Bool("w", false, "write changes to source files")
	ctxName := flag.String("ctx", "ctx", "TxContext 变量名")
	injectCtx := flag.String("inject-ctx", "", "new|pool|param:NAME")
	poolVar := flag.String("pool-var", "txPool", "sync.Pool 变量名（inject-ctx=pool）")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "usage: undorewrite [flags] ./patterns...\n")
		os.Exit(2)
	}
	cfg := Config{
		CowImport: *cowImport,
		Write:     *write,
		CtxName:   *ctxName,
		InjectCtx: *injectCtx,
		PoolVar:   *poolVar,
	}
	res, err := Run(cfg, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "undorewrite: %v\n", err)
		os.Exit(1)
	}
	if !cfg.Write {
		printDiffs(res.Diffs)
	}
	printSummary(os.Stderr, res)
	if len(res.Errors) > 0 {
		os.Exit(1)
	}
}

func printSummary(w *os.File, res *Result) {
	fmt.Fprintf(w, "undorewrite: %d files touched, %d rewrites, %d funcs skipped\n",
		res.FilesChanged, res.RewriteCount, res.SkippedFuncs)
	for _, e := range res.Errors {
		fmt.Fprintf(w, "undorewrite: %s\n", e)
	}
}
