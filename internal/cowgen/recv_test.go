package cowgen_test

import (
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func TestRecvIdent_firstRune(t *testing.T) {
	tests := []struct {
		structName, want string
	}{
		{"Player", "p"},
		{"NodeData", "n"},
		{"Item", "i"},
		{"Hero", "h"},
	}
	for _, tc := range tests {
		if got := cowgen.RecvIdent(tc.structName); got != tc.want {
			t.Fatalf("RecvIdent(%q)=%q want %q", tc.structName, got, tc.want)
		}
	}
}
