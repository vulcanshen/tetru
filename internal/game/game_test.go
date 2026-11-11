package game

import (
	"testing"
	"time"
)

// with is a game whose falling piece is p, at its spawn, on an empty field.
func with(p int) *Game {
	g := New(1)
	g.Next[0] = p
	g.come()
	return g
}

// fill takes every block of row y but those at the xs.
func (g *Game) fill(y int, holes ...int) {
	for x := 0; x < Cols; x++ {
		g.cells[y*Cols+x] = 1
	}
	for _, x := range holes {
		g.cells[y*Cols+x] = 0
	}
}

func TestPiecesComeSevenToABag(t *testing.T) {
	g := New(7)
	g.bag = nil // a fresh bag: New dealt the queue from the first
	seen := map[int]int{}
	for range 7 * 4 {
		seen[g.deal()]++
	}
	for p := range Pieces {
		if seen[p] != 4 {
			t.Fatalf("piece %d came %d times in four bags", p, seen[p])
		}
	}
}

func TestSameSeedSameGame(t *testing.T) {
	a, b := New(42), New(42)
	for range 50 {
		if a.Cur != b.Cur {
			t.Fatal("two games of one seed differ")
		}
		a.Drop()
		b.Drop()
		a.Advance(ClearTime(4))
		b.Advance(ClearTime(4))
	}
}

func TestPieceComesInAtTheTopInTheMiddle(t *testing.T) {
	for p := range Pieces {
		g := with(p)
		top, lo, hi := Rows, Cols, -1
		for _, b := range g.Blocks() {
			top, lo, hi = min(top, b[1]), min(lo, b[0]), max(hi, b[0])
		}
		if top != 0 {
			t.Errorf("piece %d comes in with its top on row %d, not 0", p, top)
		}
		if left, right := lo, Cols-1-hi; left-right < -1 || left-right > 1 {
			t.Errorf("piece %d comes in at columns %d–%d, off the middle", p, lo, hi)
		}
	}
}

func TestMoveStopsAtTheWalls(t *testing.T) {
	g := with(1) // the square
	n := 0
	for g.Move(-1) {
		n++
	}
	if n != (Cols-2)/2 {
		t.Fatalf("the square moved %d left from the middle, not %d", n, (Cols-2)/2)
	}
	for _, b := range g.Blocks() {
		if b[0] < 0 {
			t.Fatal("the square is past the left wall")
		}
	}
}

func TestRotateTurnsAndComesBack(t *testing.T) {
	g := with(2) // T
	start := g.At
	for range 4 {
		if !g.Rotate(1) {
			t.Fatal("T could not turn in the open")
		}
	}
	if g.At != start {
		t.Fatalf("four turns clockwise left T at %+v, not %+v", g.At, start)
	}
	g.Rotate(1)
	g.Rotate(-1)
	if g.At != start {
		t.Fatalf("a turn and back left T at %+v", g.At)
	}
}

func TestRotateKicksOffTheWall(t *testing.T) {
	g := with(I)
	g.Rotate(1) // upright
	for g.Move(1) {
	}
	if !g.Rotate(1) {
		t.Fatal("the long piece could not turn flat against the right wall")
	}
	for _, b := range g.Blocks() {
		if b[0] >= Cols {
			t.Fatal("the long piece turned through the wall")
		}
	}
}

func TestDownScoresARow(t *testing.T) {
	g := with(1)
	y := g.At.Y
	if !g.Down() || g.At.Y != y+1 || g.Score != 1 {
		t.Fatalf("down: at %d, score %d", g.At.Y, g.Score)
	}
}

func TestDropSetsThePieceAndScoresTwoARow(t *testing.T) {
	g := with(1) // the square, two high: all but two rows to fall
	if n := g.Drop(); n != Rows-2 {
		t.Fatalf("the square fell %d rows, not %d", n, Rows-2)
	}
	if g.Score != 2*(Rows-2) {
		t.Fatalf("score %d, not %d", g.Score, 2*(Rows-2))
	}
	if m := (Cols - 2) / 2; g.Cell(m, Rows-1) == 0 || g.Cell(m+1, Rows-2) == 0 {
		t.Fatal("the square is not set at the foot")
	}
}

func TestGravityAtTheLevelsSpeed(t *testing.T) {
	g := with(1)
	y := g.At.Y
	g.Advance(Interval(1) - time.Millisecond)
	if g.At.Y != y {
		t.Fatal("fell before its time")
	}
	g.Advance(time.Millisecond)
	if g.At.Y != y+1 {
		t.Fatal("did not fall a row in a second at level 1")
	}
}

func TestLockWaitsAndMovingRestartsIt(t *testing.T) {
	g := with(1)
	for g.Down() {
	}
	cur := g.Cur
	g.Advance(LockDelay - 100*time.Millisecond)
	g.Move(-1)
	g.Advance(LockDelay - 100*time.Millisecond)
	x := (Cols-2)/2 - 1 // the square's left column, a column left of the middle
	if g.Cur != cur || g.Cell(x, Rows-1) != 0 {
		t.Fatal("the piece was set though moving restarted the wait")
	}
	g.Advance(100 * time.Millisecond)
	if g.Cell(x, Rows-1) == 0 {
		t.Fatal("the piece was not set after lying still for the delay")
	}
}

func TestLockResetsRunOut(t *testing.T) {
	g := with(1)
	for g.Down() {
	}
	for i := range LockResets + 5 {
		g.Advance(LockDelay - time.Millisecond)
		if !g.Live() || g.At.Y < Rows-2 {
			break
		}
		g.Move(1 - 2*(i%2))
	}
	g.Advance(LockDelay)
	if g.Cell(4, Rows-1) == 0 && g.Cell(3, Rows-1) == 0 && g.Cell(5, Rows-1) == 0 {
		t.Fatal("moving kept the piece from being set for ever")
	}
}

func TestClearScoresAtTheLevelAndGoesUp(t *testing.T) {
	g := with(I)
	g.Level, g.Lines = 3, 29
	for y := Rows - 4; y < Rows; y++ {
		g.fill(y, Cols-1)
	}
	g.Rotate(1) // upright, in column 6 of its box
	for g.Move(1) {
	}
	score := g.Score
	fell := g.Drop()
	if len(g.Going) != 4 || g.Going[0] != Rows-1 || g.Going[3] != Rows-4 {
		t.Fatalf("rows going, the bottom first: %v", g.Going)
	}
	g.Advance(ClearTime(4))
	if got := g.Score - score; got != 800*3+2*fell {
		t.Fatalf("four rows at level 3 scored %d", got)
	}
	if g.Lines != 33 || g.Level != 4 {
		t.Fatalf("lines %d level %d after 33 rows", g.Lines, g.Level)
	}
	for x := 0; x < Cols; x++ {
		if g.Cell(x, Rows-1) != 0 {
			t.Fatal("a cleared row is still there")
		}
	}
}

func TestClearSlidesThenTheStackComesDown(t *testing.T) {
	g := with(I)
	g.fill(Rows-1, Cols-1)
	g.fill(Rows-3, Cols-1)
	g.Put(0, Rows-2, 1) // a block between the rows going
	g.Put(0, Rows-4, 1) // and one over them
	g.Rotate(1)         // upright, in the last column: rows Rows-1 and Rows-3 fill
	for g.Move(1) {
	}
	next := g.Next[0]
	lines := g.Lines
	g.Drop()
	if len(g.Going) != 2 || g.Going[0] != Rows-1 || g.Going[1] != Rows-3 {
		t.Fatalf("rows going, the bottom first: %v", g.Going)
	}
	if g.Cell(0, Rows-1) != 0 || g.Cell(0, Rows-3) != 0 || g.Lines != lines+2 {
		t.Fatal("the rows going are still in the stack, or not counted at once")
	}
	if g.Cur != next || g.Live() || g.Move(-1) || g.Rotate(1) || g.Down() || g.Drop() != 0 {
		t.Fatal("the next piece is not in, or it can be played while the rows go")
	}
	at := g.At
	g.Advance(Flash)
	if g.Slid(0) != 0 {
		t.Fatal("the row slides before it is white a while")
	}
	g.Advance(Stagger)
	if g.Slid(0) <= 0 || g.Slid(1) != 0 {
		t.Fatal("the bottom row does not slide first")
	}
	g.Advance(Slide)
	if g.Slid(0) != 1 || g.Slid(1) != 1 || g.Cell(0, Rows-2) == 0 {
		t.Fatal("the rows have not slid out, or the stack came down first")
	}
	g.Advance(Settle)
	if g.Cell(0, Rows-1) == 0 || g.Cell(0, Rows-2) != 0 || g.Cell(0, Rows-3) == 0 || g.Live() {
		t.Fatal("the stack did not come down a row")
	}
	g.Advance(Settle)
	if g.Cell(0, Rows-1) == 0 || g.Cell(0, Rows-2) == 0 || g.Cell(0, Rows-3) != 0 || len(g.Going) != 0 || !g.Live() {
		t.Fatal("the stack did not come down the second row, or the clear did not end")
	}
	if g.At != at {
		t.Fatal("the piece waiting came down with the stack")
	}
}

func TestStackIsWhatASaveKeepsAndResumeTakesUp(t *testing.T) {
	g := with(I)
	g.fill(Rows-1, Cols-1)
	g.Put(0, Rows-2, 1)
	g.Rotate(1) // upright, in the last column: the bottom row fills
	for g.Move(1) {
	}
	g.Drop()
	g.Advance(Flash) // the clear under way
	s := g.Stack()
	if !s[Rows-1][0] || s[Rows-2][0] || !s[Rows-1][Cols-1] || !s[Rows-3][Cols-1] || s[Rows-4][Cols-1] {
		t.Fatal("the stack saved during a clear is not as it is once the clear is over")
	}
	if g.Cell(0, Rows-2) == 0 {
		t.Fatal("Stack moved the stack itself")
	}
	r := Resume(2, 3, 25, 12340, s)
	if r.Level != 3 || r.Lines != 25 || r.Score != 12340 || !r.Live() || r.Over {
		t.Fatalf("taken up: level %d, %d lines, %d points", r.Level, r.Lines, r.Score)
	}
	for y := range Rows {
		for x := range Cols {
			if want := map[bool]uint8{true: Kept, false: 0}[s[y][x]]; r.Cell(x, y) != want {
				t.Fatalf("block %d, %d is %d, not %d", x, y, r.Cell(x, y), want)
			}
		}
	}
	if r := Resume(2, 99, 0, 0, [Rows][Cols]bool{}); r.Level != MaxLevel {
		t.Fatalf("level %d past the last", r.Level)
	}
}

func TestTheSpeedStopsAtTheLastLevel(t *testing.T) {
	g := with(I)
	g.Lines = 150
	g.fill(Rows-1, 0, 1, 2, 3)
	g.Drop() // flat, in columns 3–6: the row is not full, nothing goes
	g.Level = MaxLevel
	g.Lines = 199
	g.fill(Rows-2, Cols-1)
	g.Next[0] = I
	g.come()
	g.Rotate(1)
	for g.Move(1) {
	}
	g.Drop()
	g.Advance(ClearTime(1))
	if g.Level != MaxLevel {
		t.Fatalf("level %d past the last", g.Level)
	}
	if Interval(MaxLevel+5) != Interval(MaxLevel) {
		t.Fatal("the speed goes on past the last level")
	}
}

func TestTheStackAtTheTopEnds(t *testing.T) {
	g := with(1)
	for y := 0; y < Rows; y++ {
		g.fill(y, 0)
	}
	g.Next[0] = 1
	g.come()
	if !g.Over {
		t.Fatal("a piece with no room to come in did not end the game")
	}
	g.Advance(Curtain / 2)
	if d := g.CurtainDone(); d < 0.49 || d > 0.51 || g.Ended() {
		t.Fatalf("halfway: curtain %v", d)
	}
	g.Advance(Curtain)
	if !g.Ended() {
		t.Fatal("the curtain did not come down")
	}
}

func TestEndStopsThePiece(t *testing.T) {
	g := with(1)
	g.End()
	if g.Move(1) || g.Rotate(1) || g.Down() || g.Drop() != 0 {
		t.Fatal("the piece moved after the end")
	}
}
