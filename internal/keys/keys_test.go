package keys

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestDefaultIsTheUsersKeys(t *testing.T) {
	d := Default()
	if d[Left] != "h" || d[Right] != "l" || d[Down] != "j" || d[RotateLeft] != "d" || d[RotateRight] != "k" || d[Drop] != "Enter" {
		t.Fatalf("default %v", d)
	}
	if err := d.Check(); err != nil {
		t.Fatal(err)
	}
}

func TestArrowsKeepTheirDefault(t *testing.T) {
	d := Default()
	for a, want := range map[Action][]string{Left: {"h", "←"}, Right: {"l", "→"}, Down: {"j", "↓"}, RotateLeft: {"d", "PgDn"}, RotateRight: {"k", "↑"}, Drop: {"Enter"}} {
		if got := d.Keys(a); !slices.Equal(got, want) {
			t.Fatalf("%s: %v", a.ID(), got)
		}
	}
	// Move left on a, Rotate right on the h so freed: ← still moves left, ↑
	// still turns right (tdp #14).
	d[Left], d[RotateRight] = "a", "h"
	if !slices.Equal(d.Keys(Left), []string{"a", "←"}) || !slices.Equal(d.Keys(RotateRight), []string{"h", "↑"}) {
		t.Fatalf("the arrows follow the letters: %v %v", d.Keys(Left), d.Keys(RotateRight))
	}
	if err := d.Check(); err != nil {
		t.Fatal(err)
	}
	// Off its default letter the arrow is written after the key; on it, not.
	for a, want := range map[Action]string{Left: "a|←", Right: "l", RotateRight: "h|↑", Drop: "Enter"} {
		if got := d.Shown(a); got != want {
			t.Fatalf("%s shown as %q, not %q", a.ID(), got, want)
		}
	}
}

func TestUsedBySaysWhoHasAKey(t *testing.T) {
	d := Default()
	for k, want := range map[string]string{
		"k":     "it is Rotate right's key",
		"↑":     "it is Rotate right's key",
		"d":     "it is Rotate left's key",
		"PgDn":  "it is Rotate left's key",
		"i":     "",
		"Space": "it pauses and opens the Space menu",
		"q":     "it quits",
		"r":     "it restarts",
		"s":     "", // Save is no key: tetru saves as it quits
		"a":     "",
		"Enter": "", // its own key
	} {
		if got := d.UsedBy(Drop, k); got != want {
			t.Fatalf("%q: %q", k, got)
		}
	}
	c := Choices()
	if !slices.Contains(c, "a") || !slices.Contains(c, "k") || !slices.Contains(c, "Enter") || !slices.Contains(c, "s") || slices.Contains(c, "Tab") {
		t.Fatalf("choices %v", c)
	}
	for _, k := range []string{"Space", "q", "r"} {
		if slices.Contains(c, k) {
			t.Fatalf("%s, a key the player can't have, is offered", k)
		}
	}
}

func TestPathIsInTheConfigDirectory(t *testing.T) {
	t.Setenv("TETRU__CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if p := Path(); p != filepath.Join("/x", "tetru", "config.yaml") {
		t.Fatal(p)
	}
	t.Setenv("TETRU__CONFIG", "/y")
	if p := Path(); p != filepath.Join("/y", "config.yaml") {
		t.Fatal(p)
	}
}

func TestSaveThenLoadKeepsOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tetru", "config.yaml")
	if s, err := Load(path); err != nil || s != Default() {
		t.Fatalf("no file: %v %v", s, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("theme: dark\nkeys:\n  rotate_left: a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil || s[RotateLeft] != "a" || s[Left] != "h" {
		t.Fatalf("partial file: %v %v", s, err)
	}
	s[Drop] = "f"
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "theme: dark") {
		t.Fatalf("a key tetru does not read was lost:\n%s", b)
	}
	if got, err := Load(path); err != nil || got != s {
		t.Fatalf("saved %v, read %v %v", s, got, err)
	}
	// Windows has no Unix modes: a file there is always rw-rw-rw-.
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o644 {
		t.Fatalf("mode %v", fi.Mode())
	}
}

func TestLoadBadKeys(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"core":     "keys:\n  rotate_left: q\n",
		"twice":    "keys:\n  rotate_left: k\n",
		"not yaml": "keys: [\n",
		"not keys": "keys: 3\n",
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".yaml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("%s: read as good", name)
		}
	}
}

func TestTheHintIsOnTillSwitchedOff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tetru", "config.yaml")
	if on, err := Hint(path); err != nil || !on {
		t.Fatalf("no file: the hint %v %v", on, err)
	}
	if err := Save(path, Default()); err != nil {
		t.Fatal(err)
	}
	if on, err := Hint(path); err != nil || !on {
		t.Fatalf("a file with no hint: %v %v", on, err)
	}
	if err := SaveHint(path, false); err != nil {
		t.Fatal(err)
	}
	if on, err := Hint(path); err != nil || on {
		t.Fatalf("switched off, read %v %v", on, err)
	}
	if s, err := Load(path); err != nil || s != Default() {
		t.Fatalf("the keys went with the hint: %v %v", s, err)
	}
	if err := os.WriteFile(path, []byte("hint: maybe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if on, err := Hint(path); err == nil || !on {
		t.Fatalf("a hint neither true nor false: %v %v", on, err)
	}
}
