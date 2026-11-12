package keys

import (
	"os"
	"path/filepath"
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

func TestArrowsGoWithTheirLetters(t *testing.T) {
	d := Default()
	for a, want := range map[Action][]string{Left: {"h", "←"}, Right: {"l", "→"}, Down: {"j", "↓"}, RotateLeft: {"d", "PgDn"}, RotateRight: {"k", "↑"}, Drop: {"Enter"}} {
		if got := d.Keys(a); !slices.Equal(got, want) {
			t.Fatalf("%s: %v", a.ID(), got)
		}
	}
	d[Left], d[RotateRight] = "a", "h"
	if !slices.Equal(d.Keys(Left), []string{"a"}) || !slices.Equal(d.Keys(RotateRight), []string{"h", "←"}) {
		t.Fatal("the arrow does not follow its letter")
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
		"s":     "it saves and quits",
		"a":     "",
		"Enter": "", // its own key
	} {
		if got := d.UsedBy(Drop, k); got != want {
			t.Fatalf("%q: %q", k, got)
		}
	}
	c := Choices()
	if !slices.Contains(c, "a") || !slices.Contains(c, "k") || !slices.Contains(c, "Enter") || slices.Contains(c, "Tab") {
		t.Fatalf("choices %v", c)
	}
	for _, k := range []string{"Space", "q", "r", "s"} {
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
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o644 {
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
