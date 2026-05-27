package cow

import "testing"

func newPlayerWithHeros() *Player {
	return &Player{
		MainHero: &Hero{HeroId: 1, Level: 1},
		Heros: map[int32]*Hero{
			1: {HeroId: 1, Level: 10},
			2: {HeroId: 2, Level: 20},
		},
		Assets: map[string]int64{"gold": 100},
	}
}

func TestRollback_SetMainHero(t *testing.T) {
	p := newPlayerWithHeros()
	old := p.MainHero
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	incoming := &Hero{HeroId: 99, Level: 99}
	p.SetMainHero(ctx, incoming)
	if p.MainHero != incoming {
		t.Fatal("SetMainHero should store the same pointer as caller passed")
	}
	if p.MainHero.Level != 99 {
		t.Fatal("SetMainHero did not apply")
	}
	ctx.Rollback()
	if p.MainHero != old || p.MainHero.Level != 1 {
		t.Fatal("Rollback did not restore MainHero pointer")
	}
}

func TestRollback_SetMainHeroNil(t *testing.T) {
	p := newPlayerWithHeros()
	old := p.MainHero
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	p.SetMainHero(ctx, nil)
	if p.MainHero != nil {
		t.Fatal("expected nil MainHero")
	}
	ctx.Rollback()
	if p.MainHero != old {
		t.Fatal("Rollback did not restore MainHero")
	}
}

func TestRollback_RemoveHeros(t *testing.T) {
	p := newPlayerWithHeros()
	before := p.Heros[1]
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	p.RemoveHeros(ctx, 1)
	if _, ok := p.Heros[1]; ok {
		t.Fatal("key should be deleted")
	}
	ctx.Rollback()
	if p.Heros[1] != before {
		t.Fatal("Rollback did not restore Heros entry")
	}
}

func TestRollback_RemoveHerosNoOp(t *testing.T) {
	p := newPlayerWithHeros()
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)
	p.RemoveHeros(ctx, 999)
	if len(ctx.ops) != 0 {
		t.Fatalf("expected no undo ops, got %d", len(ctx.ops))
	}
}

func TestRollback_RemoveHerosNilMap(t *testing.T) {
	p := &Player{}
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)
	p.RemoveHeros(ctx, 1)
	if len(ctx.ops) != 0 {
		t.Fatalf("expected no undo on nil map, got %d", len(ctx.ops))
	}
}

func TestRollback_RemoveAssets(t *testing.T) {
	p := newPlayerWithHeros()
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	p.RemoveAssets(ctx, "gold")
	if _, ok := p.Assets["gold"]; ok {
		t.Fatal("gold key should be removed")
	}
	ctx.Rollback()
	if p.Assets["gold"] != 100 {
		t.Fatal("Rollback did not restore Assets")
	}
}

func TestPutHerosNilNotSameAsRemove(t *testing.T) {
	p := newPlayerWithHeros()
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer func() {
		ctx.Rollback()
		txPool.Put(ctx)
	}()
	p.PutHeros(ctx, 1, nil)
	if _, ok := p.Heros[1]; !ok {
		t.Fatal("PutHeros(nil) should keep key with nil value")
	}
	ctx.Rollback()
	if p.Heros[1] == nil || p.Heros[1].Level != 10 {
		t.Fatal("Rollback should restore previous hero")
	}
}
