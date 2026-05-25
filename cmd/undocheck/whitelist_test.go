package main

import "testing"

func TestSkipFile(t *testing.T) {
	if !skipFile("bench_fixture.go") {
		t.Fatal("fixture should skip")
	}
	if !skipFile("/repo/zz_generated.undo_proxy.go") {
		t.Fatal("generated should skip")
	}
	if skipFile("player_test.go") {
		t.Fatal("test file must not skip")
	}
}

func TestSuggestProxy(t *testing.T) {
	if got := suggestProxy("Player", "Level", writeScalar); got != "PutLevel(ctx, …)" {
		t.Fatalf("got %q", got)
	}
	if got := suggestProxy("Player", "Items", writeSliceAppend); got != "AppendItems(ctx, …)" {
		t.Fatalf("got %q", got)
	}
}
