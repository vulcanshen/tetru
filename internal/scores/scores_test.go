package scores

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestPathIsInTheDataDirectory(t *testing.T) {
	t.Setenv("TETRU__DATA", "/somewhere")
	if got := Path(); got != "/somewhere/scores.yaml" {
		t.Errorf("with TETRU__DATA: %s", got)
	}
	t.Setenv("TETRU__DATA", "")
	t.Setenv("HOME", "/home/someone")
	if got := Path(); got != "/home/someone/.tetru/scores.yaml" {
		t.Errorf("by default: %s", got)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "scores.yaml")
	if got, err := Load(path); err != nil || got != nil {
		t.Fatalf("no file yet: %v, %v", got, err)
	}
	at := time.Date(2026, 11, 11, 14, 2, 0, 0, time.FixedZone("TST", 8*3600))
	want := []Score{{at, "vulcan", 12340}, {at.Add(time.Minute), "a\tb", 50}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].Time.Equal(at) || got[0].Name != "vulcan" || got[0].Score != 12340 || got[1].Name != "a\tb" {
		t.Fatalf("read back %+v", got)
	}
	data, _ := os.ReadFile(path)
	if !slices.Contains([]string{"scores:\n"}, string(data[:8])) {
		t.Fatalf("the file starts %q", data[:8])
	}
}

func TestLoadBroken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.yaml")
	os.WriteFile(path, []byte("scores: [nope"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("a broken file read")
	}
}

func TestRankedBestFirstThenEarliest(t *testing.T) {
	at := time.Now()
	all := []Score{{at, "a", 10}, {at.Add(2), "b", 30}, {at.Add(1), "c", 30}, {at, "d", 0}}
	if got := Ranked(all); !slices.Equal(got, []int{2, 1, 0, 3}) {
		t.Fatalf("ranked %v", got)
	}
}
