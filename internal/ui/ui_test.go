package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// play starts a game: Enter on PLAYGROUND.
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
		_, d := open(t, 60, 32)
		if r := row(d, 0); !strings.Contains(r, "[T]ETRU") || !strings.Contains(r, "[R]ANK") || !strings.Contains(r, "[S]ETTINGS") {
			t.Fatalf("chips %q", r)
		}
		if r := row(d, 2); !strings.HasPrefix(r, "┏") || !strings.Contains(r, "[1] PLAYGROUND") || !strings.HasSuffix(r, "┓") {
			t.Fatalf("not PLAYGROUND alone, focused, the whole width: %q", r)
		}
		if r := row(d, 3); !strings.Contains(r, "┌─ NEXT 1 ─") || !strings.HasSuffix(r, "┐ ┃") {
			t.Fatalf("NEXT 1 is not a box in PLAYGROUND at its right: %q", r)
		}
		if !d.Has("┌─ NEXT 2 ─") || !d.Has("┌─ INFO ─") || !d.Has("┌─ KEYS ─") || d.Has("[2]") || d.Has("[3]") || d.Has("[4]") {
			t.Fatal("no NEXT 2 or INFO box, or a box numbered")
		}
		for _, c := range []struct{ w, f, c int }{{58, 34, 23}, {75, 50, 24}, {148, 98, 49}} {
			if f, cw, k := split(c.w, 27); f != c.f || cw != c.c || k != 1 {
				t.Fatalf("PLAYGROUND's body %d wide: the field's %d, the column %d, %d times", c.w, f, cw, k)
			}
		}
		if !d.Has("Enter:play") {
			t.Fatal("PLAYGROUND's hint has no Enter:play")
		}
	})
}

func TestTheBoxesAreNeverFocused(t *testing.T) {
	_, d := open(t, 77, 32)
	if d.Has("Tab|1–4:panels") || d.Has("Tab:") {
		t.Fatal("the footer has keys to other panels")
	}
	for _, k := range []string{"Tab", "2", "3", "4"} {
		d.Press(k)
		if r := row(d, 2); !strings.HasPrefix(r, "┏") || !strings.Contains(r, "[1] PLAYGROUND") {
			t.Fatalf("%s took the focus off PLAYGROUND: %q", k, r)
		}
	}
}

func TestNarrowDrawsTheBoxesWithNoFrames(t *testing.T) {
	_, d := open(t, 50, 32)
	if d.Has("NEXT") || d.Has("INFO") || !d.Has("[1] PLAYGROUND") || !d.Has("Level: 1   Lines: 0  Score: 0") {
		t.Fatal("narrow, the boxes are framed, or the level, the rows and the score are not over the field")
	}
	if got := widest(d); got != fieldW+boxW {
		t.Fatalf("narrow, PLAYGROUND draws %d pixels a row, not the field and the boxes", got)
	}
}

func TestLeastSizeIsTheGameAlone(t *testing.T) {
	_, d := open(t, minW, minH)
	if d.Has("needs a terminal") {
		t.Fatalf("%d×%d is under the least size", minW, minH)
	}
	if r := row(d, minH-2); !strings.HasPrefix(r, "┗") {
		t.Fatalf("PLAYGROUND's frame is not right over the footer: %q", r)
	}
	if got := strings.Count(strings.Join(d.Screen(), ""), pixelGlyph); got != fieldW*sceneH+2*boxW*boxH {
		t.Fatalf("narrow, at the least size, %d pixels, not all of the field and the boxes", got)
	}
	d.Resize(minW-1, minH)
	if !d.Has("needs a terminal of at least 42×32") {
		t.Fatal("no size message under the least size")
	}
	d.Resize(narrowBelow, minH)
	if !d.Has("Esc    pause") {
		t.Fatal("wide, at the least height, INFO's last row is cut")
	}
}

func TestPictureGrowsTwiceOnALargeTerminal(t *testing.T) {
	_, d := open(t, 150, 54)
	if _, _, k := split(148, 49); k != 2 || widest(d) != 2*(fieldW+boxW) {
		t.Fatalf("150×54: %d times, %d pixels a row", k, widest(d))
	}
	at := func(s string) int {
		for i, r := range d.Screen() {
			if strings.Contains(r, s) {
				return i
			}
		}
		return -1
	}
	if got := at("NEXT 2") - at("NEXT 1"); got != 2*boxH+2 {
		t.Fatalf("NEXT 1 is %d high, not twice", got)
	}
	d.Resize(150, 40)
	if _, _, k := split(148, 35); k != 1 || widest(d) != fieldW+boxW {
		t.Fatalf("150×40, too low for twice: %d times, %d pixels a row", k, widest(d))
	}
}

func TestStatsInInfo(t *testing.T) {
	tt, d := open(t, 60, 32)
	for _, s := range []string{"Level: 1", "Lines: 0", "Score: 0"} {
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
		if !tt.playing() || !strings.Contains(row(d, 2), "Gaming") {
			t.Fatal("not in the Gaming mode")
		}
		// 60 columns leave r:restart and s:save out of the footer.
		if r := row(d, 31); !strings.HasPrefix(r, "Space:menu ?:help d:rotate left k:rotate right Esc:pause") {
			t.Fatalf("footer %q", r)
		}
		if strings.Contains(row(d, 30), "Enter:") {
			t.Fatal("PLAYGROUND's hint stays in the Gaming mode")
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
	for _, s := range []string{"GAMING MODE", "[r]estart", "[s]ave"} {
		if !d.Has(s) {
			t.Fatalf("the Gaming mode's Space menu has no %q", s)
		}
	}
	// INFO beside it has the keys, "move left" among them, in lower case.
	for _, s := range []string{"ITEM OPERATION", "Rotate left", "Rotate right", "drop to the bottom", "[Esc] Pause"} {
		if d.Has(s) {
			t.Fatalf("the Gaming mode's Space menu has %q, not Restart and Save alone", s)
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

func TestSpaceMenuRunsRestartAndTheKeysAreInHelp(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("?")
	for _, s := range []string{"move left", "move right", "down a row", "rotate left", "rotate right", "drop to the bottom", "restart", "save"} {
		if !d.Has(s) {
			t.Fatalf("the Gaming mode's ? has no %q", s)
		}
	}
	d.Press("Esc")
	g := tt.g
	d.Press("Space")
	d.Press("Enter") // the first row: Restart, with no points at once
	if tt.g == g || !tt.playing() {
		t.Fatal("the menu's Restart did not start over")
	}
}

func TestEscPausesEnterResumes(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Esc")
	if tt.playing() || !tt.paused() || strings.Contains(row(d, 2), "Gaming") {
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
		t.Fatal("Esc on the popup did not leave the game paused, Enter on PLAYGROUND to resume")
	}
	d.Press("Enter")
	if !tt.playing() {
		t.Fatal("Enter on PLAYGROUND did not resume")
	}
}

func TestTabInPlaySaysToPauseFirst(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d)
	d.Press("Tab")
	if !tt.playing() || !d.Has("Press [Esc] to leave Gaming first") {
		t.Fatal("Tab in the Gaming mode left it, or said nothing")
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
		t.Fatal("still in the Gaming mode after the end")
	}
	if !d.Has("Score: " + comma(tt.g.Score)) {
		t.Fatal("PLAYGROUND lost the last game's score")
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
	if r, ok := tt.list.Current(); !ok || r.Cells[0] != "1" || r.Cells[1] != "vulcan" {
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
		t.Fatal("Esc on the popup did not leave PLAYGROUND to play from")
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
	// At 60 × 32 PLAYGROUND's body is 58 × 27, the field 24 columns in the middle
	// of its 34 at the left: at column 5 of the body, screen column 6; row
	// y of it on screen row 4 + y.
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
	white := func() bool { return strings.Contains(tt.gameP.Body.Draw(58, 27, true)[1+y], "255;255;255") }
	// A frame at a time, wherever the frames fall: white in place, then out
	// left of the field, then gone with the stack not down yet, then the
	// stack down.
	stage := 0
	for range (game.ClearTime(1) + 100*time.Millisecond) / clearFrame {
		d.Advance(clearFrame)
		c, down := first(), g.Cell(0, game.Rows-2) == 0
		switch {
		case stage == 0 && c == 6 && white() && !down:
			stage = 1
		case stage == 1 && c >= 0 && c < 6 && white() && !down:
			stage = 2
		case stage == 2 && c == 6 && !white() && !down:
			stage = 3
		case stage == 3 && down:
			stage = 4
		}
	}
	if stage != 4 {
		t.Fatalf("the clear got to %d of: white in place, out left of the field, gone with the stack up, the stack down", stage)
	}
	if len(g.Going) != 0 || !g.Live() || g.Cell(0, game.Rows-1) == 0 {
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
		t.Fatal("r in the Gaming mode with no points did not start over at once")
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
		t.Fatal("Save is not PLAYGROUND's, disabled with no game")
	}
	play(d)
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("s")
	if !d.Quit() {
		t.Fatal("s on PLAYGROUND, paused, did not save and quit")
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

func TestRankKeepsTheBestTwentyWithTheirPlaces(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TETRU__DATA", dir)
	t.Setenv("TETRU__CONFIG", t.TempDir())
	var all []scores.Score
	for i := range scores.Most + 1 { // one past the twenty, from before
		all = append(all, scores.Score{Time: time.Date(2026, 11, 1, 0, i, 0, 0, time.UTC), Name: "p" + strconv.Itoa(i), Score: 100 * (i + 1)})
	}
	if err := scores.Save(scores.Path(), all); err != nil {
		t.Fatal(err)
	}
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 80, 32)
	d.Press("R")
	place := func(n int, name, score string) bool {
		return regexp.MustCompile(`\s` + strconv.Itoa(n) + `\s+` + name + `\s+` + score + `\s`).MatchString(row(d, 3+n))
	}
	if !d.Has("#") || !place(1, "p20", "2,100") || !place(20, "p1", "200") || d.Has("p0 ") || len(tt.all) != scores.Most {
		t.Fatalf("not the best twenty, each with its place: %q, %q", row(d, 4), row(d, 23))
	}

	d.Press("T")
	play(d)
	tt.g.Score = 150 // under the twentieth
	d.Press("Esc")
	d.Press("Esc") // the popup: paused
	d.Press("x")
	if d.Has("END GAME") || !tt.g.Over || !d.Has("150 points, not in the best 20, so nothing is kept.") {
		t.Fatal("a game not in the best twenty asked who played, or did not say it is not kept")
	}
	if all := kept(t); len(all) != scores.Most+1 {
		t.Fatalf("the file was written: %d", len(all))
	}

	d.Advance(5 * time.Second) // the toast goes
	d.Press("Enter")           // a new game
	tt.g.Score = 5000
	d.Press("Esc")
	d.Press("Esc")
	d.Press("x")
	d.Press("Enter") // ANON
	all = kept(t)
	if len(all) != scores.Most || all[len(all)-1].Score != 5000 || all[0].Name != "p2" {
		t.Fatalf("kept %d, the new one %+v, the first %+v", len(all), all[len(all)-1], all[0])
	}
	if r, ok := tt.list.Current(); !ok || r.Cells[0] != "1" || r.Cells[1] != anon {
		t.Fatal("the new best is not first, the cursor on it")
	}
}

func TestRankColoursTheFirstThree(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TETRU__DATA", dir)
	t.Setenv("TETRU__CONFIG", t.TempDir())
	var all []scores.Score
	for i, n := range []int{5200, 3100, 8800, 1200} {
		all = append(all, scores.Score{Time: time.Date(2026, 11, 12, 10, i, 0, 0, time.UTC), Name: "p" + strconv.Itoa(i), Score: n})
	}
	if err := scores.Save(scores.Path(), all); err != nil {
		t.Fatal(err)
	}
	tt := NewTetru()
	d := tdptest.Start(t, tt.App, 60, 32)
	d.Press("R")
	d.Press("G") // the cursor off the first three
	rows := tt.rankP.Body.Draw(58, 27, true)
	for n, rgb := range []string{"249;226;175", "166;227;161", "137;220;235"} {
		if !strings.Contains(rows[1+n], rgb) {
			t.Fatalf("place %d is not in its colour: %q", n+1, rows[1+n])
		}
	}
	for _, rgb := range []string{"249;226;175", "166;227;161", "137;220;235"} {
		if strings.Contains(rows[4], rgb) {
			t.Fatalf("the fourth, the cursor on it, is in a place's colour: %q", rows[4])
		}
	}
}

// clearOf puts in tt a game with n rows just gone full, the long piece
// dropped upright in the last column, a block over them.
func clearOf(t *testing.T, tt *Tetru, n int) *game.Game {
	t.Helper()
	var stack [game.Rows][game.Cols]bool
	for y := game.Rows - 4; y < game.Rows; y++ {
		for x := 1; x < game.Cols-1; x++ {
			stack[y][x] = true
		}
		stack[y][0] = y >= game.Rows-n
	}
	stack[game.Rows-5][0] = true
	var g *game.Game
	for seed := uint64(1); ; seed++ {
		if g = game.Resume(seed, 1, 0, 0, stack); g.Cur == 0 {
			break
		}
	}
	g.Rotate(1)
	for g.Move(1) {
	}
	g.Drop()
	if len(g.Going) != n {
		t.Fatalf("%d rows going, not %d", len(g.Going), n)
	}
	tt.g = g
	return g
}

func TestClearsByHowManyRows(t *testing.T) {
	tt, _ := open(t, 80, 32)
	// At 80 × 32 PLAYGROUND's body is 78 × 27, the field 24 columns from
	// column 14 of its 52 at the left.
	bottom := func() string { return tt.gameP.Body.Draw(78, 27, true)[1+game.Rows-1] }
	plain := regexp.MustCompile("\x1b\\[[0-9;]*m")
	const flamingo, pink, white, ground = "242;205;205", "245;194;231", "255;255;255", "49;50;68"

	g := clearOf(t, tt, 2)
	g.Advance(game.Flash + game.Slide/2)
	r := bottom()
	rs := []rune(plain.ReplaceAllString(r, ""))
	right := false
	for i, c := range rs {
		if col := terminu.Width(string(rs[:i])); string(c) == pixelGlyph && col >= 38 && col < 52 {
			right = true
		}
	}
	if !strings.Contains(r, flamingo) || !right {
		t.Fatalf("two rows do not split in Flamingo, half going right: %q", r)
	}

	g = clearOf(t, tt, 3)
	if !strings.Contains(bottom(), pink) {
		t.Fatal("three rows are not Pink")
	}
	g.Advance(game.Flash + (game.ClearTime(3)-game.Flash)/2)
	if r := bottom(); !strings.Contains(r, pink) || !strings.Contains(r, white) || !strings.Contains(r, ground) {
		t.Fatalf("three rows half way do not crumble, some blocks Pink, some white, some gone: %q", r)
	}

	g = clearOf(t, tt, 4)
	if r := bottom(); !strings.Contains(r, white) {
		t.Fatal("four rows do not go white first")
	}
	g.Advance(game.Flash + game.Slide/3)
	colours := 0
	for _, c := range inks[inkPiece : inkPiece+7] {
		if strings.Contains(bottom(), fmt.Sprintf("%d;%d;%d", c.R(), c.G(), c.B())) {
			colours++
		}
	}
	if colours < 3 {
		t.Fatalf("four rows burst in %d of the pieces' colours", colours)
	}
}
