# Slice 04 — Load Board and Move Tickets

## Goal

Treat the filesystem as the board: state is derived from folder location, not markdown metadata.

## Scope

- Load markdown tickets from:
  - `.tickcats/backlog/`
  - `.tickcats/ready/`
  - `.tickcats/doing/`
  - `.tickcats/done/`
- Group tickets by state folder.
- Ignore or warn on invalid folders; do not treat them as board states.
- Move tickets between valid states using file moves.
- Preserve filename and file content during moves.
- Do not update `updated` on folder moves.
- Surface malformed tickets as warnings during board load.

## Out of Scope

- No TUI board rendering.
- No search/filter.
- No content edits.

## Tests

- Loading groups tickets by folder state.
- Moving `ready/foo.md` to `doing/foo.md` preserves content.
- Moving to invalid state fails clearly.
- Malformed tickets do not prevent valid tickets from loading.
- Commands targeting malformed tickets return a clear parse error.

## Acceptance Checks

- Board load works from a temp `.tickcats/` tree.
- File moves change folder-derived state only.
- `updated` remains unchanged after move.
- `go test ./...` passes.
