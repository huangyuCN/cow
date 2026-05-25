package main

import (
	"fmt"
	"strings"
)

func printDiffs(diffs []fileDiff) {
	for _, d := range diffs {
		fmt.Printf("--- %s\n+++ %s\n", d.Path, d.Path)
		fmt.Print(simpleDiff(d.Before, d.After))
	}
}

func simpleDiff(before, after string) string {
	if before == after {
		return ""
	}
	var b strings.Builder
	b.WriteString("@@ rewritten @@\n")
	for _, line := range strings.Split(after, "\n") {
		if line != "" || len(strings.Split(after, "\n")) > 1 {
			b.WriteString("+")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}
