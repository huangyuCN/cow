package cow

import "testing"

func sparseWriteDirect(p *Player) {
	p.Assets["gold"] = 500
	p.Items = append(p.Items, &Item{Id: 9999, Name: "Shield"})
	p.MainHero.Level = 2
}

func sparseWriteUndo(p *Player, ctx *TxContext) {
	applySparseWrites(p, ctx)
}

func BenchmarkUndoLog_SparseWrite_Commit(b *testing.B) {
	seed := newBenchPlayer()
	player := seed.DeepCopy()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		sparseWriteUndo(player, ctx)
		ctx.Reset()
		txPool.Put(ctx)
		b.StopTimer()
		player = seed.DeepCopy()
		b.StartTimer()
	}
}

func BenchmarkUndoLog_SparseWrite_Rollback(b *testing.B) {
	player := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		sparseWriteUndo(player, ctx)
		ctx.Rollback()
		txPool.Put(ctx)
	}
}

func BenchmarkDeepCopyGen_SparseWrite(b *testing.B) {
	seed := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		work := seed.DeepCopy()
		sparseWriteDirect(work)
	}
}
