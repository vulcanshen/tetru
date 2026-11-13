package ui

import (
	"math"
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
		{120, 40, 3, 3},
		// Narrow, as high as wide (user, 2026-11-13: bigger narrow).
		{70, 32, 1, 3},
		{150, 54, 5, 6},
	} {
		d.Resize(c.w, c.h)
		r := roomAt(c.w, c.h)
		_, _, _, k := layout(c.w, c.h)
		f := r.place(1)
		if l, rr := atSides(d, c.w, c.h); f.sx != c.sx || f.sy != c.sy || l != lit(f.rows)*f.sx*f.sy || rr != l {
			t.Fatalf("%d×%d: the 1 is %d and %d glyphs, %d × %d a pixel", c.w, c.h, l, rr, f.sx, f.sy)
		}
		// Four of the field's rows clear over them and under them (user,
		// 2026-11-13: too high, two clear).
		if f.y0 < 4*k || r.H-f.y0-5*f.sy < 4*k || 5*(f.sy+1) <= r.H-8*k || -f.lu-3*f.sx != f.ru-r.strip || f.ru-r.strip < 1 || f.y0 != (r.H-5*f.sy)/2 {
			t.Fatalf("%d×%d: the figures are not as high as four rows clear of the field's top and foot let them, as far from it on both sides, in the middle: %+v", c.w, c.h, f)
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
	// Twenty rows a level (user, 2026-11-13), over 15 lines.
	if f.sy != 3 {
		t.Fatalf("the 1 is %d lines a pixel", f.sy)
	}
	for _, c := range []struct{ lines, lit int }{{0, 0}, {3, 2}, {10, 8}, {19, 14}} {
		g.Lines = c.lines
		if got := lines(); got != c.lit {
			t.Fatalf("%d of the level's rows lit %d lines of 15, not %d", c.lines, got, c.lit)
		}
	}
	// The twentieth row: the level up, its clear not over, the 1 is full.
	g.Lines, g.Level = game.LinesPerLevel, 2
	if got := lines(); got != 5*f.sy {
		t.Fatalf("the level up lit %d lines of 15, not all", got)
	}
}

func TestTheLevelUpSweepsTheNextIn(t *testing.T) {
	tt, d := open(t, 120, 40)
	play(d)
	d.Advance(sweepTime)
	g := tt.g
	g.Lines = game.LinesPerLevel - 1
	fullFoot(g)
	p := uint8(g.Cur)
	d.Press("Enter")
	r := roomAt(120, 40)
	one, two := r.place(1), r.place(2)
	if g.Level != 2 || len(g.Going) != 1 || tt.lv.lv != 1 {
		t.Fatalf("the twentieth row: level %d, %d rows going, the sides at %d", g.Level, len(g.Going), tt.lv.lv)
	}
	tt.lv.wave = 0 // no wave over the figures yet
	if lineInk(sidesAt(tt, 120, 40), one, 0) != inkLevel {
		t.Fatal("the 1 is not full as the twentieth row goes")
	}
	d.Advance(game.ClearTime(1) + frame)
	if len(g.Going) != 0 || tt.lv.lv != 2 || tt.lv.from != 1 || tt.lv.at >= sweepTime || tt.lv.by != int(p) {
		t.Fatalf("the clear over, the 2 does not sweep in, in piece %d's colour: %+v", p, tt.lv)
	}
	d.Advance(sweepTime / 2)
	// Half way: a band across the whole of both sides (user, 2026-11-13:
	// bigger) in the colour of the piece that cleared the row (user,
	// 2026-11-13), fading under it, the 2 under that empty, the 1 over it
	// full.
	tt.lv.wave = 0
	sd := sidesAt(tt, 120, 40)
	band, fading := 0, 0
	for l := range r.H {
		switch {
		case whole(sd, r, l, inkPiece+p):
			band++
		case whole(sd, r, l, inkTrail+3*p), whole(sd, r, l, inkTrail+3*p+1), whole(sd, r, l, inkTrail+3*p+2):
			fading++
		}
	}
	w, step := r.band()
	if band != w || fading != 3*step || lineInk(sd, two, 5*two.sy-1) != inkGround || lineInk(sd, one, 0) != inkLevel {
		t.Fatalf("half way: %d lines in the piece's colour, %d fading", band, fading)
	}
	// The trail fades as the clear's white does.
	for f := range 3 {
		if toGround(inks[inkWhite], f+1) != inks[inkFade+uint8(f)] {
			t.Fatalf("a trail %d quarters to the ground is not as the white's", f+1)
		}
	}
	if inks[inkTrail+1] != terminu.Hex("#5d8798") {
		t.Fatalf("sky half way to the ground is %v", inks[inkTrail+1])
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

// waveOver walks a wave over f as sd draws it, a frame at a time, and says
// what it saw: when each line, from the foot, first turned the colour half
// way to the ground (-1 never), and how many times; what is lit is in c or
// c + 1, the wave a font pixel's lines high or less, and the rest the
// ground, or it fails.
func waveOver(t *testing.T, tt *Tetru, f figure, lit int, c uint8) (first []time.Duration, times []int) {
	t.Helper()
	h := 5 * f.sy
	first, times = make([]time.Duration, h), make([]int, h)
	was := make([]bool, h)
	for i := range first {
		first[i] = -1
	}
	for at := time.Duration(0); at < 2*waveTime; at += frame / 4 {
		tt.lv.wave = at % waveTime
		sd := sidesAt(tt, 120, 40)
		on := 0
		for up := range h {
			ink := lineInk(sd, f, h-1-up)
			switch {
			case up >= lit && ink != inkGround:
				t.Fatalf("line %d from the foot, not lit, is %d %v into the wave", up, ink, at)
			case up < lit && ink != c && ink != c+1:
				t.Fatalf("line %d from the foot, lit, is %d %v into the wave", up, ink, at)
			}
			now := up < lit && ink == c+1
			if now {
				on++
				if !was[up] {
					times[up]++
					if first[up] < 0 {
						first[up] = at
					}
				}
			}
			was[up] = now
		}
		if on > f.sy {
			t.Fatalf("a wave %d lines high %v in", on, at)
		}
	}
	return first, times
}

func TestWavesGoUpEveryLevel(t *testing.T) {
	// Every level, over what is lit of it, waves go up from its foot, one
	// after another, in the colour half way to the ground (user,
	// 2026-11-13: waves, not a breath); what is not lit stays the ground.
	tt, d := open(t, 120, 40)
	play(d)
	tt.g.Lines = game.LinesPerLevel / 2
	d.Advance(sweepTime)
	f := roomAt(120, 40).place(1)
	lit := int(math.Round(float64(5*f.sy) / 2))
	first, times := waveOver(t, tt, f, lit, inkLevel)
	for up := range lit {
		if times[up] != 2 {
			t.Fatalf("line %d from the foot went half way %d times in two waves", up, times[up])
		}
		if up > 0 && first[up] <= first[up-1] {
			t.Fatalf("the wave reached line %d from the foot at %v, no later than the one under it (%v)", up, first[up], first[up-1])
		}
	}
	if inks[inkLevel+1] != terminu.Hex("#7378a1") {
		t.Fatal("the wave is not half way from Lavender to the ground")
	}
	d.Press("Esc")
	tt.lv.wave = waveTime / 4
	for y := range lit {
		if lineInk(sidesAt(tt, 120, 40), f, 5*f.sy-1-y) != inkLevel {
			t.Fatal("a wave paused")
		}
	}
}

func TestTwoFiguresPast9WhereTheyHaveRoom(t *testing.T) {
	// 10 to 15 where the sides have room for two figures, A to F where
	// they have not (user, 2026-11-13: wide, 10 to 15).
	if two(10) != [5]string{".#.###", "##.#.#", ".#.#.#", ".#.#.#", ".#.###"} || two(11) != [5]string{".#..#", "##.##", ".#..#", ".#..#", ".#..#"} || two(15)[4] != ".#.##." {
		t.Fatalf("10, 11 and 15 are not two figures: %q %q %q", two(10), two(11), two(15))
	}
	wide, narrow := roomAt(120, 40), roomAt(70, 32)
	for lv := 10; lv <= game.MaxLevel; lv++ {
		if f := wide.place(lv); f.rows != two(lv) || f.sx < 1 {
			t.Fatalf("120×40: level %d is %q", lv, f.rows)
		}
		if f := narrow.place(lv); f.rows != font[lv-1] || f.sx < 1 {
			t.Fatalf("70×32: level %d is %q, not its letter", lv, f.rows)
		}
	}
	if roomAt(76, 32).place(10).rows != font[9] || roomAt(78, 32).place(10).rows != two(10) {
		t.Fatal("10 is not two figures from 78 columns, A under")
	}
	tt, d := open(t, 120, 40)
	play(d)
	tt.g.Level, tt.g.Lines = 10, 9*game.LinesPerLevel
	d.Advance(frame + sweepTime)
	f := wide.place(10)
	if l, r := atSides(d, 120, 40); tt.lv.lv != 10 || l != lit(f.rows)*f.sx*f.sy || r != l {
		t.Fatalf("the 10 at the sides is %d and %d glyphs", l, r)
	}
}

func TestTheLastLevelIsGold(t *testing.T) {
	// Fifteen levels, 1 to F, one character wide and narrow (user,
	// 2026-11-13).
	if game.MaxLevel != 15 || font[9] != [5]string{".#.", "#.#", "###", "#.#", "#.#"} || font[14] != [5]string{"###", "#..", "##.", "#..", "#.."} {
		t.Fatal("the levels are not 1 to F")
	}
	if f := roomAt(70, 32).place(game.MaxLevel); f.sx != 1 || f.sy != 3 {
		t.Fatalf("70×32: F is %d × %d a pixel", f.sx, f.sy)
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
	// All lit, gold, the family's icons' (user, 2026-11-13: gold at the
	// fifteenth), the waves going up it the same, half way to the ground.
	first, _ := waveOver(t, tt, f, 5*f.sy, inkGold)
	if first[0] < 0 || first[5*f.sy-1] <= first[0] || inks[inkGold] != terminu.Hex("#f2b753") || inks[inkGold+1] != terminu.Hex("#92754c") {
		t.Fatal("no wave up the F, or not gold and half way from it")
	}
	d.Press("Esc")
	tt.lv.wave = waveTime / 2
	if lineInk(sidesAt(tt, 120, 40), f, 5*f.sy/2) != inkGold {
		t.Fatal("a wave up the F paused")
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
	if tt.g.Level != 1 || tt.lv.lv != 1 || tt.lv.from != 4 || tt.lv.at != 0 || tt.lv.by != tt.g.Cur {
		t.Fatalf("restart does not sweep the 1 in over the 4 in the first piece's colour: %+v", tt.lv)
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
