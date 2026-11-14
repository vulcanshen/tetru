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
	g.PerLevel, g.Level, g.Lines, g.Score = 15, 3, 35, 12340
	// Each block its piece's (user, 2026-11-13: the colours back).
	for i := range game.Pieces {
		g.Stack[game.Rows-1][i] = uint8(1 + i)
	}
	g.Stack[game.Rows-1][game.Cols-1] = game.Kept
	g.Stack[game.Rows-2][game.Cols-1] = 1 + game.I
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "IOTSZJL....#") || !strings.Contains(string(data), "...........I") {
		t.Fatalf("not a row a line of the pieces' letters, # and .:\n%s", data)
	}
	if !strings.Contains(string(data), "lines_per_level: 15\n") {
		t.Fatalf("the rows a level not kept:\n%s", data)
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

func TestLoadAGameOfV010(t *testing.T) {
	// v0.1.0 kept where the blocks were, no colours: # each.
	path := filepath.Join(t.TempDir(), "game.yaml")
	body := "level: 2\nlines: 25\nscore: 900\nstack:\n" + strings.Repeat("  - \"............\"\n", game.Rows-1) + "  - \"##.........#\"\n"
	os.WriteFile(path, []byte(body), 0o644)
	g, err := Load(path)
	if err != nil || g.Stack[game.Rows-1][0] != game.Kept || g.Stack[game.Rows-1][2] != 0 || g.Stack[game.Rows-1][game.Cols-1] != game.Kept {
		t.Fatalf("a game of v0.1.0: %+v, %v", g, err)
	}
	// Nor did it, or v0.2.0, keep the rows a level: v0.2.0's twenty.
	if g.PerLevel != OldPerLevel || OldPerLevel != 20 {
		t.Fatalf("a game saved without its rows a level goes on at %d", g.PerLevel)
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
		"per level": "lines_per_level: 0\nlevel: 1\nstack:\n" + rows("............", game.Rows),
	} {
		path := filepath.Join(t.TempDir(), "game.yaml")
		os.WriteFile(path, []byte(body), 0o644)
		if g, err := Load(path); err == nil {
			t.Fatalf("%s: read %+v", name, g)
		}
	}
}
