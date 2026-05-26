package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitV2GraphFile_IsFocused(t *testing.T) {
	path := filepath.Join(".", "emit_v2_graph.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(string(b), "\n") + 1
	if lines > 260 {
		t.Fatalf("emit_v2_graph.go too large: got %d lines, want <= 260", lines)
	}
}
