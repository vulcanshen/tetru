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
	inkFade  = inkFrame + 1 // inkFade + f, f 0 to 2: white a quarter, a half, three quarters to the ground
	inkHint  = inkFade + 3
	inkLevel = inkHint + 1 // the level's figures lit, Lavender; inkLevel + 1 half way to the ground (heartbeat)
)

// inks are the colours of the inks, locku's: the ground surface0, the
// pieces in the colours they usually have in catppuccin-mocha's (the long
// one sky, the square yellow, T mauve, S green, Z red, J blue, L peach), a
// block of a game saved and taken up and the rows going white, a colour no
// piece has (user, 2026-11-11; 2026-11-12: the rows going all one colour,
// rosewater, then white), the curtain overlay1, three steps of white
// fading to the ground, and the hint overlay0, a grey lighter than the
// ground and a little darker than the curtain (user, 2026-11-12: a lighter
// grey; surface2 first, then overlay0, lighter still); the level's figures
// lit, lavender, as INFO's values, and half way from it to the ground, as
// the last level's heartbeats (user, 2026-11-13: not white, too much).
var inks = []terminu.Color{
	{},
	terminu.Hex("#313244"),
	terminu.Hex("#89dceb"), terminu.Hex("#f9e2af"), terminu.Hex("#cba6f7"), terminu.Hex("#a6e3a1"),
	terminu.Hex("#f38ba8"), terminu.Hex("#89b4fa"), terminu.Hex("#fab387"),
	terminu.Hex("#ffffff"),
	terminu.Hex("#7f849c"),
	terminu.Hex("#ccccd0"), terminu.Hex("#9899a2"), terminu.Hex("#656573"),
	terminu.Hex("#6c7086"),
	terminu.Hex("#b4befe"), terminu.Hex("#7378a1"),
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
// boxes, the stack, the hint under the piece to level 5 (where a drop would
// set it), the piece falling (or waiting at the top while rows
// go), the pieces to come; the rows going are not in it, they go out
// over it (clearRows);
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
		// The hint, under the piece: where a drop would set it, to level
		// 5 (user, 2026-11-12).
		for _, b := range g.Landing() {
			s.put(b[0], b[1], inkHint)
		}
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

// Draw is the field in the middle of the left of w × h cells, the level
// big at its sides where there is room (sides), and the column of boxes
// at the right, at the scale split gives (user, 2026-11-11: the field
// alone, the level, the rows and the score in INFO); narrow, the level,
// the rows and the score over the field and the boxes beside it.
func (f *field) Draw(w, h int, focused bool) []string {
	t := f.t
	sc := picture(t.g)
	if t.narrow() {
		head := append(t.statsRows(), "")
		cols := 2 * sceneW
		for _, r := range head {
			cols = max(cols, terminu.Width(r))
		}
		lines := append(head, pixels(&sc, 0, sceneW, 0, sceneH, 1)...)
		out := middle(lines, cols, w, h)
		// The boxes are right of the field: nothing goes out that way.
		clearRows(out, t.g, &sc, w, 1, sceneW, max(0, (w-cols)/2), max(0, (h-len(lines))/2)+len(head), 0, sides{})
		return out
	}
	fw, cw, k := split(w, h)
	lines := pixels(&sc, 0, fieldW, 0, sceneH, k)
	out := middle(lines, 2*fieldW*k, fw, h)
	left := max(0, (fw-2*fieldW*k)/2)
	top, right := max(0, (h-len(lines))/2), (fw-left-2*fieldW*k)/2
	sd := t.sides(k, left, right)
	// The field's lines, its glyphs and the sides'.
	lead, end := (left+1)/2, fieldW*k+right
	for l := 0; sd.on && l < sceneH*k && top+l < h; l++ {
		at := make([]uint8, lead+end)
		for u := range fieldW * k {
			at[u+lead] = sc.px[l/k][u/k]
		}
		out[top+l] = glyphs(sd.lay(at, lead, l), make([]uint8, lead+end), left, fw)
	}
	clearRows(out, t.g, &sc, fw, k, fieldW, left, top, right, sd)
	for i, c := range t.column(&sc, cw, h, k) {
		out[i] += c + " "
	}
	return out
}

// split parts PLAYGROUND's body, w × h, in the field's share, fw wide, and the
// column of boxes at its right, cw wide with their frames, a cell from
// PLAYGROUND's frame: as wide as INFO needs, or the boxes drawn twice as
// big, however wide the screen is (user, 2026-11-12: the boxes' width
// fixed, not growing with the screen; a third of it first); and gives the
// scale both are drawn at: twice where both have the room.
func split(w, h int) (fw, cw, k int) {
	cw, k = sideW, 1
	if c := max(sideW, 2*2*boxW+2); w-c-1 >= 2*2*fieldW && h >= 2*sceneH && h >= 2*(2*boxH+2)+infoH {
		cw, k = c, 2
	}
	return w - cw - 1, cw, k
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

// crumble is the order the blocks of a row go in as they crumble: block x
// of row i goes at crumble[(x+5i) mod 12], of 12.
var crumble = [game.Cols]int{7, 2, 10, 4, 0, 9, 5, 11, 1, 6, 3, 8}

// broken are the glyphs a block goes through as it crumbles, after the
// square and before it is gone: grains, dust (user, 2026-11-12: the square
// itself breaking; nf-md-grain, nf-md-blur; first nf-md-view_grid before
// them, then taken out).
var broken = [...]string{string(rune(0xF0D7C)), string(rune(0xF00B5))}

// shattered is the shape of a block gone, past the broken ones.
const shattered = uint8(len(broken)) + 1

// blown is the shape of four rows' dust as it goes out, windblown its
// glyph (user, 2026-11-12: one glyph more as it goes; nf-weather-dust,
// then nf-md-dots_horizontal tried).
const blown = shattered + 1

var windblown = string(rune(0xF01D8))

// breaking is the shape at t of a block that is gone at gone, breaking
// over span before it: 0 whole, 1 on broken[0] on, shattered gone.
func breaking(t, gone, span float64) uint8 {
	switch {
	case t >= gone:
		return shattered
	case t < gone-span:
		return 0
	}
	return 1 + uint8((t-(gone-span))/span*float64(len(broken)))
}

// clearRows draws the rows going over out, w wide, where the scene sc is
// drawn from its column 0 to its column x1, every pixel k × k glyphs, at
// column left and line top, with right glyphs blank right of the field to
// go into. However many they are, they flash white, go, and the stack
// comes down whole (user, 2026-11-12: one way for all); they go white
// (user, 2026-11-12), by how many they are, in about the same time:
//   - one fades away where it is;
//   - two burst from their middle, ring by ring outwards;
//   - three crumble, a block at a time in no order, each breaking (broken)
//     before it is gone (user, 2026-11-12);
//   - four, all together, break where they are, every block at once to
//     grains, then all to dust, then the halves go out to either side,
//     blown, the right one as far as the blank there (user, 2026-11-12:
//     not a row at a time; all breaking together, all to dust, then going
//     out, a glyph more).
//
// The ground shows where they were; the level's figures at the sides, sd,
// where nothing going out is over them.
func clearRows(out []string, g *game.Game, sc *scene, w, k, x1, left, top, right int, sd sides) {
	if g == nil || len(g.Going) == 0 {
		return
	}
	// lead is the glyphs left of the field, the first of them half cut
	// off when left is odd.
	lead := (left + 1) / 2
	strip, half := game.Cols*k, game.Cols/2*k
	end := max(x1*k, strip+right) // past the last glyph drawn
	for i, y := range g.Going {
		at := make([]uint8, lead+end) // glyph u's ink at u+lead, the field's first glyph being 0
		sh := make([]uint8, lead+end) // and its shape (breaking)
		for u := 0; u < x1*k; u++ {
			at[u+lead] = sc.px[y][u/k]
		}
		put := func(u int, ink, shape uint8) {
			if u >= -lead && u < end {
				at[u+lead], sh[u+lead] = ink, shape
			}
		}
		block := func(x int, ink, shape uint8) {
			for j := range k {
				put(x*k+j, ink, shape)
			}
		}
		n, p := len(g.Going), g.Along()
		switch {
		case p == 0: // the flash
			for x := range game.Cols {
				block(x, inkWhite, 0)
			}
		case n == 1: // fade
			if step := int(p * 4); step < 4 {
				for x := range game.Cols {
					block(x, []uint8{inkWhite, inkFade, inkFade + 1, inkFade + 2}[step], 0)
				}
			}
		case n == 2: // burst
			for x := range game.Cols {
				// The ring: twice the steps from the middle of the rows.
				d := max(2*x-(game.Cols-1), game.Cols-1-2*x) + max(2*i-(n-1), n-1-2*i)
				if p < float64(d)/float64(game.Cols-1+n-1)*0.85+0.15 {
					block(x, inkWhite, 0)
				}
			}
		case n == 3: // crumble
			for x := range game.Cols {
				// Each starts breaking after the flash, the first at once.
				gone := 0.25 + float64(crumble[(x+5*i)%game.Cols])/game.Cols*0.75
				if s := breaking(p, gone, 0.25); s < shattered {
					block(x, inkWhite, s)
				}
			}
		default: // break to grains, to dust, then the halves split
			// Every block breaks to grains at once after the flash (user,
			// 2026-11-12: all together, not half), all to dust at 40%
			// (user, 2026-11-12: the middle no different), and the halves
			// go out over the last 40%, blown.
			shape := uint8(1)
			switch {
			case p >= 0.6:
				shape = blown
			case p >= 0.4:
				shape = 2
			}
			s := max(0, (p-0.6)/0.4)
			offL, offR := int(s*float64(lead+half)+0.5), int(s*float64(right+half)+0.5)
			for x := range game.Cols {
				u := x*k + offR
				if x < game.Cols/2 {
					u = x*k - offL
				}
				for j := range k {
					if x >= game.Cols/2 && u+j >= strip+right {
						break
					}
					put(u+j, inkWhite, shape)
				}
			}
		}
		for j := range k {
			if l := top + y*k + j; l >= 0 && l < len(out) {
				out[l] = glyphs(sd.lay(at, lead, y*k+j), sh, left, w)
			}
		}
	}
}

// glyphs is a line w wide of the glyphs at, inks, shaped sh (breaking):
// the field's, the glyphs right of it and, first, those left of it, the
// field starting at column left; the first of them half cut off, a blank,
// when left is odd.
func glyphs(at, sh []uint8, left, w int) string {
	pad := strings.Repeat(" ", max(0, 2-terminu.IconWidth()))
	var b strings.Builder
	u := 0
	if left%2 == 1 {
		b.WriteString(" ")
		u++
	}
	ink, shape, run := inkNone, uint8(0), 0
	flush := func() {
		if ink == inkNone {
			b.WriteString(strings.Repeat("  ", run))
		} else {
			glyph := pixelGlyph
			switch {
			case shape == blown:
				glyph = windblown
			case shape > 0:
				glyph = broken[shape-1]
			}
			b.WriteString(terminu.Fg(inks[ink]).Render(strings.Repeat(glyph+pad, run)))
		}
		run = 0
	}
	for ; u < len(at); u++ {
		if at[u] != ink || sh[u] != shape {
			flush()
			ink, shape = at[u], sh[u]
		}
		run++
	}
	flush()
	return terminu.Pad(b.String(), w)
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

// statsRows are the stats on two lines, narrow, over the field: the level,
// a fixed width (it is at most 15), and the rows two columns apart (tdp
// components.2); the score under them. The rows and the score have no most,
// so each is last on its line (tdp L2: only the last may change width).
func (t *Tetru) statsRows() []string {
	level := 1
	if t.g != nil {
		level = t.g.Level
	}
	st := t.stats()
	q, v := terminu.Fg(terminu.Overlay0), terminu.Fg(terminu.Lavender)
	return []string{q.Render("Level: ") + v.Render(fmt.Sprintf("%-2d", level)) + "  " + st[1], st[2]}
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
