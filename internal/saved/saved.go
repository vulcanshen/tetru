// Package saved keeps the one game saved to go on with later (user,
// 2026-11-11: Save keeps the score and the blocks on the ground, and
// quits): its level, rows and score and where its stack has blocks, in the
// data directory (tdp E3: ~/.tetru/game.yaml, TETRU__DATA to move it). It
// is read back as tetru starts, and then removed.
package saved

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vulcanshen/tetru/internal/game"
	"github.com/vulcanshen/tetru/internal/scores"
)

// Game is a game saved.
type Game struct {
	Level, Lines, Score int
	Stack               [game.Rows][game.Cols]bool
}

// file is a game saved as written: the stack a row a line, top first, #
// a block and . none.
type file struct {
	Level int      `yaml:"level"`
	Lines int      `yaml:"lines"`
	Score int      `yaml:"score"`
	Stack []string `yaml:"stack"`
}

// Path is the game saved: game.yaml beside the scores.
func Path() string { return filepath.Join(filepath.Dir(scores.Path()), "game.yaml") }

// Load reads the game saved; nil when there is none.
func Load(path string) (*Game, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if len(f.Stack) != game.Rows {
		return nil, fmt.Errorf("stack: %d rows, not %d", len(f.Stack), game.Rows)
	}
	if f.Level < 1 || f.Level > game.MaxLevel || f.Lines < 0 || f.Score < 0 {
		return nil, fmt.Errorf("level %d, %d lines, %d points: out of range", f.Level, f.Lines, f.Score)
	}
	g := &Game{Level: f.Level, Lines: f.Lines, Score: f.Score}
	for y, row := range f.Stack {
		if len(row) != game.Cols || strings.Trim(row, "#.") != "" {
			return nil, fmt.Errorf("stack row %d: %q is not %d of # and .", y+1, row, game.Cols)
		}
		for x, c := range row {
			g.Stack[y][x] = c == '#'
		}
	}
	return g, nil
}

// Save writes g, the file replaced whole so a failed write leaves the old
// one as it was.
func Save(path string, g Game) error {
	f := file{Level: g.Level, Lines: g.Lines, Score: g.Score}
	for _, row := range g.Stack {
		var b strings.Builder
		for _, block := range row {
			if block {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		f.Stack = append(f.Stack, b.String())
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".game-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Remove removes the game saved, once taken up again; none there is no
// error.
func Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
