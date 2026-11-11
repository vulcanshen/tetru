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

// The field is a grid of pixels, a block a pixel, in locku's colours: the
// ground lit too, in surface0, as locku's board is; the pieces to come a box
// of ground each. Neither has a frame of its own: the panel's is it (user,
// 2026-11-11: the panel is the game; NEXT has a panel of its own). The field
// is GAME's and each box a NEXT panel's (user, 2026-11-11: Next 1 and Next 2
// at the right); narrow, where GAME is drawn alone, GAME draws the boxes
// too, a pixel from the field, a pixel apart.
//
// A pixel fills as much of GAME as it can (user, 2026-11-11: the space is
// there, use it): a solid block cw columns wide and e eighths of a row high,
// a cell a few eighths of two pixels when a pixel's edge falls inside a row
// (a lower block in the lower pixel's colour on the upper pixel's). locku's
// square glyph with a gap round it can only be a row high, which left most
// of GAME empty; pixels side by side are told apart by their shade instead,
// lighter and darker by turns, a checkerboard.
const (
	boxW, boxH = 6, 4
	// fieldW and sceneH are the field; queueX is where the column of boxes
	// starts beside it; sceneW is the two side by side, in pixels.
	fieldW = game.Cols
	sceneH = game.Rows
	queueX = fieldW + 1
	sceneW = queueX + boxW
	// rowE is a row in eighths; a pixel is an even number of them, so that
	// the field and a box end on a row's edge.
	rowE = 8
	// cellAspect is how much higher than wide a cell is, about: what makes
	// a pixel cw columns by e eighths look square.
	cellAspect = 2.2
)

// The inks a pixel can be in: none is not drawn (outside the frame).
const (
	inkNone uint8 = iota
	inkGround
	inkPiece // inkPiece + p is piece p's
	inkWhite = inkPiece + 7
	inkFrame = inkWhite + 1
	inkRed   = inkFrame + 1
	inkPause = inkRed + 1
)

// inks are the colours of the inks, locku's: the ground surface0, the
// pieces in the colours they usually have in catppuccin-mocha's (the long
// one sky, the square yellow, T mauve, S green, Z red, J blue, L peach), a
// row going white, the curtain overlay1, END red, the pause Lavender.
var inks = []terminu.Color{
	{},
	terminu.Hex("#313244"),
	terminu.Hex("#89dceb"), terminu.Hex("#f9e2af"), terminu.Hex("#cba6f7"), terminu.Hex("#a6e3a1"),
	terminu.Hex("#f38ba8"), terminu.Hex("#89b4fa"), terminu.Hex("#fab387"),
	terminu.Hex("#ffffff"),
	terminu.Hex("#7f849c"),
	terminu.Hex("#f38ba8"),
	terminu.Lavender,
}

// shades are the inks a little darker: every other pixel of the
// checkerboard.
var shades = func() []terminu.Color {
	out := make([]terminu.Color, len(inks))
	for i, c := range inks {
		if c.Set {
			out[i] = terminu.Hex(fmt.Sprintf("#%02x%02x%02x", int(c.RGB[0])*82/100, int(c.RGB[1])*82/100, int(c.RGB[2])*82/100))
		}
	}
	return out
}()

// colour is the colour of the pixel at x, y in ink: its ink, or its shade
// on every other pixel; unset for no ink.
func colour(ink uint8, x, y int) terminu.Color {
	if (x+y)%2 == 1 {
		return shades[ink]
	}
	return inks[ink]
}

// fit is the largest pixel a picture w pixels wide and h high takes in
// cols × rows cells: e eighths of a row high, an even number and a row at
// least, and cw columns wide, as many as make it about square, two at
// least; no wider than cols.
func fit(w, h, cols, rows int) (cw, e int) {
	e = max(rowE, rows*rowE/h&^1)
	cw = colsFor(e)
	if cw*w > cols {
		cw = max(2, cols/w)
		for e > rowE && colsFor(e) > cw {
			e -= 2
		}
	}
	return cw, e
}

// colsFor is the columns that make a pixel e eighths high about square.
func colsFor(e int) int { return max(2, int(float64(e)*cellAspect/rowE+0.5)) }

// rowsFor is the rows h pixels e eighths high take.
func rowsFor(h, e int) int { return (h*e + rowE - 1) / rowE }

// endWord is END in locku's short pixel face, a pixel apart, the E a pixel
// narrower than locku's: the word fits inside the field, ten pixels.
var endWord = []string{
	"##.###.##.",
	"#..#.#.#.#",
	"##.#.#.#.#",
	"#..#.#.#.#",
	"##.#.#.##.",
}

// pauseSign is two bars, the pause.
var pauseSign = []string{"##.##", "##.##", "##.##", "##.##", "##.##"}

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

// stamp lights the '#' of rows in ink, its top left at x, y.
func (s *scene) stamp(rows []string, x, y int, ink uint8) {
	for dy, row := range rows {
		for dx := range row {
			if row[dx] == '#' {
				s.put(x+dx, y+dy, ink)
			}
		}
	}
}

// picture is the scene of g, nil before the first game: the ground and the
// boxes, the stack, the rows going, the piece falling, the pieces to come;
// at the end the pieces grey as far as the curtain is down, then END; the
// pause sign over a game paused.
func picture(g *game.Game, paused bool) scene {
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
		if g.Ended() {
			s.stamp(endWord, (game.Cols-len(endWord[0]))/2, (sceneH-len(endWord))/2, inkRed)
		}
	} else if paused {
		s.stamp(pauseSign, (game.Cols-len(pauseSign[0]))/2, (game.Rows-len(pauseSign))/2, inkPause)
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

// Draw is the field in the middle of w × h cells, its pixels as large as
// the NEXT panels' (layout); narrow, where GAME is alone, the level, the
// rows and the score over the field and the boxes beside it, a pixel a row.
func (f *field) Draw(w, h int, focused bool) []string {
	t := f.t
	sc := picture(t.g, t.paused())
	if t.narrow() {
		row := t.statsRow()
		cw := colsFor(rowE)
		cols := max(cw*sceneW, terminu.Width(row))
		return middle(append([]string{row, ""}, solid(&sc, 0, sceneW, 0, sceneH, cw, rowE)...), cols, w, h)
	}
	return middle(solid(&sc, 0, fieldW, 0, sceneH, t.cw, t.e), t.cw*fieldW, w, h)
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

// Draw is the box in the middle of w × h cells, its pixels as large as
// GAME's; alone (narrow), a pixel a row.
func (n *next) Draw(w, h int, focused bool) []string {
	t := n.t
	sc := picture(t.g, t.paused())
	cw, e := t.cw, t.e
	if t.narrow() {
		cw, e = colsFor(rowE), rowE
	}
	y0 := n.i * (boxH + 1)
	return middle(solid(&sc, queueX, sceneW, y0, y0+boxH, cw, e), cw*boxW, w, h)
}

// lowerBlocks are the blocks filling the lower k eighths of a cell.
var lowerBlocks = [rowE]string{"", "▁", "▂", "▃", "▄", "▅", "▆", "▇"}

// solid is columns x0 to x1 of rows y0 to y1 of sc, every pixel a solid
// block cw columns wide and e eighths high: a cell holding one pixel is a
// full block in its colour; a cell holding the foot of one and the head of
// the next is a lower block in the lower one's colour on the upper one's.
// A pixel with no ink is blank.
func solid(sc *scene, x0, x1, y0, y1, cw, e int) []string {
	out := make([]string, rowsFor(y1-y0, e))
	for r := range out {
		top := r * rowE
		var b strings.Builder
		for x := x0; x < x1; x++ {
			yt, yb := y0+top/e, y0+(top+rowE-1)/e
			up := ink(sc, x, yt, y1)
			if yt == yb {
				b.WriteString(block(up, inkNone, x, yt, yt, 0, cw))
				continue
			}
			b.WriteString(block(up, ink(sc, x, yb, y1), x, yt, yb, (yb-y0)*e-top, cw))
		}
		out[r] = b.String()
	}
	return out
}

// ink is the ink of the pixel at x, y; none past row y1.
func ink(sc *scene, x, y, y1 int) uint8 {
	if y >= y1 {
		return inkNone
	}
	return sc.px[y][x]
}

// block is a cell cw columns wide: the pixel up at x, yt above split
// eighths and the pixel down at x, yb under it; split 0 is up alone.
func block(up, down uint8, x, yt, yb, split, cw int) string {
	if split == 0 || down == inkNone && up == inkNone {
		if up == inkNone {
			return strings.Repeat(" ", cw)
		}
		return terminu.Fg(colour(up, x, yt)).Render(strings.Repeat("█", cw))
	}
	if down == inkNone { // the foot of a picture ending mid-row: as far as it goes
		if split >= rowE/2 {
			return terminu.Fg(colour(up, x, yt)).Render(strings.Repeat("█", cw))
		}
		return strings.Repeat(" ", cw)
	}
	st := terminu.Style{Fg: colour(down, x, yb)}
	if up != inkNone {
		st.Bg = colour(up, x, yt)
	}
	return st.Render(strings.Repeat(lowerBlocks[rowE-split], cw))
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
