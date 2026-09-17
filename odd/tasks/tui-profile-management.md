# TUI Profile Management

## Objective
Evolve GitVM into a polished, keyboard-first terminal application with explicit profile-management actions, guarded deletion, semantic visual states, and a clean alternate-screen presentation.

## Problem and rationale
The current selector mixes profile browsing and activation in one plain-text view. Creating depends on a hidden `n` shortcut, profiles cannot be removed, and terminal output remains in shell scrollback. The approved issue requires clearer actions and safe profile deletion without changing activation semantics.

## Scope
- Add explicit **Switch profile**, **Create profile**, and **Delete profile** actions.
- Add a guarded delete operation for stored profiles; do not allow deletion of the active profile.
- Refactor the Bubble Tea UI into explicit interaction states with cancellation and confirmation paths.
- Use semantic terminal styling and Bubble Tea alternate-screen mode for a polished, uncluttered presentation.
- Preserve keyboard-first use, readable non-colour cues, profile format, and activation safety behavior.
- Update focused tests and user documentation.

## Non-goals
- Change profile file format or migration behavior.
- Generate keys, manage GitHub authentication, add mouse-only interaction, or delete an active profile.
- Change Git/SSH activation behavior.

## Constraints
- Technical artifacts and UI copy use English.
- UI state changes and navigation must never mutate Git, SSH, storage, or the active marker.
- Deletion requires an explicit confirmation and has a cancellation path.
- Colour is never the only signal for focus, active state, destructive actions, success, or errors.
- TDD mode: strict (inherited from the existing TUI work); record RED before each behavior change, then GREEN. Verification runner: `go test ./... -count=1`; final checks: `go fmt ./...`, `go test ./... -count=1`, `go vet ./...`, and `go build ./...`.

## Tasks
- [x] TPM-1 Add safe profile deletion storage support and its tests. (RED: undefined `Delete`; GREEN: `go test ./... -count=1` passed.)
- [x] TPM-2 Implement action-oriented styled TUI flows, alternate screen mode, and interaction tests. (Regression fixed: Switch profile preserves the newly created profile selection; focused checks passed.)
- [x] TPM-3 Update README controls, deletion behavior, and TUI presentation documentation. (Controls now match literal creation-form input behavior.)
- [x] TPM-4 Create the approved three-PR stack, apply repository policy labels, and link issue #1. (PRs #3–#5 opened; issue #1 closed.)

## Acceptance criteria
- Users can reach Switch profile, Create profile, and Delete profile using documented keyboard navigation.
- The focused item, active profile, destructive flow, successes, and failures are visually and textually distinguishable.
- Creation and switching retain existing safety behavior; selection/navigation never activates a profile.
- Deletion only occurs after confirmation, can be cancelled, clearly reports failures, and rejects the active profile.
- The application runs in a clean alternate screen and restores the terminal on exit.
- Relevant storage, model, visual, cancellation, and confirmation tests pass.

## Progress
- TPM-1 complete: `Delete` validates IDs and profile-record safety, refuses non-regular or redirected storage, and preserves the current marker and unrelated records.
- TPM-2 complete: explicit menu, selection and confirmation states preserve mutation boundaries; active-profile deletion is rejected; Lip Gloss provides framed semantic styling; alternate-screen mode is enabled; and Switch profile preserves a newly created profile selection.
- TPM-3 complete: README documents the action menu, confirmations, protected active profiles, alternate-screen presentation, and literal creation-form key behavior.
- TPM-4 complete: PR #3 provides safe deletion, PR #4 provides interaction state flows, and PR #5 provides presentation/documentation. The user repaired GitHub SSH authentication after the initial push failure; all branches were pushed, labels applied, and issue #1 was closed.

## Verification evidence
- TPM-1 RED: `go test ./... -count=1` failed with five undefined `Delete` errors.
- TPM-1 GREEN: `go test ./... -count=1` passed; `git diff --check` passed.
- TPM-2 RED: `go test ./... -count=1` failed because `WithDeleter` was undefined.
- TPM-2 GREEN: `go fmt ./...`, `go test ./... -count=1`, `go vet ./...`, and `go build ./...` passed; parent spot check reran `go test ./... -count=1` successfully.
- Independent verification initially found a creation-selection regression and inaccurate controls documentation; both were corrected.
- Regression RED: the new selection test failed because Switch profile selected the prior profile. GREEN: `go test ./... -count=1` and `git diff --check` passed after the one-line cursor fix.
- Final independent verification: read-only `gofmt -d` output was empty; `go test ./... -count=1`, `go vet ./...`, `go build ./...`, and `git diff --check` all passed. Interactive alternate-screen restoration and colour rendering remain unverified outside a real terminal.
- Native review unavailable: its package-local binary is missing, so RDD risk assessment was unassessable; the required independent verifier nevertheless completed successfully.

## Next step
- Review and merge the stack in order: PR #3, then #4, then #5.
