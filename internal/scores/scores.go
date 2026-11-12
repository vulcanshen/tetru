// Package scores keeps tetru's scores: who played, when and how many
// points, a game a record, in the data directory (tdp E3:
// ~/.tetru/scores.yaml, TETRU__DATA to move it).
package scores

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"gopkg.in/yaml.v3"
)

// Score is one game's record.
type Score struct {
	Time  time.Time `yaml:"time"`
	Name  string    `yaml:"name"`
	Score int       `yaml:"score"`
}

type file struct {
	Scores []Score `yaml:"scores"`
}

// Path is the scores' file: scores.yaml in TETRU__DATA, or in ~/.tetru.
func Path() string {
	dir := os.Getenv("TETRU__DATA")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".tetru")
	}
	return filepath.Join(dir, "scores.yaml")
}

// Load reads the records, in the order they were kept; none when the file
// is not there yet.
func Load(path string) ([]Score, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Scores, nil
}

// Save writes every record, the file replaced whole so a failed write
// leaves the old one as it was.
func Save(path string, all []Score) error {
	data, err := yaml.Marshal(file{Scores: all})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".scores-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
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

// Ranked is the records' places, best first: the higher score first, of
// two the same the earlier first.
func Ranked(all []Score) []int {
	order := make([]int, len(all))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		if c := cmp.Compare(all[b].Score, all[a].Score); c != 0 {
			return c
		}
		return all[a].Time.Compare(all[b].Time)
	})
	return order
}

// Most is how many records are kept: the best twenty (user, 2026-11-12).
const Most = 20

// Best is the records in the best Most, in the order they were kept; the
// others are no longer kept.
func Best(all []Score) []Score {
	order := Ranked(all)
	if len(order) <= Most {
		return all
	}
	in := make([]bool, len(all))
	for _, i := range order[:Most] {
		in[i] = true
	}
	var out []Score
	for i, s := range all {
		if in[i] {
			out = append(out, s)
		}
	}
	return out
}

// Makes says whether a game of score points, kept now, gets into the best
// Most: past the last of them, as of two the same the earlier is first.
func Makes(all []Score, score int) bool {
	if len(all) < Most {
		return true
	}
	return score > all[Ranked(all)[Most-1]].Score
}
