package cowmon_test

import (
	"go/types"
	"strings"
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
)

// TestCollectReachable_RejectsExternalStructField 跨包 struct 字段超出类型图能力，
// 错误须点名外部类型（而不是含糊的 "unsupported type struct{...}"）。
func TestCollectReachable_RejectsExternalStructField(t *testing.T) {
	pkg := types.NewPackage("a.example", "a")
	ext := types.NewPackage("ext.example", "ext")
	extNamed := types.NewNamed(types.NewTypeName(0, ext, "External", nil), types.NewStruct(nil, nil), nil)
	rootStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "E", extNamed, false)}, nil)
	root := types.NewNamed(types.NewTypeName(0, pkg, "Root", nil), rootStruct, nil)

	_, err := cowmon.CollectReachable(&cowmon.PackageInfo{Pkg: pkg, Roots: []*types.Named{root}})
	if err == nil {
		t.Fatal("expected error for cross-package struct field")
	}
	if !strings.Contains(err.Error(), "External") {
		t.Fatalf("error should name the external type, got: %v", err)
	}
}
