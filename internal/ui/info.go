package ui

import (
	"strings"

	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/keys"
)

// info is what the INFO box in PLAYGROUND shows, two parts (user, 2026-11-11:
// under Next 1 and Next 2; the level, the rows and the score at the top,
// beside the field, not over it; the controls under them): the level, the rows and
// the score, then, under a dashed line as a menu's sections are parted
// (tdp menu), the keys the player has set and what they do, a key a row,
// the keys in Blue as a hint's (tdp M5).
type info struct{ t *Tetru }

const (
	// infoKeyW is the key column: Enter, Space.
	infoKeyW = 5
	// infoW is INFO's widest row and a cell's margin each side: the key, two
	// columns (tdp components.2), "rotate right".
	infoW = 1 + infoKeyW + 2 + len("rotate right") + 1
	// infoRows are INFO's rows: three of the game, the line, eight keys.
	infoRows = 3 + 1 + 8
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
		{"Space", "pause, menu"},
		{"Esc", "pause"},
	}
}

// Draw is the rows from the top, a cell from the box's frame.
func (m *info) Draw(w, h int) []string {
	var rows []string
	for _, s := range m.t.stats() {
		rows = append(rows, " "+s)
	}
	rows = append(rows, " "+terminu.Fg(terminu.Overlay0).Render(strings.Repeat("╌", max(0, w-2))))
	key := terminu.Fg(terminu.Blue)
	for _, r := range m.keyRows() {
		rows = append(rows, " "+key.Render(terminu.Pad(r[0], infoKeyW))+"  "+r[1])
	}
	out := make([]string, h)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
		if i < len(rows) {
			out[i] = terminu.Pad(rows[i], w)
		}
	}
	return out
}
