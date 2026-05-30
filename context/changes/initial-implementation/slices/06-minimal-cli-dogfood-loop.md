# Slice 06 — Minimal CLI Dogfood Loop

## Goal

Expose the core engine through plain CLI commands before building the full TUI.

## Scope

Add minimal commands:

- `tickcats init`
- `tickcats new feat|task|bug`
- `tickcats list`
- `tickcats move <ticket> <state>`
- `tickcats pick-next`

Behavior:

- `new feat` generates title prefix `Feat:`.
- `new task` generates title prefix `Task:`.
- `new bug` generates title prefix `Bug:`.
- Generated filename format: proposed `YYYYMMDD-HHMM-<slug>.md` unless resolved otherwise before implementation.
- `list` can be simple text grouped by state.
- `pick-next` prints the selected ticket or tie candidates.

## Out of Scope

- No Bubble Tea board UI.
- No command palette.
- No full-screen detail view.
- No release packaging.

## Tests

- CLI init works in temp directory.
- CLI new creates a parseable markdown ticket.
- CLI move changes folder state.
- CLI pick-next returns expected ticket after setup.

## Acceptance Checks

- From a temp repo, user can initialize, create a task, move it to ready, and pick it.
- Commands work offline.
- `go test ./...` passes.
- Manual smoke test via `go run ./cmd/tickcats ...` succeeds.
