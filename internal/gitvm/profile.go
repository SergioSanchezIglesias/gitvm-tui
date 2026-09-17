package gitvm

import (
	"fmt"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Profile struct{ ID, Name, Email, Alias string }

var aliasPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func clean(s string) bool {
	return strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}
func (p Profile) Validate() error {
	if p.ID == "" || p.ID == "." || p.ID == ".." || strings.ContainsAny(p.ID, "/\\") || !clean(p.ID) || p.Name == "" || !clean(p.Name) || !clean(p.Email) {
		return fmt.Errorf("invalid profile fields")
	}
	a, err := mail.ParseAddress(p.Email)
	if err != nil || a.Address != p.Email {
		return fmt.Errorf("invalid email")
	}
	if p.Alias != "" && !aliasPattern.MatchString(p.Alias) {
		return fmt.Errorf("invalid SSH alias")
	}
	return nil
}
func Load(home string) ([]Profile, string, error) {
	dir := filepath.Join(home, ".gitvm", "profiles")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	var profiles []Profile
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, "", err
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		if len(lines) < 2 || len(lines) > 3 {
			continue
		}
		p := Profile{ID: e.Name(), Name: lines[0], Email: lines[1]}
		if len(lines) == 3 {
			p.Alias = lines[2]
		}
		if p.Validate() == nil {
			profiles = append(profiles, p)
		}
	}
	current, err := os.ReadFile(filepath.Join(home, ".gitvm", "current"))
	if err != nil && !os.IsNotExist(err) {
		return nil, "", err
	}
	return profiles, strings.TrimSuffix(string(current), "\n"), nil
}

// Create publishes a complete private record without replacing any existing entry.
func Create(home string, p Profile) error {
	if err := p.Validate(); err != nil {
		return err
	}
	dir := filepath.Join(home, ".gitvm", "profiles")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".gitvm-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	data := p.Name + "\n" + p.Email + "\n"
	if p.Alias != "" {
		data += p.Alias + "\n"
	}
	if err = f.Chmod(0600); err == nil {
		_, err = f.WriteString(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	// A same-directory hard link atomically publishes the flushed inode and
	// fails if the destination exists, including a directory or symlink.
	if err := os.Link(f.Name(), filepath.Join(dir, p.ID)); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("profile %q already exists", p.ID)
		}
		return err
	}
	return nil
}

// Delete removes a valid regular profile record. The caller owns active-profile
// policy; this operation never changes the current marker.
func Delete(home, id string) error {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, "/\\") || !clean(id) {
		return fmt.Errorf("invalid profile ID %q", id)
	}
	dir := filepath.Join(home, ".gitvm", "profiles")
	// Reject redirected storage as well as non-regular profile entries.
	for _, path := range []string{filepath.Join(home, ".gitvm"), dir} {
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("delete profile %q: %w", id, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("delete profile %q: storage %q is not a directory", id, path)
		}
	}
	path := filepath.Join(dir, id)
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("delete profile %q: %w", id, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("delete profile %q: not a regular profile record", id)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read profile %q for deletion: %w", id, err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) < 2 || len(lines) > 3 {
		return fmt.Errorf("invalid profile record %q", id)
	}
	p := Profile{ID: id, Name: lines[0], Email: lines[1]}
	if len(lines) == 3 {
		p.Alias = lines[2]
	}
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid profile record %q: %w", id, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete profile %q: %w", id, err)
	}
	return nil
}

func RunGit(args ...string) error {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// atomicWrite replaces a file only after its complete contents have been flushed.
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".gitvm-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
func Activate(home string, p Profile, run func(...string) error) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	config := filepath.Join(home, ".ssh", "config")
	var updated []byte
	mode := os.FileMode(0600)
	hasKey := false
	if p.Alias != "" {
		info, err := os.Stat(filepath.Join(home, ".ssh", "id_ed25519_"+strings.ToLower(p.Alias)))
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		hasKey = err == nil && info.Mode().IsRegular()
	}
	if hasKey {
		data, err := os.ReadFile(config)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if info, err := os.Stat(config); err == nil {
			mode = info.Mode().Perm()
		}
		text, err := updateSSH(string(data), strings.ToLower(p.Alias))
		if err != nil {
			return "", err
		}
		updated = []byte(text)
	}
	if err := run("config", "--global", "user.name", p.Name); err != nil {
		return "", err
	}
	if err := run("config", "--global", "user.email", p.Email); err != nil {
		return "", fmt.Errorf("Git name may have changed: %w", err)
	}
	if hasKey {
		if err := atomicWrite(config, updated, mode); err != nil {
			return "", fmt.Errorf("Git identity changed; SSH update failed: %w", err)
		}
	}
	if err := atomicWrite(filepath.Join(home, ".gitvm", "current"), []byte(p.ID+"\n"), 0600); err != nil {
		return "", fmt.Errorf("identity changed; current profile not saved: %w", err)
	}
	if !hasKey {
		return "Activated " + p.ID + ". SSH unchanged: no matching alias key.", nil
	}
	return "Activated " + p.ID + ". SSH updated.", nil
}
func updateSSH(data, alias string) (string, error) {
	lines := strings.SplitAfter(data, "\n")
	var out strings.Builder
	target, found := false, false
	// Global, conditional, and potentially overlapping sections cannot safely
	// supply identity options: IdentityFile accumulates even after our stanza.
	ambiguous := true
	for _, line := range lines {
		text := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		fields := strings.Fields(strings.Replace(text, "=", " ", 1))
		if len(fields) > 0 {
			key := strings.ToLower(fields[0])
			if key == "host" || key == "match" {
				target = key == "host" && len(fields) == 2 && strings.EqualFold(strings.Trim(fields[1], "\""), "github.com")
				ambiguous = key == "match"
				if key == "host" && !target {
					ambiguous = hostPatternsMatchGitHub(fields[1:])
				}
				if target {
					found = true
					out.WriteString(strings.TrimSuffix(line, "\n") + "\n")
					out.WriteString(sshIdentity(alias))
					continue
				}
			}
			// Includes may hide matching sections; do not read or rewrite them.
			if key == "include" || (ambiguous && (key == "user" || key == "identityfile" || key == "hostname" || key == "identitiesonly")) {
				return "", fmt.Errorf("cannot guarantee SSH identity precedence: %s outside an exclusive Host github.com section; SSH config unchanged", fields[0])
			}
			if target && (key == "identityfile" || key == "hostname" || key == "user" || key == "identitiesonly") {
				continue
			}
		}
		out.WriteString(line)
	}
	if !found {
		if data != "" && !strings.HasSuffix(data, "\n") {
			out.WriteByte('\n')
		}
		out.WriteString("Host github.com\n" + sshIdentity(alias))
	}
	return out.String(), nil
}

// Host patterns use only '*' and '?' as wildcards, not filesystem glob syntax.
// A matching negation vetoes the entire section; negations alone never select it.
func hostPatternsMatchGitHub(patterns []string) bool {
	matched := false
	for _, pattern := range patterns {
		pattern = strings.Trim(pattern, "\"'")
		negated := strings.HasPrefix(pattern, "!")
		if negated {
			pattern = strings.TrimPrefix(pattern, "!")
		}
		expression := regexp.QuoteMeta(strings.ToLower(pattern))
		expression = strings.ReplaceAll(expression, `\*`, `.*`)
		expression = strings.ReplaceAll(expression, `\?`, `.`)
		if regexp.MustCompile("^" + expression + "$").MatchString("github.com") {
			if negated {
				return false
			}
			matched = true
		}
	}
	return matched
}

func sshIdentity(alias string) string {
	return "  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_" + alias + "\n  IdentitiesOnly yes\n"
}
