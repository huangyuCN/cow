package cowmon_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestUnion_mergesSameSet(t *testing.T) {
	const cowPath = "github.com/huangyuCN/cow"
	info, err := cowmon.LoadPackage(cowPath)
	if err != nil {
		t.Fatal(err)
	}
	setA, err := cowmon.LoadMonitored(cowPath)
	if err != nil {
		t.Fatal(err)
	}
	player := info.Structs["Player"]
	if player == nil {
		t.Fatal("missing Player type")
	}
	merged := cowmon.Union(setA, setA)
	if merged == nil || !merged.Contains(player) {
		t.Fatal("expected Player in merged set")
	}
}

func TestUnion_emptyReturnsNil(t *testing.T) {
	if got := cowmon.Union(); got != nil {
		t.Fatalf("Union() with no args should be nil, got %v", got)
	}
	if got := cowmon.Union(nil, nil); got != nil {
		t.Fatalf("Union(nil) should be nil, got %v", got)
	}
}
