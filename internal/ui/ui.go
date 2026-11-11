// Package ui is tetru's terminal UI: the GAME panel with the field and the
// pieces to come, the SCORES panel at its left, the Play mode the game runs
// in, and the scores kept, on terminu-go.
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
	// minW, minH are tetru's least size (tdp L1, lower than 60 × 32): the
	// GAME panel alone, the picture and its panel's frame, the statusbar
	// and the footer.
	minW, minH = 2*sceneW + 4, sceneH + 4
	// scoresW is the least the SCORES panel is given beside GAME: a name and
	// a score.
	scoresW = 18
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

	list   *terminu.List
	scoreP *terminu.Panel
	gameP  *terminu.Panel
	play   *terminu.Mode

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
	t := &Tetru{App: terminu.New("tetru"), path: scores.Path(), seed: rand.Uint64}
	t.all, t.loadErr = scores.Load(t.path)
	a := t.App
	a.MinWidth, a.MinHeight = minW, minH
	a.Status = t.status
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
		EmptyHint: "Press [Enter] on [2] GAME to play",
	}
	t.scoreP = &terminu.Panel{Label: "Scores", Body: t.list}
	t.gameP = &terminu.Panel{Label: "Game", Body: &field{t}, OnEsc: func() tea.Cmd { return nil }}
	t.play = &terminu.Mode{
		Name:   "Play",
		Exit:   "pause",
		Pause:  true,
		OnExit: t.pause,
		Keys: []terminu.ModeKey{
			{Key: "h|l", Keys: []string{"h", "l", "←", "→"}, Desc: "move left, right", NoFooter: true, Do: t.move},
			{Key: "j", Keys: []string{"j", "↓"}, Desc: "down a row", NoFooter: true, Do: t.down},
			{Key: "d", Keys: []string{"d", "PgDn"}, Desc: "drop to the bottom", NoFooter: true, Do: t.drop},
			{Key: "e", Desc: "rotate left", Item: true, Label: "Rotate left", MenuDesc: "the falling piece, counterclockwise", Do: t.rotate},
			{Key: "r", Desc: "rotate right", Item: true, Label: "Rotate right", MenuDesc: "the falling piece, clockwise", Do: t.rotate},
		},
	}
	t.setActions()
	t.setRows("")
	a.AddScreen(&terminu.Screen{
		Name:        "Tetru",
		Panels:      []*terminu.Panel{t.scoreP, t.gameP},
		NarrowBelow: 60,
		Layout:      t.layout,
	})
	var err error
	if a.Splash, err = terminu.NewSplash(icon, "tetru", version.Display(), "A falling-block game for the terminal"); err != nil {
		panic(err) // the embedded icon is broken: a mistake in the program
	}
	return t
}

// start puts the focus on GAME (opening tetru is to play), and, when the
// scores could not be read, asks to move the file aside (tdp E3).
func (t *Tetru) start() tea.Cmd {
	cmd := t.App.Focus(t.gameP)
	if t.loadErr != nil {
		err := t.loadErr
		t.loadErr = nil
		return tea.Batch(cmd, t.App.MoveAside(t.path, err, func() tea.Cmd {
			t.all = nil
			t.setRows("")
			return nil
		}))
	}
	return cmd
}

// layout is SCORES at the left and GAME at the right, GAME as wide as its
// picture, SCORES what is left (user, 2026-11-11: at the game's right go
// the pieces to come; the scores matter less while playing).
func (t *Tetru) layout(w, h int) []terminu.Rect {
	gw := min(w, panelWidth(w-scoresW, h))
	return []terminu.Rect{{X: 0, Y: 0, W: w - gw, H: h}, {X: w - gw, Y: 0, W: gw, H: h}}
}

// status is the statusbar: the level, the rows cleared and the score of
// the game, or of the last one; values in Lavender, where the game is
// (tdp screen.4), Level and Lines a fixed width (tdp L2).
func (t *Tetru) status(int) []terminu.Span {
	level, lines, score := 1, 0, 0
	if t.g != nil {
		level, lines, score = t.g.Level, t.g.Lines, t.g.Score
	}
	q, v := terminu.Fg(terminu.Overlay0), terminu.Fg(terminu.Lavender)
	return []terminu.Span{
		{Text: "Level: ", Style: q}, {Text: fmt.Sprintf("%-2d", level), Style: v},
		{Text: "  Lines: ", Style: q}, {Text: fmt.Sprintf("%-5d", lines), Style: v},
		{Text: "  Score: ", Style: q}, {Text: comma(score), Style: v},
	}
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
func (t *Tetru) setActions() {
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

// The Play mode's keys.

func (t *Tetru) move(key string) tea.Cmd {
	if key == "h" || key == "←" {
		t.g.Move(-1)
	} else {
		t.g.Move(1)
	}
	return nil
}

func (t *Tetru) down(string) tea.Cmd { t.g.Down(); return t.after() }

func (t *Tetru) drop(string) tea.Cmd { t.g.Drop(); return t.after() }

func (t *Tetru) rotate(key string) tea.Cmd {
	if key == "e" {
		t.g.Rotate(-1)
	} else {
		t.g.Rotate(1)
	}
	return nil
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
