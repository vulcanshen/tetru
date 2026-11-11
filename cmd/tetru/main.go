// tetru is a falling-block game for the terminal, of the terminu family.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/vulcanshen/tetru/internal/ui"
	"github.com/vulcanshen/tetru/internal/version"
)

const usage = `tetru: a falling-block game for the terminal

Usage:
  tetru            play
  tetru version    print the version (-v, --version)
  tetru help       print this help (-h, --help)

Scores are kept in ~/.tetru/scores.yaml and a game saved in ~/.tetru/game.yaml
(TETRU__DATA to move them), the keys to play with in
~/.config/tetru/config.yaml (TETRU__CONFIG to move it).
`

func main() {
	if len(os.Args) > 1 {
		os.Exit(command(os.Args[1], os.Stdout, os.Stderr))
	}
	t := ui.NewTetru()
	err := t.App.Run()
	t.Finish() // a game still going keeps its score: Ctrl-C twice, the terminal closed (Run returns on SIGHUP)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tetru:", err)
		os.Exit(1)
	}
}

// command runs a command-line command (tdp E1), its exit status back.
func command(arg string, out, errOut io.Writer) int {
	switch arg {
	case "version", "-v", "--version":
		fmt.Fprintln(out, version.Display())
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return 0
	}
	fmt.Fprintf(errOut, "tetru: unknown command %q\n\n%s", arg, usage)
	return 2
}
