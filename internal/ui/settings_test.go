package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vulcanshen/terminu-go/tdptest"

	"github.com/vulcanshen/tetru/internal/keys"
)

func TestScreensSwitchOnTRS(t *testing.T) {
	_, d := open(t, 60, 32)
	d.Press("R")
	if r := row(d, 2); !strings.Contains(r, "[1] RANK") || !d.Has("No scores yet") || !d.Has("Press [T] to play") {
		t.Fatalf("R did not show the empty rank: %q", r)
	}
	if !d.Has("Space:menu ?:help !:log q:quit") {
		t.Fatal("a screen of one panel has the panels in its footer")
	}
	d.Press("S")
	if r := row(d, 2); !strings.Contains(r, "[1] KEYS") || !d.Has("ROTATE LEFT") {
		t.Fatalf("S did not show the settings: %q", r)
	}
	d.Press("T")
	if r := row(d, 2); !strings.Contains(r, "[1] GAME") {
		t.Fatalf("T did not go back to the game: %q", r)
	}
}

func TestInfoShowsTheKeys(t *testing.T) {
	_, d := open(t, 60, 32)
	for _, s := range []string{"h      move left", "l      move right", "j      down a row", "d      rotate left", "k      rotate right", "Enter  drop", "Space  pause, menu", "Esc    pause"} {
		if !d.Has(s) {
			t.Fatalf("INFO has no %q", s)
		}
	}
}

func TestSettingsSetAKey(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("S")
	for _, s := range []string{"ACTION", "MOVE LEFT", "h", "DROP", "Enter", "PAUSE", "Space", "Enter:edit"} {
		if !d.Has(s) {
			t.Fatalf("the settings have no %q", s)
		}
	}
	d.Press("j")
	d.Press("j")
	d.Press("j")
	d.Press("Enter")
	if !d.Has("KEY: Rotate left") {
		t.Fatal("Enter on Rotate left opened no select of keys")
	}
	d.Press("g")
	d.Press("g")
	d.Press("Enter")
	if tt.keys[keys.RotateLeft] != "a" || !strings.Contains(row(d, 7), "a") {
		t.Fatalf("rotate left is %q", tt.keys[keys.RotateLeft])
	}
	if s, err := keys.Load(tt.cfg); err != nil || s[keys.RotateLeft] != "a" {
		t.Fatalf("not kept in the settings: %v %v", s, err)
	}
	d.Press("!")
	if !d.Has("Rotate left is now a") {
		t.Fatal("no App Log entry")
	}
	d.Press("Esc")
	d.Press("T")
	if !d.Has("a      rotate left") {
		t.Fatal("INFO did not follow the new key")
	}
	play(d)
	turnable(tt, d)
	d.Press("d")
	d.Press("PgDn")
	if tt.g.At.Turn != 0 {
		t.Fatal("d or PgDn still turns")
	}
	d.Press("a")
	if tt.g.At.Turn == 0 {
		t.Fatal("a did not turn left")
	}
	if !strings.HasPrefix(row(d, 31), "Space:menu ?:help a:rotate left") {
		t.Fatalf("footer %q", row(d, 31))
	}
}

func TestPauseIsAlwaysSpace(t *testing.T) {
	_, d := open(t, 60, 32)
	d.Press("S")
	if !d.Has("PAUSE") || !d.Has("Space") {
		t.Fatal("no Pause row")
	}
	d.Press("G") // the last row the cursor stops on: Pause is passed over
	d.Press("Enter")
	if !d.Has("KEY: Drop") {
		t.Fatal("the cursor stopped on Pause, or Drop opened no select")
	}
}

func TestSettingsThatCantBeKept(t *testing.T) {
	t.Setenv("TETRU__DATA", t.TempDir())
	dir := t.TempDir()
	os.Chmod(dir, 0o555) // read, but nothing written in it
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	t.Setenv("TETRU__CONFIG", dir)
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 60, 32)
	d.Press("S")
	d.Press("Enter")
	d.Press("g")
	d.Press("g")
	d.Press("Enter")
	if !d.Has("FAILED") || tt.keys[keys.Left] != "h" {
		t.Fatalf("a key that could not be kept: %q, no error", tt.keys[keys.Left])
	}
}

func TestBrokenSettingsAskToMoveAside(t *testing.T) {
	t.Setenv("TETRU__DATA", t.TempDir())
	dir := t.TempDir()
	t.Setenv("TETRU__CONFIG", dir)
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("keys:\n  rotate_left: q\n"), 0o644)
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 60, 32)
	if !d.Has("can't be read") || !d.Has("config.yaml") {
		t.Fatal("broken settings did not ask to move them aside")
	}
	d.Press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("the broken file was not moved aside")
	}
	if tt.keys != keys.Default() {
		t.Fatal("not the default keys after starting fresh")
	}
}

func TestAKeyInUseIsRefused(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("S")
	d.Press("j")
	d.Press("j")
	d.Press("j") // Rotate left
	for _, c := range []struct{ key, why string }{
		{"k", "Rotate right's key"},
		{"Space", "the Space menu"},
	} {
		d.Press("Enter")
		d.Press("/")
		d.Type(c.key)
		d.Press("Enter")
		d.Press("Enter")
		if !d.Has("Can't use "+c.key+" for Rotate left") || !d.Has(c.why) || tt.keys[keys.RotateLeft] != "d" {
			t.Fatalf("%s for Rotate left: kept %q, no Warning", c.key, tt.keys[keys.RotateLeft])
		}
		if s, err := keys.Load(tt.cfg); err != nil || s[keys.RotateLeft] != "d" {
			t.Fatalf("a refused key was written: %v %v", s, err)
		}
		d.Advance(5 * time.Second) // the toast goes
	}
}
