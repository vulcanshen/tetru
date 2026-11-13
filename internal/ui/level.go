package ui

import (
	"math"
	"slices"
	"time"

	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/game"
)

// The level is drawn big at both sides of the field (user, 2026-11-13: what
// the blank there is for), in the field's square glyph: a 3 × 5 font, a
// pixel of it sx glyphs across and sy lines down, as high as the field
// lets it be and as wide as the sides let it be (user, 2026-11-13: bigger
// where the terminal is narrow). One character, 1 to 9 then A to F (user,
// 2026-11-13: fifteen levels, so that one character does wide and narrow;
// 10 tried before), four of the field's rows clear of its top and its foot
// (user, 2026-11-13: too high, two rows clear). It lights up from its foot
// as the level's rows are cleared, Lavender over the ground, and what is
// lit beats like a heart, going to the ground and back through the colour
// half way (user, 2026-11-13: every level breathes; not white, too much);
// the last level, F, is all lit, gold (user, 2026-11-13). As the level
// goes up, a band with a fading trail goes up over the whole of both sides
// in the colour of the piece that cleared the level's last rows (user,
// 2026-11-13: colour, the piece that fell), the next level's figure under
// it (user, 2026-11-13: the effect bigger).
const (
	// sweepTime is how long the band takes from the field's foot to past
	// its top.
	sweepTime = 800 * time.Millisecond
	// beatTime is a heartbeat of the last level: a beat to the ground, then
	// dub after it a beat a little over half as deep, each down in rise and
	// back in fall.
	beatTime = 1000 * time.Millisecond
	dub      = 280 * time.Millisecond
	rise     = 80 * time.Millisecond
	fall     = 160 * time.Millisecond
)

// font are the levels' figures, 3 × 5, # on: 1 to 9, then A to F.
var font = [game.MaxLevel][5]string{
	{".#.", "##.", ".#.", ".#.", "###"},
	{"##.", "..#", ".#.", "#..", "###"},
	{"##.", "..#", ".#.", "..#", "##."},
	{"#.#", "#.#", "###", "..#", "..#"},
	{"###", "#..", "##.", "..#", "##."},
	{".##", "#..", "###", "#.#", "###"},
	{"###", "..#", ".#.", ".#.", ".#."},
	{"###", "#.#", "###", "#.#", "###"},
	{"###", "#.#", "###", "..#", "##."},
	{".#.", "#.#", "###", "#.#", "#.#"},
	{"##.", "#.#", "##.", "#.#", "##."},
	{".##", "#..", "#..", "#..", ".##"},
	{"##.", "#.#", "#.#", "#.#", "##."},
	{"###", "#..", "##.", "#..", "###"},
	{"###", "#..", "##.", "#..", "#.."},
}

// room is the room by the field: whole glyphs L left of it and R right of
// it, the field strip glyphs wide and H lines high. The glyph next to the
// field on either side stays blank.
type room struct{ L, R, strip, H int }

// side says whether glyph u, the field's first 0, is at the sides.
func (r room) side(u int) bool { return (u >= -r.L && u < -1) || (u > r.strip && u < r.strip+r.R) }

// figure is a level's figure placed in a room: its font rows, a font pixel
// sx glyphs across and sy lines down (sx 0: no room, not drawn), the left
// one from glyph lu and the right one from ru, both from line y0.
type figure struct {
	rows       [5]string
	sx, sy     int
	lu, ru, y0 int
}

// place places level's figure in r: sy as many lines as five fit in the
// field's height short of four of its rows over them and four under them,
// sx as many glyphs as fit at the narrower side with a glyph to spare, at
// most sy; a glyph or more from the field, the same on both sides, in the
// middle of the field's height.
func (r room) place(level int) figure {
	pad := 4 * r.H / sceneH
	f := figure{rows: font[level-1], sy: (r.H - 2*pad) / 5}
	f.sx = min(f.sy, (min(r.L, r.R)-1)/len(f.rows[0]))
	if f.sx <= 0 || f.sy <= 0 {
		return figure{}
	}
	w := len(f.rows[0]) * f.sx
	gap := max(1, (min(r.L, r.R)-w)/2)
	f.lu, f.ru, f.y0 = -gap-w, r.strip+gap, (r.H-5*f.sy)/2
	return f
}

// at says whether glyph u of line l is on f's figures, and on which line
// of them.
func (f figure) at(u, l int) (on bool, y int) {
	if y = l - f.y0; f.sx == 0 || y < 0 || y >= 5*f.sy {
		return false, 0
	}
	w := len(f.rows[0]) * f.sx
	for _, x0 := range []int{f.lu, f.ru} {
		if x := u - x0; x >= 0 && x < w {
			return f.rows[y/f.sy][x/f.sx] == '#', y
		}
	}
	return false, 0
}

// banner is the level at the field's sides as it goes: the level drawn,
// none (0) before the first game, and the sweep bringing it in.
type banner struct {
	lv, from int           // the level drawn, the one the sweep takes away (0 none)
	by       int           // the piece whose colour the sweep's band is in
	at       time.Duration // time into the sweep: sweepTime, none going
	beat     time.Duration // time into the figures' heartbeat
}

// sweepTo sweeps level in over the one drawn, the band in piece by's
// colour: a new game, or the level up.
func (b *banner) sweepTo(level, by int) { b.from, b.lv, b.by, b.at = b.lv, level, by, 0 }

// show draws level at once: a game saved, taken up again.
func (b *banner) show(level int) { b.from, b.lv, b.at = 0, level, sweepTime }

// step steps the banner on dt with g: the sweep and the heartbeat go on,
// and a level gone up sweeps in once the clear that took it there is over
// (the user's choice: the game goes on under it).
func (b *banner) step(g *game.Game, dt time.Duration) {
	b.at = min(b.at+dt, sweepTime)
	b.beat = (b.beat + dt) % beatTime
	if len(g.Going) == 0 && g.Level != b.lv {
		b.sweepTo(g.Level, g.Last)
	}
}

// band is the sweep's band in r: lines in the piece's colour, then a trail
// of three steps of it fading to the ground, step lines each.
func (r room) band() (whole, step int) { return max(2, r.H/5), max(1, r.H/10) }

// The band's trails: every piece's colour a quarter, a half and three
// quarters of the way to the ground, as the clear's white fades.
func init() {
	for p := range 7 {
		for f := range 3 {
			inks = append(inks, toGround(inks[inkPiece+uint8(p)], f+1))
		}
	}
}

// toGround is c f quarters of the way to the ground.
func toGround(c terminu.Color, f int) terminu.Color {
	out := terminu.Color{Set: true}
	for i, g := range inks[inkGround].RGB {
		out.RGB[i] = uint8(math.Round(float64(c.RGB[i]) + float64(int(g)-int(c.RGB[i]))*float64(f)/4))
	}
	return out
}

// ink is the ink of glyph u of the field's line l in r with g, at the
// sides: while the sweep goes, the band over the whole of both sides, the
// level drawn under it, the one going over it, full; else the level drawn,
// filled from its foot as its rows are cleared (full while the level is
// up and the clear not over), the last level all lit, what is lit beating
// while beating.
func (b *banner) ink(g *game.Game, r room, u, l int, beating bool) uint8 {
	if !r.side(u) {
		return inkNone
	}
	if b.at < sweepTime {
		whole, step := r.band()
		front := int(float64(r.H+whole+3*step) * float64(b.at) / float64(sweepTime))
		switch d := front - (r.H - l); {
		case d < 0:
			return face(r, b.from, u, l, 1, 0)
		case d < whole:
			return inkPiece + uint8(b.by)
		case d < whole+3*step:
			return inkTrail + uint8(3*b.by+(d-whole)/step)
		}
	}
	fill := 1.0
	if b.lv == g.Level {
		fill = float64(g.Lines%game.LinesPerLevel) / game.LinesPerLevel
	}
	var lit uint8
	if beating {
		lit = heartbeat(b.beat)
	}
	return face(r, b.lv, u, l, fill, lit)
}

// face is the ink of glyph u of line l of level's figures in r: off them,
// none; on them, lit from their foot to fill (0 to 1), the ground over
// that; the last level's all lit. Lit is Lavender, the last level's gold,
// or as deep in its heartbeat as lit says: half way to the ground, or the
// ground.
func face(r room, level, u, l int, fill float64, lit uint8) uint8 {
	if level == 0 {
		return inkNone
	}
	f := r.place(level)
	on, y := f.at(u, l)
	switch {
	case !on:
		return inkNone
	case level < game.MaxLevel && 5*f.sy-y > int(math.Round(fill*float64(5*f.sy))):
		return inkGround
	}
	c := inkLevel
	if level == game.MaxLevel {
		c = inkGold
	}
	return [...]uint8{c, c + 1, inkGround}[lit]
}

// heartbeat is how deep what is lit of a figure is at t into a heartbeat:
// 0 its colour, 1 half way to the ground, 2 the ground.
func heartbeat(t time.Duration) uint8 {
	pulse := func(at time.Duration, top float64) float64 {
		switch d := t - at; {
		case d < 0, d >= rise+fall:
			return 0
		case d < rise:
			return top * float64(d) / float64(rise)
		default:
			return top * float64(rise+fall-d) / float64(fall)
		}
	}
	return uint8(math.Round(2 * max(pulse(0, 1), pulse(dub, 0.6))))
}

// sides are the level at the field's sides, once a game has started and
// where a figure has room: ink is glyph u's of the field's line l.
type sides struct {
	on  bool
	ink func(u, l int) uint8
}

// sides are the level by the field, every field pixel k × k glyphs, with
// left columns left of it and right glyphs right of it: the figures
// beating while the game is played and nothing is over it.
func (t *Tetru) sides(k, left, right int) sides {
	r := room{L: left / 2, R: right, strip: fieldW * k, H: sceneH * k}
	if t.g == nil || t.lv.lv == 0 || r.place(1).sx == 0 {
		return sides{}
	}
	beating := t.playing() && !t.App.Covered()
	return sides{on: true, ink: func(u, l int) uint8 { return t.lv.ink(t.g, r, u, l, beating) }}
}

// lay is at, the glyphs of the field's line l, glyph u at u+lead, with the
// sides' glyphs where at has none: the field, and what goes out over the
// sides, are over them.
func (sd sides) lay(at []uint8, lead, l int) []uint8 {
	if !sd.on {
		return at
	}
	out := slices.Clone(at)
	for i, c := range out {
		if c == inkNone {
			out[i] = sd.ink(i-lead, l)
		}
	}
	return out
}
