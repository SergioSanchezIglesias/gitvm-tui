# TUI Profile Creation and Legacy Script Retirement

## Objective
Add profile creation to the GitVM TUI so users can register accounts without the legacy Bash script, then remove that script from the project.

## Problem and rationale
The current TUI reads the legacy three-line profile format but cannot create profiles. Removing `gitvm` without replacing its `add` capability would force users to hand-edit files in `~/.gitvm/profiles`.

## Scope
- Add a keyboard-driven TUI flow to create a profile with a safe profile ID, Git author name, email, and optional SSH alias.
- Validate input using the existing profile rules and persist compatible records atomically under `~/.gitvm/profiles`.
- Reject duplicate profile IDs without overwriting an existing profile.
- Preserve the selector flow and list the newly created profile after success.
- Remove the legacy `gitvm` Bash script only after the new flow is implemented and tested.
- Update public documentation to replace legacy-script setup instructions.

## Non-goals
- SSH key generation, GitHub CLI authentication, profile editing/deletion, or automatic profile activation after creation.
- Migrating profile storage or changing the existing profile file format.

## Constraints
- Follow strict TDD, continuing the previously user-selected project test practice; runner: `go test ./... -count=1`.
- Use temporary homes and mocked system boundaries; never modify actual user profiles, Git config, or SSH config in tests.
- Keep the existing MIT license and public repository settings unchanged.

## Tasks
- [x] CREATE-1 Define profile-creation storage contract and focused test scenarios. (Strict-TDD scenarios added for valid, invalid, duplicate, storage failure, permissions, cancellation, and navigation behavior.)
- [x] CREATE-2 Implement the Bubble Tea profile-creation form and safe persistence. (Implemented and verified GREEN by worker.)
- [x] CREATE-3 Remove the legacy script and update README usage. (Parent removed `gitvm`; README now documents TUI-only creation.)
- [x] CREATE-4 Run focused tests, vet, build, and independent verification. (All checks passed; commit and push require separate explicit user authorization.)

## Acceptance criteria
- A user can enter profile ID, name, email, and an optional alias from the TUI.
- Invalid fields and duplicate IDs produce a useful error without changing stored data.
- Valid creation writes the compatible two- or three-line profile record atomically with private permissions.
- A newly created profile appears in the selector without restarting the program.
- `gitvm` is absent from the repository and README documents the TUI-only workflow.

## Progress
- CREATE-1 complete: strict-TDD storage and UI scenarios were defined and observed RED before implementation.
- CREATE-2 complete: profile form, validation, atomic no-overwrite persistence, and immediate selector update implemented.
- CREATE-3 complete: legacy `gitvm` script removed after creation capability was verified; README now describes TUI-only profile creation.
- CREATE-4 complete: fresh tests, vet, build, diff check, independent verification, and parent test spot check passed.
- Delivery pending: commit and push were not performed because this change has not received separate explicit delivery authorization.

## Next step
Await explicit authorization to commit and push the verified change.
