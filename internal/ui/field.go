package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/game"
)

// The field is drawn as locku's tetromino saver draws its board (user,
// 2026-11-11): a pixel is a square glyph two columns wide and a row high, a
// block a pixel; the ground lit too, in surface0, as locku's board is; the
// pieces to come a box of ground each. Neither has a frame of its own: the
// panel's is it (user, 2026-11-11: the panel is the game; NEXT has a panel
// of its own). The field is GAME's and each box a NEXT panel's (user,
// 2026-11-11: Next 1 and Next 2 at the right); narrow, where GAME is drawn
// alone, GAME draws the boxes too, a pixel from the field, a pixel apart.
// A box is 5 × 5 pixels, a square (user, 2026-11-11).
const (
	boxW, boxH = 5, 5
	// fieldW and sceneH are the field; queueX is where the column of boxes
	// starts beside it; sceneW is the two side by side, in pixels.
	fieldW = game.Cols
	sceneH = game.Rows
	queueX = fieldW + 1
	sceneW = queueX + boxW
)

// pixelGlyph is a pixel: nf-fa-square, as locku's.
var pixelGlyph = string(rune(0xf0c8))

// The inks a pixel can be in: none is not drawn (outside the frame).
const (
	inkNone uint8 = iota
	inkGround
	inkPiece // inkPiece + p is piece p's
	inkWhite = inkPiece + 7
	inkFrame = inkWhite + 1
)

// inks are the colours of the inks, locku's: the ground surface0, the
// pieces in the colours they usually have in catppuccin-mocha's (the long
// one sky, the square yellow, T mauve, S green, Z red, J blue, L peach), a
// row going white, the curtain overlay1.
var inks = []terminu.Color{
	{},
	terminu.Hex("#313244"),
	terminu.Hex("#89dceb"), terminu.Hex("#f9e2af"), terminu.Hex("#cba6f7"), terminu.Hex("#a6e3a1"),
	terminu.Hex("#f38ba8"), terminu.Hex("#89b4fa"), terminu.Hex("#fab387"),
	terminu.Hex("#ffffff"),
	terminu.Hex("#7f849c"),
}

// field is the GAME panel's body: the level, the rows and the score over
// the field of t's game; the boxes beside the field too when GAME is drawn
// alone.
type field struct{ t *Tetru }

// next is a NEXT panel's body: box i, the piece i+1 to come.
type next struct {
	t *Tetru
	i int
}

// scene is the picture, a pixel an ink.
type scene struct{ px [sceneH][sceneW]uint8 }

func (s *scene) put(x, y int, ink uint8) {
	if x >= 0 && x < sceneW && y >= 0 && y < sceneH {
		s.px[y][x] = ink
	}
}

// picture is the scene of g, nil before the first game: the ground and the
// boxes, the stack, the rows going, the piece falling, the pieces to come;
// at the end the pieces grey as far as the curtain is down. A pause and the
// end are said by popups, not drawn here (user, 2026-11-11: locku draws
// them as it has no popups; tetru does).
func picture(g *game.Game) scene {
	var s scene
	for y := range sceneH {
		for x := range fieldW {
			s.px[y][x] = inkGround
		}
	}
	// The boxes: ground, a pixel apart.
	for i := range game.Queue {
		for y := range boxH {
			for x := range boxW {
				s.put(queueX+x, i*(boxH+1)+y, inkGround)
			}
		}
	}
	if g == nil {
		return s
	}
	for i, p := range g.Next {
		preview(&s, p, queueX, i*(boxH+1))
	}
	for y := range game.Rows {
		going := slices.Contains(g.Going, y)
		for x := range game.Cols {
			c := g.Cell(x, y)
			switch {
			case c == 0:
			case !going:
				s.put(x, y, inkPiece+c-1)
			case g.Flashing() || !g.Wiped(x):
				s.put(x, y, inkWhite)
			}
		}
	}
	if g.Live() {
		for _, b := range g.Blocks() {
			if b[1] >= 0 {
				s.put(b[0], b[1], inkPiece+uint8(g.Cur))
			}
		}
	}
	if g.Over {
		// The pieces turn the frame's grey from the top, a row of pixels at
		// a time; the ground and the frame stay as they are (locku).
		grey := min(sceneH, int(float64(sceneH)*g.CurtainDone()+0.999))
		for y := range grey {
			for x := range sceneW {
				if s.px[y][x] >= inkPiece && s.px[y][x] <= inkWhite {
					s.px[y][x] = inkFrame
				}
			}
		}
	}
	return s
}

// preview draws piece p as it comes in, in the middle of the box whose
// top left is x, y.
func preview(s *scene, p, x, y int) {
	blocks := game.Shape(p)
	lx, hx, ly, hy := blocks[0][0], blocks[0][0], blocks[0][1], blocks[0][1]
	for _, b := range blocks {
		lx, hx, ly, hy = min(lx, b[0]), max(hx, b[0]), min(ly, b[1]), max(hy, b[1])
	}
	px, py := x+(boxW-(hx-lx+1))/2, y+(boxH-(hy-ly+1))/2
	for _, b := range blocks {
		s.put(px+b[0]-lx, py+b[1]-ly, inkPiece+uint8(p))
	}
}

// Draw is the field in the middle of w × h cells, at the scale the NEXT
// panels are drawn at (user, 2026-11-11: the field alone, the level, the
// rows and the score in INFO); narrow, where GAME is alone, the level, the
// rows and the score over the field and the boxes beside it.
func (f *field) Draw(w, h int, focused bool) []string {
	t := f.t
	sc := picture(t.g)
	if t.narrow() {
		row := t.statsRow()
		cols := max(2*sceneW, terminu.Width(row))
		return middle(append([]string{row, ""}, pixels(&sc, 0, sceneW, 0, sceneH, 1)...), cols, w, h)
	}
	return middle(pixels(&sc, 0, fieldW, 0, sceneH, t.scale), 2*fieldW*t.scale, w, h)
}

// stats are the level, the rows cleared and the score, of the game or of
// the last one, a line each: labels in Overlay0, values in Lavender (where
// the game is), as a statusbar's (tdp screen.4).
func (t *Tetru) stats() []string {
	level, lines, score := 1, 0, 0
	if t.g != nil {
		level, lines, score = t.g.Level, t.g.Lines, t.g.Score
	}
	q, v := terminu.Fg(terminu.Overlay0), terminu.Fg(terminu.Lavender)
	return []string{
		q.Render("Level: ") + v.Render(strconv.Itoa(level)),
		q.Render("Lines: ") + v.Render(strconv.Itoa(lines)),
		q.Render("Score: ") + v.Render(comma(score)),
	}
}

// statsRow is the stats on one line, two columns apart (tdp components.2),
// the level a fixed width (tdp L2): narrow, over the field.
func (t *Tetru) statsRow() string {
	level := 1
	if t.g != nil {
		level = t.g.Level
	}
	st := t.stats()
	q, v := terminu.Fg(terminu.Overlay0), terminu.Fg(terminu.Lavender)
	return q.Render("Level: ") + v.Render(fmt.Sprintf("%-2d", level)) + "  " + st[1] + "  " + st[2]
}

// Draw is the box in the middle of w × h cells, at GAME's scale.
func (n *next) Draw(w, h int, focused bool) []string {
	t := n.t
	sc := picture(t.g)
	k := t.scale
	if t.narrow() {
		k = 1
	}
	y0 := n.i * (boxH + 1)
	return middle(pixels(&sc, queueX, sceneW, y0, y0+boxH, k), 2*boxW*k, w, h)
}

// pixels are columns x0 to x1 of rows y0 to y1 of sc, every pixel k × k
// glyphs, two columns each.
func pixels(sc *scene, x0, x1, y0, y1, k int) []string {
	cell := pixelGlyph + strings.Repeat(" ", max(0, 2-terminu.IconWidth()))
	var out []string
	for y := y0; y < y1; y++ {
		var b strings.Builder
		for x := x0; x < x1; {
			ink := sc.px[y][x]
			run := x
			for run < x1 && sc.px[y][run] == ink {
				run++
			}
			n := (run - x) * k
			if ink == inkNone {
				b.WriteString(strings.Repeat("  ", n))
			} else {
				b.WriteString(terminu.Fg(inks[ink]).Render(strings.Repeat(cell, n)))
			}
			x = run
		}
		for range k {
			out = append(out, b.String())
		}
	}
	return out
}

// middle is lines, cols wide, in the middle of w × h cells.
func middle(lines []string, cols, w, h int) []string {
	left, top := max(0, (w-cols)/2), max(0, (h-len(lines))/2)
	pad := strings.Repeat(" ", left)
	blank := strings.Repeat(" ", w)
	out := make([]string, h)
	for i := range out {
		out[i] = blank
		if j := i - top; j >= 0 && j < len(lines) {
			out[i] = terminu.Pad(pad+lines[j], w)
		}
	}
	return out
}

// Key takes nothing out of the Play mode: the keys are the mode's.
func (f *field) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem is false: the field has no cursor; the piece falling is the
// mode's.
func (f *field) HasItem() bool { return false }

// Scroll is nothing: the picture always fits.
func (f *field) Scroll() string { return "" }

// MoveKeys are none out of the mode: nothing moves.
func (f *field) MoveKeys() []terminu.KeyRow { return nil }

// Key takes nothing: a NEXT panel has nothing to do.
func (n *next) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem is false: a NEXT panel has no cursor.
func (n *next) HasItem() bool { return false }

// Scroll is nothing: the box always fits.
func (n *next) Scroll() string { return "" }

// MoveKeys are none: nothing moves.
func (n *next) MoveKeys() []terminu.KeyRow { return nil }
