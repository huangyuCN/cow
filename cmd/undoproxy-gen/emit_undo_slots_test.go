package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestWriteRuntime_EmitsTypedKeyAndInnerSlots(t *testing.T) {
	ub := &undoBuilder{
		structs:       []string{"W"},
		kindIndex:     make(map[string]string),
		cloneHelpers:  make(map[string]struct{}),
		sliceSnaps:    make(map[string]string),
		scalarOlds:    map[string]string{"int": "oldInt"},
		keySlots:      make(map[string]string),
		innerMapSlots: make(map[string]string),
		entries:       []undoEntry{{name: "undoKindWDummy", body: "break"}},
	}
	ub.keySlot("int")
	ub.keySlot("string")
	innerName := ub.innerMapSlot("map[string]*Node")
	var buf bytes.Buffer
	if err := ub.writeRuntime(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, need := range []string{
		"key_int int",
		"key_string string",
		innerName + " map[string]*Node",
	} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in:\n%s", need, s)
		}
	}
	if strings.Contains(s, "keyI32") || strings.Contains(s, "innerMapOld") {
		t.Fatal("legacy slots must not appear")
	}
}

func TestEmit_TypedSlotsForIntKeyAndInnerMaps(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	if err := Run(out, "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, need := range []string{
		"key_int",
		"inner_map_string_int64",
		"inner_map_string__Node",
		"oldCat",
	} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q", need)
		}
	}
	// gofmt 可能对齐空白，用正则确认字段类型
	mustField := func(name, typ string) {
		t.Helper()
		re := name + `\s+` + typ
		if !regexp.MustCompile(re).MatchString(s) {
			t.Fatalf("missing field %s %s", name, typ)
		}
	}
	mustField("key_int", "int")
	mustField("inner_map_string_int64", `map\[string\]int64`)
	mustField("inner_map_string__Node", `map\[string\]\*Node`)
	mustField("oldCat", "Cat")
	if strings.Contains(s, "innerMapOld") {
		t.Fatal("legacy innerMapOld must not appear")
	}
}
