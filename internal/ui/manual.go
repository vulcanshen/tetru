package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/keys"
)

// manual is the MANUAL panel's body: how to play, a key and what it does a
// row, with the keys the player has set (user, 2026-11-11: under Next 1 and
// Next 2, the controls). The keys in Blue, as a hint's (tdp M5).
type manual struct{ t *Tetru }

const (
	// manualKeyW is the key column: Enter, Space.
	manualKeyW = 5
	// manualW is MANUAL's widest row and a cell's margin each side: the key,
	// two columns (tdp components.2), "rotate right".
	manualW = 1 + manualKeyW + 2 + len("rotate right") + 1
)

// rows are the keys and what they do.
func (m *manual) rows() [][2]string {
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

// Draw is the rows from the top, a cell from the frame.
func (m *manual) Draw(w, h int, focused bool) []string {
	key := terminu.Fg(terminu.Blue)
	rows := m.rows()
	out := make([]string, h)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
		if i < len(rows) {
			out[i] = terminu.Pad(" "+key.Render(terminu.Pad(rows[i][0], manualKeyW))+"  "+rows[i][1], w)
		}
	}
	return out
}

// Key takes nothing: MANUAL is to read.
func (m *manual) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem is false: MANUAL has no cursor.
func (m *manual) HasItem() bool { return false }

// Scroll is nothing: the rows fit at tetru's least size.
func (m *manual) Scroll() string { return "" }

// MoveKeys are none: nothing moves.
func (m *manual) MoveKeys() []terminu.KeyRow { return nil }
