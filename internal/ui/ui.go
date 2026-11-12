// Package ui is tetru's terminal UI on terminu-go: three screens (user,
// 2026-11-11), Tetru with the PLAYGROUND panel, the pieces to come and INFO,
// Rank with the scores kept, Settings with the keys to play with;
// the Gaming mode the game runs in.
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
	"github.com/vulcanshen/tetru/internal/saved"
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
	// enough for PLAYGROUND alone (narrow), the field and the boxes, 38 columns,
	// the level, the rows and the score over them, 38, its frame and a
	// cell's margin each side; high enough for the chip row, its rule and
	// the footer, PLAYGROUND's frame and the column in it at the right: Next 1
	// and Next 2, INFO and KEYS, each with its frame.
	minW, minH = 42, 3 + 2 + 2*(boxH+2) + infoH
	// narrowBelow is the width under which PLAYGROUND draws the boxes with no
	// frames and no INFO (tdp screen.5's most).
	narrowBelow = 60
	// sideW is how wide the boxes at PLAYGROUND's right are, the pieces in
	// them drawn once: INFO's rows and its frame.
	sideW = infoW + 2
	// frame is a step of the game, as locku's; clearFrame is one while rows
	// clear, a frame of a screen at 60 Hz, so the rows slide out a pixel or
	// two a frame (user, 2026-11-11: the clear not smooth).
	frame      = 40 * time.Millisecond
	clearFrame = 16 * time.Millisecond
	// nameLen is the most characters a name has.
	nameLen = 16
)

// Tetru is the app and the game in it.
type Tetru struct {
	App *terminu.App

	path     string         // the scores' file
	all      []scores.Score // the records, as kept
	loadErr  error          // why the file could not be read, at start
	cfg      string         // the settings file
	keys     keys.Set       // the keys to play with
	cfgErr   error          // why the settings could not be read, at start
	stash    string         // the game saved to go on with
	stashErr error          // why it could not be read, at start
	rmErr    error          // why it could not be removed once taken up, at start

	list  *terminu.List
	rankP *terminu.Panel
	gameP *terminu.Panel
	play  *terminu.Mode

	g      *game.Game // the game, nil before the first
	saved  bool       // its score is kept (or it was not worth keeping, or it is saved to go on with)
	topped bool       // it ended reaching the top: a popup says so once the curtain is down

	ticking bool          // a frame is on its way
	done    bool          // the program is ending: no more frames
	seed    func() uint64 // the next game's seed
}

// New is tetru, all but Run.
func New() *terminu.App { return NewTetru().App }

// NewTetru is tetru with its game, for main to keep the score of a game
// still going when the program ends (Finish).
func NewTetru() *Tetru {
	t := &Tetru{App: terminu.New("tetru"), path: scores.Path(), cfg: keys.Path(), stash: saved.Path(), seed: rand.Uint64}
	t.all, t.loadErr = scores.Load(t.path)
	// The best twenty are kept (user, 2026-11-12); a file of more, from
	// before, is cut to them the next time a score is kept.
	t.all = scores.Best(t.all)
	// A game saved is taken up again, paused, and its file removed (user,
	// 2026-11-11: Save keeps it and quits; the next start goes on with it).
	var sg *saved.Game
	if sg, t.stashErr = saved.Load(t.stash); sg != nil {
		t.g = game.Resume(t.seed(), sg.Level, sg.Lines, sg.Score, sg.Stack)
		t.rmErr = saved.Remove(t.stash)
		t.App.Log(terminu.Info, "Took up the game saved: level "+strconv.Itoa(sg.Level)+", "+comma(sg.Score)+" points")
	}
	if t.keys, t.cfgErr = keys.Load(t.cfg); t.cfgErr != nil {
		t.keys = keys.Default()
	}
	a := t.App
	a.MinWidth, a.MinHeight = minW, minH
	a.QuitFlow, a.QuitDesc = t.quit, quitDesc
	a.OnMsg = t.onMsg
	a.Start = t.start

	t.list = &terminu.List{
		Columns: []terminu.Column{
			{Title: "#", Number: true}, // the place (user, 2026-11-12)
			{Title: "Name", Shrink: true, MinWidth: 6},
			{Title: "Score", Number: true},
			{Title: "Time", Drop: 1},
		},
		EmptyFact: "No scores yet",
		EmptyHint: "Press [T] to play",
	}
	t.rankP = &terminu.Panel{Label: "Rank", Body: t.list}
	t.gameP = &terminu.Panel{Label: "Playground", Body: &field{t}, OnEsc: func() tea.Cmd { return nil }}
	keysP := &terminu.Panel{Label: "Keys", Body: &terminu.Values{Header: [2]string{"Action", "Key"}, Fields: t.keyFields()}}
	// The mode's Space menu lists Restart and Quit alone (user, 2026-11-12;
	// tdp K11: what an app's own mode lists there is the app's): not the
	// keys to play, nor [Esc] Pause, which the footer has. Tab and the
	// digits are not held: with PLAYGROUND the screen's one panel they do
	// nothing out of the mode, so nothing in it either, and say nothing
	// (tdp K11-5, its next text).
	t.play = &terminu.Mode{Name: "Gaming", Exit: "pause", NoLeaveRow: true, OnExit: t.pause}
	t.setModeKeys()
	t.setActions()
	t.setRows("")
	// PLAYGROUND is the screen's only panel: Next 1, Next 2 and INFO are boxes
	// it draws, with no [N], the focus never on them (user, 2026-11-11).
	a.AddScreen(&terminu.Screen{
		Name:        "Tetru",
		Key:         "T",
		Desc:        "the game",
		Panels:      []*terminu.Panel{t.gameP},
		NarrowBelow: narrowBelow,
	})
	a.AddScreen(&terminu.Screen{Name: "Rank", Key: "R", Desc: "the scores, best first", Panels: []*terminu.Panel{t.rankP}})
	a.AddScreen(&terminu.Screen{Name: "Settings", Key: "S", Desc: "the keys to play with", Panels: []*terminu.Panel{keysP}})
	var err error
	if a.Splash, err = terminu.NewSplash(icon, "tetru", version.Display(), "A falling-block game for the terminal"); err != nil {
		panic(err) // the embedded icon is broken: a mistake in the program
	}
	return t
}

// start puts the focus on PLAYGROUND (opening tetru is to play), and asks to move
// aside the files that could not be read.
func (t *Tetru) start() tea.Cmd {
	return tea.Batch(t.App.Focus(t.gameP), t.moveAside())
}

// moveAside asks to move aside a file that could not be read (tdp E3), one
// after the other: the settings, the scores, the game saved; then says a
// game saved and taken up could not be removed.
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
			return t.moveAside()
		})
	case t.stashErr != nil:
		err := t.stashErr
		t.stashErr = nil
		return t.App.MoveAside(t.stash, err, func() tea.Cmd { return nil })
	case t.rmErr != nil:
		err := t.rmErr
		t.rmErr = nil
		return t.App.Error("Can't remove the game saved", t.stash, err.Error(), "It is taken up again the next time tetru starts.")
	}
	return nil
}

// narrow says whether PLAYGROUND draws the boxes with no frames and no INFO
// (tdp screen.5's width).
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

// playing says whether the game is in the Gaming mode.
func (t *Tetru) playing() bool { return t.going() && t.gameP.InMode() == t.play }

// paused says whether there is a game not ended out of the Gaming mode.
func (t *Tetru) paused() bool { return t.going() && t.gameP.InMode() != t.play }

// setActions are PLAYGROUND's actions as the game stands: Enter plays a new
// game, or goes on with the one paused (tdp K3); New game and End game.
func (t *Tetru) setActions() {
	enter := terminu.Action{Key: "Enter", Label: "Play", Desc: "a new game", Scope: terminu.PanelScope, Hint: "play", Disabled: t.waiting, Do: t.enter}
	switch {
	case t.playing(), t.waiting():
		// In the Gaming mode no key of the panel's works, nor Enter while the
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

// keepGame keeps the game going in the file of the game saved, its level,
// rows, score and stack, for the next start to go on with; it is not kept
// in the rank now, but once it ends after it is taken up again. Nobody
// saves by hand: tetru does it as it quits with a game going (user,
// 2026-11-12: save and quit are two things, a stray s mid-game must not
// quit; quitting keeps the game, and a player who does not want it next
// time starts over).
func (t *Tetru) keepGame() error {
	g := t.g
	if err := saved.Save(t.stash, saved.Game{Level: g.Level, Lines: g.Lines, Score: g.Score, Stack: g.Stack()}); err != nil {
		return err
	}
	t.saved = true
	t.App.Log(terminu.Success, "Saved the game: level "+strconv.Itoa(g.Level)+", "+comma(g.Score)+" points")
	return nil
}

// anon is the name a score is kept for when none is given (user,
// 2026-11-11).
const anon = "ANON"

// enter is Enter on PLAYGROUND: go on with the game paused, or straight into a
// new one (user, 2026-11-11: no name first; it is asked as the game ends).
func (t *Tetru) enter() tea.Cmd {
	if t.paused() {
		return t.resume()
	}
	return t.begin()
}

// begin starts a new game, in the Gaming mode.
func (t *Tetru) begin() tea.Cmd {
	t.g, t.saved, t.topped = game.New(t.seed()), false, false
	return t.resume()
}

// askName is what ends a game with points the player ends (End game, New
// game): a text input (tdp text), titled by what it does (tdp popup.1),
// saying the points and asking who played, ANON offered (tdp input.6).
// Enter keeps the score for the name typed, for ANON when there is none
// (user, 2026-11-11), then runs then; Esc cancels, back to the game. A game
// not worth keeping (worth) has nothing to keep: then runs at once, an Info
// toast saying so when it has points.
func (t *Tetru) askName(title, verb string, then func() tea.Cmd) tea.Cmd {
	if !t.worth() {
		t.saved = true // nothing to keep, nobody to ask
		if t.g.Score == 0 {
			return then()
		}
		return tea.Batch(t.App.Toast(terminu.Info, t.notKept()+", so nothing is kept."), then())
	}
	keep := func(name string) tea.Cmd {
		if name = strings.TrimSpace(name); name == "" {
			name = anon
		}
		return tea.Batch(t.save(name), then())
	}
	return t.App.Text(terminu.TextInput{
		Title:     title,
		Name:      "Name",
		Prompt:    comma(t.g.Score) + " points: who played?",
		Verb:      verb,
		Suggest:   anon,
		MaxLen:    nameLen,
		Done:      keep,
		Unchanged: func() tea.Cmd { return keep("") },
	})
}

// keepThenName ends a game the stack ended that is worth keeping: its score
// is kept at once, for ANON, then a text input titled by what it does, KEEP
// SCORE, asks who played, only to put the name on it: Enter puts the name
// typed, Esc leaves it ANON, the value unchanged (tdp input.1; tdp #10: the
// game is over and kept, nothing for Esc to cancel but the name).
func (t *Tetru) keepThenName() tea.Cmd {
	if cmd := t.save(anon); cmd != nil {
		return tea.Batch(t.leaveOver(), cmd)
	}
	i := len(t.all) - 1 // the new one, last as kept
	return t.App.Text(terminu.TextInput{
		Title:     "Keep score",
		Name:      "Name",
		Prompt:    "Game over, " + comma(t.g.Score) + " points: who played?",
		Verb:      "keep",
		Suggest:   anon,
		MaxLen:    nameLen,
		Done:      func(name string) tea.Cmd { return tea.Batch(t.rename(i, name), t.leaveOver()) },
		Unchanged: t.leaveOver,
	})
}

// leaveOver leaves the Gaming mode of a game over once it is kept: back to
// PLAYGROUND, Enter to play again (tdp K11: the mode's name and colour
// back as it is left). Esc on the popup of the end only closes it, still
// in the mode, a layer at a time (tdp K4); a second Esc leaves.
func (t *Tetru) leaveOver() tea.Cmd {
	t.gameP.LeaveMode()
	t.setActions()
	return nil
}

// rename puts name on score i, kept for ANON as its game ended; no name,
// it stays ANON.
func (t *Tetru) rename(i int, name string) tea.Cmd {
	if name = strings.TrimSpace(name); name == "" || name == anon || i < 0 || i >= len(t.all) {
		return nil
	}
	t.all[i].Name = name
	t.setRows(strconv.Itoa(i))
	if err := scores.Save(t.path, t.all); err != nil {
		return t.App.Error("Can't keep the score", t.path, err.Error())
	}
	t.App.Log(terminu.Success, "Kept "+comma(t.all[i].Score)+" points for "+name+", not "+anon)
	return nil
}

// resume goes into the Gaming mode: the game runs.
func (t *Tetru) resume() tea.Cmd {
	t.gameP.EnterMode(t.play)
	t.setActions()
	return t.wake()
}

// pause is Esc leaving the Gaming mode (tdp K4): the game waits, and a
// confirm says so (user, 2026-11-11: a pause is a popup, not drawn in the
// game), the level, the rows and the score in it. A confirm is titled by
// what Enter does (tdp confirm.2): RESUME; Esc leaves the game paused,
// Enter on PLAYGROUND going on then.
func (t *Tetru) pause() tea.Cmd {
	t.setActions()
	if !t.paused() {
		return nil
	}
	return t.App.Confirm(terminu.Ask{
		Title:    "Resume",
		Question: "Go on with the game?",
		Verb:     "resume",
		Details:  []string{fmt.Sprintf("Paused: level %d, %d lines, %s points.", t.g.Level, t.g.Lines, comma(t.g.Score))},
	}, func() tea.Cmd {
		if t.paused() {
			return t.resume()
		}
		return nil
	})
}

// newGame ends the game going, its score kept for the name asked, and
// starts the next; Esc on the name goes back to the game. A game not worth
// keeping starts over at once, nothing said (user, 2026-11-12: restarting,
// no Info that it is not in the rank).
func (t *Tetru) newGame() tea.Cmd {
	if !t.going() {
		return t.begin()
	}
	if !t.worth() {
		t.saved = true
		return tea.Batch(t.finish(), t.begin())
	}
	return t.askName("New game", "start", func() tea.Cmd { return tea.Batch(t.finish(), t.begin()) })
}

// endGame ends the game going, its score kept for the name asked; Esc on
// the name goes back to the game.
func (t *Tetru) endGame() tea.Cmd {
	return t.askName("End game", "end", t.finish)
}

// finish ends the game going, if any: out of the Gaming mode, the curtain
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

// worth says whether the game's score is kept as it ends: it has points and
// gets into the best twenty (user, 2026-11-12: the best twenty are kept).
func (t *Tetru) worth() bool { return t.g.Score > 0 && scores.Makes(t.all, t.g.Score) }

// notKept says why a game not worth keeping is not: no points, or too few
// for the best twenty.
func (t *Tetru) notKept() string {
	if t.g.Score == 0 {
		return "no points"
	}
	return comma(t.g.Score) + " points, not in the best " + strconv.Itoa(scores.Most)
}

// save keeps the game's score for name, once, if it is worth keeping; the
// worst of the twenty-one then goes. The new record is the cursor's on RANK
// (tdp K3: the item added).
func (t *Tetru) save(name string) tea.Cmd {
	if t.g == nil || t.saved {
		return nil
	}
	t.saved = true
	t.setActions() // Enter plays again
	if !t.worth() {
		return nil
	}
	t.all = scores.Best(append(t.all, scores.Score{Time: time.Now().Truncate(time.Second), Name: name, Score: t.g.Score}))
	t.setRows(strconv.Itoa(len(t.all) - 1)) // the new one, last as kept
	if err := scores.Save(t.path, t.all); err != nil {
		return t.App.Error("Can't keep the score", t.path, err.Error())
	}
	t.App.Log(terminu.Success, "Kept "+comma(t.g.Score)+" points for "+name)
	return nil
}

// Finish, as the program ends (Ctrl-C pressed twice, the terminal closed),
// keeps a game going for the next start, as quitting does (keepGame); a
// game that can't be kept going, or one over and not kept yet, keeps its
// score for ANON, nobody asked (user, 2026-11-11).
func (t *Tetru) Finish() {
	t.done = true
	if t.g == nil || t.saved {
		return
	}
	if t.going() {
		if t.keepGame() == nil {
			return
		}
		t.g.End()
	}
	t.save(anon)
}

// quitDesc says what Quit does, in the global operation and the Gaming
// mode's Space menu.
const quitDesc = "leave tetru; a game going goes on next time"

// quitKeep quits keeping the game going: what the quit flow does, as a
// message, so that with a popup open it is done only once Quit tetru? is
// answered (tgo QuitFlow; tdp K9).
type quitKeep struct{}

// quit is the quit flow (tdp K9: the app's): leave, asking nothing; a game
// going is kept for the next start (quitKeep, keepGame; user, 2026-11-12:
// nobody asked, a player who does not want it then starts over).
func (t *Tetru) quit() tea.Cmd {
	if !t.going() {
		return tea.Quit
	}
	return func() tea.Msg { return quitKeep{} }
}

// keepAndQuit keeps the game going and quits; when it can't be written, an
// error popup says so and tetru stays, the game as it was.
func (t *Tetru) keepAndQuit() tea.Cmd {
	if t.going() && !t.saved {
		if err := t.keepGame(); err != nil {
			return t.App.Error("Can't save the game", t.stash, err.Error(), "[Ctrl-C] twice quits all the same, the score so far kept for ANON.")
		}
	}
	t.done = true
	return tea.Quit
}

// setRows lists the records best first, each with its place (user,
// 2026-11-12); sel is the ID of the row to put the cursor on, "" to leave
// it.
func (t *Tetru) setRows(sel string) {
	var rows []terminu.Row
	for n, i := range scores.Ranked(t.all) {
		s := t.all[i]
		r := terminu.Row{ID: strconv.Itoa(i), Cells: []string{strconv.Itoa(n + 1), s.Name, comma(s.Score), s.Time.Local().Format("2006-01-02 15:04")}}
		if n < len(podium) {
			for _, c := range r.Cells {
				r.Styled = append(r.Styled, []terminu.Span{{Text: c, Style: terminu.Fg(podium[n])}})
			}
			r.CursorColor = podium[n] // the row's own colour, none other (tdp color.2)
		}
		rows = append(rows, r)
	}
	t.list.SetRows(rows)
	if sel != "" {
		t.list.Select(sel)
	}
}

// podium are the colours of the first three places' rows on RANK (user,
// 2026-11-12): Yellow, Green, Sky; a content's colours, the app's (tdp P4,
// P5). The cursor on one of them is in its colour too, Base on it (user,
// 2026-11-12; tdp #8, color.2): Subtext1 again while RANK is in a mode,
// dimmed with it unfocused, both by tgo.
var podium = []terminu.Color{terminu.Yellow, terminu.Green, terminu.Hex("#89dceb")}

// setModeKeys are the Gaming mode's keys, the player's (Settings): the moves
// with their arrows too (tdp K12) are moving, not in the footer, as Drag's
// j|k are not (tdp screen.2); the turns and the drop are things done to the
// falling piece, in the footer as Drag's Enter:drop is. None of these is in
// the mode's Space menu (user, 2026-11-11: it has Restart and Save; tdp K11,
// an app's own mode), all of them in its ? and in KEYS. A key that is h, j,
// k or l takes its arrow too (tdp K12): by default ↑ turns left with k,
// which has no moving to do in the mode, a piece never going up.
func (t *Tetru) setModeKeys() {
	k := t.keys
	t.play.Keys = []terminu.ModeKey{
		{Key: k[keys.Left], Keys: k.Keys(keys.Left), Desc: "move left", NoFooter: true, NoMenu: true, Disabled: t.still, Do: t.do(keys.Left)},
		{Key: k[keys.Right], Keys: k.Keys(keys.Right), Desc: "move right", NoFooter: true, NoMenu: true, Disabled: t.still, Do: t.do(keys.Right)},
		{Key: k[keys.Down], Keys: k.Keys(keys.Down), Desc: "down a row", NoFooter: true, NoMenu: true, Disabled: t.still, Do: t.do(keys.Down)},
		{Key: k[keys.RotateLeft], Keys: k.Keys(keys.RotateLeft), Desc: "rotate left", Item: true, NoMenu: true, Label: keys.RotateLeft.Label(), MenuDesc: keys.RotateLeft.Desc(), Disabled: t.still, Do: t.do(keys.RotateLeft)},
		{Key: k[keys.RotateRight], Keys: k.Keys(keys.RotateRight), Desc: "rotate right", Item: true, NoMenu: true, Label: keys.RotateRight.Label(), MenuDesc: keys.RotateRight.Desc(), Disabled: t.still, Do: t.do(keys.RotateRight)},
		{Key: k[keys.Drop], Keys: k.Keys(keys.Drop), Desc: "drop", NoMenu: true, Disabled: t.still, Do: t.do(keys.Drop)},
		{Key: "r", Desc: "restart", Label: "Restart", MenuDesc: "start over from level 1", Disabled: t.waiting, Do: func(string) tea.Cmd { return t.newGame() }},
		// Quit, as q does in the mode (tdp K11), listed in its Space menu (user,
		// 2026-11-12: Restart and Quit there).
		{Key: "q", Desc: "quit", Label: "Quit", MenuDesc: quitDesc, Do: func(string) tea.Cmd { return t.quit() }},
	}
}

// do is a Gaming mode key: action a on the falling piece.
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

// over is the stack reaching the top: still in the Gaming mode, the
// curtain coming down; once it is down, who played is asked, or Play again?
// (step); keeping the score leaves the mode (leaveOver), Esc on the popup
// only closes it, a second Esc leaves (user, 2026-11-12: it is all still
// the game, so the mode, the curtain down too). Till then the game is
// waiting to be kept: its keys are disabled, nothing to move (still), and
// Restart, nor Enter and New game should Esc leave the mode. A game not
// worth keeping has nothing to keep.
func (t *Tetru) over() tea.Cmd {
	t.topped = true
	if !t.worth() {
		t.saved = true
	}
	t.setActions()
	return t.wake()
}

// still says whether the game has nothing to move now: rows going, or the
// game over. The keys to play are disabled then, in the disabled colour in
// the footer and ? (tdp M6).
func (t *Tetru) still() bool { return t.g == nil || !t.g.Live() }

// waiting says whether a game is over and not kept yet: its curtain coming
// down, or who played being asked.
func (t *Tetru) waiting() bool { return t.g != nil && t.g.Over && !t.saved }

// tick is a frame of the game, dt long.
type tick struct{ dt time.Duration }

// wake starts the frames, unless they are on their way: clearFrame while
// rows clear, frame otherwise.
func (t *Tetru) wake() tea.Cmd {
	if t.ticking {
		return nil
	}
	t.ticking = true
	dt := frame
	if t.g != nil && len(t.g.Going) > 0 {
		dt = clearFrame
	}
	return t.App.Tick(dt, func(time.Time) tea.Msg { return tick{dt} })
}

func (t *Tetru) onMsg(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case tick:
		return t.step(m.dt)
	case quitKeep:
		return t.keepAndQuit()
	}
	return nil
}

// step is a frame, dt long: the game in the Gaming mode goes on dt, unless a
// popup is over it or the terminal is too small to see it; the curtain
// comes down whatever. The frames stop with nothing left to move.
func (t *Tetru) step(dt time.Duration) tea.Cmd {
	t.ticking = false
	switch {
	case t.g == nil, t.done:
		return nil
	case t.g.Over:
		if t.g.Ended() {
			return nil
		}
		t.g.Advance(dt)
		if t.g.Ended() && t.topped {
			// The curtain is down: the end is said now, not as the stack
			// reaches the top, when the player may still be pressing Enter to
			// drop (user, 2026-11-11: the end is a popup, not drawn in the
			// game). Worth keeping, the score is kept and who played asked (keepThenName); else a confirm says
			// why nothing is kept (no points, or not in the best twenty) and
			// offers the next game, titled by what Enter does (tdp
			// confirm.2): PLAY.
			t.topped = false
			if !t.saved {
				return t.keepThenName()
			}
			return t.App.Confirm(terminu.Ask{Title: "Play", Question: "Play again?", Verb: "play", Details: []string{"Game over: " + t.notKept() + ", so nothing is kept."}}, t.begin)
		}
		return t.wake()
	case !t.playing():
		return nil
	}
	if !t.hidden() {
		t.g.Advance(dt)
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
