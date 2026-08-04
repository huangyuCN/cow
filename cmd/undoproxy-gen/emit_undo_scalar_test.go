package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/huangyuCN/cow/internal/cowgen"
)

// TestUndoBuilder_ScalarOldField_emptyPanics 空类型串属于上游分类缺陷，
// 必须显式失败而不是静默落到 oldI64 槽生成错误代码。
func TestUndoBuilder_ScalarOldField_emptyPanics(t *testing.T) {
	g := &cowgen.Graph{Structs: []*cowgen.StructPlan{{Name: "Skill"}}}
	ub := newUndoBuilder(g)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty scalar type")
		}
	}()
	ub.scalarOldField("")
}

func TestUndoBuilder_ScalarOldField_float32(t *testing.T) {
	g := &cowgen.Graph{Structs: []*cowgen.StructPlan{{Name: "Skill"}}}
	ub := newUndoBuilder(g)
	if got := ub.scalarOldField("float32"); got != "oldF32" {
		t.Fatalf("scalarOldField(float32)=%q want oldF32", got)
	}
	var buf bytes.Buffer
	if err := ub.writeRuntime(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "oldF32 float32") {
		t.Fatalf("writeRuntime missing oldF32 field:\n%s", out)
	}
}
