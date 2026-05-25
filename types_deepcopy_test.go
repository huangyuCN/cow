package cow

import "testing"

func TestPlayerDeepCopy_Isolated(t *testing.T) {
	src := &Player{
		Uid:    1,
		Assets: map[string]int64{"gold": 1},
		MainHero: &Hero{Level: 1},
	}
	dst := src.DeepCopy()
	src.Assets["gold"] = 999
	if dst.Assets["gold"] == 999 {
		t.Fatal("deep copy shares map")
	}
}
