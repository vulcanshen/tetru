// tetru is a falling-block game for the terminal, of the terminu family.
package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/vulcanshen/tetru/internal/ui"
	"github.com/vulcanshen/tetru/internal/version"
)

const usage = `tetru: a falling-block game for the terminal

Usage:
  tetru            play
  tetru version    print the version (-v, --version)
  tetru help       print this help (-h, --help)

Scores are kept in ~/.tetru/scores.yaml (TETRU__DATA to move it).
`

func main() {
	if len(os.Args) > 1 {
		os.Exit(command(os.Args[1], os.Stdout, os.Stderr))
	}
	t := ui.NewTetru()
	endOnHangup()
	err := t.App.Run()
	t.Finish() // a game still going keeps its score, Ctrl-C twice too
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

// endOnHangup ends the program as SIGTERM does when its terminal is closed
// (SIGHUP), so that Run returns and the game going keeps its score (user,
// 2026-11-11: leaving keeps the score as it is then).
//
// TODO(tgo): terminu-go's Run ends on SIGTERM (bubbletea's) but not on
// SIGHUP, which kills the program before anything after Run.
func endOnHangup() {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		<-hup
		syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()
}
