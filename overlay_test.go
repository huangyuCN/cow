package cow

import "testing"

func TestOverlayCommitKeepsUntouchedComponentShared(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	before := store.Load()

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	bag := mutableBag(sess)
	bag.Gold += 5
	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	after := store.Load()
	if after.Bag == before.Bag {
		t.Fatal("expected bag component to be replaced after write")
	}
	if after.Quest != before.Quest {
		t.Fatal("expected untouched quest component to remain shared")
	}
}

func TestOverlayDoesNotCloneQuestWhenBagChanges(t *testing.T) {
	store := newMemoryStore(newTestRoot())

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	bag := mutableBag(sess)
	bag.Gold = 20
	if got := sess.Dirty(); len(got) != 1 || got[0] != "bag" {
		t.Fatalf("dirty = %v, want [bag]", got)
	}
	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
}

func TestMutableBagItemsCloneMapOnFirstWrite(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	before := store.Load()

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	items := mutableBagItems(sess)
	items[1001]++
	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	after := store.Load()
	if after.Bag.Items[1001] != 2 {
		t.Fatalf("item count = %d, want 2", after.Bag.Items[1001])
	}
	if before.Bag.Items[1001] != 1 {
		t.Fatalf("base item count = %d, want 1", before.Bag.Items[1001])
	}
	before.Bag.Items[2002] = 7
	if _, ok := after.Bag.Items[2002]; ok {
		t.Fatal("expected committed items map to be detached from base map")
	}
}

func TestMutableBagGoldDoesNotCloneItemsMap(t *testing.T) {
	store := newMemoryStore(newTestRoot())
	before := store.Load()

	sess, err := Begin(store, cloneTestRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	bag := mutableBag(sess)
	bag.Gold++
	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	after := store.Load()
	before.Bag.Items[2002] = 7
	if after.Bag.Items[2002] != 7 {
		t.Fatal("expected items map to stay shared when only gold changes")
	}
}
