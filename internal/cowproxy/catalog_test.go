package cowproxy_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowproxy"
)

func TestCatalog_PlayerMainHero(t *testing.T) {
	cat, err := cowproxy.NewCatalog("github.com/huangyuCN/cow")
	if err != nil {
		t.Fatal(err)
	}
	mh, ok := cat.Lookup("Player", "MainHero")
	if !ok || mh.GetForWrite != "GetMainHeroForWrite" {
		t.Fatalf("MainHero: %+v ok=%v", mh, ok)
	}
	st, ok := cat.Lookup("Player", "Stats")
	if !ok || st.MapPutKeyCount != 2 {
		t.Fatalf("Stats: %+v", st)
	}
	it, ok := cat.Lookup("Player", "Items")
	if !ok || it.Append != "AppendItems" {
		t.Fatalf("Items: %+v", it)
	}
}
