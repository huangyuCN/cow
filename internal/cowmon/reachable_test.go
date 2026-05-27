package cowmon_test

import (
	"go/types"
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestMonitoredSet_Contains_distinguishesSameNameAcrossPackages(t *testing.T) {
	set, err := cowmon.LoadMonitored("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	pkgA := types.NewPackage("a.example", "a")
	pkgB := types.NewPackage("b.example", "b")
	heroA := types.NewNamed(types.NewTypeName(0, pkgA, "Hero", nil), types.NewStruct(nil, nil), nil)
	heroB := types.NewNamed(types.NewTypeName(0, pkgB, "Hero", nil), types.NewStruct(nil, nil), nil)

	if !set.ContainsName("Hero") {
		t.Fatal("expected cow Hero in monitored set")
	}
	if set.Contains(heroB) {
		t.Fatal("unrelated Hero in another package must not match by name")
	}
	if set.Contains(heroA) {
		t.Fatal("synthetic Hero must not match cow monitored Hero")
	}
}
