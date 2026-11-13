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

// roomAt is the room by the field on a w × h terminal.
func roomAt(w, h int) room {
	_, left, right, k := layout(w, h)
	return room{L: left / 2, R: right, strip: fieldW * k, H: sceneH * k}
}

// sidesAt are the sides as tt draws them on a w × h terminal.
func sidesAt(tt *Tetru, w, h int) sides {
	_, left, right, k := layout(w, h)
	return tt.sides(k, left, right)
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

// lit is how many pixels of a figure's font are on.
func lit(rows [5]string) int {
	n := 0
	for _, r := range rows {
		n += strings.Count(r, "#")
	}
	return n
}

// lineInk is the ink of line y of f's left figure as sd draws it: the one
// ink all its glyphs on the font have, 255 when they differ.
func lineInk(sd sides, f figure, y int) uint8 {
	ink := inkNone
	for u := f.lu; u < f.lu+len(f.rows[0])*f.sx; u++ {
		if on, _ := f.at(u, f.y0+y); !on {
			continue
		}
		switch c := sd.ink(u, f.y0+y); {
		case ink == inkNone:
			ink = c
		case c != ink:
			return 255
		}
	}
	return ink
}

// whole says whether every glyph at the sides of line l is ink.
func whole(sd sides, r room, l int, ink uint8) bool {
	for u := -r.L; u < r.strip+r.R; u++ {
		if r.side(u) && sd.ink(u, l) != ink {
			return false
		}
	}
	return true
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

func TestTheLevelIsBigAtTheSides(t *testing.T) {
	_, d := open(t, 120, 40)
	if l, r := atSides(d, 120, 40); l+r != 0 {
		t.Fatalf("figures before a game: %d and %d glyphs", l, r)
	}
	play(d)
	d.Advance(sweepTime)
	for _, c := range []struct{ w, h, sx, sy int }{
		{120, 40, 4, 4},
		// Narrow, as high as wide (user, 2026-11-13: bigger narrow).
		{70, 32, 1, 4},
		{150, 54, 5, 9},
	} {
		d.Resize(c.w, c.h)
		r := roomAt(c.w, c.h)
		f := r.place(1)
		if l, rr := atSides(d, c.w, c.h); f.sx != c.sx || f.sy != c.sy || l != lit(f.rows)*f.sx*f.sy || rr != l {
			t.Fatalf("%d×%d: the 1 is %d and %d glyphs, %d × %d a pixel", c.w, c.h, l, rr, f.sx, f.sy)
		}
		if f.sy != r.H/5 || -f.lu-3*f.sx != f.ru-r.strip || f.ru-r.strip < 1 || f.y0 != (r.H-5*f.sy)/2 {
			t.Fatalf("%d×%d: the figures are not as high as the field lets them, as far from it on both sides, in the middle: %+v", c.w, c.h, f)
		}
	}
	// 60 columns leave 2 glyphs at either side: no room for 1 and one more.
	d.Resize(60, 32)
	if l, r := atSides(d, 60, 32); l+r != 0 {
		t.Fatalf("60×32: figures with no room, %d and %d glyphs", l, r)
	}
}

func TestTheLevelFillsFromItsFoot(t *testing.T) {
	g := game.New(1)
	var b banner
	b.show(1)
	r := room{L: 17, R: 17, strip: fieldW, H: sceneH}
	f := r.place(1)
	lines := func() int {
		top := -1
		for y := range 5 * f.sy {
			for u := f.lu; u < f.lu+3*f.sx; u++ {
				switch b.ink(g, r, u, f.y0+y, false) {
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
		if top < 0 {
			return 0
		}
		return 5*f.sy - top
	}
	// Twenty rows a level (user, 2026-11-13): a row lights a line of 20.
	for _, c := range []struct{ lines, lit int }{{0, 0}, {3, 3}, {19, 19}} {
		g.Lines = c.lines
		if got := lines(); got != c.lit {
			t.Fatalf("%d of the level's rows lit %d lines of 20, not %d", c.lines, got, c.lit)
		}
	}
	// The twentieth row: the level up, its clear not over, the 1 is full.
	g.Lines, g.Level = game.LinesPerLevel, 2
	if got := lines(); got != 5*f.sy {
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
	r := roomAt(120, 40)
	one, two := r.place(1), r.place(2)
	if g.Level != 2 || len(g.Going) != 1 || tt.lv.lv != 1 {
		t.Fatalf("the tenth row: level %d, %d rows going, the sides at %d", g.Level, len(g.Going), tt.lv.lv)
	}
	if lineInk(sidesAt(tt, 120, 40), one, 0) != inkLevel {
		t.Fatal("the 1 is not full as the tenth row goes")
	}
	d.Advance(game.ClearTime(1) + frame)
	if len(g.Going) != 0 || tt.lv.lv != 2 || tt.lv.from != 1 || tt.lv.at >= sweepTime {
		t.Fatalf("the clear over, the 2 does not sweep in: %+v", tt.lv)
	}
	d.Advance(sweepTime / 2)
	// Half way: a band of white across the whole of both sides (user,
	// 2026-11-13: bigger), fading under it, the 2 under that empty, the 1
	// over it full.
	sd := sidesAt(tt, 120, 40)
	white, fading := 0, 0
	for l := range r.H {
		switch {
		case whole(sd, r, l, inkWhite):
			white++
		case whole(sd, r, l, inkFade), whole(sd, r, l, inkFade+1), whole(sd, r, l, inkFade+2):
			fading++
		}
	}
	w, step := r.band()
	if white != w || fading != 3*step || lineInk(sd, two, 5*two.sy-1) != inkGround || lineInk(sd, one, 0) != inkLevel {
		t.Fatalf("half way: %d lines white, %d fading", white, fading)
	}
	at := g.At
	d.Press("h")
	if g.At == at {
		t.Fatal("the game stopped under the sweep")
	}
	d.Advance(sweepTime)
	sd = sidesAt(tt, 120, 40)
	for y := range 5 * two.sy {
		if c := lineInk(sd, two, y); c != inkGround {
			t.Fatalf("the sweep over, the 2's line %d is %d, not the ground", y, c)
		}
	}
}

func TestTheLastLevelIsFBreathing(t *testing.T) {
	// Fifteen levels, 1 to F, one character wide and narrow (user,
	// 2026-11-13).
	if game.MaxLevel != 15 || font[9] != [5]string{".#.", "#.#", "###", "#.#", "#.#"} || font[14] != [5]string{"###", "#..", "##.", "#..", "#.."} {
		t.Fatal("the levels are not 1 to F")
	}
	if f := roomAt(70, 32).place(game.MaxLevel); f.sx != 1 || f.sy != 4 {
		t.Fatalf("70×32: F is %d × %d a pixel", f.sx, f.sy)
	}
	// Lavender, half way to the ground, the ground: no white (user,
	// 2026-11-13: white too much).
	for _, c := range []struct {
		at  time.Duration
		lit uint8
	}{{0, 0}, {rise, 2}, {dub + rise, 1}, {dub + rise + fall, 0}, {beatTime - frame, 0}} {
		if got := heartbeat(c.at); got != c.lit {
			t.Fatalf("the heartbeat at %v: %d, not %d", c.at, got, c.lit)
		}
	}
	tt, d := open(t, 120, 40)
	play(d)
	g := tt.g
	g.Level, g.Lines = game.MaxLevel, (game.MaxLevel-1)*game.LinesPerLevel
	d.Advance(frame + sweepTime)
	f := roomAt(120, 40).place(game.MaxLevel)
	if tt.lv.lv != game.MaxLevel {
		t.Fatal("the last level did not sweep in")
	}
	seen := map[uint8]bool{}
	for at := time.Duration(0); at < beatTime; at += frame {
		tt.lv.beat = at
		sd := sidesAt(tt, 120, 40)
		top, foot := lineInk(sd, f, 0), lineInk(sd, f, 5*f.sy-1)
		if top != foot {
			t.Fatalf("the F is not all one ink %v into its heartbeat", at)
		}
		seen[top] = true
	}
	if len(seen) != 3 || !seen[inkLevel] || !seen[inkLevel+1] || !seen[inkGround] || inks[inkLevel+1] != terminu.Hex("#7378a1") {
		t.Fatalf("the F beats through %v, not Lavender, half way and the ground", seen)
	}
	d.Press("Esc")
	tt.lv.beat = rise
	if lineInk(sidesAt(tt, 120, 40), f, 0) != inkLevel {
		t.Fatal("the F beats paused")
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
	f := roomAt(120, 40).place(3)
	sd := sidesAt(tt, 120, 40)
	if lineInk(sd, f, 5*f.sy-1) != inkLevel || lineInk(sd, f, 0) != inkGround {
		t.Fatal("the 3 is not half full")
	}
	if l, r := atSides(d, 120, 40); l != lit(f.rows)*f.sx*f.sy || r != l {
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
	d.Press("Enter") // no points: the confirm
	if tt.g.Level != 1 || tt.lv.lv != 1 || tt.lv.from != 4 || tt.lv.at != 0 {
		t.Fatalf("restart does not sweep the 1 in over the 4: %+v", tt.lv)
	}
}

func TestWhatGoesOutIsOverTheFigures(t *testing.T) {
	const lead = 6
	sd := sides{on: true, ink: func(u, l int) uint8 {
		if u < 0 && l == 2 {
			return inkLevel
		}
		return inkNone
	}}
	at := make([]uint8, lead+fieldW+6)
	for u := range fieldW {
		at[u+lead] = inkGround
	}
	at[lead-3] = inkWhite // dust going out over the left figure
	got := sd.lay(at, lead, 2)
	for u := -lead; u < 0; u++ {
		want := inkLevel
		if u == -3 {
			want = inkWhite
		}
		if got[u+lead] != want {
			t.Fatalf("glyph %d left of the field is %d, not %d", u, got[u+lead], want)
		}
	}
	if got[lead] != inkGround || got[lead+fieldW] != inkNone {
		t.Fatal("the field or the right is drawn over")
	}
	if (sides{}).lay(at, lead, 2)[lead-1] != inkNone {
		t.Fatal("figures with no sides")
	}
}
