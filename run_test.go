package cow

import "testing"

func TestCommitAppliesChanges(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold += 5
	items := mutableBagItems(sess)
	items[1001] = 2

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	committed := store.Load()
	if committed.Bag.Gold != 15 {
		t.Fatalf("gold = %d, want 15", committed.Bag.Gold)
	}
	if committed.Bag.Items[1001] != 2 {
		t.Fatalf("item count = %d, want 2", committed.Bag.Items[1001])
	}
}

func TestRollbackDiscardsChanges(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold = 99
	sess.Rollback()

	committed := store.Load()
	if committed.Bag.Gold != 10 {
		t.Fatalf("gold = %d, want 10", committed.Bag.Gold)
	}
}

func TestCommitMarksDirtyComponents(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold++
	dirty := sess.Dirty()

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if len(dirty) != 1 || dirty[0] != "bag" {
		t.Fatalf("dirty = %v, want [bag]", dirty)
	}
}

func TestBeginStartsReadOnly(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if sess.work != nil {
		t.Fatal("work should stay nil before first write")
	}
	if sess.dirty != nil {
		t.Fatal("dirty should stay nil before first write")
	}
	if sess.cloned != nil {
		t.Fatal("cloned should stay nil before first write")
	}
	if sess.checkpoints != nil {
		t.Fatal("checkpoints should stay nil before first savepoint")
	}
}

func TestReadOnlyCommitSkipsWorkAllocation(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	base := store.Load()

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if store.Load() != base {
		t.Fatal("read-only commit should keep committed root pointer unchanged")
	}
	if sess.work != nil {
		t.Fatal("read-only commit should not materialize work root")
	}
	if !sess.finished {
		t.Fatal("session should be finished after commit")
	}
}

func TestDirtyOnReadOnlySessionReturnsEmpty(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	dirty := sess.Dirty()
	if len(dirty) != 0 {
		t.Fatalf("dirty = %v, want empty", dirty)
	}
	if sess.dirty != nil {
		t.Fatal("Dirty() should not allocate dirty set on read-only session")
	}
}
