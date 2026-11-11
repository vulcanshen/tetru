package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// play starts a game for name: Enter on GAME, the name, Enter.
func play(d *tdptest.Driver, name string) {
	d.Press("Enter")
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
		if !d.Has("[3] NEXT 2") || !d.Has("[4] MANUAL") || !d.Has("Tab|1–4:panels") {
			t.Fatal("no NEXT 2 or MANUAL")
		}
		for _, c := range []struct{ w, g, r int }{{60, 37, 23}, {77, 52, 25}, {150, 100, 50}} {
			got := tt.layout(c.w, 29)
			nh := boxTall + 2
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
	if d.Has("NEXT") || d.Has("MANUAL") || !d.Has("[1] GAME") {
		t.Fatal("narrow, another panel shows beside the game")
	}
	if got := widest(d); got != sceneW {
		t.Fatalf("narrow, GAME draws %d pixels a row, not the field and the boxes", got)
	}
	for _, p := range []string{"[2] NEXT 1", "[3] NEXT 2", "[4] MANUAL"} {
		d.Press("Tab")
		if r := row(d, 2); !strings.Contains(r, p) || strings.Contains(r, "GAME") {
			t.Fatalf("Tab did not show %s alone: %q", p, r)
		}
	}
}

func TestNextStaysBrightWhilePlaying(t *testing.T) {
	tt, d := open(t, 77, 32)
	streams := func() bool { return tt.next1P.Stream && tt.next2P.Stream }
	if tt.next1P.Stream || tt.next2P.Stream {
		t.Fatal("NEXT is streaming with no game")
	}
	play(d, "vulcan")
	if !streams() {
		t.Fatal("the NEXT panels dim while playing")
	}
	d.Press("Esc")
	if tt.next1P.Stream || tt.next2P.Stream {
		t.Fatal("NEXT is streaming with the game paused")
	}
	d.Press("Enter")
	topOut(t, tt, d)
	if tt.next1P.Stream || tt.next2P.Stream {
		t.Fatal("NEXT is streaming with the game over")
	}
}

func TestLeastSizeIsTheGameAlone(t *testing.T) {
	_, d := open(t, minW, minH)
	if d.Has("needs a terminal") {
		t.Fatalf("%d×%d is under the least size", minW, minH)
	}
	d.Resize(minW-1, minH)
	if !d.Has("needs a terminal of at least 42×30") {
		t.Fatal("no size message under the least size")
	}
}

func TestPictureGrowsTwiceOnALargeTerminal(t *testing.T) {
	tt, d := open(t, 150, 52)
	if tt.scale != 2 || widest(d) != 2*(fieldW+queueW) {
		t.Fatalf("150×52: GAME and NEXT %d times, %d pixels a row", tt.scale, widest(d))
	}
	if got := tt.layout(150, 49); got[1].H != 2*boxTall+2 {
		t.Fatalf("NEXT 1 is %d high, twice", got[1].H)
	}
	d.Resize(150, 40)
	if tt.scale != 1 || widest(d) != fieldW+queueW {
		t.Fatalf("150×40, too low for twice: %d times, %d pixels a row", tt.scale, widest(d))
	}
}

func TestStatsInGame(t *testing.T) {
	tt, d := open(t, 60, 32)
	if !strings.Contains(row(d, 4), "Level: 1   Lines: 0") || !strings.Contains(row(d, 5), "Score: 0") {
		t.Fatalf("no level, rows and score over the field:\n%s\n%s", row(d, 4), row(d, 5))
	}
	if strings.Contains(row(d, 0), "Level") {
		t.Fatal("the level is still at the top")
	}
	play(d, "vulcan")
	d.Press("Enter")
	if !d.Has("Score: " + comma(tt.g.Score)) {
		t.Fatal("the score is not GAME's")
	}
	if comma(1234567) != "1,234,567" || comma(12) != "12" {
		t.Fatal("comma")
	}
}

func TestPictureIsLockus(t *testing.T) {
	s := picture(nil, false)
	if s.px[0][0] != inkFrame || s.px[sceneH-1][sceneW-1] != inkFrame || s.px[1][1] != inkGround {
		t.Fatal("no grey frame round a ground")
	}
	fw := 2*edge + game.Cols
	for i := 0; i <= game.Queue; i++ {
		if s.px[i*(boxH+edge)][fw+1] != inkFrame {
			t.Fatalf("no line over box %d", i)
		}
	}
	g := game.New(1)
	s = picture(g, false)
	for i, p := range g.Next {
		found := false
		for y := edge + i*(boxH+edge); y < edge+i*(boxH+edge)+boxH; y++ {
			for x := fw; x < sceneW-1; x++ {
				found = found || s.px[y][x] == inkPiece+uint8(p)
			}
		}
		if !found {
			t.Fatalf("box %d does not show piece %d", i, p)
		}
	}
	for _, b := range g.Blocks() {
		if s.px[edge+b[1]][edge+b[0]] != inkPiece+uint8(g.Cur) {
			t.Fatal("the falling piece is not drawn")
		}
	}
}

func TestPlay(t *testing.T) {
	tdptest.EachIconWidth(t, func(t *testing.T) {
		tt, d := open(t, 60, 32)
		play(d, "vulcan")
		turnable(tt, d)
		score := tt.g.Score
		if !tt.playing() || !strings.Contains(row(d, 2), "Play") {
			t.Fatal("not in the Play mode")
		}
		if r := row(d, 31); !strings.HasPrefix(r, "Space:menu ?:help e:rotate left i:rotate right Esc:pause") {
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
		d.Press("k")
		d.Press("↑")
		if tt.g.At.Y != at.Y+2 || tt.g.At.Turn != at.Turn {
			t.Fatal("k or ↑ did something")
		}
		d.Press("i")
		if tt.g.At.Turn != 1 {
			t.Fatal("i did not turn clockwise")
		}
		d.Press("e")
		if tt.g.At.Turn != 0 {
			t.Fatal("e did not turn back")
		}
		before := tt.g.Score
		d.Press("d")
		if tt.g.Score != before {
			t.Fatal("d did something")
		}
		d.Press("Enter")
		if tt.g.Score-before < 2*10 {
			t.Fatal("Enter did not drop the piece")
		}
	})
}

func TestTheGameFalls(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	y := tt.g.At.Y
	d.Advance(game.Interval(1) + frame)
	if tt.g.At.Y != y+1 {
		t.Fatalf("a second on, the piece is at %d, from %d", tt.g.At.Y, y)
	}
}

func TestTheGameStopsUnderAPopup(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	d.Press("Space")
	for _, s := range []string{"ITEM OPERATION", "Rotat[e] left", "Rotate r[i]ght", "PLAY MODE", "move left", "drop to the bottom", "[Esc] Pause"} {
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
	play(d, "vulcan")
	turnable(tt, d)
	d.Press("Space")
	d.Press("Enter") // the first row: Rotate left
	if tt.g.At.Turn == 0 || !tt.playing() {
		t.Fatal("the menu's Rotate left did not turn the piece and go back to playing")
	}
}

func TestEscPausesEnterResumes(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	d.Press("Esc")
	if tt.playing() || !tt.paused() || !d.Has("Enter:resume") || strings.Contains(row(d, 2), "Play") {
		t.Fatal("Esc did not pause")
	}
	s := picture(tt.g, true)
	if s.px[edge+game.Rows/2][edge+game.Cols/2-2] != inkPause {
		t.Fatal("no pause sign over the game paused")
	}
	y := tt.g.At.Y
	d.Advance(3 * time.Second)
	if tt.g.At.Y != y {
		t.Fatal("the game went on paused")
	}
	d.Press("Enter")
	if !tt.playing() {
		t.Fatal("Enter did not resume")
	}
}

func TestTabInPlaySaysToPauseFirst(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	d.Press("Tab")
	if !tt.playing() || !d.Has("Press [Esc] to leave Play first") {
		t.Fatal("Tab in the Play mode left it, or said nothing")
	}
}

func TestNameIsAskedFirstTheLastOffered(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("Enter")
	if !d.Has("NAME") || !d.Has("Who is playing?") {
		t.Fatal("no name asked")
	}
	d.Press("Enter")
	if !d.Has("Type a name") || tt.g != nil {
		t.Fatal("an empty name started a game")
	}
	d.Type("  amy  ")
	d.Press("Enter")
	if tt.name != "amy" {
		t.Fatalf("name %q, not trimmed", tt.name)
	}
	topOut(t, tt, d)
	d.Press("Enter")
	if !d.Has("Tab:edit Backspace:clear") || !d.Has("amy") {
		t.Fatal("the last name is not offered")
	}
	d.Press("Enter")
	if tt.name != "amy" || !tt.playing() {
		t.Fatal("Enter did not take the name offered")
	}
}

func TestEscOnTheNameStartsNothing(t *testing.T) {
	tt, d := open(t, 60, 32)
	d.Press("Enter")
	d.Press("Esc")
	if tt.g != nil || d.Has("NAME") {
		t.Fatal("Esc on the name started a game")
	}
}

func TestNameWithTrickyCharacters(t *testing.T) {
	for _, name := range tdptest.Tricky {
		if strings.TrimSpace(name) == "" || strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) }) {
			continue // control characters are held back on Enter (tdp input.8)
		}
		tt, d := open(t, 60, 32)
		d.Note("name " + name)
		d.Press("Enter")
		d.Paste(name)
		d.Press("Enter")
		if tt.g == nil {
			continue // longer than a name may be: the input says so
		}
		d.Press("j")
		d.Press("Esc")
		d.Press("x")
		d.Press("Enter")
		want := []rune(name) // a paste is cut at the most a name has
		if all := kept(t); len(all) != 1 || all[0].Name != strings.TrimSpace(string(want[:min(len(want), nameLen)])) {
			t.Fatalf("name %q kept as %+v", name, all)
		}
	}
}

func TestTopOutKeepsTheScore(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	topOut(t, tt, d)
	all := kept(t)
	if len(all) != 1 || all[0].Name != "vulcan" || all[0].Score != tt.g.Score || all[0].Score == 0 {
		t.Fatalf("kept %+v", all)
	}
	if tt.playing() || !d.Has("Enter:play") {
		t.Fatal("still in the Play mode after the end")
	}
	if r, ok := tt.list.Current(); !ok || r.Cells[0] != "vulcan" {
		t.Fatal("the cursor is not on the new score")
	}
	if !d.Has("Score: " + comma(tt.g.Score)) {
		t.Fatal("GAME lost the last game's score")
	}
	d.Advance(game.Curtain / 2)
	if tt.g.Ended() {
		t.Fatal("END before the curtain came down")
	}
	d.Advance(game.Curtain)
	s := picture(tt.g, false)
	red := 0
	for y := range sceneH {
		for x := range sceneW {
			if s.px[y][x] == inkRed {
				red++
				if x < edge || x >= fieldW-edge {
					t.Fatalf("END is out of the field at pixel %d", x)
				}
			}
		}
	}
	if !tt.g.Ended() || red == 0 {
		t.Fatal("no END once the curtain is down")
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
	play(d, "vulcan")
	d.Press("Esc")
	d.Press("x")
	d.Press("Enter")
	if !tt.g.Over || len(kept(t)) != 0 {
		t.Fatal("a game with no points was kept, or not ended")
	}
}

func TestEndGameAsksAndKeeps(t *testing.T) {
	tt, d := open(t, 60, 32)
	if a := tt.gameP.Actions[2]; a.Key != "x" || !a.Disabled() {
		t.Fatal("End game is not disabled with no game")
	}
	play(d, "vulcan")
	d.Press("j")
	d.Press("Esc")
	d.Press("x")
	if !d.Has("End this game?") || !d.Has("This game ends with 1 points, kept for vulcan.") || !d.Has("Enter:end") {
		t.Fatal("End game did not ask, saying what is kept")
	}
	d.Press("Esc")
	if tt.g.Over {
		t.Fatal("Esc on the confirm ended the game")
	}
	d.Press("x")
	d.Press("Enter")
	if !tt.g.Over || len(kept(t)) != 1 {
		t.Fatal("End game did not end the game and keep it")
	}
}

func TestNewGameAsksThenAsksTheName(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	d.Press("j")
	d.Press("Esc")
	d.Press("n")
	if !d.Has("Start a new game?") {
		t.Fatal("New game did not ask with a game going")
	}
	d.Press("Enter")
	if len(kept(t)) != 1 || !d.Has("Who is playing?") {
		t.Fatal("New game did not keep the game going and ask the name")
	}
	d.Press("Enter")
	if !tt.playing() || tt.g.Score != 0 {
		t.Fatal("no new game")
	}
}

func TestQuitWithAGameAsksAndKeeps(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
	d.Press("j")
	d.Press("q")
	if !d.Has("Quit tetru?") || !d.Has("This game ends with 1 points, kept for vulcan.") {
		t.Fatal("q with a game going did not ask")
	}
	y := tt.g.At.Y
	d.Advance(3 * time.Second)
	if tt.g.At.Y != y {
		t.Fatal("the game went on under the quit confirm")
	}
	d.Press("Enter")
	if len(kept(t)) != 1 {
		t.Fatal("quitting did not keep the score")
	}
	d.Advance(closing)
	if !d.Quit() {
		t.Fatal("Enter on Quit tetru? did not leave once the confirm closed")
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
	play(d, "vulcan")
	d.Press("Enter")
	tt.Finish() // what main does after Run: Ctrl-C twice, the terminal closed
	tt.Finish()
	if all := kept(t); len(all) != 1 || all[0].Score != tt.g.Score {
		t.Fatalf("kept %+v", all)
	}
}

func TestLineClearFlashesThenGoes(t *testing.T) {
	tt, d := open(t, 60, 32)
	play(d, "vulcan")
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
	d.Press("Enter")
	if len(g.Going) != 1 {
		t.Fatalf("rows going %v", g.Going)
	}
	y := edge + game.Rows - 1
	if picture(g, false).px[y][edge] != inkWhite {
		t.Fatal("the row going is not white first")
	}
	d.Advance(game.Flash + game.Wipe/2)
	half := picture(g, false)
	if half.px[y][edge+game.Cols/2] == inkWhite || half.px[y][edge] != inkWhite {
		t.Fatal("halfway, the middle is not gone while the ends stay")
	}
	d.Advance(game.Wipe)
	if g.Lines != 1 || len(g.Going) != 0 || !d.Has("Lines: 1") {
		t.Fatal("the row did not go")
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
	play(d, "vulcan")
	topOut(t, tt, d)
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
			play(d, "vulcan")
			d.Press("Space")
			d.Press("Esc")
			d.Press("?")
			d.Press("Esc")
			d.Press("q")
			d.Press("Esc")
			d.Press("Esc")
			d.Press("x")
			d.Press("Esc")
			d.Press("n")
			d.Press("Esc")
			d.Press("Enter")
			topOut(t, tt, d)
			d.Advance(game.Curtain)
		}
	})
}
