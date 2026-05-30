# Slice 09.1 — TUI Edit Ticket

## Goal

Support `e` in board mode to edit selected ticket in external editor.

## Scope

- Press `e` on selected ticket to open `$EDITOR`.
- If `$EDITOR` is empty, fallback to `vi`.
- Suspend/exit Bubble Tea cleanly while editor runs.
- Reload board after editor exits.
- Show success/error status message.
- If no ticket selected, show `No ticket selected`.

## Out of Scope

- No inline editing.
- No metadata-only TUI form.
- No edit from detail view yet unless cheap.

## Tests

- `e` on empty column shows no-selection status.
- Editor command resolution uses `$EDITOR`, fallback `vi`.
- Edit command targets selected ticket path.
- After edit callback, board reloads.

## Acceptance Checks

- In TUI, select ticket, press `e`; external editor opens ticket file.
- Returning from editor reloads board.
- `go test ./...` passes.
