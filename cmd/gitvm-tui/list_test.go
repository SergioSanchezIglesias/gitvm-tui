package main

import (
	"bytes"
	"reflect"
	"testing"

	"gitvm/internal/gitvm"
)

func TestRunList(t *testing.T) {
	for _, flag := range []string{"--list", "-l"} {
		for _, scenario := range []string{"profiles", "empty", "stale marker", "load error", "extra argument"} {
			t.Run(flag+"/"+scenario, func(t *testing.T) {
				home := t.TempDir()
				wantOut, wantErr, wantCode := "No profiles found.\n", "", 0
				args := []string{flag}
				switch scenario {
				case "profiles", "stale marker":
					put(t, home, ".gitvm/profiles/work", "Work User\nwork@example.com\n")
					put(t, home, ".gitvm/profiles/personal", "Personal User\npersonal@example.com\n")
					put(t, home, ".gitvm/profiles/broken", "bad record\n")
					put(t, home, ".gitvm/current", "work\n")
					wantOut = "  personal\tPersonal User\tpersonal@example.com\n* work\tWork User\twork@example.com\n"
					if scenario == "stale marker" {
						put(t, home, ".gitvm/current", "missing\n")
						wantOut = "  personal\tPersonal User\tpersonal@example.com\n  work\tWork User\twork@example.com\n"
					}
				case "load error":
					put(t, home, ".gitvm/profiles", "not a directory")
					// Use a deterministic storage failure without depending on OS error text.
					wantCode = 1
					wantOut = ""
				case "extra argument":
					args = append(args, "--user", "personal")
					wantCode = 1
					wantOut = ""
					wantErr = "Error: " + flag + " must be used alone\nRun gitvm-tui --help for usage.\n"
				}
				put(t, home, ".gitconfig", "original Git config\n")
				put(t, home, ".ssh/config", "original SSH config\n")
				before := snapshot(t, home)
				var out, stderr bytes.Buffer
				code := run(args, &out, &stderr, func() (string, error) { return home, nil }, func(...string) error { t.Fatal("list ran Git"); return nil }, func(gitvm.Model) error { t.Fatal("list opened TUI"); return nil })
				if scenario == "load error" {
					_, _, err := gitvm.Load(home)
					if err == nil {
						t.Fatal("expected load failure")
					}
					wantErr = "Load profiles: " + err.Error() + "\n"
				}
				if code != wantCode || out.String() != wantOut || stderr.String() != wantErr {
					t.Fatalf("code=%d stdout=%q stderr=%q; want %d %q %q", code, out.String(), stderr.String(), wantCode, wantOut, wantErr)
				}
				if !reflect.DeepEqual(before, snapshot(t, home)) {
					t.Fatal("list changed state")
				}
			})
		}
	}
}
