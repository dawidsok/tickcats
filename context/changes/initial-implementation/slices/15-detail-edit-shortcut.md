# Slice 15 — Detail Edit Shortcut

## Goal

Allow editing the currently opened ticket directly from detail view.

## Scope

- In detail view, press `e` to open selected ticket in `$EDITOR`.
- Reuse existing board edit flow and editor command behavior.
- After editor exits, reload board and remain in detail view if the ticket still exists.
- If edited ticket becomes invalid or disappears, return to board with a status message.
- Keep `esc` returning from detail to board.

## Out of Scope

- No inline detail editing.
- No metadata form.
- No edit conflict handling.
- No edit from delete confirmation/move modes.

## Tests

- Detail mode `e` starts editor command for selected ticket.
- Editor completion reloads board and keeps detail mode when selected ticket still exists.
- Editor completion returns to board when selected ticket is gone/invalid.
- `esc` behavior unchanged.

## Acceptance Checks

- Open ticket details with `o`/`enter`, press `e`, edit file, quit editor.
- TickCats returns to detail view with reloaded content.
- `go test ./...` passes.
