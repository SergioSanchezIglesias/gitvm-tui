# TUI Profile Selector

## Objective
Replace the shell-only profile switching experience with a keyboard-driven Go/Bubble Tea TUI that lists registered GitVM profiles and safely activates one.

## Problem and rationale
`gitvm` can switch profiles, but users must remember aliases and cannot inspect all registered profiles while choosing. The TUI should make the active profile and profile metadata visible before an explicit switch.

## Scope
- Read legacy profiles from `~/.gitvm/profiles` and the active profile from `~/.gitvm/current`.
- Show profile name, email, alias, and active state in a Bubble Tea list.
- Provide keyboard selection, confirmation, cancellation, and status/error feedback.
- On confirmation, update Git global `user.name` and `user.email`, write the selected profile ID to `~/.gitvm/current`, and update only the `Host github.com` section in `~/.ssh/config` when an expected SSH key exists.
- Provide a standalone executable command while preserving the legacy `gitvm` script untouched.

## Non-goals
- Creating, editing, or removing profiles.
- Generating SSH keys, authenticating GitHub CLI, cloning repositories, or migrating stored profiles.
- Rewriting all of `~/.ssh/config`.

## Constraints
- Technical artifacts and UI copy use English.
- Preserve compatibility with the existing three-line profile format.
- Validate profile path components and never treat profile data as shell code or SSH configuration syntax.
- TDD mode: strict (user-selected); record an observed failing `go test ./...` run before implementation, then GREEN and run `go fmt ./...`, `go test ./...`, `go vet ./...`, and `go build ./...`.

## Tasks
- [x] TUI-1 Define Go module, domain model, and safe legacy-profile storage operations. (Implemented by worker; baseline Go checks passed.)
- [x] TUI-2 Implement the Bubble Tea profile selector and safe profile activation flow. (Implemented and corrected for SSH precedence and wildcard compatibility.)
- [x] TUI-3 Add focused tests and run formatting, tests, vet, and build checks. (Strict RED/GREEN evidence and independent verification completed.)

## Acceptance criteria
- The executable lists valid legacy profiles and clearly marks the active one.
- A user can navigate with arrows or j/k, confirm an activation, and quit without changing anything.
- Activation persists Git identity and selected profile, and updates SSH only when the alias-derived key exists.
- Invalid profile data is rejected without executing shell commands or rewriting unrelated SSH configuration.
- The original `gitvm` script remains unchanged.

## Progress
- TUI-1 complete: module, model, profile storage, activation logic, and tests were implemented.
- TUI-2 complete: SSH matching now rejects only identity-relevant Host sections that can match `github.com`; unrelated wildcard sections are preserved. `Match` and `Include` remain fail-safe rejection cases.
- TUI-3 complete: focused tests, vet, and build passed after a fresh independent verification.

## Verification evidence
- Worker RED: `go test ./...` initially failed with undefined `Load`, `Profile`, `Activate`, and `NewModel`; GREEN then passed.
- Worker: `go mod tidy`, `go fmt ./...`, `go test ./...`, `go vet ./...`, and `go build ./...` passed.
- Independent verification after the final correction: `go test ./... -count=1` passed (`internal/gitvm` in 0.364s); `go vet ./...` and `go build ./...` passed. Parent spot check: `go test ./... -count=1` passed (`internal/gitvm` in 0.269s).

## Next step
Completed. The new command is ready for use as `go run ./cmd/gitvm-tui` or after building it.
