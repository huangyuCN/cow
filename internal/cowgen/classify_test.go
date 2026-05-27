package cowgen_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestBuildGraph_mapSliceTypeAlias(t *testing.T) {
	pkg, err := cowmon.LoadPackage("github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata")
	if err != nil {
		t.Fatal(err)
	}
	g, err := cowgen.BuildGraph(pkg)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	var back *cowgen.StructPlan
	for _, sp := range g.Structs {
		if sp.Name == "EquipBack" {
			back = sp
			break
		}
	}
	if back == nil {
		t.Fatal("EquipBack not in graph")
	}
	var equips, spares bool
	for _, p := range back.Plans {
		switch p.FieldName {
		case "Equips":
			equips = p.Kind == cowgen.KindMapPtrStruct && p.DeclaredType == "Equips"
		case "Spares":
			spares = p.Kind == cowgen.KindSlicePtr && p.DeclaredType == "ItemList"
		}
	}
	if !equips {
		t.Fatal("Equips: want KindMapPtrStruct with DeclaredType Equips")
	}
	if !spares {
		t.Fatal("Spares: want KindSlicePtr with DeclaredType ItemList")
	}
}
