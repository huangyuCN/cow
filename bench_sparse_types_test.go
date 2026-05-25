package cow

import (
	"maps"
	"testing"
)

const benchSparseMapSize = 128

type benchSparseRoot struct {
	Comps []*benchSparseComp
}

type benchSparseComp struct {
	Gold  int
	Items map[int]int
}

func newBenchSparseRoot(compCount int) *benchSparseRoot {
	root := &benchSparseRoot{
		Comps: make([]*benchSparseComp, 0, compCount),
	}
	for i := 0; i < compCount; i++ {
		items := make(map[int]int, benchSparseMapSize)
		for key := 0; key < benchSparseMapSize; key++ {
			items[key] = key + i
		}
		root.Comps = append(root.Comps, &benchSparseComp{
			Gold:  i,
			Items: items,
		})
	}
	return root
}

func cloneBenchSparseRoot(src *benchSparseRoot) *benchSparseRoot {
	next := &benchSparseRoot{
		Comps: make([]*benchSparseComp, 0, len(src.Comps)),
	}
	for _, comp := range src.Comps {
		next.Comps = append(next.Comps, &benchSparseComp{
			Gold:  comp.Gold,
			Items: maps.Clone(comp.Items),
		})
	}
	return next
}

func mutableSparseComp(sess *TxSession[benchSparseRoot], idx int) *benchSparseComp {
	root := sess.ensureWritable()
	if _, ok := sess.cloned["comps"]; !ok {
		root.Comps = append([]*benchSparseComp(nil), sess.base.Comps...)
		sess.markCloned("comps")
	}
	name := "comp"
	sess.markDirty(name)

	if root.Comps[idx] == sess.base.Comps[idx] {
		baseComp := sess.base.Comps[idx]
		root.Comps[idx] = &benchSparseComp{
			Gold:  baseComp.Gold,
			Items: baseComp.Items,
		}
	}

	return root.Comps[idx]
}

func mutableSparseItems(sess *TxSession[benchSparseRoot], idx int) map[int]int {
	comp := mutableSparseComp(sess, idx)
	name := "comp.items"
	if _, ok := sess.cloned[name]; !ok {
		comp.Items = maps.Clone(sess.base.Comps[idx].Items)
		sess.markCloned(name)
	}
	return comp.Items
}

func TestBenchSparseWriteCommitKeepsUntouchedComponentsShared(t *testing.T) {
	store := newMemoryStore(newBenchSparseRoot(16))
	before := store.Load()

	sess, err := Begin(store, cloneBenchSparseRoot)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	comp := mutableSparseComp(sess, 0)
	comp.Gold++
	items := mutableSparseItems(sess, 0)
	items[0]++

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	after := store.Load()
	if after.Comps[0] == before.Comps[0] {
		t.Fatal("expected written component to be replaced")
	}
	if after.Comps[1] != before.Comps[1] {
		t.Fatal("expected untouched component to remain shared")
	}
	if before.Comps[0].Items[0] == after.Comps[0].Items[0] {
		t.Fatal("expected written component items map to be detached")
	}
}
