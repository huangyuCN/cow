package aftershop

import (
	"testing"
)

func TestHandlePurchaseFail_Rollback(t *testing.T) {
	p := NewDemoPlayer()
	want := clonePlayer(p)
	err := runScopedWithRollback(func(ctx *TxContext) error {
		return HandlePurchaseFail(p, ctx)
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !playersEqual(p, want) {
		t.Fatal("player not restored after rollback")
	}
}

func TestHandlePurchaseSuccess_Commit(t *testing.T) {
	p := NewDemoPlayer()
	beforeGold := p.Gold
	beforeWallet := p.Wallet["gold"]
	beforeHero := p.MainHero.Level
	beforeItems := len(p.Items)

	if err := runScopedCommit(func(ctx *TxContext) error {
		HandlePurchase(p, ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if p.Gold != beforeGold-100 {
		t.Fatalf("gold got %d want %d", p.Gold, beforeGold-100)
	}
	if p.Wallet["gold"] != beforeWallet+50 {
		t.Fatalf("wallet[gold] got %d want %d", p.Wallet["gold"], beforeWallet+50)
	}
	if p.MainHero.Level != beforeHero+1 {
		t.Fatalf("hero level got %d want %d", p.MainHero.Level, beforeHero+1)
	}
	if len(p.Items) != beforeItems+1 {
		t.Fatalf("items len got %d want %d", len(p.Items), beforeItems+1)
	}
}

