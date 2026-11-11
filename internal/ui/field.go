package ui

import (
	"slices"
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
// locku's board is. The field is GAME's and the boxes NEXT's (user,
// 2026-11-11: three columns); narrow, where GAME is drawn alone, GAME draws
// both, as locku's board is.
const (
	edge       = 1 // the frame's width in pixels
	boxW, boxH = 6, 4
	// fieldW is the field and its frame, queueW the column of boxes and its
	// frame, its left side the field's right; sceneW and sceneH are the two
	// side by side, in pixels.
	fieldW = 2*edge + game.Cols
	queueW = 2*edge + boxW
	sceneW = fieldW + queueW - edge
	sceneH = 2*edge + game.Rows
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

// field is the GAME panel's body: the field of t's game, and the boxes
// too when GAME is drawn alone.
type field struct{ t *Tetru }

// queue is the NEXT panel's body: the boxes of the pieces to come.
type queue struct{ t *Tetru }

// fits is the most a pixel can be grown by for a picture pw pixels wide in
// w × h cells: 2 when it fits twice as large, else 1.
func fits(pw, w, h int) int {
	if w >= 2*2*pw && h >= 2*sceneH {
		return 2
	}
	return 1
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

// Draw is the field in the middle of w × h cells, at the scale NEXT is
// drawn at; narrow, where GAME is alone, the field and the boxes, at the
// largest scale that fits.
func (f *field) Draw(w, h int, focused bool) []string {
	t := f.t
	sc := picture(t.g, t.paused())
	if t.narrow() {
		return draw(&sc, 0, sceneW, fits(sceneW, w, h), w, h)
	}
	return draw(&sc, 0, fieldW, min(t.scale, fits(fieldW, w, h)), w, h)
}

// draw is the pixels x0 to x1 of every row of sc in the middle of w × h
// cells, every pixel k × k glyphs, two columns each.
func draw(sc *scene, x0, x1, k, w, h int) []string {
	cell := pixelGlyph + strings.Repeat(" ", max(0, 2-terminu.IconWidth()))
	pw, ph := 2*(x1-x0)*k, sceneH*k
	left, top := max(0, (w-pw)/2), max(0, (h-ph)/2)
	out := make([]string, h)
	blank := strings.Repeat(" ", w)
	for i := range out {
		out[i] = blank
	}
	for y := range sceneH {
		var b strings.Builder
		b.WriteString(strings.Repeat(" ", left))
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
		row := terminu.Pad(b.String(), w)
		for dy := range k {
			if r := top + y*k + dy; r < h {
				out[r] = row
			}
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

// Draw is the boxes in the middle of w × h cells, at the scale GAME is
// drawn at; alone (narrow), at the largest scale that fits.
func (q *queue) Draw(w, h int, focused bool) []string {
	t := q.t
	sc := picture(t.g, t.paused())
	k := fits(queueW, w, h)
	if !t.narrow() {
		k = min(t.scale, k)
	}
	return draw(&sc, fieldW-edge, sceneW, k, w, h)
}

// Key takes nothing: NEXT has nothing to do.
func (q *queue) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem is false: NEXT has no cursor.
func (q *queue) HasItem() bool { return false }

// Scroll is nothing: the boxes always fit.
func (q *queue) Scroll() string { return "" }

// MoveKeys are none: nothing moves.
func (q *queue) MoveKeys() []terminu.KeyRow { return nil }
