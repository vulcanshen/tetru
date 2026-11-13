// Package saved keeps the one game saved to go on with later (user,
// 2026-11-11: Save keeps the score and the blocks on the ground, and
// quits): its level, rows and score and its stack, each block's piece too
// (user, 2026-11-13: the colours back as they were), in the data directory (tdp E3: ~/.tetru/game.yaml, TETRU__DATA to move it). It
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

// Game is a game saved: its stack's blocks as game.Stack gives them.
type Game struct {
	Level, Lines, Score int
	Stack               [game.Rows][game.Cols]uint8
}

// file is a game saved as written: the stack a row a line, top first, a
// block its piece's letter (game.Letters: I O T S Z J L), # a block whose
// piece is not known (game.Kept: saved by v0.1.0, which kept no colours),
// and . none.
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
		if len(row) != game.Cols || strings.Trim(row, marks) != "" {
			return nil, fmt.Errorf("stack row %d: %q is not %d of %s", y+1, row, game.Cols, marks)
		}
		for x, c := range row {
			g.Stack[y][x] = cell(c)
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
		for _, c := range row {
			b.WriteByte(mark(c))
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

// marks are what a block of the stack is written as: none, the pieces'
// letters, a block whose piece is not known.
const marks = "." + game.Letters + "#"

// mark is how block c (game.Cell's) is written.
func mark(c uint8) byte {
	switch {
	case c == 0:
		return '.'
	case int(c) <= game.Pieces:
		return game.Letters[c-1]
	}
	return '#'
}

// cell is block m as game.Cell has it.
func cell(m rune) uint8 {
	switch i := strings.IndexRune(game.Letters, m); {
	case m == '.':
		return 0
	case i >= 0:
		return uint8(1 + i)
	}
	return game.Kept
}

// Remove removes the game saved, once taken up again; none there is no
// error.
func Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
