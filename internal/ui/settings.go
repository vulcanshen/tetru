package ui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/tetru/internal/keys"
)

// keyFields are the Settings screen's rows, a value a row (tdp panel.12):
// an action and its key, Enter to choose another. No row for pausing: Esc
// pauses, leaving the Gaming mode (tdp K4), and Space opens the Space menu,
// which holds the game as any popup does (user, 2026-11-12: two things; no
// pause key of its own, Esc is it).
func (t *Tetru) keyFields() []*terminu.Field {
	var fs []*terminu.Field
	for _, a := range keys.Actions {
		fs = append(fs, &terminu.Field{
			Label: a.Label(),
			Desc:  a.Desc(),
			Value: func() string { return t.keys[a] },
			Edit:  func() tea.Cmd { return t.chooseKey(a) },
		})
	}
	return fs
}

// hintField is PLAY's row: the hint's switch, one checkbox flipped in
// place with Enter (tdp checkbox: a switch drawn on the panel), in effect
// and saved as it is flipped (tdp panel.12). On at first, and on, the hint
// shows at every level (user, 2026-11-13: a switch in Settings, not the
// first five levels by themselves). A switch that can't be saved flips
// back, an error popup saying why (tdp E3).
func (t *Tetru) hintField() *terminu.Field {
	f := &terminu.Field{
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

// setHint sets the hint's switch, and PLAY's row with it.
func (t *Tetru) setHint(on bool) {
	t.hint = on
	if t.hintF != nil {
		t.hintF.Checked = []bool{on}
	}
}

// settingsLayout lays the Settings screen out (tdp L1: the app's): KEYS the
// whole width, as high as its rows, PLAY under it, the rest.
// TODO(tdp #15): one panel, the keys and the hint two sections of it (user,
// 2026-11-13), once tdp and terminu-go have sections in a panel of a value
// a row.
func settingsLayout(w, h int) []terminu.Rect {
	top := min(h-3, 3+len(keys.Actions)) // the frame, the header, a row an action
	return []terminu.Rect{{W: w, H: top}, {Y: top, W: w, H: h - top}}
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
