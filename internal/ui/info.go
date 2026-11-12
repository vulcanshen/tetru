package ui

import (
	"strings"

	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/keys"
)

// info is what the two boxes under Next 1 and Next 2 in PLAYGROUND show
// (user, 2026-11-11: the level, the rows and the score at the top, beside
// the field, not over it; the controls under them; 2026-11-12: a box each,
// now that they are boxes, not panels): INFO the level, the rows and the
// score; KEYS the keys the player has set and what they do, a key a row,
// the keys in Blue as a hint's (tdp M5).
type info struct{ t *Tetru }

const (
	// infoKeyW is the key column: Enter, Space.
	infoKeyW = 5
	// infoW is the widest row of the boxes and a cell's margin each side:
	// the key, two columns (tdp components.2), "rotate right".
	infoW = 1 + infoKeyW + 2 + len("rotate right") + 1
	// statRows are INFO's rows: the level, the rows, the score.
	statRows = 3
	// keyLines are KEYS' rows: eight keys.
	keyLines = 8
	// infoH is INFO's and KEYS' least height, with their frames.
	infoH = statRows + 2 + keyLines + 2
)

// keyRows are the keys and what they do.
func (m *info) keyRows() [][2]string {
	k := m.t.keys
	return [][2]string{
		{k[keys.Left], "move left"},
		{k[keys.Right], "move right"},
		{k[keys.Down], "down a row"},
		{k[keys.RotateLeft], "rotate left"},
		{k[keys.RotateRight], "rotate right"},
		{k[keys.Drop], "drop"},
		{"Space", "menu"}, // it holds the game as any popup (user, 2026-11-12)
		{"Esc", "pause"},
	}
}

// Stats is INFO's rows, w × h, a cell from the box's frame.
func (m *info) Stats(w, h int) []string {
	var rows []string
	for _, s := range m.t.stats() {
		rows = append(rows, " "+s)
	}
	return fill(rows, w, h)
}

// Keys is KEYS' rows, w × h, a cell from the box's frame.
func (m *info) Keys(w, h int) []string {
	var rows []string
	key := terminu.Fg(terminu.Blue)
	for _, r := range m.keyRows() {
		rows = append(rows, " "+key.Render(terminu.Pad(r[0], infoKeyW))+"  "+r[1])
	}
	return fill(rows, w, h)
}

// fill is rows from the top of w × h cells, blank under them.
func fill(rows []string, w, h int) []string {
	out := make([]string, h)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
		if i < len(rows) {
			out[i] = terminu.Pad(rows[i], w)
		}
	}
	return out
}
