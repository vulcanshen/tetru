package saved

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vulcanshen/tetru/internal/game"
)

func TestPathIsBesideTheScores(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TETRU__DATA", dir)
	if Path() != filepath.Join(dir, "game.yaml") {
		t.Fatalf("path %s", Path())
	}
}

func TestSaveLoadRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "game.yaml")
	if g, err := Load(path); g != nil || err != nil {
		t.Fatalf("no file: %v, %v", g, err)
	}
	var g Game
	g.Level, g.Lines, g.Score = 3, 25, 12340
	g.Stack[game.Rows-1][0] = true
	g.Stack[game.Rows-2][game.Cols-1] = true
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "#...........") || !strings.Contains(string(data), "...........#") {
		t.Fatalf("not a row a line of # and .:\n%s", data)
	}
	// Windows has no Unix modes: a file there is always rw-rw-rw-.
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o644 {
		t.Fatalf("mode %v", fi.Mode().Perm())
	}
	back, err := Load(path)
	if err != nil || *back != g {
		t.Fatalf("read back %+v, %v", back, err)
	}
	if err := Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("not removed")
	}
	if err := Remove(path); err != nil {
		t.Fatal("removing none is an error")
	}
}

func TestLoadBroken(t *testing.T) {
	rows := func(r string, n int) string {
		return strings.Repeat("  - \""+r+"\"\n", n)
	}
	for name, body := range map[string]string{
		"yaml":      "level: [\n",
		"rows":      "level: 1\nstack:\n" + rows("............", 3),
		"width":     "level: 1\nstack:\n" + rows("....", game.Rows),
		"character": "level: 1\nstack:\n" + rows("....x.......", game.Rows),
		"level":     "level: 16\nstack:\n" + rows("............", game.Rows),
		"score":     "level: 1\nscore: -1\nstack:\n" + rows("............", game.Rows),
	} {
		path := filepath.Join(t.TempDir(), "game.yaml")
		os.WriteFile(path, []byte(body), 0o644)
		if g, err := Load(path); err == nil {
			t.Fatalf("%s: read %+v", name, g)
		}
	}
}
