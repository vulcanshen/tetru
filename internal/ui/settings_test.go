package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vulcanshen/terminu-go/tdptest"

	"github.com/vulcanshen/tetru/internal/keys"
)

// checkOn and checkOff are a checkbox ticked and not (tdp checkbox).
const checkOn, checkOff = "\U000f0132", "\U000f0131"

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
	if r := row(d, 2); !strings.Contains(r, "[1] SETTINGS") || !d.Has("ROTATE LEFT") {
		t.Fatalf("S did not show the settings: %q", r)
	}
	d.Press("T")
	if r := row(d, 2); !strings.Contains(r, "[1] PLAYGROUND") {
		t.Fatalf("T did not go back to the game: %q", r)
	}
}

func TestInfoShowsTheKeys(t *testing.T) {
	_, d := open(t, 60, 32)
	for _, s := range []string{"h      move left", "l      move right", "j      down a row", "d      rotate left", "k      rotate right", "Enter  drop", "Space  menu", "Esc    pause"} {
		if !d.Has(s) {
			t.Fatalf("INFO has no %q", s)
		}
	}
}

func TestSettingsSetAKey(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("S")
	for _, s := range []string{"KEYS", "MOVE LEFT", "h", "DROP", "Enter", "PLAY", "HINT", "Enter:edit"} {
		if !d.Has(s) {
			t.Fatalf("the settings have no %q", s)
		}
	}
	// One panel, two groups, no header (user, 2026-11-13; tdp panel.12).
	if d.Has("ACTION") || d.Has("[2]") || !strings.Contains(row(d, 3), "KEYS") || !strings.Contains(row(d, 4), "  MOVE LEFT") {
		t.Fatal("the settings are not one panel of groups, the keys first, with no header")
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
	if !d.Has("Rotate left is now [a]") {
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
	if tt.g.At.Turn != 0 {
		t.Fatal("d still turns")
	}
	// No one chooses the arrows: PgDn keeps turning left (tdp #14).
	d.Press("PgDn")
	if tt.g.At.Turn != 3 {
		t.Fatal("PgDn did not turn left")
	}
	d.Press("a")
	if tt.g.At.Turn != 2 {
		t.Fatal("a did not turn left")
	}
	if !strings.HasPrefix(row(d, 31), "Space:menu ?:help a|k:rotate Enter:drop") {
		t.Fatalf("footer %q", row(d, 31))
	}
}

// TestEnterAndTheMovesAreThePlayers: in an app's own mode Enter and the
// moves are the player's to set too (tdp v1.2.7 K11, K12, K11-25; tdp #14).
// Moved off, a move still moves and ← with it; a letter so freed and Enter
// can go to another action, a single g too; the turns and the drop stay in
// the footer at 60 columns (tdp K11-20).
func TestEnterAndTheMovesAreThePlayers(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("S")
	set := func(row int, key string) {
		t.Helper()
		d.Press("g")
		d.Press("g")
		for range row {
			d.Press("j")
		}
		d.Press("Enter")
		d.Press("g")
		d.Press("g")
		for _, c := range keys.Choices() {
			if c == key {
				break
			}
			d.Press("j")
		}
		d.Press("Enter")
	}
	set(5, "s") // Drop off Enter
	set(0, "a") // Move left off h
	set(4, "h") // Rotate right on the h so freed
	set(3, "Enter")
	set(2, "g") // Down a row on a single g: Gaming has no chord starting with g
	want := keys.Set{"a", "l", "g", "Enter", "h", "s"}
	if tt.keys != want {
		t.Fatalf("keys %v, not %v", tt.keys, want)
	}
	if s, err := keys.Load(tt.cfg); err != nil || s != want {
		t.Fatalf("not kept in the settings: %v %v", s, err)
	}
	d.Press("T")
	play(d)
	for tt.g.Cur == 1 { // an O does not turn
		d.Press("s")
	}
	if r := row(d, 31); !strings.HasPrefix(r, "Space:menu ?:help Enter|h:rotate s:drop") {
		t.Fatalf("footer %q", r)
	}
	// The arrows of the actions off their letters are written after the keys
	// in ?; the one on its letter is not (tdp K11, K11-25). The turns are two
	// rows there, | one way only (tdp M5).
	d.Press("?")
	if !d.Has("a|←") || !d.Has("g|↓") || d.Has("l|→") {
		t.Fatal("? does not write the arrow after a move off its letter, or writes one on it")
	}
	rowWith := func(key string) string {
		for _, r := range d.Screen() {
			if strings.Contains(r, " "+key+" ") {
				return r
			}
		}
		return ""
	}
	if r := rowWith("Enter|PgDn"); !strings.Contains(r, "rotate left") {
		t.Fatalf("? has no row Enter|PgDn rotate left: %q", r)
	}
	if r := rowWith("h|↑"); !strings.Contains(r, "rotate right") {
		t.Fatalf("? has no row h|↑ rotate right: %q", r)
	}
	d.Press("Esc")
	at := tt.g.At
	d.Press("a")
	d.Press("←")
	if tt.g.At.X != at.X-2 {
		t.Fatal("a and ← do not move left")
	}
	d.Press("g")
	d.Press("↓")
	if tt.g.At.Y != at.Y+2 {
		t.Fatal("g and ↓ do not move down a row")
	}
	d.Press("Enter")
	if tt.g.At.Turn != 3 {
		t.Fatal("Enter did not turn left")
	}
	d.Press("h")
	d.Press("↑")
	if tt.g.At.Turn != 1 {
		t.Fatal("h and ↑ did not turn right")
	}
	before := tt.g.Score
	d.Press("s")
	if tt.g.Score <= before {
		t.Fatal("s did not drop")
	}
}

func TestSettingsSwitchTheHint(t *testing.T) {
	tt, d := open(t, 60, 32)
	if !tt.hint {
		t.Fatal("the hint is off at first")
	}
	d.Press("S")
	if !d.Has("PLAY") || !d.Has("HINT") || !d.Has("where a drop lands") || !d.Has(checkOn) {
		t.Fatal("the settings have no hint switched on under the keys")
	}
	d.Press("G") // the last row
	if !d.Has("Enter:toggle") {
		t.Fatal("the hint's switch has no Enter:toggle")
	}
	d.Press("Enter")
	if tt.hint || !d.Has(checkOff) {
		t.Fatal("Enter did not switch the hint off")
	}
	if on, err := keys.Hint(tt.cfg); err != nil || on {
		t.Fatalf("the hint off is not kept in the settings: %v %v", on, err)
	}
	if again := NewTetru(); again.hint || again.hintF.Checked[0] {
		t.Fatal("the next start has the hint on")
	}
	d.Press("!")
	if !d.Has("The hint is now off") {
		t.Fatal("no App Log entry")
	}
	d.Press("Esc")
	d.Press("Enter")
	if on, err := keys.Hint(tt.cfg); !tt.hint || err != nil || !on {
		t.Fatal("Enter did not switch the hint back on")
	}
	if s, err := keys.Load(tt.cfg); err != nil || s != keys.Default() {
		t.Fatalf("the keys are not as they were: %v %v", s, err)
	}
}

func TestSettingsHaveNoPauseRow(t *testing.T) {
	_, d := open(t, 60, 32)
	d.Press("S")
	if d.Has("PAUSE") {
		t.Fatal("a Pause row: Esc pauses, Space opens the menu")
	}
	d.Press("G") // the last row: the hint
	d.Press("k") // Drop, the last key
	d.Press("Enter")
	if !d.Has("KEY: Drop") {
		t.Fatal("Drop is not the last row, or opened no select")
	}
}

func TestSettingsThatCantBeKept(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a directory can't be made read-only on Windows")
	}
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
	d.Press("Esc")
	d.Press("Esc")
	d.Press("G")
	d.Press("Enter")
	if !d.Has("SAVE THE HINT") || !tt.hint || !tt.hintF.Checked[0] {
		t.Fatal("a hint that could not be kept: switched off, or no error")
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
	if !d.Has("MOVE ASIDE") || !d.Has("config.yaml.bak") {
		t.Fatal("broken settings did not ask to move them aside")
	}
	d.Press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("the broken file was not moved aside")
	}
	if tt.keys != keys.Default() {
		t.Fatal("not the default keys after starting fresh")
	}
	// A hint neither true nor false: the same.
	os.WriteFile(path, []byte("hint: maybe\n"), 0o644)
	tt = NewTetru()
	d = tdptest.Start(t, tt.App, 60, 32)
	if !d.Has("MOVE ASIDE") || !tt.hint {
		t.Fatal("a broken hint did not ask to move the settings aside")
	}
}

func TestAKeyInUseIsRefused(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("S")
	d.Press("j")
	d.Press("j")
	d.Press("j") // Rotate left
	for _, c := range []struct{ key, why string }{
		{"k", "Rotate right's"}, // key wraps to the next line at 60 columns
	} {
		d.Press("Enter")
		d.Press("/")
		d.Type(c.key)
		d.Press("Enter")
		d.Press("Enter")
		if !d.Has("Can't use ["+c.key+"] for Rotate left") || !d.Has(c.why) || tt.keys[keys.RotateLeft] != "d" {
			t.Fatalf("%s for Rotate left: kept %q, no Warning", c.key, tt.keys[keys.RotateLeft])
		}
		if s, err := keys.Load(tt.cfg); err != nil || s[keys.RotateLeft] != "d" {
			t.Fatalf("a refused key was written: %v %v", s, err)
		}
		d.Advance(5 * time.Second) // the toast goes
	}
}
