package cowgen_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestBuildGraph_Player(t *testing.T) {
	pkg, err := cowmon.LoadPackage("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	g, err := cowgen.BuildGraph(pkg)
	if err != nil {
		t.Fatal(err)
	}
	var player *cowgen.StructPlan
	for _, sp := range g.Structs {
		if sp.Name == "Player" {
			player = sp
			break
		}
	}
	if player == nil {
		t.Fatal("no Player plan")
	}
	hasItems, hasAssets, hasMainHero, hasStats := false, false, false, false
	for _, p := range player.Plans {
		switch p.FieldName {
		case "Items":
			hasItems = p.Kind == cowgen.KindSlicePtr
		case "Assets":
			hasAssets = p.Kind == cowgen.KindMapScalar
		case "MainHero":
			hasMainHero = p.Kind == cowgen.KindPtrStruct
		case "Stats":
			hasStats = p.Kind == cowgen.KindMapMapScalar
		}
	}
	if !hasItems || !hasAssets || !hasMainHero || !hasStats {
		t.Fatalf("missing plans: items=%v assets=%v hero=%v stats=%v", hasItems, hasAssets, hasMainHero, hasStats)
	}
}
