package cow

import "testing"

func applyMegaSparseWritesV2(p *Player, ctx *TxContextV2) {
	p.PutAssetsV2(ctx, "gold", 500)
	h := p.GetHeroForWriteV2(ctx, 1)
	if h != nil {
		h.PutLevelV2(ctx, 99)
	}
	p.AppendItemsV2(ctx, newTestItem(9999, "mega_probe"))
	p.AppendBagsAtV2(ctx, 1, newTestItem(8888, "bag_probe"))
	p.PutStatsV2(ctx, 1, "atk", 100)
}

func BenchmarkMega_UndoLogV2_SparseWrite_Rollback(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPoolV2.Get().(*TxContextV2)
		ctx.Reset()
		applyMegaSparseWritesV2(player, ctx)
		ctx.Rollback()
		txPoolV2.Put(ctx)
	}
}

func BenchmarkMega_UndoLogV2_SparseWrite_Commit(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPoolV2.Get().(*TxContextV2)
		ctx.Reset()
		applyMegaSparseWritesV2(player, ctx)
		ctx.Reset()
		txPoolV2.Put(ctx)
	}
}
