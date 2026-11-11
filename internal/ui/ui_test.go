package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"

	"github.com/vulcanshen/tetru/internal/game"
	"github.com/vulcanshen/tetru/internal/scores"
)

// newTetru is tetru keeping its scores in a directory of the test's own,
// every game the same.
func newTetru(t *testing.T) *Tetru {
	t.Helper()
	t.Setenv("TETRU__DATA", t.TempDir())
	t.Setenv("TETRU__CONFIG", t.TempDir())
	tt := NewTetru()
	tt.seed = func() uint64 { return 1 }
	return tt
}

// open is tetru on a w × h terminal.
func open(t *testing.T, w, h int) (*Tetru, *tdptest.Driver) {
	t.Helper()
	tt := newTetru(t)
	return tt, tdptest.Start(t, tt.App, w, h)
}

// play starts a game: Enter on GAME.
func play(d *tdptest.Driver) { d.Press("Enter") }

// keepAs answers who played the game over: once the curtain is down, the
// name typed, Enter.
func keepAs(d *tdptest.Driver, name string) {
	d.Advance(game.Curtain)
	d.Type(name)
	d.Press("Enter")
}

// kept is what the scores' file holds.
func kept(t *testing.T) []scores.Score {
	t.Helper()
	all, err := scores.Load(scores.Path())
	if err != nil {
		t.Fatal(err)
	}
	return all
}

// turnable drops pieces till the one falling is not the square, which
// has one turn.
func turnable(tt *Tetru, d *tdptest.Driver) {
	for tt.g.Cur == 1 {
		d.Press("Enter")
	}
}

// row is the screen's row i.
func row(d *tdptest.Driver, i int) string { return d.Screen()[i] }

// topOut drops pieces in the middle till the stack reaches the top.
func topOut(t *testing.T, tt *Tetru, d *tdptest.Driver) {
	t.Helper()
	for range 60 {
		if tt.g.Over {
			return
		}
		d.Press("Enter")
	}
	t.Fatal("the stack never reached the top")
}

// topOutBare fills the well but its left column and the piece falling, then
// drops the piece where it is: it falls no row, so no points, and the next
// has no room.
func topOutBare(t *testing.T, tt *Tetru, d *tdptest.Driver) {
	t.Helper()
	g := tt.g
	piece := map[[2]int]bool{}
	for _, b := range g.Blocks() {
		piece[b] = true
	}
	for y := range game.Rows {
		for x := 1; x < game.Cols; x++ {
			if !piece[[2]int{x, y}] {
				g.Put(x, y, 1)
			}
		}
	}
	d.Press("Enter")
	if !g.Over || g.Score != 0 {
		t.Fatalf("not over with no points: over %v, %d points", g.Over, g.Score)
	}
}

// widest is the most pixels on a row of the screen.
func widest(d *tdptest.Driver) int {
	most := 0
	for _, r := range d.Screen() {
		most = max(most, strings.Count(r, pixelGlyph))
	}
	return most
}

func TestTetruScreen(t *testing.T) {
	tdptest.EachIconWidth(t, func(t *testing.T) {
		tt, d := open(t, 60, 32)
		if r := row(d, 0); !strings.Contains(r, "[T]ETRU") || !strings.Contains(r, "[R]ANK") || !strings.Contains(r, "[S]ETTINGS") {
			t.Fatalf("chips %q", r)
		}
		if r := row(d, 2); !strings.HasPrefix(r, "┏") || !strings.Contains(r, "[1] GAME") || !strings.Contains(r, "[2] NEXT 1") {
			t.Fatalf("not GAME, focused, and NEXT 1 at its right: %q", r)
		}
		if !d.Has("[3] NEXT 2") || !d.Has("[4] INFO") || !d.Has("Tab|1–4:panels") {
			t.Fatal("no NEXT 2 or INFO")
		}
		for _, c := range []struct{ w, g, r int }{{60, 37, 23}, {77, 52, 25}, {150, 100, 50}} {
			got := tt.layout(c.w, 29)
			nh := boxH + 2
			if got[0].W != c.g || got[1].X != c.g || got[1].W != c.r || got[1].H != nh || got[2].Y != nh || got[3].Y != 2*nh || got[3].H != 29-2*nh {
				t.Fatalf("at %d columns: %+v", c.w, got)
			}
		}
		if !d.Has("Enter:play") {
			t.Fatal("GAME's hint has no Enter:play")
		}
	})
}

func TestNarrowShowsTheFocusedPanelOnly(t *testing.T) {
	_, d := open(t, 50, 31)
	if d.Has("NEXT") || d.Has("INFO") || !d.Has("[1] GAME") || !d.Has("Level: 1   Lines: 0  Score: 0") {
		t.Fatal("narrow, another panel shows beside the game")
	}
	if got := widest(d); got != fieldW+boxW {
		t.Fatalf("narrow, GAME draws %d pixels a row, not the field and the boxes", got)
	}
	for _, p := range []string{"[2] NEXT 1", "[3] NEXT 2", "[4] INFO"} {
		d.Press("Tab")
		if r := row(d, 2); !strings.Contains(r, p) || strings.Contains(r, "GAME") {
			t.Fatalf("Tab did not show %s alone: %q", p, r)
		}
	}
}

func TestNextStaysBrightWhilePlaying(t *testing.T) {
	tt, d := open(t, 77, 32)
	streams := func() bool { return tt.next1P.Stream && tt.next2P.Stream && tt.infoP.Stream }
	if tt.next1P.Stream || tt.next2P.Stream || tt.infoP.Stream {
		t.Fatal("NEXT is streaming with no game")
	}
	play(d)
	if !streams() {
		t.Fatal("the NEXT panels dim while playing")
	}
	d.Press("Esc")
	if tt.next1P.Stream || tt.next2P.Stream || tt.infoP.Stream {
		t.Fatal("NEXT is streaming with the game paused")
	}
	d.Press("Enter")
	topOut(t, tt, d)
	if tt.next1P.Stream || tt.next2P.Stream || tt.infoP.Stream {
		t.Fatal("NEXT is streaming with the game over")
	}
}

func TestLeastSizeIsTheGameAlone(t *testing.T) {
	_, d := open(t, minW, minH)
	if d.Has("needs a terminal") {
		t.Fatalf("%d×%d is under the least size", minW, minH)
	}
	d.Resize(minW-1, minH)
	if !d.Has("needs a terminal of at least 42×29") {
		t.Fatal("no size message under the least size")
	}
}

func TestPictureGrowsTwiceOnALargeTerminal(t *testing.T) {
	tt, d := open(t, 150, 54)
	if tt.scale != 2 || widest(d) != 2*(fieldW+boxW) {
		t.Fatalf("150×54: GAME and NEXT %d times, %d pixels a row", tt.scale, widest(d))
	}
	if got := tt.layout(150, 51); got[1].H != 2*boxH+2 {
		t.Fatalf("NEXT 1 is %d high, twice", got[1].H)
	}
	d.Resize(150, 40)
	if tt.scale != 1 || widest(d) != fieldW+boxW {
		t.Fatalf("150×40, too low for twice: %d times, %d pixels a row", tt.scale, widest(d))
	}
}

func TestStatsInInfo(t *testing.T) {
	tt, d := open(t, 60, 32)
	for _, s := range []string{"Level: 1", "Lines: 0", "Score: 0", "╌╌╌"} {
		if !d.Has(s) {
			t.Fatalf("INFO has no %q", s)
		}
	}
	for i, r := range d.Screen() {
		if strings.Contains(r, "Level") && i < 3 {
			t.Fatal("the level is still at the top")
		}
	}
	play(d)
	d.Press("Enter")
	if !d.Has("Score: " + comma(tt.g.Score)) {
		t.Fatal("INFO does not follow the score")
	}
	if comma(1234567) != "1,234,567" || comma(12) != "12" {
		t.Fatal("comma")
	}
}

func TestPictureIsLockus(t *testing.T) {
	if boxW != 6 || boxH != 4 {
		t.Fatalf("a box is %d wide and %d high, not 6 and 4 (user, 2026-11-11)", boxW, boxH)
	}
	s := picture(nil)
	if s.px[0][0] != inkGround || s.px[sceneH-1][fieldW-1] != inkGround || s.px[0][fieldW] != inkNone {
		t.Fatal("the field is not the ground alone, unframed")
	}
	for i := range game.Queue {
		y := i * (boxH + 1)
		if s.px[y][queueX] != inkGround || s.px[y+boxH-1][sceneW-1] != inkGround {
			t.Fatalf("box %d is not the ground alone, unframed", i)
		}
	}
	if s.px[boxH][queueX] != inkNone {
		t.Fatal("no pixel between the boxes")
	}
	g := game.New(1)
	s = picture(g)
	for i, p := range g.Next {
		found := false
		for y := i * (boxH + 1); y < i*(boxH+1)+boxH; y++ {
			for x := queueX; x < sceneW; x++ {
				found = found || s.px[y][x] == inkPiece+uint8(p)
			}
		}
		if !found {
			t.Fatalf("box %d does not show piece %d", i, p)
		}
	}
	for _, b := range g.Blocks() {
		if s.px[b[1]][b[0]] != inkPiece+uint8(g.Cur) {
			t.Fatal("the falling piece is not drawn")
		}
	}
}

func TestPlay(t *testing.T) {
	tdptest.EachIconWidth(t, func(t *testing.T) {
		tt, d := open(t, 60, 32)
		play(d)
		turnable(tt, d)
		score := tt.g.Score
		if !tt.playing() || !strings.Contains(row(d, 2), "Play") {
			t.Fatal("not in the Play mode")
		}
		// 60 columns leave r:restart and s:save out of the footer.
		if r := row(d, 31); !strings.HasPrefix(r, "Space:menu ?:help d:rotate left k:rotate right Esc:pause") {
			t.Fatalf("footer %q", r)
		}
		if strings.Contains(row(d, 30), "Enter:") {
			t.Fatal("GAME's hint stays in the Play mode")
		}
		at := tt.g.At
		d.Press("h")
		d.Press("←")
		if tt.g.At.X != at.X-2 {
			t.Fatal("h and ← do not move left")
		}
		d.Press("l")
		d.Press("→")
		if tt.g.At.X != at.X {
			t.Fatal("l and → do not move right")
		}
		d.Press("j")
		d.Press("↓")
		if tt.g.At.Y != at.Y+2 || tt.g.Score != score+2 {
			t.Fatal("j and ↓ do not move down a row")
		}
		d.Press("e")
		if tt.g.At.Y != at.Y+2 || tt.g.At.Turn != at.Turn {
			t.Fatal("e did something")
		}
		d.Press("k")
		if tt.g.At.Turn != 1 {
			t.Fatal("k did not turn clockwise")
		}
		d.Press("d")
		if tt.g.At.Turn != 0 {
			t.Fatal("d did not turn back")
		}
		d.Press("PgDn")
		if tt.g.At.Turn != 3 || tt.g.At.Y != at.Y+2 {
			t.Fatal("PgDn did not turn left as d does (tdp K12)")
		}
		d.Press("↑")
		if tt.g.At.Turn != 0 || tt.g.At.Y != at.Y+2 {
			t.Fatal("↑ did not turn right as k does (tdp K12)")
		}
		before := tt.g.Score
		d.Press("i")
		if tt.g.Score != before || tt.g.At.Turn != 0 {
			t.Fatal("i did something")
		}
		d.Press("Enter")
		if tt.g.Score-before < 2*10 {
			t.Fatal("Enter did not drop the piece")
		}
	})
}

func TestTheGameFalls(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	y := tt.g.At.Y
	d.Advance(game.Interval(1) + frame)
	if tt.g.At.Y != y+1 {
		t.Fatalf("a second on, the piece is at %d, from %d", tt.g.At.Y, y)
	}
}

func TestTheGameStopsUnderAPopup(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Space")
	for _, s := range []string{"ITEM OPERATION", "[d] Rotate left", "[k] Rotate right", "PLAY MODE", "move left", "drop to the bottom", "[r]estart", "[s]ave", "[Esc] Pause"} {
		if !d.Has(s) {
			t.Fatalf("the Play mode's Space menu has no %q", s)
		}
	}
	y := tt.g.At.Y
	d.Advance(3 * time.Second)
	if tt.g.At.Y != y {
		t.Fatal("the game went on under the Space menu")
	}
	d.Press("Esc")
	d.Advance(game.Interval(1) + frame)
	if tt.g.At.Y == y || !tt.playing() {
		t.Fatal("the game did not go on as the menu closed")
	}
	y = tt.g.At.Y
	d.Press("?")
	d.Advance(3 * time.Second)
	if tt.g.At.Y != y {
		t.Fatal("the game went on under the key reference")
	}
}

func TestSpaceMenuRotates(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	turnable(tt, d)
	d.Press("Space")
	d.Press("Enter") // the first row: Rotate left
	if tt.g.At.Turn == 0 || !tt.playing() {
		t.Fatal("the menu's Rotate left did not turn the piece and go back to playing")
	}
}

func TestEscPausesEnterResumes(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Esc")
	if tt.playing() || !tt.paused() || strings.Contains(row(d, 2), "Play") {
		t.Fatal("Esc did not pause")
	}
	if !d.Has("RESUME") || !d.Has("Go on with the game?") || !d.Has("Paused: level 1, 0 lines, 0 points.") || !d.Has("Enter:resume") {
		t.Fatal("no popup says the game is paused")
	}
	y := tt.g.At.Y
	d.Advance(3 * time.Second)
	if tt.g.At.Y != y {
		t.Fatal("the game went on paused")
	}
	d.Press("Enter")
	if !tt.playing() || d.Has("RESUME") {
		t.Fatal("Enter on the popup did not resume")
	}
	d.Press("Esc")
	d.Press("Esc")
	if !tt.paused() || d.Has("RESUME") || !d.Has("Enter:resume") {
		t.Fatal("Esc on the popup did not leave the game paused, Enter on GAME to resume")
	}
	d.Press("Enter")
	if !tt.playing() {
		t.Fatal("Enter on GAME did not resume")
	}
}

func TestTabInPlaySaysToPauseFirst(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Tab")
	if !tt.playing() || !d.Has("Press [Esc] to leave Play first") {
		t.Fatal("Tab in the Play mode left it, or said nothing")
	}
}

func TestNameIsAskedAsTheGameEnds(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("Enter")
	if !tt.playing() || d.Has("NAME") {
		t.Fatal("Enter did not go straight into a game")
	}
	topOut(t, tt, d)
	if len(kept(t)) != 0 || d.Has("GAME OVER") {
		t.Fatal("kept, or asked, before the curtain came down")
	}
	g := tt.g
	d.Press("Enter")
	d.Press("n")
	if tt.g != g || d.Has("Enter:play") {
		t.Fatal("a new game while the last waits to be kept")
	}
	d.Advance(game.Curtain)
	if !d.Has("GAME OVER") || !d.Has(comma(g.Score)+" points: who played?") || !d.Has("ANON") || !d.Has("Enter:keep") {
		t.Fatal("no name asked once the curtain is down")
	}
	d.Press("Enter")
	if all := kept(t); len(all) != 1 || all[0].Name != "ANON" || all[0].Score != g.Score {
		t.Fatalf("Enter on the name offered kept %+v", all)
	}
	if !d.Has("Enter:play") {
		t.Fatal("Enter does not play once the game is kept")
	}
	d.Press("Enter")
	if !tt.playing() || tt.g == g {
		t.Fatal("Enter did not start the next game")
	}
}

func TestNameTypedIsKept(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	topOut(t, tt, d)
	keepAs(d, "  amy  ")
	if all := kept(t); len(all) != 1 || all[0].Name != "amy" {
		t.Fatalf("kept %+v, not the name trimmed", all)
	}
}

func TestEscOnTheNameKeepsAnon(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	topOut(t, tt, d)
	d.Advance(game.Curtain)
	d.Press("Esc")
	if all := kept(t); len(all) != 1 || all[0].Name != "ANON" {
		t.Fatalf("Esc on the name kept %+v", all)
	}
}

func TestNameWithTrickyCharacters(t *testing.T) {
	for _, name := range tdptest.Tricky {
		if strings.TrimSpace(name) == "" || strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) }) {
			continue // control characters are held back on Enter (tdp input.8)
		}
		_, d := open(t, 60, 32)
		d.Note("name " + name)
		play(d)
		d.Press("j")
		d.Press("Esc")
		d.Press("Esc") // the popup: paused
		d.Press("x")
		d.Paste(name)
		d.Press("Enter")
		want := []rune(name) // a paste is cut at the most a name has
		if all := kept(t); len(all) != 1 || all[0].Name != strings.TrimSpace(string(want[:min(len(want), nameLen)])) {
			t.Fatalf("name %q kept as %+v", name, all)
		}
	}
}

func TestTopOutKeepsTheScore(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	topOut(t, tt, d)
	if tt.playing() {
		t.Fatal("still in the Play mode after the end")
	}
	if !d.Has("Score: " + comma(tt.g.Score)) {
		t.Fatal("GAME lost the last game's score")
	}
	d.Advance(game.Curtain / 2)
	if tt.g.Ended() || d.Has("GAME OVER") {
		t.Fatal("the end was said before the curtain came down")
	}
	d.Advance(game.Curtain)
	if !tt.g.Ended() || !d.Has("GAME OVER") {
		t.Fatal("no popup once the curtain is down")
	}
	s := picture(tt.g)
	for y := range sceneH {
		for x := range fieldW {
			if p := s.px[y][x]; p != inkGround && p != inkFrame {
				t.Fatalf("pixel %d,%d is neither the ground nor the grey stack: the end is drawn in the game", x, y)
			}
		}
	}
	d.Type("vulcan")
	d.Press("Enter")
	all := kept(t)
	if len(all) != 1 || all[0].Name != "vulcan" || all[0].Score != tt.g.Score || all[0].Score == 0 {
		t.Fatalf("kept %+v", all)
	}
	if r, ok := tt.list.Current(); !ok || r.Cells[0] != "vulcan" {
		t.Fatal("the cursor is not on the new score")
	}
}

func TestTopOutWithNoPoints(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	g := tt.g
	topOutBare(t, tt, d)
	d.Advance(game.Curtain / 2)
	if d.Has("Play again?") {
		t.Fatal("the end was said before the curtain came down")
	}
	d.Advance(game.Curtain)
	if !d.Has("PLAY") || !d.Has("Play again?") || !d.Has("Game over: no points, so nothing is kept.") || !d.Has("Enter:play") || d.Has("who played") {
		t.Fatal("no popup saying the game is over with nothing kept")
	}
	d.Press("Esc")
	if d.Has("Play again?") || !d.Has("Enter:play") || len(kept(t)) != 0 {
		t.Fatal("Esc on the popup did not leave GAME to play from")
	}
	topOutAgain := func() {
		play(d)
		g = tt.g
		topOutBare(t, tt, d)
		d.Advance(game.Curtain)
	}
	topOutAgain()
	d.Press("Enter")
	if !tt.playing() || tt.g == g {
		t.Fatal("Enter on the popup did not start the next game")
	}
}

func TestScoresBestFirst(t *testing.T) {
	t.Setenv("TETRU__DATA", t.TempDir())
	t.Setenv("TETRU__CONFIG", t.TempDir())
	at := time.Date(2026, 11, 11, 9, 0, 0, 0, time.Local)
	scores.Save(scores.Path(), []scores.Score{
		{Time: at, Name: "amy", Score: 300},
		{Time: at.Add(time.Hour), Name: "bob", Score: 9800},
		{Time: at.Add(2 * time.Hour), Name: "cat", Score: 300},
	})
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 100, 32)
	d.Press("R")
	r := d.Screen()
	if !strings.Contains(r[2], "[1] RANK") || !strings.Contains(r[4], "bob") || !strings.Contains(r[4], "9,800") || !strings.Contains(r[5], "amy") || !strings.Contains(r[6], "cat") {
		t.Fatalf("not best first, the earlier of two the same first:\n%s", strings.Join(r[2:7], "\n"))
	}
	if !strings.Contains(r[3], "TIME") || !strings.Contains(r[4], "2026-11-11 10:00") {
		t.Fatal("no time column at 100 columns")
	}
}

func TestNoPointsNotKept(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("x")
	if !tt.g.Over || len(kept(t)) != 0 || d.Has("END GAME") {
		t.Fatal("a game with no points asked, was kept, or did not end")
	}
	d.Advance(game.Curtain)
	if d.Has("GAME OVER") || !d.Has("Enter:play") {
		t.Fatal("a game with no points asked who played")
	}
	play(d)
	d.Press("q")
	if d.Quit() || !d.Has("Quit tetru?") || !d.Has("No points so far, so nothing is kept.") {
		t.Fatal("q with a game of no points going did not confirm")
	}
	d.Press("Enter")
	if !d.Quit() || len(kept(t)) != 0 {
		t.Fatal("Enter did not leave, or kept a game of no points")
	}
}

func TestEndGameAsksAndKeeps(t *testing.T) {
	tt, d := open(t, 60, 32)
	if a := tt.gameP.Actions[2]; a.Key != "x" || !a.Disabled() {
		t.Fatal("End game is not disabled with no game")
	}
	play(d)
	d.Press("j")
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("x")
	if !d.Has("END GAME") || !d.Has("1 points: who played?") || !d.Has("Enter:end") {
		t.Fatal("End game did not ask who played, saying the points")
	}
	d.Press("Esc")
	if tt.g.Over || len(kept(t)) != 0 {
		t.Fatal("Esc on the name ended the game")
	}
	d.Press("x")
	d.Type("amy")
	d.Press("Enter")
	if all := kept(t); !tt.g.Over || len(all) != 1 || all[0].Name != "amy" {
		t.Fatalf("End game kept %+v", all)
	}
	d.Advance(game.Curtain)
	if d.Has("GAME OVER") {
		t.Fatal("asked again as the curtain came down")
	}
}

func TestNewGameAsksWhoPlayed(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("j")
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("n")
	if !d.Has("NEW GAME") || !d.Has("Enter:start") {
		t.Fatal("New game did not ask who played the game going")
	}
	d.Press("Enter")
	if all := kept(t); len(all) != 1 || all[0].Name != "ANON" {
		t.Fatalf("New game kept %+v", all)
	}
	if !tt.playing() || tt.g.Score != 0 {
		t.Fatal("no new game")
	}
}

func TestQuitWithAGameConfirmsAndKeepsAnon(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("j")
	d.Press("q")
	if !d.Has("QUIT") || !d.Has("Quit tetru?") || !d.Has("The score so far, 1, is kept for ANON.") || !d.Has("Enter:quit") || d.Has("who played") {
		t.Fatal("q with a game going did not confirm, saying the score is kept for ANON")
	}
	y := tt.g.At.Y
	d.Advance(3 * time.Second)
	if tt.g.At.Y != y {
		t.Fatal("the game went on under the confirm")
	}
	d.Press("Esc")
	if d.Quit() || !tt.playing() || len(kept(t)) != 0 {
		t.Fatal("Esc on the confirm did not go back to the game")
	}
	d.Press("q")
	d.Press("Enter")
	if all := kept(t); len(all) != 1 || all[0].Name != "ANON" || all[0].Score != 1 {
		t.Fatalf("quitting kept %+v", all)
	}
	if !d.Quit() {
		t.Fatal("Enter did not leave")
	}
}

func TestQuitWithNoGameLeaves(t *testing.T) {
	_, d := open(t, 60, 32)
	d.Press("q")
	if !d.Quit() {
		t.Fatal("q with no game going did not leave")
	}
}

func TestFinishKeepsTheGameGoing(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Enter")
	tt.Finish() // what main does after Run: Ctrl-C twice, the terminal closed
	tt.Finish()
	if all := kept(t); len(all) != 1 || all[0].Score != tt.g.Score || all[0].Name != "ANON" {
		t.Fatalf("kept %+v", all)
	}
}

func TestLineClearSlidesOutToTheLeft(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	g := tt.g
	// The foot full but where the piece coming in falls.
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
	g.Put(0, game.Rows-2, 1) // a block over the row: it comes down after
	next := g.Next[0]
	d.Press("Enter")
	if len(g.Going) != 1 || g.Lines != 1 || !d.Has("Lines: 1") {
		t.Fatalf("rows going %v, %d lines", g.Going, g.Lines)
	}
	if g.Cur != next {
		t.Fatal("the next piece is not in at the top")
	}
	at := g.At
	for _, k := range []string{"h", "l", "j", "d", "k", "Enter"} {
		d.Press(k)
	}
	if g.At != at || !tt.playing() {
		t.Fatal("the piece was played while the row goes")
	}
	// At 60 × 32 GAME's body is 35 × 27, the field 24 columns in its middle:
	// at column 5 of the body, screen column 6; row y of it on screen row
	// 4 + y.
	y := game.Rows - 1
	first := func() int {
		r := []rune(row(d, 4+y))
		for i, c := range r {
			if string(c) == pixelGlyph {
				return terminu.Width(string(r[:i]))
			}
		}
		return -1
	}
	if first() != 6 {
		t.Fatalf("the row going starts at column %d, not in place", first())
	}
	white := func() bool { return strings.Contains(tt.gameP.Body.Draw(35, 27, true)[1+y], "255;255;255") }
	d.Advance(game.Flash / 2)
	if first() != 6 || !white() {
		t.Fatal("the row going is not white in place first")
	}
	d.Advance(game.Flash/2 + game.Slide*3/4)
	if c := first(); c < 0 || c >= 6 || !white() {
		t.Fatalf("the row going is not out left of the field: column %d", c)
	}
	d.Advance(game.Slide / 4)
	if first() != 6 || g.Cell(0, game.Rows-2) == 0 {
		t.Fatal("the row did not slide out, or the stack came down before it had")
	}
	d.Advance(game.ClearTime(1))
	if len(g.Going) != 0 || !g.Live() || g.Cell(0, game.Rows-1) == 0 || g.Cell(0, game.Rows-2) != 0 {
		t.Fatal("the stack did not come down after the row")
	}
	if g.At != at {
		t.Fatal("the piece came down with the stack")
	}
}

func TestRestartFromThePlayModesSpaceMenu(t *testing.T) {
	tt, d := open(t, 80, 32)
	play(d)
	d.Press("j")
	g := tt.g
	d.Press("Space")
	d.Press("r")
	if !d.Has("NEW GAME") || !d.Has("1 points: who played?") {
		t.Fatal("Restart did not ask who played the game going")
	}
	d.Press("Enter")
	if all := kept(t); len(all) != 1 || tt.g == g || !tt.playing() {
		t.Fatalf("Restart kept %+v, or no new game going", all)
	}
	d.Press("r")
	if tt.g.Score != 0 || !tt.playing() || d.Has("NEW GAME") {
		t.Fatal("r in the Play mode with no points did not start over at once")
	}
}

func TestSaveQuitsAndTheNextStartGoesOn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TETRU__DATA", dir)
	t.Setenv("TETRU__CONFIG", t.TempDir())
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 80, 32)
	play(d)
	d.Press("j")
	tt.g.Put(0, game.Rows-1, 3)
	tt.g.Put(1, game.Rows-1, 4)
	level, lines, score := tt.g.Level, tt.g.Lines, tt.g.Score
	d.Press("Space")
	d.Press("s")
	if !d.Quit() {
		t.Fatal("Save did not quit")
	}
	tt.Finish()
	if len(kept(t)) != 0 {
		t.Fatal("a game saved was kept in the rank")
	}
	path := filepath.Join(dir, "game.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("no game saved")
	}

	tt = NewTetru()
	d = tdptest.Start(t, tt.App, 80, 32)
	g := tt.g
	if g == nil || g.Level != level || g.Lines != lines || g.Score != score || !tt.paused() || !d.Has("Enter:resume") || !d.Has("Score: 1") {
		t.Fatal("the next start did not take the game saved up, paused")
	}
	if g.Cell(0, game.Rows-1) != game.Kept || g.Cell(1, game.Rows-1) != game.Kept || g.Cell(2, game.Rows-1) != 0 {
		t.Fatal("the stack saved is not back, its blocks Kept")
	}
	if p := picture(g); p.px[game.Rows-1][0] != inkWhite || p.px[game.Rows-1][1] != inkWhite {
		t.Fatal("the blocks of the game saved are not white")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the game saved was not removed once taken up")
	}
	d.Press("Enter")
	if !tt.playing() {
		t.Fatal("Enter did not go on with it")
	}
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("x")
	d.Type("amy")
	d.Press("Enter")
	if all := kept(t); len(all) != 1 || all[0].Score != score {
		t.Fatalf("the game taken up and ended kept %+v", all)
	}
}

func TestSaveFromThePanelWhenPaused(t *testing.T) {
	tt, d := open(t, 80, 32)
	if a := tt.gameP.Actions[3]; a.Key != "s" || !a.Disabled() {
		t.Fatal("Save is not GAME's, disabled with no game")
	}
	play(d)
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("s")
	if !d.Quit() {
		t.Fatal("s on GAME, paused, did not save and quit")
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("TETRU__DATA"), "game.yaml")); err != nil {
		t.Fatal("no game saved")
	}
}

func TestBrokenSaveAsksToMoveAside(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TETRU__DATA", dir)
	t.Setenv("TETRU__CONFIG", t.TempDir())
	path := filepath.Join(dir, "game.yaml")
	os.WriteFile(path, []byte("level: [nope"), 0o644)
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 60, 32)
	if !d.Has("MOVE ASIDE") || !d.Has("game.yaml.bak") || tt.g != nil {
		t.Fatal("a broken game saved did not ask to move it aside")
	}
	d.Press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("the broken file was not moved aside")
	}
	play(d)
	if !tt.playing() {
		t.Fatal("no game after starting fresh")
	}
}

func TestBrokenScoresAskToMoveAside(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TETRU__DATA", dir)
	t.Setenv("TETRU__CONFIG", t.TempDir())
	path := filepath.Join(dir, "scores.yaml")
	os.WriteFile(path, []byte("scores: [nope"), 0o644)
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 60, 32)
	if !d.Has("can't be read") {
		t.Fatal("a broken scores file did not ask to move it aside")
	}
	d.Press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("the broken file was not moved aside")
	}
	play(d)
	topOut(t, tt, d)
	keepAs(d, "vulcan")
	if len(kept(t)) != 1 {
		t.Fatal("no score kept after starting fresh")
	}
}

func TestSplashIsTheIcon(t *testing.T) {
	docs, err := os.ReadFile("../../docs/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if string(docs) != string(icon) {
		t.Fatal("internal/ui/icon.svg is not docs/icon.svg")
	}
	_, d := open(t, 60, 40)
	d.Press("V")
	d.Advance(5 * time.Second)
	if !d.Has("tetru") || !d.Has("A falling-block game for the terminal") || !d.Has("developed by") {
		t.Fatal("V did not open the splash")
	}
}

func TestPopupsOverTheGameAtEveryWidth(t *testing.T) {
	tdptest.EachIconWidth(t, func(t *testing.T) {
		for _, size := range [][2]int{{60, 32}, {minW, minH}, {120, 50}, {200, 60}} {
			tt, d := open(t, size[0], size[1])
			play(d)
			d.Press("j") // a point: the game is worth asking about
			d.Press("Space")
			d.Press("Esc")
			d.Press("?")
			d.Press("Esc")
			d.Press("q")
			d.Press("Esc")
			d.Press("Esc") // pauses: the popup
			d.Press("Esc")
			d.Press("x")
			d.Press("Esc")
			d.Press("n")
			d.Press("Esc")
			d.Press("Enter")
			topOut(t, tt, d)
			keepAs(d, "vulcan")
			play(d)
			topOutBare(t, tt, d)
			d.Advance(game.Curtain) // game over with no points: the popup
			d.Press("Enter")
		}
	})
}
