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
