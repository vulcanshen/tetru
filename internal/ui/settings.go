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

// chooseKey opens a select of the keys offered (tdp select; tdp K11: which
// keys and how are the app's): the small letters and Enter, not the keys
// the Gaming mode keeps (Space, q, r, s), the key now chosen. A key another
// action has (or its arrow) is refused with a Warning saying whose it is,
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
				return t.App.Toast(terminu.Warning, "Can't use "+opts[i]+" for "+a.Label()+": "+t.keys.UsedBy(a, opts[i]))
			}
			t.setModeKeys()
			t.App.Log(terminu.Success, a.Label()+" is now "+opts[i])
			return nil
		},
	})
}
