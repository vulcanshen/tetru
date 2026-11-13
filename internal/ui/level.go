package ui

import (
	"math"
	"slices"
	"time"

	"github.com/vulcanshen/tetru/internal/game"
)

// The level is drawn big at both sides of the field (user, 2026-11-13: what
// the blank there is for), one figure, in the field's square glyph: a 3 × 5
// font, a pixel of it s × s glyphs. It fills from its foot as the level's
// rows are cleared, Lavender over the ground; as the level goes up, a white
// line goes up over it, the next level's figure under the line; the last
// level is A, lit and beating like a heart (the user's choices, having
// seen it drawn).
const (
	// sweepTime is how long the white line takes from the figures' foot
	// to past their top.
	sweepTime = 600 * time.Millisecond
	// beatTime is a heartbeat of the A: a beat to white, then dub after it
	// a beat a little over half as bright, each up in rise and down in fall.
	beatTime = 1000 * time.Millisecond
	dub      = 280 * time.Millisecond
	rise     = 80 * time.Millisecond
	fall     = 160 * time.Millisecond
)

// font are the figures, 3 × 5, # on: 1 to 9, then A for the last level
// (user, 2026-11-13: one character, so the tenth is A).
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
}

// banner is the level at the field's sides as it goes: the level drawn,
// none (0) before the first game, and the sweep bringing it in.
type banner struct {
	lv, from int           // the level drawn, the one the sweep takes away (0 none)
	at       time.Duration // time into the sweep: sweepTime, none going
	beat     time.Duration // time into the A's heartbeat
}

// sweepTo sweeps level in over the one drawn: a new game, or the level up.
func (b *banner) sweepTo(level int) { b.from, b.lv, b.at = b.lv, level, 0 }

// show draws level at once: a game saved, taken up again.
func (b *banner) show(level int) { b.from, b.lv, b.at = 0, level, sweepTime }

// step steps the banner on dt with g: the sweep and the heartbeat go on,
// and a level gone up sweeps in once the clear that took it there is over
// (the user's choice: the game goes on under it).
func (b *banner) step(g *game.Game, dt time.Duration) {
	b.at = min(b.at+dt, sweepTime)
	b.beat = (b.beat + dt) % beatTime
	if len(g.Going) == 0 && g.Level != b.lv {
		b.sweepTo(g.Level)
	}
}

// ink is the ink of glyph x, line y of the figures at scale s with g:
// under the white line going up, the level drawn; over it, the one going,
// full. The level drawn fills from its foot as its rows are cleared, full
// while the level is up and the clear not over; the A is full, beating
// while beating.
func (b *banner) ink(g *game.Game, x, y, s int, beating bool) uint8 {
	h := 5 * s
	up := h - y // lines from the foot, the lowest 1
	if b.at < sweepTime {
		band := max(1, s/2)
		top := int(float64(h+band) * float64(b.at) / float64(sweepTime))
		switch {
		case up > top:
			return face(b.from, x, y, s, 1, 0)
		case up > top-band:
			return inkWhite
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
	return face(b.lv, x, y, s, fill, lit)
}

// face is the ink of glyph x, line y of level's figure at scale s: off the
// figure, none; on it, Lavender from its foot to fill (0 to 1), the ground
// over that; the last level's all lit, lit up to white.
func face(level, x, y, s int, fill float64, lit uint8) uint8 {
	switch {
	case level == 0 || font[level-1][y/s][x/s] != '#':
		return inkNone
	case level == game.MaxLevel:
		return inkLevel + lit
	case 5*s-y <= int(math.Round(fill*float64(5*s))):
		return inkLevel
	}
	return inkGround
}

// heartbeat is how lit the A is at t into a heartbeat, 0 (Lavender) to 3
// (white).
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
	return uint8(math.Round(3 * max(pulse(0, 1), pulse(dub, 0.6))))
}

// sides is where the level's figures go at the field's sides: a figure
// pixel s × s glyphs, none (0) without the room; the left figure's first
// glyph lu and the right one's ru, the field's first glyph 0; their first
// line y0, the field's first line 0; ink is glyph x, line y of a figure.
type sides struct {
	s, lu, ru, y0 int
	ink           func(x, y int) uint8
}

// place places the figures by the field, every field pixel k × k glyphs,
// with left columns left of it and right glyphs right of it: as big as
// both sides have the room for, a glyph or more from the field, the same
// on both sides, and no higher than the field, in the middle of its height.
func place(k, left, right int) sides {
	room := min(left/2, right) // whole glyphs at either side
	var sd sides
	for s := 1; 3*s+1 <= room && 5*s <= sceneH*k; s++ {
		sd.s = s
	}
	if sd.s == 0 {
		return sd
	}
	gap := max(1, (room-3*sd.s)/2)
	sd.lu, sd.ru = -gap-3*sd.s, fieldW*k+gap
	sd.y0 = (sceneH*k - 5*sd.s) / 2
	return sd
}

// sides are the level's figures by the field (place), once a game has
// started: the A beating while it is played and nothing is over it.
func (t *Tetru) sides(k, left, right int) sides {
	if t.g == nil || t.lv.lv == 0 {
		return sides{}
	}
	sd := place(k, left, right)
	beating := t.playing() && !t.App.Covered()
	sd.ink = func(x, y int) uint8 { return t.lv.ink(t.g, x, y, sd.s, beating) }
	return sd
}

// lay is at, the glyphs of the field's line l, glyph u at u+lead, with the
// figures' glyphs where at has none: the field, and what goes out over the
// sides, are over them.
func (sd sides) lay(at []uint8, lead, l int) []uint8 {
	if sd.s == 0 || l < sd.y0 || l >= sd.y0+5*sd.s {
		return at
	}
	out := slices.Clone(at)
	for _, x0 := range []int{sd.lu, sd.ru} {
		for x := range 3 * sd.s {
			if i := x0 + x + lead; i >= 0 && i < len(out) && out[i] == inkNone {
				out[i] = sd.ink(x, l-sd.y0)
			}
		}
	}
	return out
}
