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

func TestBestKeepsTheTwenty(t *testing.T) {
	var all []Score
	for i := range Most + 2 {
		all = append(all, Score{Time: time.Date(2026, 11, 12, 0, i, 0, 0, time.UTC), Name: "p", Score: 10 * (i % 4)})
	}
	best := Best(all)
	if len(best) != Most {
		t.Fatalf("%d kept", len(best))
	}
	// Six of 0 points, the two last of them out: of two the same, the
	// earlier is first.
	if !slices.Contains(best, all[0]) || slices.Contains(best, all[16]) || slices.Contains(best, all[20]) || !slices.Contains(best, all[21]) {
		t.Fatalf("not the best twenty: %v", best)
	}
	for i := 1; i < len(best); i++ {
		if best[i].Time.Before(best[i-1].Time) {
			t.Fatal("not in the order they were kept")
		}
	}
	if Makes(best, 0) || !Makes(best, 10) || !Makes(best[:Most-1], 0) {
		t.Fatal("Makes: past the last of the twenty, or room left")
	}
	if few := all[:3]; len(Best(few)) != 3 {
		t.Fatal("fewer than twenty not all kept")
	}
}
