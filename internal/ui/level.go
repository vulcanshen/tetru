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
// where the terminal is narrow). 1 to 9, then 10 to 15 where the sides
// have room for two figures, A to F where they have not (user, 2026-11-13:
// fifteen levels, wide 10 to 15, narrow A to F; A to F everywhere was a
// misreading), four of the field's rows clear of its top and its foot
// (user, 2026-11-13: too high, two rows clear). It lights up from its foot
// as the level's rows are cleared, Lavender over the ground, and over what
// is lit waves go up from its foot, one after another, in the colour half
// way to the ground (user, 2026-11-13: the waves, not a breath; the breath,
// a heartbeat a second, too often, a blinking; not white, too much);
// the last level, 15 or F, is all lit, gold (user, 2026-11-13). As the level
// goes up, a band with a fading trail goes up over the whole of both sides
// in the colour of the piece that cleared the level's last rows (user,
// 2026-11-13: colour, the piece that fell), the next level's figure under
// it (user, 2026-11-13: the effect bigger).
const (
	// sweepTime is how long the band takes from the field's foot to past
	// its top.
	sweepTime = 800 * time.Millisecond
	// waveTime is a wave of the figures: from under their foot to past
	// their top, evenly, the next coming in as it goes out.
	waveTime = 2 * time.Second
)

// font are the levels' figures, 3 × 5, # on: 1 to 9, then A to F, 10 to
// 15 where two figures have no room.
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

// one and zero are the figures of 10 to 15 besides font's: a 1 two pixels
// wide, as their first figure and for 11 their second, so that two figures
// have room from 78 columns, and a 0.
var (
	one  = [5]string{".#", "##", ".#", ".#", ".#"}
	zero = [5]string{"###", "#.#", "#.#", "#.#", "###"}
)

// two is level, 10 to 15, in two figures a pixel apart.
func two(level int) (rows [5]string) {
	second := one
	switch d := level - 10; d {
	case 0:
		second = zero
	case 1:
	default:
		second = font[d-1]
	}
	for i := range rows {
		rows[i] = one[i] + "." + second[i]
	}
	return rows
}

// place places level's figure in r: sy as many lines as five fit in the
// field's height short of four of its rows over them and four under them,
// sx as many glyphs as fit at the narrower side with a glyph to spare, at
// most sy; a glyph or more from the field, the same on both sides, in the
// middle of the field's height. Past 9, two figures where they have room,
// a glyph a pixel at least, else its letter.
func (r room) place(level int) figure {
	pad := 4 * r.H / sceneH
	f := figure{sy: (r.H - 2*pad) / 5}
	shapes := [][5]string{font[level-1]}
	if level >= 10 {
		shapes = [][5]string{two(level), font[level-1]}
	}
	for _, rows := range shapes {
		if f.rows, f.sx = rows, min(f.sy, (min(r.L, r.R)-1)/len(rows[0])); f.sx > 0 {
			break
		}
	}
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
	wave     time.Duration // time into the figures' wave
}

// sweepTo sweeps level in over the one drawn, the band in piece by's
// colour: a new game, or the level up.
func (b *banner) sweepTo(level, by int) { b.from, b.lv, b.by, b.at = b.lv, level, by, 0 }

// show draws level at once: a game saved, taken up again.
func (b *banner) show(level int) { b.from, b.lv, b.at = 0, level, sweepTime }

// step steps the banner on dt with g: the sweep and the waves go on,
// and a level gone up sweeps in once the clear that took it there is over
// (the user's choice: the game goes on under it).
func (b *banner) step(g *game.Game, dt time.Duration) {
	b.at = min(b.at+dt, sweepTime)
	b.wave = (b.wave + dt) % waveTime
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
// up and the clear not over), the last level all lit, a wave over what is
// lit while waving.
func (b *banner) ink(g *game.Game, r room, u, l int, waving bool) uint8 {
	if !r.side(u) {
		return inkNone
	}
	if b.at < sweepTime {
		whole, step := r.band()
		front := int(float64(r.H+whole+3*step) * float64(b.at) / float64(sweepTime))
		switch d := front - (r.H - l); {
		case d < 0:
			return face(r, b.from, u, l, 1, -1)
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
	wave := -1.0
	if waving {
		wave = float64(b.wave) / float64(waveTime)
	}
	return face(r, b.lv, u, l, fill, wave)
}

// face is the ink of glyph u of line l of level's figures in r: off them,
// none; on them, lit from their foot to fill (0 to 1), the ground over
// that; the last level's all lit. Lit is Lavender, the last level's gold;
// wave (0 to 1, none under 0) as far on its way up, a font pixel's lines
// of it are half way to the ground.
func face(r room, level, u, l int, fill, wave float64) uint8 {
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
	if wave >= 0 {
		// The wave's top, lines up from under the foot, to past the top.
		front := int(wave * float64(6*f.sy))
		if up := 5*f.sy - y; up <= front && up > front-f.sy {
			return c + 1
		}
	}
	return c
}

// sides are the level at the field's sides, once a game has started and
// where a figure has room: ink is glyph u's of the field's line l.
type sides struct {
	on  bool
	ink func(u, l int) uint8
}

// sides are the level by the field, every field pixel k × k glyphs, with
// left columns left of it and right glyphs right of it: the waves going
// while the game is played and nothing is over it.
func (t *Tetru) sides(k, left, right int) sides {
	r := room{L: left / 2, R: right, strip: fieldW * k, H: sceneH * k}
	if t.g == nil || t.lv.lv == 0 || r.place(1).sx == 0 {
		return sides{}
	}
	waving := t.playing() && !t.App.Covered()
	return sides{on: true, ink: func(u, l int) uint8 { return t.lv.ink(t.g, r, u, l, waving) }}
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
