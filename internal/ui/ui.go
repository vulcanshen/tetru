// Package ui is tetru's terminal UI on terminu-go: three screens (user,
// 2026-11-11), Tetru with the GAME panel, the pieces to come and INFO,
// Rank with the scores kept, Settings with the keys to play with;
// the Play mode the game runs in.
package ui

import (
	_ "embed"
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
	// minW, minH are tetru's least size (tdp L1, lower than 60 × 32): wide
	// enough for GAME alone (narrow), the field and the boxes, 38 columns,
	// the level, the rows and the score over them, 38, its frame and a
	// cell's margin each side; high enough for the chip row, its rule and
	// the footer, and the column at GAME's right: NEXT 1 and NEXT 2 and INFO,
	// each with its frame.
	minW, minH = 42, 3 + 2*(boxH+2) + infoRows + 2
	// narrowBelow is the width under which only the focused panel is drawn
	// (tdp screen.5's most).
	narrowBelow = 60
	// sideW is the least the column at GAME's right is given: INFO's rows
	// and its frame.
	sideW = infoW + 2
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
	infoP  *terminu.Panel
	play   *terminu.Mode
	scale  int // what a pixel of GAME and the NEXT panels is grown by

	g     *game.Game // the game, nil before the first
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
	t.infoP = &terminu.Panel{Label: "Info", Body: &info{t}}
	keysP := &terminu.Panel{Label: "Keys", Body: &terminu.Values{Header: [2]string{"Action", "Key"}, Fields: t.keyFields()}}
	t.play = &terminu.Mode{Name: "Play", Exit: "pause", Pause: true, OnExit: t.pause}
	t.setModeKeys()
	t.setActions()
	t.setRows("")
	a.AddScreen(&terminu.Screen{
		Name:        "Tetru",
		Key:         "T",
		Desc:        "the game",
		Panels:      []*terminu.Panel{t.gameP, t.next1P, t.next2P, t.infoP},
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
// of NEXT 1, NEXT 2 and INFO under one another (user, 2026-11-11), the
// column a third of the width and INFO's at least, the NEXT panels as
// high as a box. GAME and the boxes are grown twice when GAME and the
// column both have room for it.
func (t *Tetru) layout(w, h int) []terminu.Rect {
	rw := max(sideW, w/3)
	gw := w - rw
	t.scale = 1
	if gw-2 >= 2*2*fieldW && h-2 >= 2*sceneH && rw-2 >= 2*2*boxW && h >= 2*(2*boxH+2)+infoRows+2 {
		t.scale = 2
	}
	nh := boxH*t.scale + 2
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
// While playing, the NEXT panels and INFO are streaming content (tdp T2):
// the pieces to come, the level, the rows and the score change as the
// player plays on GAME, read from the corner of an eye.
func (t *Tetru) setActions() {
	t.next1P.Stream, t.next2P.Stream, t.infoP.Stream = t.playing(), t.playing(), t.playing()
	enter := terminu.Action{Key: "Enter", Label: "Play", Desc: "a new game", Scope: terminu.PanelScope, Hint: "play", Disabled: t.waiting, Do: t.enter}
	switch {
	case t.playing(), t.waiting():
		// In the Play mode no key of the panel's works, nor Enter while the
		// game over waits to be kept: no hint (tdp panel.1); the mode's keys
		// are in the footer.
		enter.Hint = ""
	case t.paused():
		enter.Label, enter.Desc, enter.Hint = "Resume", "this game, where it was", "resume"
	}
	t.gameP.Actions = []terminu.Action{
		enter,
		{Key: "n", Label: "New game", Desc: "start over from level 1", Scope: terminu.PanelScope, Disabled: t.waiting, Do: t.newGame},
		{Key: "x", Label: "End game", Desc: "keep this score and stop", Scope: terminu.PanelScope,
			Disabled: func() bool { return !t.going() }, Do: t.endGame},
	}
}

// anon is the name a score is kept for when none is given (user,
// 2026-11-11).
const anon = "ANON"

// enter is Enter on GAME: go on with the game paused, or straight into a
// new one (user, 2026-11-11: no name first; it is asked as the game ends).
func (t *Tetru) enter() tea.Cmd {
	if t.paused() {
		return t.resume()
	}
	return t.begin()
}

// begin starts a new game, in the Play mode.
func (t *Tetru) begin() tea.Cmd {
	t.g, t.saved = game.New(t.seed()), false
	return t.resume()
}

// askName is what ends a game with points: a text input (tdp text) saying
// the points and asking who played, ANON offered (tdp input.6). Enter keeps
// the score for the name typed, for ANON when there is none (user,
// 2026-11-11), then runs then; Esc runs cancel. A game with no points has
// nothing to keep: then runs at once.
func (t *Tetru) askName(title, verb string, then, cancel func() tea.Cmd) tea.Cmd {
	if t.g.Score == 0 {
		t.saved = true // nothing to keep, nobody to ask
		return then()
	}
	keep := func(name string) tea.Cmd {
		if name = strings.TrimSpace(name); name == "" {
			name = anon
		}
		return tea.Batch(t.save(name), then())
	}
	return t.App.Text(terminu.TextInput{
		Title:     title,
		Prompt:    comma(t.g.Score) + " points: who played?",
		Verb:      verb,
		Suggest:   anon,
		MaxLen:    nameLen,
		Done:      keep,
		Unchanged: func() tea.Cmd { return keep("") },
		OnCancel:  cancel,
	})
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

// newGame ends the game going, its score kept for the name asked, and
// starts the next; Esc on the name goes back to the game.
func (t *Tetru) newGame() tea.Cmd {
	if !t.going() {
		return t.begin()
	}
	return t.askName("New game", "start", func() tea.Cmd { return tea.Batch(t.finish(), t.begin()) }, nil)
}

// endGame ends the game going, its score kept for the name asked; Esc on
// the name goes back to the game.
func (t *Tetru) endGame() tea.Cmd {
	return t.askName("End game", "end", t.finish, nil)
}

// finish ends the game going, if any: out of the Play mode, the curtain
// coming down.
func (t *Tetru) finish() tea.Cmd {
	if !t.going() {
		return nil
	}
	t.g.End()
	t.gameP.LeaveMode()
	t.setActions()
	return t.wake()
}

// save keeps the game's score for name, once: a game with no points is not
// kept. The new record is the cursor's on RANK (tdp K3: the item added).
func (t *Tetru) save(name string) tea.Cmd {
	if t.g == nil || t.saved {
		return nil
	}
	t.saved = true
	t.setActions() // Enter plays again
	if t.g.Score == 0 {
		return nil
	}
	t.all = append(t.all, scores.Score{Time: time.Now().Truncate(time.Second), Name: name, Score: t.g.Score})
	t.setRows(strconv.Itoa(len(t.all) - 1))
	if err := scores.Save(t.path, t.all); err != nil {
		return t.App.Error("Can't keep the score", t.path, err.Error())
	}
	t.App.Log(terminu.Success, "Kept "+comma(t.g.Score)+" points for "+name)
	return nil
}

// Finish keeps the score of a game not kept yet when the program ends, for
// ANON: nobody was asked (Ctrl-C pressed twice, the terminal closed; user,
// 2026-11-11: leaving keeps the score as it is then, ANON for no name).
func (t *Tetru) Finish() {
	t.done = true
	if t.g != nil && !t.saved {
		if t.going() {
			t.g.End()
		}
		t.save(anon)
	}
}

// quit is the quit flow (tdp K9): with a game with points going, ask who
// played, as the game ending does, Esc staying; else leave.
func (t *Tetru) quit() tea.Cmd {
	if !t.going() || t.g.Score == 0 {
		return tea.Quit
	}
	return t.askName("Quit", "quit", func() tea.Cmd {
		t.Finish()
		return tea.Quit
	}, nil)
}

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
// from the Space menu too (tdp K11). A key that is h, j, k or l takes its
// arrow too (tdp K12): by default ↑ turns left with k, which has no moving
// to do in the mode, a piece never going up.
func (t *Tetru) setModeKeys() {
	k := t.keys
	t.play.Keys = []terminu.ModeKey{
		{Key: k[keys.Left], Keys: k.Keys(keys.Left), Desc: "move left", NoFooter: true, Do: t.do(keys.Left)},
		{Key: k[keys.Right], Keys: k.Keys(keys.Right), Desc: "move right", NoFooter: true, Do: t.do(keys.Right)},
		{Key: k[keys.Down], Keys: k.Keys(keys.Down), Desc: "down a row", NoFooter: true, Do: t.do(keys.Down)},
		{Key: k[keys.RotateLeft], Keys: k.Keys(keys.RotateLeft), Desc: "rotate left", Item: true, Label: keys.RotateLeft.Label(), MenuDesc: keys.RotateLeft.Desc(), Do: t.do(keys.RotateLeft)},
		{Key: k[keys.RotateRight], Keys: k.Keys(keys.RotateRight), Desc: "rotate right", Item: true, Label: keys.RotateRight.Label(), MenuDesc: keys.RotateRight.Desc(), Do: t.do(keys.RotateRight)},
		{Key: k[keys.Drop], Keys: k.Keys(keys.Drop), Desc: "drop to the bottom", NoFooter: true, Do: t.do(keys.Drop)},
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

// over is the stack reaching the top: out of the Play mode, the curtain
// coming down; once END is up, who played is asked (step), Esc keeping the
// score for ANON. Till then the game is waiting to be kept: Enter and New
// game do nothing. A game with no points has nothing to keep.
func (t *Tetru) over() tea.Cmd {
	t.gameP.LeaveMode()
	t.setActions()
	if t.g.Score == 0 {
		t.saved = true
	}
	return t.wake()
}

// waiting says whether a game is over and not kept yet: its curtain coming
// down, or who played being asked.
func (t *Tetru) waiting() bool { return t.g != nil && t.g.Over && !t.saved }

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
		if t.g.Ended() && !t.saved {
			// END is up: ask who played now, not as the stack reaches the
			// top, when the player may still be pressing Enter to drop.
			return t.askName("Game over", "keep", func() tea.Cmd { return nil }, func() tea.Cmd { return t.save(anon) })
		}
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
	return t.App.Covered() || w < minW || h < minH
}
