package cow

import "testing"

func TestRollbackRejectsConsumedSavepoint(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	sp, err := sess.Savepoint()
	if err != nil {
		t.Fatalf("Savepoint() error = %v", err)
	}
	if err := sess.RollbackTo(sp); err != nil {
		t.Fatalf("RollbackTo() error = %v", err)
	}
	if err := sess.RollbackTo(sp); err != ErrInvalidSavepoint {
		t.Fatalf("error = %v, want %v", err, ErrInvalidSavepoint)
	}
}

func TestCommitAfterRollbackReturnsSessionClosed(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	sess.Rollback()

	if err := sess.Commit(); err != ErrSessionClosed {
		t.Fatalf("error = %v, want %v", err, ErrSessionClosed)
	}
}

func TestWriteAfterRollbackPanicsWithSessionClosed(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	sess.Rollback()

	defer func() {
		got := recover()
		if got != ErrSessionClosed {
			t.Fatalf("panic = %v, want %v", got, ErrSessionClosed)
		}
	}()

	mutableBag(sess).Gold++
}
