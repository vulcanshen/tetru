package ui

import (
	"slices"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/keys"
)

// keyFields are the Settings panel's first group, KEYS, a value a row (tdp
// panel.12): an action and its key, Enter to choose another. No row for pausing: Esc
// pauses, leaving the Gaming mode (tdp K4), and Space opens the Space menu,
// which holds the game as any popup does (user, 2026-11-12: two things; no
// pause key of its own, Esc is it).
func (t *Tetru) keyFields() []*terminu.Field {
	var fs []*terminu.Field
	for _, a := range keys.Actions {
		fs = append(fs, &terminu.Field{
			Group: "Keys",
			Label: a.Label(),
			Desc:  a.Desc(),
			Value: func() string { return t.keys[a] },
			Edit:  func() tea.Cmd { return t.chooseKey(a) },
		})
	}
	return fs
}

// hintField is the Settings panel's second group, PLAY, one row: the
// hint's switch, one checkbox flipped in place with Enter (tdp checkbox: a switch drawn on the panel), in effect
// and saved as it is flipped (tdp panel.12). On at first, and on, the hint
// shows at every level (user, 2026-11-13: a switch in Settings, not the
// first five levels by themselves). A switch that can't be saved flips
// back, an error popup saying why (tdp E3).
func (t *Tetru) hintField() *terminu.Field {
	f := &terminu.Field{
		Group:   "Play",
		Label:   "Hint",
		Desc:    "a lighter grey where a drop would set the piece",
		Options: []string{"where a drop lands"},
		Multi:   true,
		Checked: []bool{t.hint},
	}
	f.OnChange = func() tea.Cmd {
		on := f.Checked[0]
		if err := keys.SaveHint(t.cfg, on); err != nil {
			t.setHint(t.hint)
			return t.App.Error("Can't save the hint", t.cfg, err.Error())
		}
		t.setHint(on)
		t.App.Log(terminu.Success, "The hint is now "+map[bool]string{true: "on", false: "off"}[on])
		return nil
	}
	return f
}

// setHint sets the hint's switch, and its row with it.
func (t *Tetru) setHint(on bool) {
	t.hint = on
	if t.hintF != nil {
		t.hintF.Checked = []bool{on}
	}
}

// perField is PLAY's second row: the rows a level, a radio of the five
// offered drawn in place (tdp radio.1: five at most), Enter on one choosing
// it, in effect and saved at once (tdp radio.3, panel.12) (user,
// 2026-11-14: in Settings, ten at first, the player's to set). It is the
// next game's: a game going, or saved, keeps the one it began with, its
// level counted from its rows by it. One that can't be saved goes back, an
// error popup saying why (tdp E3).
func (t *Tetru) perField() *terminu.Field {
	f := &terminu.Field{
		Group: "Play",
		Label: "Level up",
		Desc:  "the lines to clear for the next level, from the next game",
	}
	for _, n := range keys.PerLevels {
		f.Options = append(f.Options, "every "+strconv.Itoa(n)+" lines")
	}
	f.Selected = slices.Index(keys.PerLevels, t.per)
	f.OnChange = func() tea.Cmd {
		n := keys.PerLevels[f.Selected]
		if err := keys.SavePerLevel(t.cfg, n); err != nil {
			t.setPerLevel(t.per)
			return t.App.Error("Can't save the lines a level", t.cfg, err.Error())
		}
		t.setPerLevel(n)
		t.App.Log(terminu.Success, "A level is now "+strconv.Itoa(n)+" lines, from the next game")
		return nil
	}
	return f
}

// setPerLevel sets the rows a level of the next game, and its row with it.
func (t *Tetru) setPerLevel(n int) {
	t.per = n
	if t.perF != nil {
		t.perF.Selected = slices.Index(keys.PerLevels, n)
	}
}

// chooseKey opens a select of the keys offered (tdp select; tdp K11: which
// keys and how are the app's): the small letters and Enter, not the keys
// the Gaming mode keeps (Space, q, r), the key now chosen. A key another
// action has is refused with a Warning saying whose it is,
// and not set (tdp K11). Another is written to the settings file
// before the popup closes: a failure keeps it open under an error popup
// (tdp popup.7).
func (t *Tetru) chooseKey(a keys.Action) tea.Cmd {
	opts := keys.Choices()
	return t.App.Select(terminu.Select{
		Title:    "Key",
		Subject:  a.Label(),
		Options:  opts,
		Selected: slices.Index(opts, t.keys[a]),
		Commit: func(i int) error {
			if t.keys.UsedBy(a, opts[i]) != "" {
				return nil // refused, said by Done
			}
			s := t.keys
			s[a] = opts[i]
			if err := keys.Save(t.cfg, s); err != nil {
				return err
			}
			t.keys = s
			return nil
		},
		Done: func(i int) tea.Cmd {
			if t.keys[a] != opts[i] {
				return t.App.Toast(terminu.Warning, "Can't use ["+opts[i]+"] for "+a.Label()+": "+t.keys.UsedBy(a, opts[i]))
			}
			t.setModeKeys()
			t.App.Log(terminu.Success, a.Label()+" is now ["+opts[i]+"]")
			return nil
		},
	})
}
