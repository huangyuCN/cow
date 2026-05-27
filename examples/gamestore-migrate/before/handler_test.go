package beforeshop

import (
	"testing"
)

func TestHandlePurchase_Semantics(t *testing.T) {
	p := NewDemoPlayer()
	beforeGold := p.Gold
	beforeWalletGold := p.Wallet["gold"]
	beforeHeroLevel := p.MainHero.Level
	beforeItemsLen := len(p.Items)

	HandlePurchase(p)
	if p.Gold != beforeGold-100 {
		t.Fatalf("gold got %d want %d", p.Gold, beforeGold-100)
	}
	if p.Wallet["gold"] != beforeWalletGold+50 {
		t.Fatalf("wallet[gold] got %d want %d", p.Wallet["gold"], beforeWalletGold+50)
	}
	if len(p.Items) != beforeItemsLen+1 {
		t.Fatalf("items len got %d want %d", len(p.Items), beforeItemsLen+1)
	}
	if p.MainHero.Level != beforeHeroLevel+1 {
		t.Fatalf("hero level got %d want %d", p.MainHero.Level, beforeHeroLevel+1)
	}
}

func TestHandlePurchaseFail_SideEffectsOnError(t *testing.T) {
	p := NewDemoPlayer()
	beforeHeroLevel := p.MainHero.Level
	beforeItemsLen := len(p.Items)

	gotErr := HandlePurchaseFail(p)
	if gotErr == nil {
		t.Fatal("expected error")
	}
	if p.Gold != int64(99999) {
		t.Fatalf("gold got %d want %d", p.Gold, 99999)
	}
	if p.Wallet["gold"] != 0 {
		t.Fatalf("wallet[gold] got %d want %d", p.Wallet["gold"], 0)
	}
	if len(p.Items) != beforeItemsLen+1 {
		t.Fatalf("items len got %d want %d", len(p.Items), beforeItemsLen+1)
	}
	if p.MainHero.Level != beforeHeroLevel+100 {
		t.Fatalf("hero level got %d want %d", p.MainHero.Level, beforeHeroLevel+100)
	}
}

