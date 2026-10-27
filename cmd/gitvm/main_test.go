package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/SergioSanchezIglesias/gitvm-tui/internal/gitvm"
)

func put(t *testing.T, home, path, data string) {
	t.Helper()
	path = filepath.Join(home, path)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, home string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(home, path)
			if err != nil {
				return err
			}
			files[rel] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			var out, stderr bytes.Buffer
			code := run([]string{arg}, &out, &stderr, func() (string, error) { t.Fatal("help accessed home"); return "", nil }, nil, nil)
			if code != 0 || out.String() != usage || stderr.String() != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
			}
		})
	}
}

func TestRunRejectedArguments(t *testing.T) {
	cases := []struct {
		args    []string
		message string
	}{
		{[]string{"--user"}, "--user requires a profile ID"},
		{[]string{"-u"}, "-u requires a profile ID"},
		{[]string{"--user", ""}, "invalid profile ID \"\""},
		{[]string{"-u", "../personal"}, "invalid profile ID \"../personal\""},
		{[]string{"--user", " padded"}, "invalid profile ID \" padded\""},
		{[]string{"--user", "--help"}, "profile \"--help\" not found; create or check its ID in the TUI"},
		{[]string{"--unknown"}, "unknown argument \"--unknown\""},
		{[]string{"personal"}, "unknown argument \"personal\""},
		{[]string{"--user", "personal", "extra"}, "unexpected argument \"extra\""},
		{[]string{"-u", "personal", "--user", "personal"}, "unexpected argument \"--user\""},
		{[]string{"--help", "-u", "personal"}, "--help must be used alone"},
		{[]string{"--user", "PERSONAL"}, "profile \"PERSONAL\" not found; create or check its ID in the TUI"},
		{[]string{"--user", "missing"}, "profile \"missing\" not found; create or check its ID in the TUI"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			home := t.TempDir()
			put(t, home, ".gitvm/profiles/personal", "Example User\nuser@example.com\n")
			put(t, home, ".gitvm/current", "previous\n")
			put(t, home, ".gitconfig", "original git config\n")
			put(t, home, ".ssh/config", "original SSH config\n")
			before := snapshot(t, home)
			var out, stderr bytes.Buffer
			code := run(tc.args, &out, &stderr, func() (string, error) { return home, nil }, func(...string) error { t.Fatal("rejected input ran Git"); return nil }, func(gitvm.Model) error { t.Fatal("rejected input opened TUI"); return nil })
			wantErr := "Error: " + tc.message + "\nRun gitvm --help for usage.\n"
			if code != 1 || out.String() != "" || stderr.String() != wantErr {
				t.Fatalf("code=%d stdout=%q stderr=%q; want %q", code, out.String(), stderr.String(), wantErr)
			}
			if !reflect.DeepEqual(before, snapshot(t, home)) {
				t.Fatal("rejected input changed state")
			}
		})
	}
}

func TestRunActivation(t *testing.T) {
	cases := []struct {
		name, alias, ssh string
		key              bool
		failAt           int
		wantOut, wantErr string
		calls            int
	}{
		{name: "no alias", wantOut: "Activated personal. SSH unchanged: no matching alias key.\n", calls: 2},
		{name: "missing key", alias: "Personal", wantOut: "Activated personal. SSH unchanged: no matching alias key.\n", calls: 2},
		{name: "matching key", alias: "Personal", key: true, wantOut: "Activated personal. SSH updated.\n", calls: 2},
		{name: "SSH precedence rejection", alias: "Personal", key: true, ssh: "Include other.conf\n", wantErr: "Activation failed: cannot guarantee SSH identity precedence: Include outside an exclusive Host github.com section; SSH config unchanged\n"},
		{name: "Git name failure", failAt: 1, wantErr: "Activation failed: controlled Git failure\n", calls: 1},
		{name: "Git email partial failure", failAt: 2, wantErr: "Activation failed: Git name may have changed: controlled Git failure\n", calls: 2},
	}
	for _, option := range []string{"--user", "-u"} {
		for _, tc := range cases {
			t.Run(option+"/"+tc.name, func(t *testing.T) {
				home := t.TempDir()
				profile := "Example User\nuser@example.com\n"
				if tc.alias != "" {
					profile += tc.alias + "\n"
				}
				put(t, home, ".gitvm/profiles/personal", profile)
				put(t, home, ".gitvm/current", "previous\n")
				put(t, home, ".gitconfig", "original\n")
				ssh := tc.ssh
				if ssh == "" {
					ssh = "Host elsewhere\n  User other\n"
				}
				put(t, home, ".ssh/config", ssh)
				if tc.key {
					put(t, home, ".ssh/id_ed25519_personal", "existing key")
				}
				before := snapshot(t, home)
				var calls [][]string
				runner := func(args ...string) error {
					calls = append(calls, append([]string(nil), args...))
					if len(calls) == tc.failAt {
						return errors.New("controlled Git failure")
					}
					f, err := os.OpenFile(filepath.Join(home, ".gitconfig"), os.O_APPEND|os.O_WRONLY, 0600)
					if err != nil {
						return err
					}
					_, err = f.WriteString(strings.Join(args, " ") + "\n")
					closeErr := f.Close()
					if err != nil {
						return err
					}
					return closeErr
				}
				var out, stderr bytes.Buffer
				code := run([]string{option, "personal"}, &out, &stderr, func() (string, error) { return home, nil }, runner, func(gitvm.Model) error { t.Fatal("activation opened TUI"); return nil })
				wantCode := 0
				if tc.wantErr != "" {
					wantCode = 1
				}
				if code != wantCode || out.String() != tc.wantOut || stderr.String() != tc.wantErr {
					t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
				}
				expectedCalls := [][]string{{"config", "--global", "user.name", "Example User"}, {"config", "--global", "user.email", "user@example.com"}}
				if tc.calls == 0 {
					expectedCalls = nil
				} else {
					expectedCalls = expectedCalls[:tc.calls]
				}
				if !reflect.DeepEqual(calls, expectedCalls) {
					t.Fatalf("Git calls: %v", calls)
				}
				after := snapshot(t, home)
				want := before
				if tc.wantErr == "" {
					want[".gitvm/current"] = "personal\n"
				}
				successfulCalls := tc.calls
				if tc.failAt > 0 {
					successfulCalls--
				}
				for _, args := range expectedCalls[:successfulCalls] {
					want[".gitconfig"] += strings.Join(args, " ") + "\n"
				}
				if tc.key && tc.wantErr == "" {
					want[".ssh/config"] = ssh + "Host github.com\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_personal\n  IdentitiesOnly yes\n"
				}
				if !reflect.DeepEqual(after, want) {
					t.Fatalf("state=%v; want=%v", after, want)
				}
			})
		}
	}
}

func TestRunTUI(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[failure], func(t *testing.T) {
			home := t.TempDir()
			put(t, home, ".gitvm/profiles/personal", "Example User\nuser@example.com\n")
			before := snapshot(t, home)
			var out, stderr bytes.Buffer
			opened := false
			code := run(nil, &out, &stderr, func() (string, error) { return home, nil }, func(...string) error { t.Fatal("startup ran Git"); return nil }, func(model gitvm.Model) error {
				opened = true
				if !strings.Contains(model.View(), "Switch profile") {
					t.Fatal("not the existing action menu")
				}
				if failure {
					return errors.New("controlled TUI failure")
				}
				return nil
			})
			wantCode, wantErr := 0, ""
			if failure {
				wantCode, wantErr = 1, "controlled TUI failure\n"
			}
			if !opened || code != wantCode || out.String() != "" || stderr.String() != wantErr || !reflect.DeepEqual(before, snapshot(t, home)) {
				t.Fatalf("opened=%v code=%d stdout=%q stderr=%q", opened, code, out.String(), stderr.String())
			}
		})
	}
}
