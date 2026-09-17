# GitVM TUI

A small terminal interface for selecting and activating existing GitVM account profiles. Built with Go and Bubble Tea, it reads the legacy `gitvm` profile files without migration.

## Quick start

Requirements: Go 1.23 or newer, Git on `PATH`, and an interactive terminal. From this source directory:

```sh
go run ./cmd/gitvm-tui
```

Choose a profile, press **Enter** to select it, then **Enter** again to activate it. Merely opening or navigating the selector does not change your identity.

If you have no profiles, create one with the included legacy script before starting the TUI:

```sh
bash ./gitvm add "Example User" user@example.com personal
```

This creates profile metadata, not an SSH key. Existing SSH keys are optional for activation; see the safety behavior below.

### Build or install

Build a local executable:

```sh
go build -o gitvm-tui ./cmd/gitvm-tui
./gitvm-tui
```

Or install from this source checkout:

```sh
go install ./cmd/gitvm-tui
```

The installed executable goes to `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. Add that directory to `PATH` to run `gitvm-tui` anywhere.

## Legacy profile storage

The TUI uses the current user's home directory:

| Path | Purpose |
| --- | --- |
| `~/.gitvm/profiles/<profile-id>` | Plain-text profile; the filename is its ID |
| `~/.gitvm/current` | Active profile ID, followed by a newline |
| `~/.ssh/id_ed25519_<lowercase-alias>` | Optional existing key used for SSH activation |
| `~/.ssh/config` | SSH configuration updated when a matching key exists |

Each profile contains two required lines and an optional third line:

```text
Example User
user@example.com
personal
```

The lines are Git author name, email address, and optional SSH alias. The alias may contain only ASCII letters, digits, underscores, and hyphens. Names and emails must not contain control characters or surrounding whitespace; the email must parse as a bare address. Profile IDs must be safe single filenames.

Only regular files with valid fields and two or three lines are loaded; malformed profiles and non-regular entries are skipped. A missing profiles directory produces an empty selector. Other read errors can prevent startup. The `[active]` marker comes from `~/.gitvm/current`, not a live check of Git or GitHub authentication.

## Controls

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Move through profiles |
| `Enter` | Select, then press again to confirm activation |
| `Esc`, `q`, `Ctrl+C` | Cancel a pending confirmation; otherwise quit |

Input is ignored while activation is running. Success and failure messages appear in the interface.

## What activation changes

After validating the profile and any required SSH rewrite, the TUI performs these operations in order:

1. Sets `git config --global user.name` and `git config --global user.email`.
2. If a matching alias key exists as a regular file, updates `~/.ssh/config` for `github.com`.
3. Saves the profile ID to `~/.gitvm/current`.

**This changes your global Git defaults**, not repository-local settings. Local Git configuration can still override them. The TUI does not switch GitHub CLI accounts.

SSH configuration and the current-profile marker are each written using atomic file replacement. The entire activation is **not a transaction**: a later failure can leave earlier changes applied. Read the error message before retrying. Existing SSH configuration permissions are retained; a new SSH configuration and the current marker use mode `0600`. No backup is created by the TUI.

## SSH safety behavior

With no alias or no matching regular key file, activation still sets the Git identity and current marker, but leaves SSH unchanged and reports that explicitly. It does not generate keys, inspect private-key contents, load an SSH agent, upload keys, or test authentication. Provision and register keys separately.

With a matching key, the TUI updates exclusive `Host github.com` sections, or appends one if absent, with:

```sshconfig
Host github.com
  HostName github.com
  User git
  IdentityFile ~/.ssh/id_ed25519_personal
  IdentitiesOnly yes
```

Unrelated host sections and other options are preserved. Safety checks reject configurations whose identity precedence cannot be guaranteed:

- Any `Include` directive is rejected; included files are not read or rewritten.
- `User`, `IdentityFile`, `HostName`, or `IdentitiesOnly` outside a host section, inside a `Match` section, or in another host section matching `github.com` cause rejection.
- Matching wildcard and multi-host sections are checked even after the exclusive section, because `IdentityFile` can accumulate. Matching negations exclude a section. A `Match` section containing only unrelated options is preserved rather than rejected unconditionally.

These precedence rejections happen **before Git, SSH, or the current marker is changed**. Simplify the conflicting configuration manually if you want the TUI to manage the GitHub identity; it does not attempt to resolve arbitrary SSH configuration semantics.

## Scope and non-goals

The TUI lists existing profiles, shows the saved active selection, and activates a confirmed selection. It does not add, edit, or delete profiles; clone repositories; generate SSH keys; authenticate with GitHub; or manage per-repository identities.

The included Bash `gitvm` script remains a separate legacy tool with broader commands. Its behavior and safety guarantees differ: notably, `setup` replaces SSH configuration after making a backup, and legacy switching may invoke `gh`. The TUI's SSH preservation and rejection rules do not apply to that script.

## Tests

Run the full Go test suite from the source directory:

```sh
go test ./... -count=1
```

The tests use temporary home directories and injected Git runners to exercise profile loading, selection, activation, and SSH safety without changing your real Git or SSH configuration.

## License

[MIT](LICENSE) — Copyright (c) 2026 Sergio Sanchez Iglesias.
