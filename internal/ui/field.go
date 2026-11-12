package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/game"
)

// The field is drawn as locku's tetromino saver draws its board (user,
// 2026-11-11): a pixel is a square glyph two columns wide and a row high, a
// block a pixel; the ground lit too, in surface0, as locku's board is; the
// pieces to come a box of ground each. The field has no frame of its own:
// PLAYGROUND's is it (user, 2026-11-11: the panel is the game). PLAYGROUND is the only
// panel (user, 2026-11-11: NEXT 1, NEXT 2 and INFO with no [N], never
// focused): at its right are three boxes of its own, framed as content is
// (tdp F1: a frame a panel's content draws is content, how it is drawn
// the app's), Next 1 and Next 2 and INFO (user, 2026-11-11: at the
// right); narrow, PLAYGROUND draws the boxes with no frames, a pixel from the
// field, a pixel apart, and no INFO.
// A box is 6 pixels wide and 4 high (user, 2026-11-11: 4 × 6, after 5 × 5).
const (
	boxW, boxH = 6, 4
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
	inkPiece                // inkPiece + p is piece p's
	inkWhite = inkPiece + 7 // game.Kept's too: inkPiece + game.Kept - 1
	inkFrame = inkWhite + 1
)

// inks are the colours of the inks, locku's: the ground surface0, the
// pieces in the colours they usually have in catppuccin-mocha's (the long
// one sky, the square yellow, T mauve, S green, Z red, J blue, L peach), a
// row going and a block of a game saved and taken up white (user,
// 2026-11-11), the curtain overlay1.
var inks = []terminu.Color{
	{},
	terminu.Hex("#313244"),
	terminu.Hex("#89dceb"), terminu.Hex("#f9e2af"), terminu.Hex("#cba6f7"), terminu.Hex("#a6e3a1"),
	terminu.Hex("#f38ba8"), terminu.Hex("#89b4fa"), terminu.Hex("#fab387"),
	terminu.Hex("#ffffff"),
	terminu.Hex("#7f849c"),
}

// field is the PLAYGROUND panel's body: the field of t's game and, at its right,
// the boxes of the pieces to come and INFO; narrow, the level, the rows and
// the score over the field and the boxes beside it.
type field struct{ t *Tetru }

// scene is the picture, a pixel an ink.
type scene struct{ px [sceneH][sceneW]uint8 }

func (s *scene) put(x, y int, ink uint8) {
	if x >= 0 && x < sceneW && y >= 0 && y < sceneH {
		s.px[y][x] = ink
	}
}

// picture is the scene of g, nil before the first game: the ground and the
// boxes, the stack, the piece falling (or waiting at the top while rows
// go), the pieces to come; the rows going are not in it, they slide out
// over it (slideRows);
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
		for x := range game.Cols {
			if c := g.Cell(x, y); c != 0 {
				s.put(x, y, inkPiece+c-1)
			}
		}
	}
	if !g.Over {
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
				if s.px[y][x] >= inkPiece && s.px[y][x] < inkFrame {
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

// Draw is the field in the middle of the left of w × h cells and the
// column of boxes at the right, at the scale split gives (user,
// 2026-11-11: the field alone, the level, the rows and the score in
// INFO); narrow, the level, the rows and the score over the field and the
// boxes beside it.
func (f *field) Draw(w, h int, focused bool) []string {
	t := f.t
	sc := picture(t.g)
	if t.narrow() {
		row := t.statsRow()
		cols := max(2*sceneW, terminu.Width(row))
		lines := append([]string{row, ""}, pixels(&sc, 0, sceneW, 0, sceneH, 1)...)
		out := middle(lines, cols, w, h)
		slideRows(out, t.g, &sc, w, 1, sceneW, max(0, (w-cols)/2), max(0, (h-len(lines))/2)+2)
		return out
	}
	fw, cw, k := split(w, h)
	lines := pixels(&sc, 0, fieldW, 0, sceneH, k)
	out := middle(lines, 2*fieldW*k, fw, h)
	slideRows(out, t.g, &sc, fw, k, fieldW, max(0, (fw-2*fieldW*k)/2), max(0, (h-len(lines))/2))
	for i, c := range t.column(&sc, cw, h, k) {
		out[i] += c + " "
	}
	return out
}

// split parts PLAYGROUND's body, w × h, in the field's share, fw wide, and the
// column of boxes at its right, cw wide with their frames and a cell from
// PLAYGROUND's frame, a third of the screen or INFO's least; and gives the scale
// both are drawn at: twice where both have the room.
func split(w, h int) (fw, cw, k int) {
	side := max(sideW+1, (w+2)/3)
	fw, cw, k = w-side, side-1, 1
	if fw >= 2*2*fieldW && h >= 2*sceneH && cw-2 >= 2*2*boxW && h >= 2*(2*boxH+2)+infoH {
		k = 2
	}
	return fw, cw, k
}

// column is the boxes at PLAYGROUND's right, cw wide and h high, every pixel k ×
// k glyphs: Next 1 and Next 2, a box each in the middle of its frame; INFO
// under them, as high as its rows; KEYS under it down to the bottom.
func (t *Tetru) column(sc *scene, cw, h, k int) []string {
	nh := boxH*k + 2
	var out []string
	for i, title := range []string{"Next 1", "Next 2"} {
		y0 := i * (boxH + 1)
		out = append(out, framed(title, middle(pixels(sc, queueX, sceneW, y0, y0+boxH, k), 2*boxW*k, cw-2, nh-2), cw)...)
	}
	m := &info{t}
	out = append(out, framed("Info", m.Stats(cw-2, statRows), cw)...)
	return append(out, framed("Keys", m.Keys(cw-2, h-2*nh-(statRows+2)-2), cw)...)
}

// framed is body in a frame w wide, the title upper case after the frame's
// top left corner and a line, as a panel's are kept apart from its corners
// (tdp panel.1); the frame in Surface2 as an unfocused panel's, thin, the
// title in Overlay0, as quiet text: the boxes are to read, never focused.
func framed(title string, body []string, w int) []string {
	line := terminu.Fg(terminu.Surface2)
	name := " " + strings.ToUpper(title) + " "
	out := []string{line.Render("┌─") + terminu.Fg(terminu.Overlay0).Render(name) + line.Render(strings.Repeat("─", max(0, w-3-terminu.Width(name)))+"┐")}
	for _, b := range body {
		out = append(out, line.Render("│")+terminu.Pad(b, w-2)+line.Render("│"))
	}
	return append(out, line.Render("└"+strings.Repeat("─", max(0, w-2))+"┘"))
}

// slideRows draws the rows going over out, w wide, where the scene sc is
// drawn from its column 0 to its column x1, every pixel k × k glyphs, at
// column left and line top: each row, white, slides out to the left as far
// as it has gone, the bottom one first, over the blank left of the field
// and out past PLAYGROUND's frame (user, 2026-11-11), the ground showing where
// it was.
func slideRows(out []string, g *game.Game, sc *scene, w, k, x1, left, top int) {
	if g == nil {
		return
	}
	cell := pixelGlyph + strings.Repeat(" ", max(0, 2-terminu.IconWidth()))
	// lead is the glyphs left of the field, the first of them half cut
	// off when left is odd.
	lead := (left + 1) / 2
	strip := game.Cols * k
	for i, y := range g.Going {
		f := g.Slid(i)
		if f >= 1 {
			continue
		}
		from := -int(f*float64(lead+strip) + 0.5) // the row's first glyph, the field's first being 0
		var b strings.Builder
		u := -lead
		if left%2 == 1 {
			b.WriteString(" ")
			u++
		}
		ink, n := inkNone, 0
		flush := func() {
			if ink == inkNone {
				b.WriteString(strings.Repeat("  ", n))
			} else {
				b.WriteString(terminu.Fg(inks[ink]).Render(strings.Repeat(cell, n)))
			}
			n = 0
		}
		for ; u < x1*k; u++ {
			c := inkNone
			switch {
			case u >= from && u < from+strip:
				c = inkWhite
			case u >= 0:
				c = sc.px[y][u/k]
			}
			if c != ink {
				flush()
				ink = c
			}
			n++
		}
		flush()
		line := terminu.Pad(b.String(), w)
		for j := range k {
			if l := top + y*k + j; l >= 0 && l < len(out) {
				out[l] = line
			}
		}
	}
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

// Key takes nothing out of the Gaming mode: the keys are the mode's.
func (f *field) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem is false: the field has no cursor; the piece falling is the
// mode's.
func (f *field) HasItem() bool { return false }

// Scroll is nothing: the picture always fits.
func (f *field) Scroll() string { return "" }

// MoveKeys are none out of the mode: nothing moves.
func (f *field) MoveKeys() []terminu.KeyRow { return nil }
