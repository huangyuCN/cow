package cowgen_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func TestSingular(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Heros", "Hero"},
		{"Items", "Item"},
		{"Assets", "Asset"},
	}
	for _, tc := range tests {
		if got := cowgen.Singular(tc.in); got != tc.want {
			t.Fatalf("Singular(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSliceMethodNames_FieldSlice(t *testing.T) {
	m := cowgen.SliceMethodNames("Items")
	if m.Append != "AppendItems" || m.SetAt != "SetItemsAt" ||
		m.RemoveAt != "RemoveItemsAt" || m.Truncate != "TruncateItems" {
		t.Fatalf("got %+v", m)
	}
}

func TestMapForWriteName(t *testing.T) {
	if got := cowgen.MapForWriteName("Buffs"); got != "GetBuffsMapForWrite" {
		t.Fatalf("got %q", got)
	}
}

func TestPtrSetName(t *testing.T) {
	if got := cowgen.PtrSetName("MainHero"); got != "SetMainHero" {
		t.Fatalf("got %q", got)
	}
}

func TestMapRemoveName(t *testing.T) {
	if got := cowgen.MapRemoveName("Heros"); got != "RemoveHeros" {
		t.Fatalf("got %q", got)
	}
}

func TestMapKeyGetForWriteName_UsesField(t *testing.T) {
	cases := []struct{ field, want string }{
		{"Heros", "GetHerosForWrite"},
		{"VirtualStackable", "GetVirtualStackableForWrite"},
		{"Stackable", "GetStackableForWrite"},
		{"Skills", "GetSkillsForWrite"},
	}
	for _, tc := range cases {
		if got := cowgen.MapKeyGetForWriteName(tc.field); got != tc.want {
			t.Fatalf("MapKeyGetForWriteName(%q)=%q want %q", tc.field, got, tc.want)
		}
	}
}
