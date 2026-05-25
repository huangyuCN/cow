package main

import "github.com/huangyuCN/cow/internal/cowmon"

// PackageInfo 已加载的目标包信息。
type PackageInfo = cowmon.PackageInfo

func loadPackage(importPath string) (*PackageInfo, error) {
	return cowmon.LoadPackage(importPath)
}
