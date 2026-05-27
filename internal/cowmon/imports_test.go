package cowmon_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
	"golang.org/x/tools/go/packages"
)

func TestImports_cowRoot(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedImports | packages.NeedTypes | packages.NeedDeps,
	}, "github.com/huangyuCN/cow/cmd/undorewrite/testdata/legacy")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages")
	}
	if !cowmon.Imports(pkgs[0].Types, "github.com/huangyuCN/cow") {
		t.Fatal("legacy should import cow")
	}
}
