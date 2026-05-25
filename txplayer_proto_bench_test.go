package cow

import "testing"

func BenchmarkTxPlayerReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		tx := BeginPlayer(base)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		_ = clonePlayer(base)
	}
}

func BenchmarkTxPlayerReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		tx := BeginPlayer(base)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		_ = clonePlayer(base)
	}
}

func BenchmarkTxPlayerReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		tx := BeginPlayer(base)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		_ = clonePlayer(base)
	}
}

func BenchmarkTxPlayerSparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		tx := BeginPlayer(base)
		tx.SetLevel(20)
		tx.SetItem(0, 99)
		tx.SetSkill(0, 77)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneSparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(16))
		out.Level = 20
		out.Items[0] = 99
		out.Skills[0] = 77
	}
}

func BenchmarkTxPlayerSparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		tx := BeginPlayer(base)
		tx.SetLevel(20)
		tx.SetItem(0, 99)
		tx.SetSkill(0, 77)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneSparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(64))
		out.Level = 20
		out.Items[0] = 99
		out.Skills[0] = 77
	}
}

func BenchmarkTxPlayerSparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		tx := BeginPlayer(base)
		tx.SetLevel(20)
		tx.SetItem(0, 99)
		tx.SetSkill(0, 77)
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneSparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(256))
		out.Level = 20
		out.Items[0] = 99
		out.Skills[0] = 77
	}
}

func BenchmarkTxPlayerHotWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(16)
		tx := BeginPlayer(base)
		for j := 0; j < 8; j++ {
			tx.SetItem(0, j)
			tx.SetSkill(0, j)
		}
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneHotWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(16))
		for j := 0; j < 8; j++ {
			out.Items[0] = j
			out.Skills[0] = j
		}
	}
}

func BenchmarkTxPlayerHotWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(64)
		tx := BeginPlayer(base)
		for j := 0; j < 8; j++ {
			tx.SetItem(0, j)
			tx.SetSkill(0, j)
		}
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneHotWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(64))
		for j := 0; j < 8; j++ {
			out.Items[0] = j
			out.Skills[0] = j
		}
	}
}

func BenchmarkTxPlayerHotWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base := newBenchPlayer(256)
		tx := BeginPlayer(base)
		for j := 0; j < 8; j++ {
			tx.SetItem(0, j)
			tx.SetSkill(0, j)
		}
		_ = tx.Commit()
	}
}

func BenchmarkEagerCloneHotWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		out := clonePlayer(newBenchPlayer(256))
		for j := 0; j < 8; j++ {
			out.Items[0] = j
			out.Skills[0] = j
		}
	}
}
