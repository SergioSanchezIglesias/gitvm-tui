package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/SergioSanchezIglesias/gitvm-tui/internal/gitvm"
	"github.com/charmbracelet/x/ansi"
)

func TestColoredOutput(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "--list", "-l"} {
		t.Run(flag, func(t *testing.T) {
			home := t.TempDir()
			put(t, home, ".gitvm/profiles/work", "Work User\nwork@example.com\n")
			put(t, home, ".gitvm/current", "work\n")
			before := snapshot(t, home)
			var plain, colored, stderr bytes.Buffer
			getHome := func() (string, error) {
				if strings.Contains(flag, "h") {
					t.Fatal("help accessed home")
				}
				return home, nil
			}
			git := func(...string) error { t.Fatal("output ran Git"); return nil }
			tui := func(gitvm.Model) error { t.Fatal("output opened TUI"); return nil }
			if code := runCLI([]string{flag}, &plain, &stderr, getHome, git, tui, false); code != 0 {
				t.Fatalf("plain exit %d", code)
			}
			if code := runCLI([]string{flag}, &colored, &stderr, getHome, git, tui, true); code != 0 {
				t.Fatalf("colored exit %d", code)
			}
			if !strings.Contains(colored.String(), "\x1b[") || ansi.Strip(colored.String()) != plain.String() || stderr.Len() != 0 {
				t.Fatalf("plain=%q colored=%q stderr=%q", plain.String(), colored.String(), stderr.String())
			}
			if strings.Contains(flag, "h") {
				for _, token := range []string{"\x1b[1;36mUsage:", "\x1b[36mgitvm", "\x1b[33m--help"} {
					if !strings.Contains(colored.String(), token) {
						t.Errorf("missing style %q", token)
					}
				}
			} else if !strings.Contains(colored.String(), "\x1b[1;32m* work") {
				t.Error("active profile not highlighted")
			}
			if !reflect.DeepEqual(before, snapshot(t, home)) {
				t.Fatal("output changed state")
			}
		})
	}
}

func TestColorPolicy(t *testing.T) {
	for _, tc := range []struct {
		name          string
		terminal      bool
		term, noColor string
		want          bool
	}{
		{"terminal", true, "xterm-256color", "", true},
		{"redirected", false, "xterm-256color", "", false},
		{"NO_COLOR", true, "xterm-256color", "1", false},
		{"dumb terminal", true, "dumb", "", false},
		{"unknown terminal", true, "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := useColor(tc.terminal, tc.term, tc.noColor); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
