package cow

import (
	"errors"
	"testing"
)

func runScopedWithRollback(p *Player, fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	return fn(ctx)
}

func runScopedCommit(p *Player, fn func(ctx *TxContext) error) error {
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)
	if err := fn(ctx); err != nil {
		ctx.Rollback()
		return err
	}
	ctx.Reset()
	return nil
}

func TestRollback_RestoresInitialState(t *testing.T) {
	player := newBenchPlayer()
	want := clonePlayerSnapshot(player)

	err := runScopedWithRollback(player, func(ctx *TxContext) error {
		applySparseWrites(player, ctx)
		return errors.New("business error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, player, want)
}

func TestCommit_KeepsMutations(t *testing.T) {
	player := newBenchPlayer()
	before := clonePlayerSnapshot(player)

	err := runScopedCommit(player, func(ctx *TxContext) error {
		applySparseWrites(player, ctx)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if player.Assets["gold"] == before.Assets["gold"] {
		t.Fatal("gold should change after commit")
	}
	if len(player.Items) != len(before.Items)+1 {
		t.Fatalf("items len got %d want %d", len(player.Items), len(before.Items)+1)
	}
	if player.MainHero.Level != 2 {
		t.Fatalf("hero level got %d want 2", player.MainHero.Level)
	}
}
