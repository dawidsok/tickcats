# Slice 07 — TUI Board Skeleton

## Goal

Render first interactive terminal board using loaded `.tickcats/` data.

## Scope

- Add Bubble Tea stack dependencies.
- Add `tickcats tui` command.
- Load board with existing `store.LoadBoard`.
- Render columns: Backlog, Ready, Doing, Done.
- Show pick-next banner above board using `store.PickNext`.
- Support keyboard navigation:
  - `h` / `l`: move between columns
  - `j` / `k`: move within selected column
  - `q`: quit
- Highlight selected column/ticket in simple text style.
- Show parse warnings in a small footer or warning section.

## Out of Scope

- No full-screen ticket detail.
- No command palette.
- No moving tickets from TUI yet.
- No search/filter.
- No label toggling.

## Tests

- Model initializes from board data.
- Navigation clamps at column and row boundaries.
- Empty columns do not panic.
- Pick-next banner renders no-pick state and selected-ticket state.

## Acceptance Checks

- `tickcats tui` opens a board without crashing in an initialized repo.
- `h/l/j/k/q` work.
- `go test ./...` passes.
