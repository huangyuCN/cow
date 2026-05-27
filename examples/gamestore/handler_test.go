package gamestore

import (
	"errors"
	"os"
	"strings"
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
	g := NewDemoGuild()
	beforeGold := p.Gold
	err := runScopedCommit(func(ctx *TxContext) error {
		return HandlePurchaseSuccess(p, g, ctx)
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Gold != beforeGold-100 {
		t.Fatalf("gold got %d want %d", p.Gold, beforeGold-100)
	}
	if g.Members[2] == nil || g.Members[2].Name != "bob" {
		t.Fatal("guild member 2 not committed")
	}
}

func TestGuildMember_Write_Rollback(t *testing.T) {
	g := NewDemoGuild()
	want := cloneGuild(g)
	err := runScopedWithRollback(func(ctx *TxContext) error {
		g.PutMembers(ctx, 99, newTempMember())
		return errors.New("abort")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !guildsEqual(g, want) {
		t.Fatal("guild not restored")
	}
}

func TestGenerated_contract(t *testing.T) {
	b, err := os.ReadFile("zz_generated.undo_proxy.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, needle := range []string{"type TxContext struct", "func (ctx *TxContext) Rollback()"} {
		if !strings.Contains(s, needle) {
			t.Fatalf("generated missing %q", needle)
		}
	}
	if strings.Contains(s, "AddUndo") {
		t.Fatal("generated should not contain AddUndo")
	}
}
