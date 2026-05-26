package cow

import "testing"

func BenchmarkUndoLogV2_SparseWrite_Commit(b *testing.B) {
	seed := newBenchPlayer()
	player := seed.DeepCopy()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPoolV2.Get().(*TxContextV2)
		ctx.Reset()
		applySparseWritesV2(player, ctx)
		ctx.Reset()
		txPoolV2.Put(ctx)
		b.StopTimer()
		player = seed.DeepCopy()
		b.StartTimer()
	}
}

func BenchmarkUndoLogV2_SparseWrite_Rollback(b *testing.B) {
	player := newBenchPlayer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := txPoolV2.Get().(*TxContextV2)
		ctx.Reset()
		applySparseWritesV2(player, ctx)
		ctx.Rollback()
		txPoolV2.Put(ctx)
	}
}
