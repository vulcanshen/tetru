package scores

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestPathIsInTheDataDirectory(t *testing.T) {
	t.Setenv("TETRU__DATA", "/somewhere")
	if got := Path(); got != filepath.Join("/somewhere", "scores.yaml") {
		t.Errorf("with TETRU__DATA: %s", got)
	}
	t.Setenv("TETRU__DATA", "")
	// The home is HOME, and on Windows USERPROFILE (tdp E3).
	t.Setenv("HOME", "/home/someone")
	t.Setenv("USERPROFILE", "/home/someone")
	if got := Path(); got != filepath.Join("/home/someone", ".tetru", "scores.yaml") {
		t.Errorf("by default: %s", got)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "scores.yaml")
	if got, err := Load(path); err != nil || got != nil {
		t.Fatalf("no file yet: %v, %v", got, err)
	}
	at := time.Date(2026, 11, 11, 14, 2, 0, 0, time.FixedZone("TST", 8*3600))
	lines := 212
	// The rows and the clears (user, 2026-11-16); none in a record of
	// v0.3.1 or before, the second.
	want := []Score{
		{Time: at, Name: "vulcan", Score: 12340, Lines: &lines, Clears: &[4]int{61, 28, 11, 15}},
		{Time: at.Add(time.Minute), Name: "a\tb", Score: 50},
	}
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
	if got[0].Lines == nil || *got[0].Lines != 212 || got[0].Clears == nil || *got[0].Clears != [4]int{61, 28, 11, 15} {
		t.Fatalf("the rows and the clears read back %v, %v", got[0].Lines, got[0].Clears)
	}
	if got[1].Lines != nil || got[1].Clears != nil {
		t.Fatal("a record with no rows or clears read back with some")
	}
	data, _ := os.ReadFile(path)
	if !slices.Contains([]string{"scores:\n"}, string(data[:8])) {
		t.Fatalf("the file starts %q", data[:8])
	}
	if s := string(data); !strings.Contains(s, "lines: 212\n") || !strings.Contains(s, "clears: [61, 28, 11, 15]\n") || strings.Count(s, "clears:") != 1 {
		t.Fatalf("the rows and the clears not written as a line each, once:\n%s", s)
	}
}

func TestLoadBroken(t *testing.T) {
	for name, body := range map[string]string{
		"yaml":         "scores: [nope",
		"lines":        "scores:\n  - name: a\n    score: 1\n    lines: -1\n",
		"clears":       "scores:\n  - name: a\n    score: 1\n    clears: [1, -2, 0, 0]\n",
		"three clears": "scores:\n  - name: a\n    score: 1\n    clears: [1, 2, 0]\n",
	} {
		path := filepath.Join(t.TempDir(), "scores.yaml")
		os.WriteFile(path, []byte(body), 0o644)
		if got, err := Load(path); err == nil {
			t.Fatalf("%s: a broken file read: %+v", name, got)
		}
	}
}

func TestRankedBestFirstThenEarliest(t *testing.T) {
	at := time.Now()
	all := []Score{{Time: at, Name: "a", Score: 10}, {Time: at.Add(2), Name: "b", Score: 30}, {Time: at.Add(1), Name: "c", Score: 30}, {Time: at, Name: "d"}}
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
