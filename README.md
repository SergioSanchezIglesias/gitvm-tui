# GitVM TUI

A small terminal interface for creating, selecting, and activating GitVM account profiles. Built with Go and Bubble Tea, it uses a simple local profile format without migration.

## Quick start

Requirements: Go 1.23 or newer, Git on `PATH`, and an interactive terminal. From this source directory:

```sh
go run ./cmd/gitvm
```

GitVM opens in a clean alternate terminal screen and restores your shell when you exit. Its action menu provides **Switch profile**, **Create profile**, and **Delete profile**. Use **↑/↓** or **j/k** to focus an action and **Enter** to choose it; focus, active profiles, destructive actions, successes, and errors have both textual and colour cues.

The action menu includes an original ASCII GitVM banner that stays visible while choosing an action and reappears when you return. It is readable without colour and adds no splash screen, delay, or extra keypress.

To switch, choose **Switch profile**, select a profile, press **Enter**, then press **Enter** again to confirm activation. Merely opening or navigating any view does not change your identity.

Choose **Create profile** (or press **n** from the action menu) to enter a profile ID, Git author name, email, and optional SSH alias. Use **Tab/Shift+Tab** or **↑/↓** to change fields. **Enter** advances to the next field and saves from the alias field; **Backspace** erases the last character. **Esc/Ctrl+C** cancels without saving. Letters such as `q`, `j`, and `k` are ordinary text inside the form.

Choose **Delete profile** to select a stored profile. Deletion requires a second **Enter** confirmation and **Esc**, **q**, or **Ctrl+C** cancels it. The active profile is protected: switch to another profile before deleting it. Validation and storage errors remain visible so you can correct, cancel, or retry. Existing IDs are rejected without overwrite. Successful creation returns to the action menu with the new profile selected, but does not activate it or change Git, SSH, or the current-profile marker.

This creates profile metadata, not an SSH key. Existing SSH keys are optional for activation; see the safety behavior below.

## Install and update

Published packages become available after the first stable release. Git and an
interactive terminal are needed for the TUI; non-interactive commands need no TTY.
Choose one installation method to avoid competing executables on `PATH`.

### Homebrew (macOS and Linux)

```sh
brew install SergioSanchezIglesias/tap/gitvm
# Later:
brew update
brew upgrade gitvm
```

Homebrew installs a prebuilt binary for Intel/ARM64 macOS or Linux and manages
version replacement. It does not require Go. If installation or linking fails,
read Homebrew's error and retry; do not use `sudo` or `brew link --overwrite` to
replace an unrelated executable.

### Go

Requires Go 1.23 or newer:

```sh
go install github.com/SergioSanchezIglesias/gitvm-tui/cmd/gitvm@latest
```

Repeat the command to update, or replace `latest` with a released version such as
`v0.1.0` to install that version explicitly. The binary goes to `GOBIN`, or
`$(go env GOPATH)/bin` when unset. Keep this user-owned directory before any
legacy installation on `PATH`; the command writes only to the selected Go bin
directory. Avoid a system-wide `GOBIN`.

Run `command -v gitvm` to check which installation wins. Neither method needs
`sudo` or modifies profile data during installation. A failed download/build
does not require deleting a working installation; correct the error and retry.
To return to a previous release, install its explicit Go version.

### Release downloads

[GitHub Releases](https://github.com/SergioSanchezIglesias/gitvm-tui/releases)
provide `gitvm_<version>_<darwin|linux>_<amd64|arm64>.tar.gz` and `checksums.txt`.
Verify the archive with `sha256sum --check --ignore-missing checksums.txt` on
Linux, or `shasum -a 256 -c checksums.txt --ignore-missing` on macOS, before
extracting. For manual installation, extract into a temporary directory and
copy `gitvm` into a user-owned bin directory. Leave any system-wide script alone.

### Build from source

Build a local executable:

```sh
go build -o gitvm ./cmd/gitvm
./gitvm
```

Or install from this source checkout:

```sh
go install ./cmd/gitvm
```

The installed executable uses the same Go bin directory described above.
The command was previously named `gitvm-tui`; new builds and packages use `gitvm`.

## Command-line activation

Activate an existing profile without opening the TUI or asking for confirmation:

```sh
gitvm --user personal
# Short form:
gitvm -u personal
# From the source checkout:
go run ./cmd/gitvm --user personal
```

`personal` is the exact, case-sensitive stored **profile ID** (the filename under
`~/.gitvm/profiles`), not a Git author name, email, or GitHub account. Create the
profile in the TUI first. Activation needs no interactive terminal. Running
`gitvm` without arguments still opens the TUI.

Use `gitvm --help` or `gitvm -h` for usage without changing anything.
Help and list output use colors on supported terminals: cyan headings and commands,
yellow options and email addresses, bold profile names, and green for the saved
active profile. Text and the `*` marker remain readable without color. Set
`NO_COLOR=1` to disable styling. Redirected output, `TERM=dumb`, and an unset or
empty `TERM` always produce plain text, preserving tab-separated list fields.

The user option takes exactly one ID as a separate argument; help must be used
alone. Unknown options, extra arguments, missing/invalid IDs, and unknown profiles
produce an error on stderr and exit with status `1`, without modifying state.
Success prints the activation result (including whether SSH stayed unchanged)
and exits with status `0`; help also exits with `0`.

This uses the same activation as the TUI: **global Git name/email defaults**, the
current-profile marker, and SSH only under the existing safety rules below.
Repository-local Git settings and GitHub CLI authentication are not changed.
Activation failures report an error on stderr and exit with status `1`. Activation
is not a transaction: earlier changes may remain applied after a later failure;
there is no whole-operation rollback.

## List profiles

Run `gitvm --list` or `gitvm -l` to print valid stored profiles without
opening the TUI or changing Git, SSH, profiles, or the current marker:

```text
  personal    Personal User    personal@example.com
* work        Work User        work@example.com
```

Each line contains the profile ID, name, and email, separated by tabs. `*` marks
the saved active profile from `~/.gitvm/current`, not a live Git identity check.
Profiles are listed in filename order; malformed records are skipped, just as in
the TUI. A missing or empty profiles directory prints `No profiles found.`.
If the saved active ID is missing, no profile is marked.

The option must be used alone. Listing succeeds with exit status `0`, including
an empty list; extra arguments and storage failures report errors on stderr and
exit with status `1`.

## Profile storage

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

Creation writes a complete record with mode `0600` and publishes it atomically without overwriting an existing entry. New storage directories use mode `0700`; permissions of existing directories are unchanged. Publication requires filesystem hard-link support. Records without an alias contain only two lines.

Only regular files with valid fields and two or three lines are loaded; malformed profiles and non-regular entries are skipped. A missing profiles directory produces an empty selector. Other read errors can prevent startup. The `[active]` marker comes from `~/.gitvm/current`, not a live check of Git or GitHub authentication.

## Controls

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Move through menu actions and profiles |
| `Enter` | Choose the focused action or profile; press again to confirm activation or deletion |
| `n` | Open the create-profile form from the action menu |
| `Tab` / `Shift+Tab`, `↑` / `↓` | Move between create-profile form fields |
| `Esc` / `Ctrl+C` | Cancel a form or confirmation; return to the action menu; quit from the menu |
| `q` | Return to the action menu or quit from it; ordinary text in the create form |

Input is ignored while activation or profile creation is running. Success and failure messages appear in the interface.

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

The TUI creates, lists, safely deletes inactive profiles, shows the saved active selection, and activates a confirmed selection. It does not edit profiles, delete the active profile, clone repositories, generate SSH keys, authenticate with GitHub, or manage per-repository identities.

## Tests

Run the full Go test suite from the source directory:

```sh
go test ./... -count=1
```

The tests use temporary home directories and injected Git runners to exercise profile creation, loading, form navigation, selection, activation, and SSH safety without changing your real Git or SSH configuration.

## License

[MIT](LICENSE) — Copyright (c) 2026 Sergio Sanchez Iglesias.

## Publishing releases (maintainers)

1. Create a fine-grained GitHub token scoped only to
   `SergioSanchezIglesias/homebrew-tap`, with **Contents: read and write**.
   Add it as the `HOMEBREW_TAP_TOKEN` Actions secret in this repository.
   Never commit or paste the token into source or issue comments.
2. Merge the release configuration and run the checks above, plus
   `python3 -m unittest discover -s scripts -p 'test_*.py'`.
3. Tag the reviewed commit with stable SemVer (for example `v0.1.0`) and push
   that tag. The Release workflow tests, builds four archives with GoReleaser
   2.19.0, publishes checksums, and updates `Formula/gitvm.rb` in the tap.

No versions are published just by building locally. Prerelease tags are rejected.
The normal repository token publishes releases; the separate tap token is used
only by the Homebrew job. If that job fails, the release remains available:
correct the credential or tap error and rerun **only the failed Homebrew job**.
Do not recreate the tag or rerun the successful release job.
