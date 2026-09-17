package gitvm

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, home, path, data string) {
	t.Helper()
	p := filepath.Join(home, path)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestProfiles(t *testing.T) {
	h := t.TempDir()
	fixture(t, h, ".gitvm/profiles/Person", "Person\nperson@example.com\nWork\n")
	fixture(t, h, ".gitvm/profiles/bad", "Bad\nbad\nx *\n")
	fixture(t, h, ".gitvm/current", "Person\n")
	ps, current, err := Load(h)
	if err != nil || len(ps) != 1 || current != "Person" || ps[0].Alias != "Work" {
		t.Fatalf("%v %q %v", ps, current, err)
	}
	for _, p := range []Profile{{"../x", "Name", "a@b.com", "ok"}, {"x", "Name\nHost *", "a@b.com", "ok"}, {"x", "Name", "a@b.com", "x *"}} {
		if p.Validate() == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
}
func TestActivate(t *testing.T) {
	h := t.TempDir()
	p := Profile{"Person", "Name $(touch nope)", "a@example.com", "Work"}
	fixture(t, h, ".ssh/id_ed25519_work", "")
	before := "# global\nHost elsewhere\n  User keep\nHost github.com\n  IdentityFile old\n  Port 22\nMatch all\n  ServerAliveInterval 30\n"
	fixture(t, h, ".ssh/config", before)
	var calls [][]string
	message, err := Activate(h, p, func(args ...string) error { calls = append(calls, args); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0][3] != p.Name {
		t.Fatalf("unsafe args: %v", calls)
	}
	b, _ := os.ReadFile(filepath.Join(h, ".gitvm/current"))
	if string(b) != "Person\n" {
		t.Fatalf("current %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(h, ".ssh/config"))
	s := string(b)
	if !strings.HasPrefix(s, "# global\nHost elsewhere\n  User keep\n") || !strings.Contains(s, "  Port 22\nMatch all\n  ServerAliveInterval 30\n") || !strings.Contains(s, "id_ed25519_work") || message == "" {
		t.Fatalf("ssh %q", s)
	}
}
func TestActivateRejectsOverlappingSSHIdentity(t *testing.T) {
	for _, section := range []string{
		"Host *\n  User other\n  IdentityFile ~/.ssh/other\n",
		"Host *.com\n  IdentityFile ~/.ssh/other\n",
		"Host github.com elsewhere\n  User other\n",
		"Host github.?om\n  IdentityFile ~/.ssh/other\n",
		"Host github.com\n  User git\nHost *\n  IdentityFile ~/.ssh/other\n",
		"User other\n",
		"Match all\n  IdentityFile ~/.ssh/other\n",
		"Include conf.d/*\n",
	} {
		for _, suffix := range []string{"", "Host github.com\n  User git\n"} {
			t.Run(section+suffix, func(t *testing.T) {
				h := t.TempDir()
				before := section + suffix
				fixture(t, h, ".ssh/id_ed25519_work", "")
				fixture(t, h, ".ssh/config", before)
				fixture(t, h, ".gitvm/current", "old\n")
				calls := 0
				msg, err := Activate(h, Profile{"new", "Name", "a@example.com", "work"}, func(...string) error { calls++; return nil })
				if err == nil || !strings.Contains(err.Error(), "SSH identity precedence") || msg != "" {
					t.Fatalf("expected clear precedence error, got message %q, error %v", msg, err)
				}
				if calls != 0 {
					t.Fatal("Git changed before SSH validation")
				}
				for path, want := range map[string]string{".ssh/config": before, ".gitvm/current": "old\n"} {
					got, err := os.ReadFile(filepath.Join(h, path))
					if err != nil || string(got) != want {
						t.Fatalf("%s changed: %q, %v", path, got, err)
					}
				}
			})
		}
	}
}

func TestActivatePreservesUnrelatedSSHOptions(t *testing.T) {
	h := t.TempDir()
	before := "Host *\n  ServerAliveInterval 30\nHost elsewhere\n  User keep\n  IdentityFile ~/.ssh/elsewhere\n"
	fixture(t, h, ".ssh/id_ed25519_work", "")
	fixture(t, h, ".ssh/config", before)
	_, err := Activate(h, Profile{"new", "Name", "a@example.com", "work"}, func(...string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(h, ".ssh/config"))
	if err != nil || string(got) != before+"Host github.com\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_work\n  IdentitiesOnly yes\n" {
		t.Fatalf("unexpected SSH configuration: %q, %v", got, err)
	}
}

func TestSSHHostPatterns(t *testing.T) {
	for _, tc := range []struct {
		patterns string
		reject   bool
	}{
		{"*.internal", false},
		{"gitlab.?om", false},
		{"*.internal elsewhere", false},
		{"* !github.com", false},
		{"!github.?om *", false},
		{"!elsewhere", false},
		{"*.com", true},
		{"github.?om", true},
		{"*.internal github.com", true},
		{"* !*.internal", true},
		{"g*hub.c?m", true},
	} {
		t.Run(tc.patterns, func(t *testing.T) {
			for _, directive := range []string{"User other", "IdentityFile ~/.ssh/other"} {
				section := "Host " + tc.patterns + "\n  " + directive + "\n"
				for _, before := range []string{section, "Host github.com\n  User git\n" + section} {
					got, err := updateSSH(before, "work")
					if tc.reject {
						if err == nil || !strings.Contains(err.Error(), "SSH identity precedence") {
							t.Fatalf("expected precedence rejection, got %q, %v", got, err)
						}
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					identity := "Host github.com\n" + sshIdentity("work")
					want := section + identity
					if before != section {
						want = identity + section
					}
					if got != want {
						t.Fatalf("unrelated section changed: got %q, want %q", got, want)
					}
				}
			}
		})
	}
}

func TestFailureAndMissingKey(t *testing.T) {
	h := t.TempDir()
	fixture(t, h, ".gitvm/current", "old\n")
	fixture(t, h, ".ssh/config", "Host other\n User keep\n")
	p := Profile{"new", "Name", "a@example.com", ""}
	_, err := Activate(h, p, func(...string) error { return errors.New("git failed") })
	if err == nil {
		t.Fatal("expected failure")
	}
	b, _ := os.ReadFile(filepath.Join(h, ".gitvm/current"))
	if string(b) != "old\n" {
		t.Fatal("changed current")
	}
	msg, err := Activate(h, p, func(...string) error { return nil })
	if err != nil || !strings.Contains(msg, "SSH unchanged") {
		t.Fatalf("%s %v", msg, err)
	}
	b, _ = os.ReadFile(filepath.Join(h, ".ssh/config"))
	if string(b) != "Host other\n User keep\n" {
		t.Fatal("changed SSH")
	}
}
func TestSelector(t *testing.T) {
	count := 0
	m := NewModel([]Profile{{"one", "One", "one@example.com", ""}, {"two", "Two", "two@example.com", "work"}}, "one", func(Profile) (string, error) { count++; return "Activated", nil })
	if !strings.Contains(m.View(), "[active]") || !strings.Contains(m.View(), "one@example.com") {
		t.Fatal(m.View())
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = next.(Model)
	if m.cursor != 1 {
		t.Fatal("navigation")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd != nil || count != 0 || !strings.Contains(m.View(), "Confirm") {
		t.Fatal("missing confirmation")
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("missing activation")
	}
	result := cmd()
	next, _ = m.Update(result)
	if count != 1 || !strings.Contains(next.View(), "Activated") {
		t.Fatal("activation feedback")
	}
	m = NewModel(nil, "", nil)
	if !strings.Contains(m.View(), "No valid profiles") {
		t.Fatal(m.View())
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("cancel")
	}
}
