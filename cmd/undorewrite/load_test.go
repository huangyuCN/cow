package main

import (
	"testing"
)

func TestResolvePackageEnv_consumer(t *testing.T) {
	cfg := Config{CowImport: "github.com/huangyuCN/cow"}
	ws, err := loadWorkspace(cfg, []string{"./testdata/consumer"})
	if err != nil {
		t.Fatal(err)
	}
	env, ok := ws.envForPkgPath("github.com/huangyuCN/cow/cmd/undorewrite/testdata/consumer")
	if !ok {
		t.Fatal("missing consumer env")
	}
	if env.Mon == nil || env.Catalog == nil {
		t.Fatal("nil mon or catalog")
	}
	if !env.Mon.ContainsName("Player") {
		t.Fatal("Player not monitored")
	}
	if _, ok := env.Catalog.Lookup("Player", "Assets"); !ok {
		t.Fatal("Assets methods missing")
	}
	if env.TxPkgPath != "github.com/huangyuCN/cow/cmd/undorewrite/testdata/consumer" {
		t.Fatalf("TxPkgPath=%q", env.TxPkgPath)
	}
}

func TestResolvePackageEnv_legacyFallback(t *testing.T) {
	cfg := Config{CowImport: "github.com/huangyuCN/cow"}
	ws, err := loadWorkspace(cfg, []string{"./testdata/legacy"})
	if err != nil {
		t.Fatal(err)
	}
	env, ok := ws.envForPkgPath("github.com/huangyuCN/cow/cmd/undorewrite/testdata/legacy")
	if !ok || env.Mon == nil {
		t.Fatal("legacy env")
	}
	if !env.Mon.ContainsName("Player") {
		t.Fatal("cow Player via fallback")
	}
	if env.TxPkgPath != cfg.CowImport {
		t.Fatalf("TxPkgPath=%q want cow import", env.TxPkgPath)
	}
}
