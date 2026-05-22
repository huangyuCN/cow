package cow

import "testing"

func TestSavepointRollbackToLatest(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold = 20

	sp1, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}

	bag = mutableBag(sess)
	bag.Gold = 30

	sp2, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}

	bag = mutableBag(sess)
	bag.Gold = 40

	if err := sess.RollbackTo(sp2); err != nil {
		t.Fatalf("RollbackTo(sp2) error = %v", err)
	}
	if got := mutableBag(sess).Gold; got != 30 {
		t.Fatalf("gold after sp2 rollback = %d, want 30", got)
	}

	if err := sess.RollbackTo(sp1); err != nil {
		t.Fatalf("RollbackTo(sp1) error = %v", err)
	}
	if got := mutableBag(sess).Gold; got != 20 {
		t.Fatalf("gold after sp1 rollback = %d, want 20", got)
	}
}

func TestSavepointRejectsOutOfOrderRollback(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	sp1, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}
	sp2, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}
	if err := sess.RollbackTo(sp1); err == nil {
		t.Fatal("expected out-of-order rollback error")
	}
	if err := sess.RollbackTo(sp2); err != nil {
		t.Fatalf("RollbackTo(sp2) error = %v", err)
	}
}

func TestReadOnlySavepointDoesNotUpgradeSession(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	sp, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}
	if sp == 0 {
		t.Fatal("savepoint id should be assigned")
	}
	if sess.work != nil {
		t.Fatal("read-only savepoint should not materialize work root")
	}
	if len(sess.checkpoints) != 1 {
		t.Fatalf("checkpoint count = %d, want 1", len(sess.checkpoints))
	}
	if sess.checkpoints[0].writable {
		t.Fatal("read-only savepoint should record writable=false")
	}
}

func TestRollbackToReadOnlySavepointRestoresReadOnlyState(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	sp, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}

	bag := mutableBag(sess)
	bag.Gold = 99
	if sess.work == nil {
		t.Fatal("write should materialize work root")
	}

	if err := sess.RollbackTo(sp); err != nil {
		t.Fatalf("RollbackTo() error = %v", err)
	}
	if sess.work != nil {
		t.Fatal("rollback to read-only savepoint should clear work root")
	}
	if sess.dirty != nil {
		t.Fatal("rollback to read-only savepoint should clear dirty set")
	}
	if sess.cloned != nil {
		t.Fatal("rollback to read-only savepoint should clear cloned set")
	}

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got := store.Load().Bag.Gold; got != 10 {
		t.Fatalf("gold after commit = %d, want 10", got)
	}
}
