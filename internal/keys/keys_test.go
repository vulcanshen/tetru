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
	if d[Left] != "h" || d[Right] != "l" || d[Down] != "j" || d[RotateLeft] != "e" || d[RotateRight] != "i" || d[Drop] != "Enter" {
		t.Fatalf("default %v", d)
	}
	if err := d.Check(); err != nil {
		t.Fatal(err)
	}
}

func TestChoicesKeepTdpsKeys(t *testing.T) {
	for _, a := range Actions {
		c := Choices(a)
		for _, k := range []string{"q", "k", "u", "d", "g", "G", "Space", "Tab", "Esc", "1", "?", "!"} {
			if slices.Contains(c, k) {
				t.Fatalf("%s can be %q", a.ID(), k)
			}
		}
		if !slices.Contains(c, "e") || !slices.Contains(c, "z") {
			t.Fatalf("%s: %v", a.ID(), c)
		}
	}
	if !slices.Contains(Choices(Left), "h") || slices.Contains(Choices(Left), "l") || slices.Contains(Choices(RotateLeft), "h") {
		t.Fatal("a moving letter goes to its own way only")
	}
	if !slices.Contains(Choices(Drop), "Enter") || slices.Contains(Choices(RotateLeft), "Enter") {
		t.Fatal("Enter is Drop's only")
	}
}

func TestFreeLeavesOutTheOthersKeys(t *testing.T) {
	s := Default()
	f := s.Free(RotateLeft)
	if slices.Contains(f, "i") || !slices.Contains(f, "e") || !slices.Contains(f, "a") {
		t.Fatalf("free for rotate left: %v", f)
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
	s[Drop] = "s"
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
		"reserved": "keys:\n  rotate_left: k\n",
		"twice":    "keys:\n  rotate_left: i\n",
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
