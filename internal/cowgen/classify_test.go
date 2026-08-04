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

func TestBuildGraph_exactKeyAndNamedScalar(t *testing.T) {
	pkg, err := cowmon.LoadPackage("github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata")
	if err != nil {
		t.Fatal(err)
	}
	g, err := cowgen.BuildGraph(pkg)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	var root *cowgen.StructPlan
	for _, sp := range g.Structs {
		if sp.Name == "TypedSlotsRoot" {
			root = sp
			break
		}
	}
	if root == nil {
		t.Fatal("TypedSlotsRoot not in graph")
	}
	var byInt, score bool
	for _, p := range root.Plans {
		switch p.FieldName {
		case "ByInt":
			byInt = len(p.Keys) == 2 &&
				p.Keys[0].KeyType == "int" &&
				p.Keys[1].KeyType == "string"
		case "Score":
			score = p.Kind == cowgen.KindScalar && p.LeafType == "Cat"
		}
	}
	if !byInt {
		t.Fatal("ByInt: want Keys[0]=int Keys[1]=string")
	}
	if !score {
		t.Fatal("Score: want KindScalar with LeafType Cat")
	}
}
