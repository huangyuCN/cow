package main

import (
	"strings"
	"testing"
)

func TestRewriteLegacyDryRun(t *testing.T) {
	cfg := Config{
		CowImport: "github.com/huangyuCN/cow",
		Write:     false,
		CtxName:   "ctx",
	}
	res, err := Run(cfg, []string{"./testdata/legacy"})
	if err != nil {
		t.Fatal(err)
	}
	if res.RewriteCount == 0 {
		t.Fatal("expected rewrites")
	}
	all := ""
	for _, d := range res.Diffs {
		all += d.After
	}
	for _, want := range []string{
		"PutLevel(ctx,",
		"PutAssets(ctx,",
		"GetMainHeroForWrite(ctx)",
		"AppendItems(ctx,",
	} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q in output:\n%s", want, all)
		}
	}
}
