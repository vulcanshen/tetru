// Package keys is the keys tetru is played with: the player chooses them
// (user, 2026-11-11: all but Space, which pauses and opens the Space
// menu), and they are kept in tetru's settings (tdp E3), with the one
// other setting, the hint's switch (user, 2026-11-13).
package keys

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Action is a thing the player does to the falling piece.
type Action int

// The actions, in the order the settings list them.
const (
	Left Action = iota
	Right
	Down
	RotateLeft
	RotateRight
	Drop
)

// Actions are all of them, in that order.
var Actions = []Action{Left, Right, Down, RotateLeft, RotateRight, Drop}

var about = [...]struct{ id, label, desc string }{
	{"move_left", "Move left", "the falling piece a column left"},
	{"move_right", "Move right", "the falling piece a column right"},
	{"down", "Down a row", "the falling piece a row down"},
	{"rotate_left", "Rotate left", "the falling piece, counterclockwise"},
	{"rotate_right", "Rotate right", "the falling piece, clockwise"},
	{"drop", "Drop", "the falling piece to the bottom, set there"},
}

// ID is a's key in the settings file: move_left.
func (a Action) ID() string { return about[a].id }

// Label is a's name: Move left.
func (a Action) Label() string { return about[a].label }

// Desc says what a does.
func (a Action) Desc() string { return about[a].desc }

// Set is a key for each action.
type Set [len(about)]string

// Default is the keys the user asked for (2026-11-11): h left, l right, j a
// row down, d rotate left, k rotate right, Enter drop. d and k have no moving
// to do in the Gaming mode, a piece never going up nor a half page down: the
// mode may take them (tdp K12).
func Default() Set { return Set{"h", "l", "j", "d", "k", "Enter"} }

// arrows are the keys that move as a letter does: h|j|k|l are ←|↓|↑|→,
// u|d are PgUp|PgDn (tdp K12).
var arrows = map[string]string{"h": "←", "j": "↓", "k": "↑", "l": "→", "u": "PgUp", "d": "PgDn"}

// Keys are the keys that do a in s: its key, and the arrow of its default
// key when it has one: ← left, → right, ↓ a row down, ↑ rotate right as k
// does, PgDn rotate left as d does (tdp K12). The player chooses no arrow,
// so each keeps its default whatever letter the action is on (tdp #14,
// user 2026-11-13: Move left on a, ← still moves left).
func (s Set) Keys(a Action) []string {
	if arrow, ok := arrows[Default()[a]]; ok {
		return []string{s[a], arrow}
	}
	return []string{s[a]}
}

// Shown is how a's keys are written in its own row of the Gaming mode's ?:
// its key, and, once a is off its default letter, the arrow that keeps
// doing it after a | (Enter|PgDn). On its default letter the arrow goes
// unwritten, the same key as the letter (tdp K11, K12). For the turns, two
// rows tetru writes itself (tgo ModeKey.Ref); tgo writes the moves' arrows.
func (s Set) Shown(a Action) string {
	if arrow, ok := arrows[Default()[a]]; ok && s[a] != Default()[a] {
		return s[a] + "|" + arrow
	}
	return s[a]
}

// core are the keys the Gaming mode keeps for itself, and what each does
// there (tdp K11): the core keys, and Restart (user, 2026-11-11; Save, s,
// gone 2026-11-12: tetru saves as it quits).
// A key the player can't have, so not offered (Choices).
var core = map[string]string{"Space": "it pauses and opens the Space menu", "q": "it quits", "r": "it restarts"}

// Choices are the keys offered to the player for an action: the small
// letters and Enter (tdp K11: which keys the player may choose is the
// app's), none of the core ones (user, 2026-11-12: a key that can't be
// used is not in the list). One another action has is refused (UsedBy).
// Enter and the moves are the player's too in an app's own mode, a letter
// so freed having nothing to move; a single g as well, Gaming having no
// chord starting with g (tdp v1.2.7 K11, K12; tdp #14).
func Choices() []string {
	var out []string
	for c := 'a'; c <= 'z'; c++ {
		if core[string(c)] == "" {
			out = append(out, string(c))
		}
	}
	return append(out, "Enter")
}

// UsedBy says what already has key k in the Gaming mode, other than action a:
// another action or a core key (tdp K11); "" when nothing has it.
func (s Set) UsedBy(a Action, k string) string {
	if why, ok := core[k]; ok {
		return why
	}
	for _, b := range Actions {
		if b != a && slices.Contains(s.Keys(b), k) {
			return "it is " + b.Label() + "'s key"
		}
	}
	return ""
}

// Check says what is wrong with s: a key its action can't have, or a key
// two actions have.
func (s Set) Check() error {
	by := map[string]Action{}
	for _, a := range Actions {
		k := s[a]
		if !slices.Contains(Choices(), k) || core[k] != "" {
			return fmt.Errorf("%s: %q can't be a key to %s", a.ID(), k, strings.ToLower(a.Label()))
		}
		if b, ok := by[k]; ok {
			return fmt.Errorf("%s and %s are both %q", b.ID(), a.ID(), k)
		}
		by[k] = a
	}
	return nil
}

// Path is the settings file: config.yaml in TETRU__CONFIG, else in tetru
// under $XDG_CONFIG_HOME, else under ~/.config (tdp E2, E3).
func Path() string {
	dir := os.Getenv("TETRU__CONFIG")
	if dir == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				home = "."
			}
			base = filepath.Join(home, ".config")
		}
		dir = filepath.Join(base, "tetru")
	}
	return filepath.Join(dir, "config.yaml")
}

// Load reads the keys from the settings file at path: the default for an
// action the file does not set, all of them when there is no file. A file
// that can't be read, or keys that can't be, is an error (tdp E3).
func Load(path string) (Set, error) {
	s := Default()
	m, err := read(path)
	if err != nil {
		return s, err
	}
	raw, ok := m["keys"]
	if !ok {
		return s, nil
	}
	set, ok := raw.(map[string]any)
	if !ok {
		return s, fmt.Errorf("keys: not a list of actions and keys")
	}
	for _, a := range Actions {
		if v, ok := set[a.ID()]; ok {
			k, ok := v.(string)
			if !ok {
				return s, fmt.Errorf("%s: not a key", a.ID())
			}
			s[a] = k
		}
	}
	return s, s.Check()
}

// read is the settings file as a map; empty when there is none.
func read(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]any{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Hint says whether the hint shows, from the settings file at path (hint:
// true or false): on when the file does not say (user, 2026-11-13: on at
// first). A file that can't be read, or a hint neither true nor false, is
// an error (tdp E3), the hint on.
func Hint(path string) (bool, error) {
	m, err := read(path)
	if err != nil {
		return true, err
	}
	v, ok := m["hint"]
	if !ok {
		return true, nil
	}
	on, ok := v.(bool)
	if !ok {
		return true, fmt.Errorf("hint: %v is neither true nor false", v)
	}
	return on, nil
}

// SaveHint writes on as the hint's switch into the settings file at path,
// the rest as it is.
func SaveHint(path string, on bool) error {
	m, err := read(path)
	if err != nil {
		return err
	}
	m["hint"] = on
	return write(path, m)
}

// Save writes s into the settings file at path, leaving whatever else it
// holds as it is (tdp E3: a key tetru does not read stays).
func Save(path string, s Set) error {
	m, err := read(path)
	if err != nil {
		return err
	}
	set := map[string]string{}
	for _, a := range Actions {
		set[a.ID()] = s[a]
	}
	m["keys"] = set
	return write(path, m)
}

// write writes m as the settings file at path: to a file beside it first,
// then in its place, so that a failure leaves the old one.
func write(path string, m map[string]any) error {
	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
