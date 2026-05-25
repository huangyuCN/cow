package cow

import "testing"

func BenchmarkFrameworkBeginCommitRollback(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
		sess, err := Begin(store, cloneTestRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrameworkEmptyClosure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = func() error {
			return nil
		}()
	}
}

func BenchmarkCowWritePathOnSession(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sess := newBenchSession()
		bag := mutableBag(sess)
		bag.Gold++
		items := mutableBagItems(sess)
		items[1001]++
	}
}

func BenchmarkCowWritePathInSessionLifecycle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
		sess, err := Begin(store, cloneTestRoot)
		if err != nil {
			b.Fatal(err)
		}
		bag := mutableBag(sess)
		bag.Gold++
		items := mutableBagItems(sess)
		items[1001]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeepCopyWritePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneTestRoot(newTestRoot())
		root.Bag.Gold++
		root.Bag.Items[1001]++
	}
}

func BenchmarkEndToEndSessionWithCow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newTestRoot())
		sess, err := Begin(store, cloneTestRoot)
		if err != nil {
			b.Fatal(err)
		}
		bag := mutableBag(sess)
		bag.Gold++
		items := mutableBagItems(sess)
		items[1001]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEndToEndSessionWithDeepCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneTestRoot(newTestRoot())
		root.Bag.Gold++
		root.Bag.Items[1001]++
	}
}

func BenchmarkCowSparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(16))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		comp := mutableSparseComp(sess, 0)
		comp.Gold++
		items := mutableSparseItems(sess, 0)
		items[0]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowSparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(64))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		comp := mutableSparseComp(sess, 0)
		comp.Gold++
		items := mutableSparseItems(sess, 0)
		items[0]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowSparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(256))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		comp := mutableSparseComp(sess, 0)
		comp.Gold++
		items := mutableSparseItems(sess, 0)
		items[0]++
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeepCopySparseWrite16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneBenchSparseRoot(newBenchSparseRoot(16))
		root.Comps[0].Gold++
		root.Comps[0].Items[0]++
	}
}

func BenchmarkDeepCopySparseWrite64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneBenchSparseRoot(newBenchSparseRoot(64))
		root.Comps[0].Gold++
		root.Comps[0].Items[0]++
	}
}

func BenchmarkDeepCopySparseWrite256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := cloneBenchSparseRoot(newBenchSparseRoot(256))
		root.Comps[0].Gold++
		root.Comps[0].Items[0]++
	}
}

func BenchmarkCowReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(16))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(64))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCowReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := newMemoryStore(newBenchSparseRoot(256))
		sess, err := Begin(store, cloneBenchSparseRoot)
		if err != nil {
			b.Fatal(err)
		}
		if err := sess.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeepCopyReadOnly16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloneBenchSparseRoot(newBenchSparseRoot(16))
	}
}

func BenchmarkDeepCopyReadOnly64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloneBenchSparseRoot(newBenchSparseRoot(64))
	}
}

func BenchmarkDeepCopyReadOnly256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cloneBenchSparseRoot(newBenchSparseRoot(256))
	}
}
