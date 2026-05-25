package main

import "testing"

func TestSingular(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Heros", "Hero"},
		{"Items", "Item"},
		{"Assets", "Asset"},
	}
	for _, tc := range tests {
		if got := singular(tc.in); got != tc.want {
			t.Fatalf("singular(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestMethodNames_FieldSlice(t *testing.T) {
	m := sliceMethodNames("Items")
	if m.Append != "AppendItems" || m.SetAt != "SetItemsAt" ||
		m.RemoveAt != "RemoveItemsAt" || m.Truncate != "TruncateItems" {
		t.Fatalf("got %+v", m)
	}
}

func TestMapForWriteName(t *testing.T) {
	if got := mapForWriteName("Buffs"); got != "GetBuffsMapForWrite" {
		t.Fatalf("got %q", got)
	}
}
