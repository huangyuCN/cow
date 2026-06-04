package cowmon_test

import (
	"path/filepath"
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestLoadMonitoredFromSrcDir_importscope(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(root, "cmd", "undocheck", "testdata", "src")
	set, err := cowmon.LoadMonitoredFromSrcDir("importscope/defpkg", srcDir)
	if err != nil {
		t.Fatal(err)
	}
	if set == nil || !set.ContainsName("Player") {
		t.Fatal("expected Player in set")
	}
}
