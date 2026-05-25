package cowmon_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestLoadMonitored_cow(t *testing.T) {
	set, err := cowmon.LoadMonitored("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Player", "Hero", "Item", "Skill", "Mail", "Quest"} {
		if !set.ContainsName(name) {
			t.Fatalf("missing monitored type %s", name)
		}
	}
}
