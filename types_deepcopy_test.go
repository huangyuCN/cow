package cow

import "testing"

func TestPlayerDeepCopy_Isolated(t *testing.T) {
	src := newPlayerForDeepCopyTest()
	dst := src.DeepCopy()
	ctx := &TxContext{}
	src.PutAssets(ctx, "gold", 999)
	if dst.Assets["gold"] == 999 {
		t.Fatal("deep copy shares map")
	}
}
