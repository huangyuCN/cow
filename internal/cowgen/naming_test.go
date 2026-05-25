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
