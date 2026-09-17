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
func TestCreateProfile(t *testing.T) {
	for _, alias := range []string{"", "work"} {
		t.Run("alias="+alias, func(t *testing.T) {
			h := t.TempDir()
			p := Profile{"new", "New User", "new@example.com", alias}
			if err := Create(h, p); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(h, ".gitvm/profiles/new")
			want := "New User\nnew@example.com\n"
			if alias != "" {
				want += alias + "\n"
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != want {
				t.Fatalf("record %q: %v", data, err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("profile must be private", err)
			}
			info, err = os.Stat(filepath.Dir(path))
			if err != nil || info.Mode().Perm() != 0700 {
				t.Fatal("directory must be private", err)
			}
			p.Name = "Replacement"
			if err := Create(h, p); err == nil {
				t.Fatal("duplicate accepted")
			}
			data, _ = os.ReadFile(path)
			if string(data) != want {
				t.Fatal("duplicate overwrote profile")
			}
			profiles, current, err := Load(h)
			if err != nil || len(profiles) != 1 || current != "" {
				t.Fatalf("load %v %q %v", profiles, current, err)
			}
			entries, _ := os.ReadDir(filepath.Dir(path))
			if len(entries) != 1 {
				t.Fatal("temporary file leaked")
			}
		})
	}
	for _, p := range []Profile{{"../bad", "Name", "a@example.com", ""}, {"bad", "Name\nInjected", "a@example.com", ""}, {"bad", "Name", "invalid", ""}, {"bad", "Name", "a@example.com", "bad alias"}} {
		h := t.TempDir()
		if err := Create(h, p); err == nil {
			t.Fatalf("accepted %+v", p)
		}
		entries, _ := os.ReadDir(h)
		if len(entries) != 0 {
			t.Fatal("invalid input changed storage")
		}
	}
	h := t.TempDir()
	fixture(t, h, ".gitvm/profiles", "blocked")
	if err := Create(h, Profile{"new", "Name", "a@example.com", ""}); err == nil {
		t.Fatal("storage error hidden")
	}
	data, _ := os.ReadFile(filepath.Join(h, ".gitvm/profiles"))
	if string(data) != "blocked" {
		t.Fatal("storage changed")
	}
}

func TestDeleteProfile(t *testing.T) {
	for _, alias := range []string{"", "Work"} {
		t.Run("alias="+alias, func(t *testing.T) {
			h := t.TempDir()
			if err := Create(h, Profile{"chosen", "Name", "a@example.com", alias}); err != nil {
				t.Fatal(err)
			}
			fixture(t, h, ".gitvm/profiles/keep", "Keep\nkeep@example.com\n")
			fixture(t, h, ".gitvm/current", "chosen\n")
			if err := Delete(h, "chosen"); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(h, ".gitvm/profiles/chosen")); !os.IsNotExist(err) {
				t.Fatalf("profile remains: %v", err)
			}
			for path, want := range map[string]string{".gitvm/current": "chosen\n", ".gitvm/profiles/keep": "Keep\nkeep@example.com\n"} {
				data, err := os.ReadFile(filepath.Join(h, path))
				if err != nil || string(data) != want {
					t.Fatalf("changed %s: %q, %v", path, data, err)
				}
			}
			if err := Delete(h, "chosen"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing profile error: %v", err)
			}
		})
	}
	for _, id := range []string{"", ".", "..", "../outside", "a/b", `a\b`, "/absolute", " padded", "line\nfeed", "nul\x00"} {
		t.Run("invalid ID="+id, func(t *testing.T) {
			if err := Delete(t.TempDir(), id); err == nil || !strings.Contains(err.Error(), "invalid profile ID") {
				t.Fatalf("invalid ID error: %v", err)
			}
		})
	}
	for _, data := range []string{"", "Name\n", "Name\nbad\n", "Name\na@example.com\nbad alias\n", "Name\na@example.com\nWork\nextra\n"} {
		t.Run("invalid record="+data, func(t *testing.T) {
			h := t.TempDir()
			fixture(t, h, ".gitvm/profiles/bad", data)
			if err := Delete(h, "bad"); err == nil || !strings.Contains(err.Error(), "invalid profile record") {
				t.Fatalf("invalid record error: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(h, ".gitvm/profiles/bad"))
			if err != nil || string(got) != data {
				t.Fatal("invalid record changed", err)
			}
		})
	}
	for _, kind := range []string{"directory", "symlink", "dangling symlink", "profiles symlink", "storage symlink"} {
		t.Run(kind, func(t *testing.T) {
			h := t.TempDir()
			outside := t.TempDir()
			fixture(t, outside, "profiles/target", "Name\na@example.com\n")
			path := filepath.Join(h, ".gitvm/profiles/target")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink", "dangling symlink":
				target := filepath.Join(outside, "profiles/target")
				if kind == "dangling symlink" {
					target += "-missing"
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			default:
				h = t.TempDir()
				path = filepath.Join(h, ".gitvm")
				target := outside
				if kind == "profiles symlink" {
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
					path = filepath.Join(path, "profiles")
					target = filepath.Join(outside, "profiles")
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			if err := Delete(h, "target"); err == nil {
				t.Fatal("non-regular storage accepted")
			}
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("entry removed", err)
			}
			data, err := os.ReadFile(filepath.Join(outside, "profiles/target"))
			if err != nil || string(data) != "Name\na@example.com\n" {
				t.Fatal("outside record changed", err)
			}
		})
	}
}

func TestCreationForm(t *testing.T) {
	h := t.TempDir()
	m := NewModel(nil, "old", func(Profile) (string, error) { t.Fatal("creation activated profile"); return "", nil }).WithCreator(func(p Profile) error { return Create(h, p) })
	key := func(k tea.KeyMsg) tea.Cmd { next, cmd := m.Update(k); m = next.(Model); return cmd }
	text := func(s string) { key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}) }
	text("n")
	if !strings.Contains(m.View(), "Create profile") {
		t.Fatal(m.View())
	}
	text("discard")
	key(tea.KeyMsg{Type: tea.KeyEsc})
	if strings.Contains(m.View(), "Profile ID:") {
		t.Fatal("cancel failed")
	}
	text("n")
	for _, s := range []string{"new", "Name qjk", "bad"} {
		text(s)
		key(tea.KeyMsg{Type: tea.KeyTab})
	}
	cmd := key(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || !strings.Contains(m.View(), "invalid email") {
		t.Fatal("validation missing", m.View())
	}
	key(tea.KeyMsg{Type: tea.KeyShiftTab})
	for range "bad" {
		key(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	text("new@example.com")
	key(tea.KeyMsg{Type: tea.KeyDown})
	cmd = key(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("missing create command")
	}
	if c := key(tea.KeyMsg{Type: tea.KeyEnter}); c != nil {
		t.Fatal("duplicate submit")
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	if len(m.profiles) != 1 || m.profiles[0].Name != "Name qjk" || m.current != "old" || !strings.Contains(m.View(), "Created new") {
		t.Fatal(m.View())
	}
	text("n")
	for _, s := range []string{"new", "Other", "other@example.com"} {
		text(s)
		key(tea.KeyMsg{Type: tea.KeyEnter})
	}
	cmd = key(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("missing duplicate attempt")
	}
	next, _ = m.Update(cmd())
	m = next.(Model)
	if !strings.Contains(m.View(), "already exists") || !strings.Contains(m.View(), "Create profile") || len(m.profiles) != 1 {
		t.Fatal(m.View())
	}
}

func TestSwitchSelectionRetainsCreatedProfile(t *testing.T) {
	prior := Profile{"old", "Old", "old@example.com", ""}
	created := Profile{"new", "New", "new@example.com", ""}
	m := NewModel([]Profile{prior}, prior.ID, func(Profile) (string, error) {
		t.Fatal("navigation activated a profile")
		return "", nil
	}).WithCreator(func(Profile) error {
		t.Fatal("navigation created a profile")
		return nil
	}).WithDeleter(func(string) error {
		t.Fatal("navigation deleted a profile")
		return nil
	})
	update := func(msg tea.Msg) {
		t.Helper()
		next, cmd := m.Update(msg)
		m = next.(Model)
		if cmd != nil || m.current != prior.ID {
			t.Fatal("navigation crossed a mutation boundary")
		}
	}
	update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != switchSelection || m.cursor != 0 {
		t.Fatal("first selection must start at the first profile")
	}
	update(tea.KeyMsg{Type: tea.KeyEsc})
	update(tea.KeyMsg{Type: tea.KeyDown})
	update(tea.KeyMsg{Type: tea.KeyEnter})
	update(creationResult{profile: created})
	if m.state != actionMenu || m.action != 1 || m.cursor != 1 {
		t.Fatal("creation must return to actions with the new profile selected")
	}
	update(tea.KeyMsg{Type: tea.KeyUp})
	update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != switchSelection || m.cursor != 1 || !strings.Contains(m.View(), "> new") {
		t.Fatalf("Switch profile lost the created profile selection: %s", m.View())
	}
	update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Fatal("profile navigation did not move up")
	}
	update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatal("profile navigation did not move down")
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
func TestActionMenuAndDeletion(t *testing.T) {
	for _, outcome := range []string{"success", "failure", "cancel", "active"} {
		t.Run(outcome, func(t *testing.T) {
			calls := 0
			m := NewModel([]Profile{{"one", "One", "one@example.com", ""}, {"two", "Two", "two@example.com", ""}}, "one", func(Profile) (string, error) { t.Fatal("unexpected activation"); return "", nil }).WithCreator(func(Profile) error { t.Fatal("unexpected creation"); return nil }).WithDeleter(func(id string) error {
				calls++
				if id != "two" {
					t.Fatal(id)
				}
				if outcome == "failure" {
					return errors.New("disk denied")
				}
				return nil
			})
			key := func(k tea.KeyType) tea.Cmd { next, cmd := m.Update(tea.KeyMsg{Type: k}); m = next.(Model); return cmd }
			for _, text := range []string{"GitVM", "> Switch profile", "Create profile", "Delete profile", "╭", "Enter"} {
				if !strings.Contains(m.View(), text) {
					t.Fatalf("missing %q: %s", text, m.View())
				}
			}
			key(tea.KeyDown)
			key(tea.KeyDown)
			key(tea.KeyEnter)
			if !strings.Contains(m.View(), "[active]") {
				t.Fatal(m.View())
			}
			if outcome == "active" {
				if key(tea.KeyEnter) != nil || calls != 0 || !strings.Contains(m.View(), "Cannot delete active profile") {
					t.Fatal(m.View())
				}
				return
			}
			key(tea.KeyDown)
			if key(tea.KeyEnter) != nil || calls != 0 || !strings.Contains(m.View(), "Confirm deletion of two") {
				t.Fatal(m.View())
			}
			if outcome == "cancel" {
				key(tea.KeyEsc)
				key(tea.KeyEsc)
				if calls != 0 || !strings.Contains(m.View(), "> Delete profile") {
					t.Fatal(m.View())
				}
				return
			}
			cmd := key(tea.KeyEnter)
			if cmd == nil || calls != 0 {
				t.Fatal("deletion not deferred")
			}
			if key(tea.KeyEnter) != nil {
				t.Fatal("duplicate deletion")
			}
			next, _ := m.Update(cmd())
			m = next.(Model)
			if calls != 1 || m.current != "one" {
				t.Fatal("unsafe deletion")
			}
			if outcome == "failure" {
				key(tea.KeyUp)
				if len(m.profiles) != 2 || !strings.Contains(m.View(), "Deletion failed: disk denied") {
					t.Fatal(m.View())
				}
			} else if len(m.profiles) != 1 || !strings.Contains(m.View(), "Deleted two") {
				t.Fatal(m.View())
			}
		})
	}
}

func TestActionCancellationAndActivationFailure(t *testing.T) {
	for _, cancel := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}, {Type: tea.KeyRunes, Runes: []rune{'q'}}} {
		m := NewModel([]Profile{{"one", "One", "one@example.com", ""}}, "old", func(Profile) (string, error) { return "", errors.New("git denied") })
		key := func(k tea.KeyMsg) tea.Cmd { next, cmd := m.Update(k); m = next.(Model); return cmd }
		enter := tea.KeyMsg{Type: tea.KeyEnter}
		key(enter)
		key(enter)
		if cmd := key(cancel); cmd != nil || strings.Contains(m.View(), "Confirm activation") {
			t.Fatal("confirmation cancellation", m.View())
		}
		key(enter)
		cmd := key(enter)
		if cmd == nil {
			t.Fatal("missing command")
		}
		if key(enter) != nil {
			t.Fatal("duplicate activation")
		}
		next, _ := m.Update(cmd())
		m = next.(Model)
		if m.current != "old" || !strings.Contains(m.View(), "Activation failed: git denied") {
			t.Fatal(m.View())
		}
		key(cancel)
		if !strings.Contains(m.View(), "> Switch profile") {
			t.Fatal("selection cancellation", m.View())
		}
	}
	m := NewModel(nil, "", nil).WithCreator(func(Profile) error { t.Fatal("navigation created profile"); return nil })
	for _, k := range []tea.KeyType{tea.KeyDown, tea.KeyEnter} {
		next, cmd := m.Update(tea.KeyMsg{Type: k})
		m = next.(Model)
		if cmd != nil {
			t.Fatal("menu side effect")
		}
	}
	if !strings.Contains(m.View(), "Profile ID:") {
		t.Fatal("create action unreachable", m.View())
	}
	for _, action := range []int{0, 2} {
		m = NewModel(nil, "", nil)
		m.action = action
		for range 3 {
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(Model)
			if cmd != nil {
				t.Fatal("empty selection command")
			}
		}
		if !strings.Contains(m.View(), "No valid profiles") {
			t.Fatal(m.View())
		}
	}
}

func TestSelector(t *testing.T) {
	count := 0
	m := NewModel([]Profile{{"one", "One", "one@example.com", ""}, {"two", "Two", "two@example.com", "work"}}, "one", func(Profile) (string, error) { count++; return "Activated", nil })
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !strings.Contains(m.View(), "[active]") || !strings.Contains(m.View(), "one@example.com") {
		t.Fatal(m.View())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
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
