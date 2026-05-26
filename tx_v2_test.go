package cow

import (
	"errors"
	"testing"
)

func runScopedWithRollbackV2(p *Player, fn func(ctx *TxContextV2) error) error {
	ctx := txPoolV2.Get().(*TxContextV2)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPoolV2.Put(ctx)
	}()
	return fn(ctx)
}

func runScopedCommitV2(p *Player, fn func(ctx *TxContextV2) error) error {
	ctx := txPoolV2.Get().(*TxContextV2)
	ctx.Reset()
	defer txPoolV2.Put(ctx)
	if err := fn(ctx); err != nil {
		ctx.Rollback()
		return err
	}
	ctx.Reset()
	return nil
}

// applySparseWritesV2 模拟一次请求的三处稀疏写（V2 实验路径）。
func applySparseWritesV2(p *Player, ctx *TxContextV2) {
	p.PutAssetsV2(ctx, "gold", 500)
	p.AppendItemsV2(ctx, newTestItem(9999, "Shield"))
	h := p.GetMainHeroForWriteV2(ctx)
	if h != nil {
		h.PutLevelV2(ctx, 2)
	}
}

func TestRollbackV2_RestoresInitialState(t *testing.T) {
	player := newBenchPlayer()
	want := clonePlayerSnapshot(player)

	err := runScopedWithRollbackV2(player, func(ctx *TxContextV2) error {
		applySparseWritesV2(player, ctx)
		return errors.New("business error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, player, want)
}

func TestCommitV2_KeepsMutations(t *testing.T) {
	player := newBenchPlayer()
	before := clonePlayerSnapshot(player)

	err := runScopedCommitV2(player, func(ctx *TxContextV2) error {
		applySparseWritesV2(player, ctx)
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
