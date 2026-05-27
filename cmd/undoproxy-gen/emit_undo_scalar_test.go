package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func TestUndoBuilder_ScalarOldField_float32(t *testing.T) {
	g := &cowgen.Graph{Structs: []*cowgen.StructPlan{{Name: "Skill"}}}
	ub := newUndoBuilder(g)
	if got := ub.scalarOldField("float32"); got != "oldF32" {
		t.Fatalf("scalarOldField(float32)=%q want oldF32", got)
	}
	var buf bytes.Buffer
	ub.writeRuntime(&buf)
	out := buf.String()
	if !strings.Contains(out, "oldF32 float32") {
		t.Fatalf("writeRuntime missing oldF32 field:\n%s", out)
	}
}
