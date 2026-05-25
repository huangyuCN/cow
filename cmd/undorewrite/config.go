package main

// Config CLI 配置。
type Config struct {
	CowImport string
	Write     bool
	CtxName   string
	InjectCtx string
	PoolVar   string
}

// Result 一次运行结果。
type Result struct {
	FilesChanged int
	RewriteCount int
	SkippedFuncs int
	Errors       []string
	Diffs        []fileDiff
}

type fileDiff struct {
	Path   string
	Before string
	After  string
}
