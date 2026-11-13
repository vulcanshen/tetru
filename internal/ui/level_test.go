package ui

import (
	"strings"
	"testing"
	"time"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"

	"github.com/vulcanshen/tetru/internal/game"
	"github.com/vulcanshen/tetru/internal/saved"
)

// layout is how a w × h terminal lays PLAYGROUND's body out: the field's
// share fw wide, the field at its column left, right glyphs right of it,
// every pixel k × k glyphs.
func layout(w, h int) (fw, left, right, k int) {
	fw, _, k = split(w-2, h-5)
	left = (fw - 2*fieldW*k) / 2
	return fw, left, (fw - left - 2*fieldW*k) / 2, k
}

// atSides counts the square glyphs on screen left of the field and right
// of it, short of the boxes.
func atSides(d *tdptest.Driver, w, h int) (l, r int) {
	fw, left, _, k := layout(w, h)
	for _, row := range d.Screen() {
		col := 0
		for _, c := range row {
			if string(c) == pixelGlyph {
				switch {
				case col < 1+left:
					l++
				case col >= 1+left+2*fieldW*k && col < 1+fw:
					r++
				}
			}
			col += terminu.Width(string(c))
		}
	}
	return l, r
}

// on is how many pixels of level's figure are on.
func on(level int) int {
	n := 0
	for _, r := range font[level-1] {
		n += strings.Count(r, "#")
	}
	return n
}

// fullFoot fills the field's foot but where the piece falling lands on it:
// a drop clears it.
func fullFoot(g *game.Game) {
	foot := -1
	for _, b := range g.Blocks() {
		foot = max(foot, b[1])
	}
	lo, hi := game.Cols, -1
	for _, b := range g.Blocks() {
		if b[1] == foot {
			lo, hi = min(lo, b[0]), max(hi, b[0])
		}
	}
	for x := range game.Cols {
		if x < lo || x > hi {
			g.Put(x, game.Rows-1, 1)
		}
	}
}

// figure is the level's figures as tt draws them on a w × h terminal.
func figure(tt *Tetru, w, h int) sides {
	_, left, right, k := layout(w, h)
	return tt.sides(k, left, right)
}

// lineInk is the ink of line y of the figure across it: the one ink all
// its glyphs that are on the font have, or 255 when they differ.
func lineInk(sd sides, level, y int) uint8 {
	ink := inkNone
	for x := range 3 * sd.s {
		if font[level-1][y/sd.s][x/sd.s] != '#' {
			continue
		}
		switch c := sd.ink(x, y); {
		case ink == inkNone:
			ink = c
		case c != ink:
			return 255
		}
	}
	return ink
}

func TestTheLevelIsBigAtTheSides(t *testing.T) {
	_, d := open(t, 120, 40)
	if l, r := atSides(d, 120, 40); l+r != 0 {
		t.Fatalf("figures before a game: %d and %d glyphs", l, r)
	}
	play(d)
	d.Advance(sweepTime)
	_, left, right, k := layout(120, 40)
	sd := place(k, left, right)
	if l, r := atSides(d, 120, 40); sd.s != 4 || l != on(1)*sd.s*sd.s || r != l {
		t.Fatalf("120×40: the 1 at the sides is %d and %d glyphs, not %d each, %d times", l, r, on(1)*16, sd.s)
	}
	if -sd.lu-3*sd.s != sd.ru-fieldW*k || sd.ru-fieldW*k < 1 || sd.y0 != (sceneH-5*sd.s)/2 {
		t.Fatalf("the figures are not as far from the field on both sides, a glyph or more, in the middle of its height: %+v", sd)
	}
	// 60 columns leave 5 at either side: no room for 3 glyphs and one more.
	d.Resize(60, 32)
	if l, r := atSides(d, 60, 32); l+r != 0 {
		t.Fatalf("60×32: figures with no room, %d and %d glyphs", l, r)
	}
	d.Resize(150, 54)
	_, left, right, k = layout(150, 54)
	sd = place(k, left, right)
	if l, r := atSides(d, 150, 54); k != 2 || sd.s != 5 || l != on(1)*25 || r != l {
		t.Fatalf("150×54: the 1 is %d and %d glyphs, %d times, the field %d times", l, r, sd.s, k)
	}
}

func TestTheLevelFillsFromItsFoot(t *testing.T) {
	g := game.New(1)
	var b banner
	b.show(1)
	const s = 4
	lit := func() int {
		n, top := 0, -1
		for y := range 5 * s {
			for x := range 3 * s {
				switch b.ink(g, x, y, s, false) {
				case inkLevel:
					if top < 0 {
						top = y
					}
				case inkGround:
					if top >= 0 {
						t.Fatalf("the ground under the lit at line %d", y)
					}
				}
			}
		}
		if top >= 0 {
			n = 5*s - top
		}
		return n
	}
	for _, c := range []struct{ lines, lit int }{{0, 0}, {3, 6}, {9, 18}} {
		g.Lines = c.lines
		if got := lit(); got != c.lit {
			t.Fatalf("%d of the level's rows lit %d lines of 20, not %d", c.lines, got, c.lit)
		}
	}
	// The tenth row: the level up, its clear not over, the 1 is full.
	g.Lines, g.Level = 10, 2
	if got := lit(); got != 5*s {
		t.Fatalf("the level up lit %d lines of 20, not all", got)
	}
}

func TestTheLevelUpSweepsTheNextIn(t *testing.T) {
	tt, d := open(t, 120, 40)
	play(d)
	d.Advance(sweepTime)
	g := tt.g
	g.Lines = game.LinesPerLevel - 1
	fullFoot(g)
	d.Press("Enter")
	if g.Level != 2 || len(g.Going) != 1 || tt.lv.lv != 1 {
		t.Fatalf("the tenth row: level %d, %d rows going, the sides at %d", g.Level, len(g.Going), tt.lv.lv)
	}
	if sd := figure(tt, 120, 40); lineInk(sd, 1, 0) != inkLevel {
		t.Fatal("the 1 is not full as the tenth row goes")
	}
	d.Advance(game.ClearTime(1) + frame)
	if len(g.Going) != 0 || tt.lv.lv != 2 || tt.lv.from != 1 || tt.lv.at >= sweepTime {
		t.Fatalf("the clear over, the 2 does not sweep in: %+v", tt.lv)
	}
	d.Advance(sweepTime / 2)
	sd := figure(tt, 120, 40)
	white := -1
	for y := range 5 * sd.s {
		if sd.ink(0, y) == inkWhite && sd.ink(3*sd.s-1, y) == inkWhite {
			white = y
		}
	}
	if white <= 0 || white >= 5*sd.s-1 || lineInk(sd, 2, 5*sd.s-1) != inkGround || lineInk(sd, 1, 0) != inkLevel {
		t.Fatalf("half way: the white line at %d, the 2 under it empty, the 1 over it full", white)
	}
	at := g.At
	d.Press("h")
	if g.At == at {
		t.Fatal("the game stopped under the sweep")
	}
	d.Advance(sweepTime)
	sd = figure(tt, 120, 40)
	for y := range 5 * sd.s {
		if c := lineInk(sd, 2, y); c != inkGround {
			t.Fatalf("the sweep over, the 2's line %d is %d, not the ground", y, c)
		}
	}
}

func TestTheLastLevelIsABeatingA(t *testing.T) {
	for _, c := range []struct {
		at  time.Duration
		lit uint8
	}{{0, 0}, {rise, 3}, {dub + rise, 2}, {dub + rise + fall, 0}, {beatTime - frame, 0}} {
		if got := heartbeat(c.at); got != c.lit {
			t.Fatalf("the heartbeat at %v: %d, not %d", c.at, got, c.lit)
		}
	}
	tt, d := open(t, 120, 40)
	play(d)
	g := tt.g
	g.Level, g.Lines = game.MaxLevel, 90
	d.Advance(frame + sweepTime)
	if tt.lv.lv != game.MaxLevel || font[game.MaxLevel-1] != [5]string{".#.", "#.#", "###", "#.#", "#.#"} {
		t.Fatal("the last level is not drawn A")
	}
	tt.lv.beat = rise
	sd := figure(tt, 120, 40)
	if lineInk(sd, game.MaxLevel, 0) != inkLevel+3 || lineInk(sd, game.MaxLevel, 5*sd.s-1) != inkLevel+3 {
		t.Fatal("the A is not all white at the top of its heartbeat")
	}
	tt.lv.beat = beatTime - frame
	sd = figure(tt, 120, 40)
	if lineInk(sd, game.MaxLevel, 0) != inkLevel || lineInk(sd, game.MaxLevel, 5*sd.s-1) != inkLevel {
		t.Fatal("the A at no rows of its own is not all lit")
	}
	d.Press("Esc")
	tt.lv.beat = rise
	if sd = figure(tt, 120, 40); lineInk(sd, game.MaxLevel, 0) != inkLevel {
		t.Fatal("the A beats paused")
	}
}

func TestAGameTakenUpShowsItsLevelAtOnce(t *testing.T) {
	t.Setenv("TETRU__DATA", t.TempDir())
	t.Setenv("TETRU__CONFIG", t.TempDir())
	if err := saved.Save(saved.Path(), saved.Game{Level: 3, Lines: 25, Score: 100}); err != nil {
		t.Fatal(err)
	}
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 120, 40)
	if tt.lv.lv != 3 || tt.lv.at != sweepTime {
		t.Fatalf("the game taken up does not show its level at once: %+v", tt.lv)
	}
	sd := figure(tt, 120, 40)
	if lineInk(sd, 3, 5*sd.s-1) != inkLevel || lineInk(sd, 3, 0) != inkGround {
		t.Fatal("the 3 is not half full")
	}
	if l, r := atSides(d, 120, 40); l != on(3)*sd.s*sd.s || r != l {
		t.Fatalf("the 3 at the sides is %d and %d glyphs", l, r)
	}
}

func TestRestartSweepsOneIn(t *testing.T) {
	tt, d := open(t, 120, 40)
	play(d)
	tt.g.Level, tt.g.Lines = 4, 30
	d.Advance(frame + sweepTime)
	if tt.lv.lv != 4 {
		t.Fatal("the 4 did not sweep in")
	}
	d.Press("r")
	if tt.g.Level != 1 || tt.lv.lv != 1 || tt.lv.from != 4 || tt.lv.at != 0 {
		t.Fatalf("restart does not sweep the 1 in over the 4: %+v", tt.lv)
	}
}

func TestWhatGoesOutIsOverTheFigures(t *testing.T) {
	const lead = 6
	sd := sides{s: 1, lu: -4, ru: fieldW + 1, y0: 2, ink: func(x, y int) uint8 { return inkLevel }}
	at := make([]uint8, lead+fieldW+6)
	for u := range fieldW {
		at[u+lead] = inkGround
	}
	at[lead-3] = inkWhite // dust going out over the left figure
	got := sd.lay(at, lead, 2)
	for u := -4; u < -1; u++ {
		want := inkLevel
		if u == -3 {
			want = inkWhite
		}
		if got[u+lead] != want {
			t.Fatalf("glyph %d of the left figure is %d, not %d", u, got[u+lead], want)
		}
	}
	if got[lead] != inkGround || got[fieldW+1+lead] != inkLevel || got[lead-1] != inkNone {
		t.Fatal("the field, the right figure or the gap is wrong")
	}
	if line := sd.lay(at, lead, 1); line[lead-4] != inkNone {
		t.Fatal("a figure on a line over it")
	}
}
