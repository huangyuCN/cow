package cowmon_test

import (
	"errors"
	"testing"

	"github.com/huangyuCN/cow/internal/cowmon"
)

func TestLoadMonitoredCached_noTagPackage(t *testing.T) {
	const path = "encoding/json"
	_, err := cowmon.LoadMonitoredCached(path, "")
	if err == nil {
		t.Fatal("expected error for package without tag")
	}
	_, err2 := cowmon.LoadMonitoredCached(path, "")
	if err2 == nil {
		t.Fatal("expected cached error")
	}
	if !errors.Is(err, err2) && err.Error() != err2.Error() {
		t.Fatalf("cached error mismatch: %v vs %v", err, err2)
	}
}

func TestLoadMonitoredCached_hitForTaggedPackage(t *testing.T) {
	const path = "github.com/huangyuCN/cow"
	set1, err := cowmon.LoadMonitoredCached(path, "")
	if err != nil {
		t.Fatal(err)
	}
	set2, err := cowmon.LoadMonitoredCached(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if set1 == nil || set2 == nil {
		t.Fatal("expected non-nil sets")
	}
	if !set1.ContainsName("Player") || !set2.ContainsName("Player") {
		t.Fatal("expected Player in cached set")
	}
}
