package cow

import (
	"errors"
	"testing"
)

func TestRollback_SliceRemoveRestore(t *testing.T) {
	p := &Player{Items: []*Item{{Id: 1}, {Id: 2}}}
	want := clonePlayerSnapshot(p)

	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		p.RemoveItemsAt(ctx, 0)
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, p, want)
}

func TestRollback_SliceTruncateRestore(t *testing.T) {
	p := &Player{Items: []*Item{{Id: 1}, {Id: 2}, {Id: 3}}}
	want := clonePlayerSnapshot(p)

	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		p.TruncateItems(ctx, 1)
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, p, want)
}
