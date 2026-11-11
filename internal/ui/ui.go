// Package ui is tetru's terminal UI on terminu-go: three screens (user,
// 2026-11-11), Tetru with the GAME panel, the pieces to come and the
// manual, Rank with the scores kept, Settings with the keys to play with;
// the Play mode the game runs in.
package ui

import (
	_ "embed"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/game"
	"github.com/vulcanshen/tetru/internal/keys"
	"github.com/vulcanshen/tetru/internal/scores"
	"github.com/vulcanshen/tetru/internal/version"
)

// icon is tetru's family mark (tdp S5), drawn by the splash: a copy of
// docs/icon.svg, which go:embed can't reach from here (a test holds the two
// the same).
//
//go:embed icon.svg
var icon []byte

const (
	// minW, minH are tetru's least size (tdp L1, lower than 60 × 32): GAME
	// alone (narrow) drawing the field and the boxes, its frame and a cell's
	// margin each side; the chip row, its rule and the footer, GAME's frame,
	// the rows over the field and the picture.
	minW, minH = 2*sceneW + 4, 3 + 2 + statsH + sceneH
	// narrowBelow is the width under which only the focused panel is drawn
	// (tdp screen.5's most).
	narrowBelow = 60
	// sideW is the least the column at GAME's right is given: MANUAL's rows
	// and its frame.
	sideW = manualW + 2
	// frame is a step of the game: the pace of the clear, as locku's.
	frame = 40 * time.Millisecond
	// nameLen is the most characters a name has.
	nameLen = 16
)

// Tetru is the app and the game in it.
type Tetru struct {
	App *terminu.App

	path    string         // the scores' file
	all     []scores.Score // the records, as kept
	loadErr error          // why the file could not be read, at start
	cfg     string         // the settings file
	keys    keys.Set       // the keys to play with
	cfgErr  error          // why the settings could not be read, at start

	list   *terminu.List
	rankP  *terminu.Panel
	gameP  *terminu.Panel
	next1P *terminu.Panel
	next2P *terminu.Panel
	play   *terminu.Mode
	scale  int // what a pixel of GAME and the NEXT panels is grown by

	g     *game.Game // the game, nil before the first
	name  string     // who plays it
	saved bool       // its score is kept (or it was not worth keeping)

	ticking bool          // a frame is on its way
	done    bool          // the program is ending: no more frames
	seed    func() uint64 // the next game's seed
}

// New is tetru, all but Run.
func New() *terminu.App { return NewTetru().App }

// NewTetru is tetru with its game, for main to keep the score of a game
// still going when the program ends (Finish).
func NewTetru() *Tetru {
	t := &Tetru{App: terminu.New("tetru"), path: scores.Path(), cfg: keys.Path(), seed: rand.Uint64, scale: 1}
	t.all, t.loadErr = scores.Load(t.path)
	if t.keys, t.cfgErr = keys.Load(t.cfg); t.cfgErr != nil {
		t.keys = keys.Default()
	}
	a := t.App
	a.MinWidth, a.MinHeight = minW, minH
	a.QuitFlow = t.quit
	a.OnMsg = t.onMsg
	a.Start = t.start

	t.list = &terminu.List{
		Columns: []terminu.Column{
			{Title: "Name", Shrink: true, MinWidth: 6},
			{Title: "Score", Number: true},
			{Title: "Time", Drop: 1},
		},
		EmptyFact: "No scores yet",
		EmptyHint: "Press [T] to play",
	}
	t.rankP = &terminu.Panel{Label: "Rank", Body: t.list}
	t.gameP = &terminu.Panel{Label: "Game", Body: &field{t}, OnEsc: func() tea.Cmd { return nil }}
	t.next1P = &terminu.Panel{Label: "Next 1", Body: &next{t, 0}}
	t.next2P = &terminu.Panel{Label: "Next 2", Body: &next{t, 1}}
	manualP := &terminu.Panel{Label: "Manual", Body: &manual{t}}
	keysP := &terminu.Panel{Label: "Keys", Body: &terminu.Values{Header: [2]string{"Action", "Key"}, Fields: t.keyFields()}}
	t.play = &terminu.Mode{Name: "Play", Exit: "pause", Pause: true, OnExit: t.pause}
	t.setModeKeys()
	t.setActions()
	t.setRows("")
	a.AddScreen(&terminu.Screen{
		Name:        "Tetru",
		Key:         "T",
		Desc:        "the game",
		Panels:      []*terminu.Panel{t.gameP, t.next1P, t.next2P, manualP},
		NarrowBelow: narrowBelow,
		Layout:      t.layout,
	})
	a.AddScreen(&terminu.Screen{Name: "Rank", Key: "R", Desc: "the scores, best first", Panels: []*terminu.Panel{t.rankP}})
	a.AddScreen(&terminu.Screen{Name: "Settings", Key: "S", Desc: "the keys to play with", Panels: []*terminu.Panel{keysP}})
	var err error
	if a.Splash, err = terminu.NewSplash(icon, "tetru", version.Display(), "A falling-block game for the terminal"); err != nil {
		panic(err) // the embedded icon is broken: a mistake in the program
	}
	return t
}

// start puts the focus on GAME (opening tetru is to play), and asks to move
// aside the files that could not be read.
func (t *Tetru) start() tea.Cmd {
	return tea.Batch(t.App.Focus(t.gameP), t.moveAside())
}

// moveAside asks to move aside a file that could not be read (tdp E3), one
// after the other: the settings, then the scores.
func (t *Tetru) moveAside() tea.Cmd {
	switch {
	case t.cfgErr != nil:
		err := t.cfgErr
		t.cfgErr = nil
		return t.App.MoveAside(t.cfg, err, func() tea.Cmd {
			t.keys = keys.Default()
			t.setModeKeys()
			return t.moveAside()
		})
	case t.loadErr != nil:
		err := t.loadErr
		t.loadErr = nil
		return t.App.MoveAside(t.path, err, func() tea.Cmd {
			t.all = nil
			t.setRows("")
			return nil
		})
	}
	return nil
}

// layout is the Tetru screen: GAME at the left, and at its right a column
// of NEXT 1, NEXT 2 and MANUAL under one another (user, 2026-11-11), the
// column a third of the width and MANUAL's at least, the NEXT panels as
// high as a box. GAME and the boxes are grown twice when GAME and the
// column both have room for it.
func (t *Tetru) layout(w, h int) []terminu.Rect {
	rw := max(sideW, w/3)
	gw := w - rw
	t.scale = 1
	if gw-2 >= 2*2*fieldW && h-2 >= statsH+2*sceneH && rw-2 >= 2*2*queueW {
		t.scale = 2
	}
	nh := boxTall*t.scale + 2
	return []terminu.Rect{
		{W: gw, H: h},
		{X: gw, W: rw, H: nh},
		{X: gw, Y: nh, W: rw, H: nh},
		{X: gw, Y: 2 * nh, W: rw, H: h - 2*nh},
	}
}

// narrow says whether only the focused panel is drawn (tdp screen.5).
func (t *Tetru) narrow() bool {
	w, _ := t.App.Size()
	return t.App.ForceNarrow || w < narrowBelow
}

// comma writes n with its thousands apart: 12,340.
func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// going says whether there is a game not ended: playing or paused.
func (t *Tetru) going() bool { return t.g != nil && !t.g.Over }

// playing says whether the game is in the Play mode.
func (t *Tetru) playing() bool { return t.going() && t.gameP.InMode() == t.play }

// paused says whether there is a game not ended out of the Play mode.
func (t *Tetru) paused() bool { return t.going() && t.gameP.InMode() != t.play }

// setActions are GAME's actions as the game stands: Enter plays a new
// game, or goes on with the one paused (tdp K3); New game and End game.
// While playing, the NEXT panels are streaming content (tdp T2): the pieces
// to come change as the player plays on GAME, read from the corner of an
// eye.
func (t *Tetru) setActions() {
	t.next1P.Stream, t.next2P.Stream = t.playing(), t.playing()
	enter := terminu.Action{Key: "Enter", Label: "Play", Desc: "a new game", Scope: terminu.PanelScope, Hint: "play", Do: t.enter}
	switch {
	case t.playing():
		// In the Play mode no key of the panel's works: no hint (tdp
		// panel.1); the mode's keys are in the footer.
		enter.Hint = ""
	case t.paused():
		enter.Label, enter.Desc, enter.Hint = "Resume", "this game, where it was", "resume"
	}
	t.gameP.Actions = []terminu.Action{
		enter,
		{Key: "n", Label: "New game", Desc: "start over from level 1", Scope: terminu.PanelScope,
			AskFor: t.askEnding("New game", "Start a new game?", "start"), Do: t.newGame},
		{Key: "x", Label: "End game", Desc: "keep this score and stop", Scope: terminu.PanelScope,
			Disabled: func() bool { return !t.going() },
			AskFor:   t.askEnding("End game", "End this game?", "end"), Do: t.endGame},
	}
}

// askEnding is the confirm before what ends the game going, saying what is
// kept (tdp F6); nothing to ask with no game going.
func (t *Tetru) askEnding(title, question, verb string) func() *terminu.Ask {
	return func() *terminu.Ask {
		if !t.going() {
			return nil
		}
		return &terminu.Ask{Title: title, Question: question, Verb: verb, Details: []string{t.keeping()}}
	}
}

// keeping says what ending the game keeps.
func (t *Tetru) keeping() string {
	if t.g.Score == 0 {
		return "This game has no points yet: nothing is kept."
	}
	return "This game ends with " + comma(t.g.Score) + " points, kept for " + t.name + "."
}

// enter is Enter on GAME: go on with the game paused, or a new one.
func (t *Tetru) enter() tea.Cmd {
	if t.paused() {
		return t.resume()
	}
	return t.askName()
}

// askName asks who plays the new game, the last name kept offered (tdp
// input.6); a name is asked before the game, not after it: a game left
// half way is kept too, and leaving is no time to ask (user, 2026-11-11).
func (t *Tetru) askName() tea.Cmd {
	last := ""
	if n := len(t.all); n > 0 {
		last = t.all[n-1].Name
	}
	return t.App.Text(terminu.TextInput{
		Title:   "Name",
		Prompt:  "Who is playing?",
		Verb:    "play",
		Suggest: last,
		MaxLen:  nameLen,
		Check: func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("Type a name")
			}
			return nil
		},
		Done: func(s string) tea.Cmd { return t.begin(strings.TrimSpace(s)) },
	})
}

// begin starts a game for name, in the Play mode.
func (t *Tetru) begin(name string) tea.Cmd {
	t.g, t.name, t.saved = game.New(t.seed()), name, false
	return t.resume()
}

// resume goes into the Play mode: the game runs.
func (t *Tetru) resume() tea.Cmd {
	t.gameP.EnterMode(t.play)
	t.setActions()
	return t.wake()
}

// pause is Esc leaving the Play mode (tdp K4): the game waits, the pause
// sign on it, Enter to go on.
func (t *Tetru) pause() tea.Cmd {
	t.setActions()
	return nil
}

// newGame ends the game going, its score kept, and asks who plays the next.
func (t *Tetru) newGame() tea.Cmd {
	cmd := t.finish()
	return tea.Batch(cmd, t.askName())
}

// endGame ends the game going, its score kept.
func (t *Tetru) endGame() tea.Cmd {
	return t.finish()
}

// finish ends the game going, if any: out of the Play mode, the curtain
// coming down, its score kept.
func (t *Tetru) finish() tea.Cmd {
	if !t.going() {
		return nil
	}
	t.g.End()
	t.gameP.LeaveMode()
	t.setActions()
	return tea.Batch(t.save(), t.wake())
}

// save keeps the game's score, once: a game with no points is not kept.
// The new record is the cursor's on SCORES (tdp K3: the item added).
func (t *Tetru) save() tea.Cmd {
	if t.g == nil || t.saved {
		return nil
	}
	t.saved = true
	if t.g.Score == 0 {
		return nil
	}
	t.all = append(t.all, scores.Score{Time: time.Now().Truncate(time.Second), Name: t.name, Score: t.g.Score})
	t.setRows(strconv.Itoa(len(t.all) - 1))
	if err := scores.Save(t.path, t.all); err != nil {
		return t.App.Error("Can't keep the score", t.path, err.Error())
	}
	t.App.Log(terminu.Success, "Kept "+comma(t.g.Score)+" points for "+t.name)
	return nil
}

// Finish keeps the score of a game still going when the program ends: q
// and Ctrl-C ask first and keep it; Ctrl-C pressed twice does not ask
// (user, 2026-11-11: leaving keeps the score as it is then).
func (t *Tetru) Finish() {
	t.done = true
	if t.going() {
		t.g.End()
		t.save()
	}
}

// quit is the quit flow (tdp K9): with a game going, ask, saying what is
// kept; else leave.
func (t *Tetru) quit() tea.Cmd {
	if !t.going() {
		return tea.Quit
	}
	return t.App.Confirm(terminu.Ask{Title: "Quit", Question: "Quit tetru?", Verb: "quit", Details: []string{t.keeping()}},
		func() tea.Cmd {
			t.Finish()
			// TODO(tgo): leave once the confirm has closed (tdp F2: 9
			// frames of 32ms): tdptest cannot settle a confirm of the app's
			// whose accept quits while it closes; the lib's own quit
			// confirm does not have it.
			return t.App.Tick(closing, func(time.Time) tea.Msg { return tea.QuitMsg{} })
		})
}

// closing is how long a popup takes to close (tdp F2), and a frame more.
const closing = 10 * 32 * time.Millisecond

// setRows lists the records best first; sel is the ID of the row to put
// the cursor on, "" to leave it.
func (t *Tetru) setRows(sel string) {
	var rows []terminu.Row
	for _, i := range scores.Ranked(t.all) {
		s := t.all[i]
		rows = append(rows, terminu.Row{ID: strconv.Itoa(i), Cells: []string{s.Name, comma(s.Score), s.Time.Local().Format("2006-01-02 15:04")}})
	}
	t.list.SetRows(rows)
	if sel != "" {
		t.list.Select(sel)
	}
}

// setModeKeys are the Play mode's keys, the player's (Settings): the moves
// with their arrows too (tdp K12) and the drop, all the way down, are
// moving: shown in the mode's Space menu, not in the footer, which keeps
// room for Esc:pause; the turns are things done to the falling piece, run
// from the Space menu too (tdp K11). k and ↑ do nothing.
//
// TODO(tdp #4): rotate left on k and ↑, as the user asked.
func (t *Tetru) setModeKeys() {
	k := t.keys
	t.play.Keys = []terminu.ModeKey{
		{Key: k[keys.Left], Keys: []string{k[keys.Left], keys.Arrow(keys.Left)}, Desc: "move left", NoFooter: true, Do: t.do(keys.Left)},
		{Key: k[keys.Right], Keys: []string{k[keys.Right], keys.Arrow(keys.Right)}, Desc: "move right", NoFooter: true, Do: t.do(keys.Right)},
		{Key: k[keys.Down], Keys: []string{k[keys.Down], keys.Arrow(keys.Down)}, Desc: "down a row", NoFooter: true, Do: t.do(keys.Down)},
		{Key: k[keys.RotateLeft], Desc: "rotate left", Item: true, Label: keys.RotateLeft.Label(), MenuDesc: keys.RotateLeft.Desc(), Do: t.do(keys.RotateLeft)},
		{Key: k[keys.RotateRight], Desc: "rotate right", Item: true, Label: keys.RotateRight.Label(), MenuDesc: keys.RotateRight.Desc(), Do: t.do(keys.RotateRight)},
		{Key: k[keys.Drop], Desc: "drop to the bottom", NoFooter: true, Do: t.do(keys.Drop)},
	}
}

// do is a Play mode key: action a on the falling piece.
func (t *Tetru) do(a keys.Action) func(string) tea.Cmd {
	return func(string) tea.Cmd {
		switch a {
		case keys.Left:
			t.g.Move(-1)
		case keys.Right:
			t.g.Move(1)
		case keys.Down:
			t.g.Down()
			return t.after()
		case keys.RotateLeft:
			t.g.Rotate(-1)
		case keys.RotateRight:
			t.g.Rotate(1)
		case keys.Drop:
			t.g.Drop()
			return t.after()
		}
		return nil
	}
}

// after is what follows a move that may have set the piece: the stack at
// the top ends the game.
func (t *Tetru) after() tea.Cmd {
	if t.g.Over {
		return t.over()
	}
	return nil
}

// over is the stack reaching the top: out of the Play mode, the score kept,
// the curtain coming down.
func (t *Tetru) over() tea.Cmd {
	t.gameP.LeaveMode()
	t.setActions()
	return tea.Batch(t.save(), t.wake())
}

// tick is a frame of the game.
type tick struct{}

// wake starts the frames, unless they are on their way.
func (t *Tetru) wake() tea.Cmd {
	if t.ticking {
		return nil
	}
	t.ticking = true
	return t.App.Tick(frame, func(time.Time) tea.Msg { return tick{} })
}

func (t *Tetru) onMsg(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tick); ok {
		return t.step()
	}
	return nil
}

// step is a frame: the game in the Play mode goes on a frame, unless a
// popup is over it or the terminal is too small to see it; the curtain
// comes down whatever. The frames stop with nothing left to move.
func (t *Tetru) step() tea.Cmd {
	t.ticking = false
	switch {
	case t.g == nil, t.done:
		return nil
	case t.g.Over:
		if t.g.Ended() {
			return nil
		}
		t.g.Advance(frame)
		return t.wake()
	case !t.playing():
		return nil
	}
	if !t.hidden() {
		t.g.Advance(frame)
		if t.g.Over {
			return t.over()
		}
	}
	return t.wake()
}

// hidden says whether the game cannot be seen: a popup over it (the Space
// menu, ?, a confirm), or a terminal smaller than tetru's least size.
func (t *Tetru) hidden() bool {
	w, h := t.App.Size()
	return covered(t.App) || w < minW || h < minH
}
