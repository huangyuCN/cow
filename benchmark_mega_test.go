package cow

import "testing"

func BenchmarkMega_UndoLog_SparseWrite_Rollback(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		applyMegaSparseWrites(player, ctx)
		ctx.Rollback()
		txPool.Put(ctx)
	}
}

func BenchmarkMega_UndoLog_SparseWrite_Commit(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		applyMegaSparseWrites(player, ctx)
		ctx.Reset()
		txPool.Put(ctx)
	}
}

func BenchmarkMega_DeepCopyGen_SparseWrite(b *testing.B) {
	seed := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		work := seed.DeepCopy()
		sparseWriteMegaDirect(work)
	}
}

func BenchmarkMega_UndoLog_SparseWrite32_Rollback(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		applyMegaSparseWrites32(player, ctx)
		ctx.Rollback()
		txPool.Put(ctx)
	}
}

func BenchmarkMega_UndoLog_SparseWrite32_Commit(b *testing.B) {
	player := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		applyMegaSparseWrites32(player, ctx)
		ctx.Reset()
		txPool.Put(ctx)
	}
}

func BenchmarkMega_DeepCopyGen_SparseWrite32(b *testing.B) {
	seed := newMegaBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		work := seed.DeepCopy()
		sparseWriteMegaDirect32(work)
	}
}
