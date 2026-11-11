// Package game is tetru's game: the field, the pieces falling into it, the
// rows they fill going, the levels and the score. It knows nothing of a
// terminal: the ui steps it a frame at a time and draws what it holds.
//
// The pieces, their turns and the bag they come from are locku's tetromino
// saver's (internal/saver/tetromino.go), played by a player instead of by
// itself.
package game

import (
	"math/rand/v2"
	"slices"
	"time"
)

const (
	// Cols and Rows are the field, in blocks.
	Cols, Rows = 12, 24
	// Queue is how many pieces to come are shown, a box each (user,
	// 2026-11-11: Next 1 and Next 2).
	Queue = 2
	// MaxLevel is the last level: the speed goes no higher after it (user,
	// 2026-11-11: the ten levels are speeds).
	MaxLevel = 10
	// LinesPerLevel are the rows to clear for the next level.
	LinesPerLevel = 10
	// LockDelay is how long a piece lies on the stack before it is set;
	// moving or turning it starts the wait again, LockResets times at most.
	LockDelay  = 500 * time.Millisecond
	LockResets = 15
	// Flash and Wipe are the clear, as locku's: the rows going white, then
	// going from the middle out to both ends.
	Flash = 280 * time.Millisecond
	Wipe  = 400 * time.Millisecond
	// Curtain is the end: the pieces going grey from the top.
	Curtain = 5 * time.Second
)

// The pieces in their boxes as they come in: I, O, T, S, Z, J, L.
var art = [][]string{
	{"....", "####", "....", "...."},
	{"##", "##"},
	{".#.", "###", "..."},
	{".##", "##.", "..."},
	{"##.", ".##", "..."},
	{"#..", "###", "..."},
	{"..#", "###", "..."},
}

// Pieces is how many kinds of piece there are.
var Pieces = len(art)

// I is the long piece, the one that kicks two blocks.
const I = 0

// turns are each piece's blocks in its box, a turn clockwise after the
// other till it is back as it came; the square has the one.
var turns = func() [][][][2]int {
	out := make([][][][2]int, len(art))
	for p, rows := range art {
		var blocks [][2]int
		for y, row := range rows {
			for x := range row {
				if row[x] == '#' {
					blocks = append(blocks, [2]int{x, y})
				}
			}
		}
		n := len(rows)
		for {
			out[p] = append(out[p], blocks)
			next := make([][2]int, len(blocks))
			for i, b := range blocks {
				next[i] = [2]int{n - 1 - b[1], b[0]}
			}
			if sameBlocks(next, out[p][0]) {
				break
			}
			blocks = next
		}
	}
	return out
}()

// sameBlocks says whether a and b are the same blocks, in any order.
func sameBlocks(a, b [][2]int) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return len(a) == len(b)
}

// Shape is piece p's blocks as it comes in, for a box of the queue.
func Shape(p int) [][2]int { return turns[p][0] }

// intervals are the time a piece takes to fall a row at each level.
var intervals = [MaxLevel]time.Duration{1000, 800, 650, 520, 420, 330, 260, 200, 150, 100}

// Interval is the time a piece takes to fall a row at level; the last
// level's past it.
func Interval(level int) time.Duration {
	return intervals[min(max(level, 1), MaxLevel)-1] * time.Millisecond
}

// clearScores are the points for clearing 1, 2, 3 or 4 rows at once, times
// the level.
var clearScores = [5]int{0, 100, 300, 500, 800}

// Spot is where a piece is: its turn and its box's top-left, in blocks;
// a y under 0 is over the field.
type Spot struct{ Turn, X, Y int }

// Game is one game in progress, or ended.
type Game struct {
	rng   *rand.Rand
	cells []uint8 // the stack: 1 + the piece a block is of, 0 none
	bag   []int   // the pieces still to come from this bag

	Cur  int   // the piece falling
	At   Spot  // where it is
	Next []int // the pieces to come, the next first

	Level, Lines, Score int

	// Going are the rows full and going, top first, while the clear is
	// under way.
	Going []int
	// Over is the game ended: the stack reached the top, or the player
	// ended it.
	Over bool

	fall   time.Duration // time since the piece last fell a row
	lock   time.Duration // time the piece has lain on the stack
	resets int           // times moving or turning started the lock again
	fx     time.Duration // time into the clear, or into the end
}

// New is a game from its first piece. The same seed is the same game.
func New(seed uint64) *Game {
	g := &Game{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), cells: make([]uint8, Cols*Rows), Level: 1}
	for range Queue {
		g.Next = append(g.Next, g.deal())
	}
	g.come()
	return g
}

// deal is the next piece from the bag: seven to a bag, one of each.
func (g *Game) deal() int {
	if len(g.bag) == 0 {
		g.bag = g.rng.Perm(len(art))
	}
	p := g.bag[0]
	g.bag = g.bag[1:]
	return p
}

// come brings the next piece in, in the middle, its top row the field's
// top row. One that has no room there is the end.
func (g *Game) come() {
	g.Cur, g.Next = g.Next[0], append(g.Next[1:], g.deal())
	blocks := turns[g.Cur][0]
	lo, hi, top := blocks[0][0], blocks[0][0], blocks[0][1]
	for _, b := range blocks {
		lo, hi, top = min(lo, b[0]), max(hi, b[0]), min(top, b[1])
	}
	g.At = Spot{0, (Cols-(hi-lo+1))/2 - lo, -top}
	g.fall, g.lock, g.resets = 0, 0, 0
	if !g.fits(g.At) {
		g.end()
	}
}

// Put sets block x, y of the stack to piece p's, a stack to start from (a
// test's).
func (g *Game) Put(x, y, p int) { g.cells[y*Cols+x] = uint8(1 + p) }

// Cell is the piece block x, y of the stack is of, 1 + its number; 0 none.
func (g *Game) Cell(x, y int) uint8 { return g.cells[y*Cols+x] }

// Blocks are the falling piece's blocks on the field (a y under 0 is over
// it).
func (g *Game) Blocks() [][2]int {
	var out [][2]int
	for _, b := range turns[g.Cur][g.At.Turn] {
		out = append(out, [2]int{g.At.X + b[0], g.At.Y + b[1]})
	}
	return out
}

// Live says whether the piece is falling: not while rows go, nor after the
// end.
func (g *Game) Live() bool { return !g.Over && len(g.Going) == 0 }

// open says whether block x, y can take a piece's block: in the field and
// not the stack's, or over it.
func (g *Game) open(x, y int) bool {
	if x < 0 || x >= Cols || y >= Rows {
		return false
	}
	return y < 0 || g.cells[y*Cols+x] == 0
}

// fits says whether the piece can be at s.
func (g *Game) fits(s Spot) bool {
	for _, b := range turns[g.Cur][s.Turn] {
		if !g.open(s.X+b[0], s.Y+b[1]) {
			return false
		}
	}
	return true
}

// falls says whether the piece has room a row down.
func (g *Game) falls() bool { return g.fits(Spot{g.At.Turn, g.At.X, g.At.Y + 1}) }

// moved starts the lock's wait again after the piece moved or turned on
// the stack, as long as it has resets left.
func (g *Game) moved() {
	if g.lock > 0 && g.resets < LockResets {
		g.lock = 0
		g.resets++
	}
}

// Move moves the piece a block across, dx -1 left and 1 right, if it has
// room; it reports whether it moved.
func (g *Game) Move(dx int) bool {
	if !g.Live() {
		return false
	}
	s := Spot{g.At.Turn, g.At.X + dx, g.At.Y}
	if !g.fits(s) {
		return false
	}
	g.At = s
	g.moved()
	return true
}

// Rotate turns the piece in its box, dir 1 clockwise and -1 the other way.
// With no room as it is, it tries a block left, right and up, the long one
// two left and right as well; it reports whether it turned.
func (g *Game) Rotate(dir int) bool {
	if !g.Live() {
		return false
	}
	n := len(turns[g.Cur])
	t := (g.At.Turn + dir + n) % n
	kicks := [][2]int{{0, 0}, {-1, 0}, {1, 0}, {0, -1}}
	if g.Cur == I {
		kicks = append(kicks, [2]int{-2, 0}, [2]int{2, 0})
	}
	for _, k := range kicks {
		s := Spot{t, g.At.X + k[0], g.At.Y + k[1]}
		if g.fits(s) {
			g.At = s
			g.moved()
			return true
		}
	}
	return false
}

// Down moves the piece a row down, a point for it; it reports whether it
// had room.
func (g *Game) Down() bool {
	if !g.Live() || !g.falls() {
		return false
	}
	g.At.Y++
	g.Score++
	g.fall = 0
	return true
}

// Drop drops the piece as far as it goes and sets it there, two points a
// row; it reports how many rows it fell.
func (g *Game) Drop() int {
	if !g.Live() {
		return 0
	}
	n := 0
	for g.falls() {
		g.At.Y++
		n++
	}
	g.Score += 2 * n
	g.set()
	return n
}

// set puts the piece in the stack: a block over the field is the end; the
// rows it fills start going; else the next piece comes in.
func (g *Game) set() {
	over := false
	for _, b := range g.Blocks() {
		if b[1] < 0 {
			over = true
			continue
		}
		g.cells[b[1]*Cols+b[0]] = uint8(1 + g.Cur)
	}
	if over {
		g.end()
		return
	}
	if g.Going = g.full(); len(g.Going) > 0 {
		g.fx = 0
		return
	}
	g.come()
}

// full are the rows with every block taken, the top first.
func (g *Game) full() []int {
	var out []int
	for y := 0; y < Rows; y++ {
		whole := true
		for x := 0; x < Cols && whole; x++ {
			whole = g.cells[y*Cols+x] != 0
		}
		if whole {
			out = append(out, y)
		}
	}
	return out
}

// clear takes the rows that went out, every row over them down as far;
// scores them at the level they were cleared at, and the level goes up a
// level every ten rows, up to the last; and brings the next piece in.
func (g *Game) clear() {
	out := make([]uint8, len(g.cells))
	y := Rows - 1
	for from := Rows - 1; from >= 0; from-- {
		if slices.Contains(g.Going, from) {
			continue
		}
		copy(out[y*Cols:(y+1)*Cols], g.cells[from*Cols:(from+1)*Cols])
		y--
	}
	g.cells = out
	g.Lines += len(g.Going)
	g.Score += clearScores[min(len(g.Going), 4)] * g.Level
	g.Level = min(MaxLevel, 1+g.Lines/LinesPerLevel)
	g.Going, g.fx = nil, 0
	g.come()
}

// End ends the game where it is: the player ended it.
func (g *Game) End() {
	if !g.Over {
		g.end()
	}
}

func (g *Game) end() { g.Over, g.Going, g.fx = true, nil, 0 }

// Advance steps the game on by dt: the end's curtain, or the clear, or the
// piece falling at the level's speed and, once it lies on the stack long
// enough, set.
func (g *Game) Advance(dt time.Duration) {
	switch {
	case g.Over:
		g.fx = min(g.fx+dt, Curtain)
	case len(g.Going) > 0:
		if g.fx += dt; g.fx >= Flash+Wipe {
			g.clear()
		}
	case g.falls():
		g.lock = 0
		g.fall += dt
		for every := Interval(g.Level); g.fall >= every && g.falls(); g.fall -= every {
			g.At.Y++
		}
		if !g.falls() {
			g.fall = 0
		}
	default:
		g.fall = 0
		if g.lock += dt; g.lock >= LockDelay {
			g.set()
		}
	}
}

// Flashing says whether the rows going are still white, not yet going
// out.
func (g *Game) Flashing() bool { return len(g.Going) > 0 && g.fx < Flash }

// Wiped says whether block x of a row going is gone by now: the row goes
// from the middle of the field out, the blocks the same distance from the
// ends at once, after the flash.
func (g *Game) Wiped(x int) bool {
	k := g.fx - Flash
	if len(g.Going) == 0 || k < 0 {
		return false
	}
	rings := (Cols-1)/2 + 1
	gone := int((int64(rings)*int64(k) + int64(Wipe) - 1) / int64(Wipe))
	return min(x, Cols-1-x) >= rings-gone
}

// CurtainDone is how far the end has gone, 0 to 1: the share of the field,
// from the top, whose pieces are grey.
func (g *Game) CurtainDone() float64 {
	if !g.Over {
		return 0
	}
	return float64(g.fx) / float64(Curtain)
}

// Ended says whether the end's curtain is all down: END is lettered.
func (g *Game) Ended() bool { return g.Over && g.fx >= Curtain }
