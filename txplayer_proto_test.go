package cow

import "testing"

func TestTxPlayerReadsFallbackToBase(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)

	if got := tx.Name(); got != "hero" {
		t.Fatalf("Name() = %q, want hero", got)
	}
	if got := tx.Level(); got != 10 {
		t.Fatalf("Level() = %d, want 10", got)
	}
	if got, ok := tx.Item(1001); !ok || got != 1 {
		t.Fatalf("Item(1001) = (%d, %v), want (1, true)", got, ok)
	}
	if got, ok := tx.Skill(1); !ok || got != 22 {
		t.Fatalf("Skill(1) = (%d, %v), want (22, true)", got, ok)
	}
}

func TestTxPlayerSetScalarDoesNotMutateBase(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)
	tx.SetName("mage")
	tx.SetLevel(20)

	if base.Name != "hero" {
		t.Fatalf("base.Name = %q, want hero", base.Name)
	}
	if base.Level != 10 {
		t.Fatalf("base.Level = %d, want 10", base.Level)
	}
}

func TestTxPlayerMapAndSliceWritesDoNotMutateBase(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)
	tx.SetItem(1001, 9)
	tx.DeleteItem(1002)
	tx.SetSkill(1, 77)
	tx.AppendSkill(99)

	if got := base.Items[1001]; got != 1 {
		t.Fatalf("base.Items[1001] = %d, want 1", got)
	}
	if got := base.Items[1002]; got != 2 {
		t.Fatalf("base.Items[1002] = %d, want 2", got)
	}
	if got := base.Skills[1]; got != 22 {
		t.Fatalf("base.Skills[1] = %d, want 22", got)
	}
	if got := len(base.Skills); got != 3 {
		t.Fatalf("len(base.Skills) = %d, want 3", got)
	}
}

func TestTxPlayerCommitRebuildsTopLevelAndReusesUntouchedFields(t *testing.T) {
	base := newPlayer()

	tx := BeginPlayer(base)
	tx.SetName("mage")
	tx.SetItem(1001, 9)

	out := tx.Commit()

	if out == base {
		t.Fatal("Commit() should return a new top-level Player")
	}
	if out.Name != "mage" {
		t.Fatalf("out.Name = %q, want mage", out.Name)
	}
	if out.Level != base.Level {
		t.Fatalf("out.Level = %d, want %d", out.Level, base.Level)
	}
	if out.Items[1001] != 9 {
		t.Fatalf("out.Items[1001] = %d, want 9", out.Items[1001])
	}
	if out.Skills == nil || len(out.Skills) != len(base.Skills) {
		t.Fatalf("out.Skills should reuse base slice")
	}
	if &out.Skills[0] != &base.Skills[0] {
		t.Fatal("untouched slice field should reuse base backing data")
	}
}
