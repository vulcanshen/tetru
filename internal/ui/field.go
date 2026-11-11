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
// block a pixel; the field framed in grey a pixel thick, the pieces to come
// in a column of boxes at its right, a box each, the side by the field the
// field's own. Inside the frame the ground is lit too, in surface0, as
// locku's board is. The field is GAME's and each box a NEXT panel's (user,
// 2026-11-11: Next 1 and Next 2 at the right); narrow, where GAME is drawn
// alone, GAME draws both, as locku's board is.
const (
	edge       = 1 // the frame's width in pixels
	boxW, boxH = 6, 4
	// fieldW is the field and its frame, queueW the column of boxes and its
	// frame, its left side the field's right; sceneW and sceneH are the two
	// side by side, in pixels; boxTall is a box and its frame.
	fieldW  = 2*edge + game.Cols
	queueW  = 2*edge + boxW
	sceneW  = fieldW + queueW - edge
	sceneH  = 2*edge + game.Rows
	boxTall = boxH + 2*edge
	// statsH is the rows over the field in GAME: the level and the rows
	// cleared, the score, and a row between them and the field (user,
	// 2026-11-11: in the game panel).
	statsH = 3
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
	inkRed   = inkFrame + 1
	inkPause = inkRed + 1
)

// inks are the colours of the inks, locku's: the ground surface0, the
// pieces in the colours they usually have in catppuccin-mocha's (the long
// one sky, the square yellow, T mauve, S green, Z red, J blue, L peach), a
// row going white, the frame overlay1, END red, the pause Lavender.
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

// picture is the scene of g, nil before the first game: the frame and the
// boxes, the stack, the rows going, the piece falling, the pieces to come;
// at the end the pieces grey as far as the curtain is down, then END; the
// pause sign over a game paused.
func picture(g *game.Game, paused bool) scene {
	var s scene
	fw := fieldW
	for y := range sceneH {
		for x := range sceneW {
			s.px[y][x] = inkGround
		}
	}
	for x := range sceneW {
		s.px[0][x], s.px[sceneH-1][x] = inkFrame, inkFrame
	}
	for y := range sceneH {
		s.px[y][0], s.px[y][fw-1], s.px[y][sceneW-1] = inkFrame, inkFrame, inkFrame
	}
	// The boxes: a line over each and under the last; under the last box
	// whole, down to the foot, the frame's grey (locku).
	bh := boxH + edge
	for i := 0; i <= game.Queue; i++ {
		for x := fw - 1; x < sceneW; x++ {
			s.put(x, i*bh, inkFrame)
		}
	}
	for y := game.Queue*bh + 1; y < sceneH; y++ {
		for x := fw - 1; x < sceneW; x++ {
			s.put(x, y, inkFrame)
		}
	}
	if g == nil {
		return s
	}
	for i, p := range g.Next {
		preview(&s, p, fw, edge+i*bh)
	}
	for y := range game.Rows {
		going := slices.Contains(g.Going, y)
		for x := range game.Cols {
			c := g.Cell(x, y)
			switch {
			case c == 0:
			case !going:
				s.put(edge+x, edge+y, inkPiece+c-1)
			case g.Flashing() || !g.Wiped(x):
				s.put(edge+x, edge+y, inkWhite)
			}
		}
	}
	if g.Live() {
		for _, b := range g.Blocks() {
			if b[1] >= 0 {
				s.put(edge+b[0], edge+b[1], inkPiece+uint8(g.Cur))
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
			s.stamp(endWord, edge+(game.Cols-len(endWord[0]))/2, (sceneH-len(endWord))/2, inkRed)
		}
	} else if paused {
		s.stamp(pauseSign, edge+(game.Cols-len(pauseSign[0]))/2, edge+(game.Rows-len(pauseSign))/2, inkPause)
	}
	return s
}

// preview draws piece p as it comes in, in the middle of the box whose
// inside's top left is x, y.
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

// Draw is the level, the rows and the score over the field, in the middle
// of w × h cells, at the scale the NEXT panels are drawn at; narrow, where
// GAME is alone, the boxes beside the field.
func (f *field) Draw(w, h int, focused bool) []string {
	t := f.t
	sc := picture(t.g, t.paused())
	x1, k := fieldW, t.scale
	if t.narrow() {
		x1, k = sceneW, 1
	}
	cols := 2 * x1 * k
	return middle(append(t.stats(cols), pixels(&sc, 0, x1, 0, sceneH, k)...), cols, w, h)
}

// stats are the rows over the field, cols wide: the level and the rows
// cleared, the score, of the game or of the last one, values in Lavender
// (where the game is), the level a fixed width (tdp L2); a row blank.
func (t *Tetru) stats(cols int) []string {
	level, lines, score := 1, 0, 0
	if t.g != nil {
		level, lines, score = t.g.Level, t.g.Lines, t.g.Score
	}
	q, v := terminu.Fg(terminu.Overlay0), terminu.Fg(terminu.Lavender)
	return []string{
		terminu.Pad(q.Render("Level: ")+v.Render(fmt.Sprintf("%-2d", level))+q.Render("  Lines: ")+v.Render(strconv.Itoa(lines)), cols),
		terminu.Pad(q.Render("Score: ")+v.Render(comma(score)), cols),
		strings.Repeat(" ", cols),
	}
}

// Draw is the box in the middle of w × h cells, at GAME's scale.
func (n *next) Draw(w, h int, focused bool) []string {
	t := n.t
	sc := picture(t.g, t.paused())
	k := t.scale
	if t.narrow() {
		k = 1
	}
	y0 := n.i * (boxH + edge)
	return middle(pixels(&sc, fieldW-edge, sceneW, y0, y0+boxTall, k), 2*queueW*k, w, h)
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
