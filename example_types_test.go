package cow

import "maps"

type testRoot struct {
	Bag   *testBagComp
	Quest *testQuestComp
}

type testBagComp struct {
	Gold  int
	Items map[int]int
}

type testQuestComp struct {
	Stage int
	Flags map[string]bool
}

func newTestRoot() *testRoot {
	return &testRoot{
		Bag: &testBagComp{
			Gold:  10,
			Items: map[int]int{1001: 1},
		},
		Quest: &testQuestComp{
			Stage: 1,
			Flags: map[string]bool{"daily": true},
		},
	}
}

func cloneTestRoot(src *testRoot) *testRoot {
	next := &testRoot{}
	if src.Bag != nil {
		next.Bag = &testBagComp{
			Gold:  src.Bag.Gold,
			Items: maps.Clone(src.Bag.Items),
		}
	}
	if src.Quest != nil {
		next.Quest = &testQuestComp{
			Stage: src.Quest.Stage,
			Flags: maps.Clone(src.Quest.Flags),
		}
	}
	return next
}

func mutableBag(sess *TxSession[testRoot]) *testBagComp {
	root := sess.ensureWritable()
	sess.markDirty("bag")
	if root.Bag == sess.base.Bag {
		baseBag := sess.base.Bag
		root.Bag = &testBagComp{
			Gold:  baseBag.Gold,
			Items: baseBag.Items,
		}
	}
	return Mutable(sess, func(root *testRoot) *testBagComp { return root.Bag })
}

func mutableBagItems(sess *TxSession[testRoot]) map[int]int {
	bag := mutableBag(sess)
	if _, ok := sess.cloned["bag.items"]; !ok {
		bag.Items = maps.Clone(sess.base.Bag.Items)
		sess.markCloned("bag.items")
	}
	return bag.Items
}
