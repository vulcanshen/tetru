package main

import (
	"strings"
	"testing"
)

func TestVersionAndHelp(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		var out, errOut strings.Builder
		if code := command(arg, &out, &errOut); code != 0 || out.String() != "dev\n" {
			t.Errorf("%s: %d %q", arg, code, out.String())
		}
	}
	for _, arg := range []string{"help", "-h", "--help"} {
		var out, errOut strings.Builder
		if code := command(arg, &out, &errOut); code != 0 || !strings.Contains(out.String(), "tetru version") || !strings.Contains(out.String(), "scores.yaml") {
			t.Errorf("%s: %d %q", arg, code, out.String())
		}
	}
	var out, errOut strings.Builder
	if code := command("nope", &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "unknown command") {
		t.Errorf("an unknown command: %d %q", code, errOut.String())
	}
}
