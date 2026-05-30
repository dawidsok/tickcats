# Slice 09.2 — TUI Delete Ticket

## Goal

Support safe delete from TUI with confirmation.

## Scope

- Press `x` on selected ticket to enter delete confirmation mode.
- Confirmation mode:
  - `y` confirms delete
  - `n` or `esc` cancels
  - `q` quits
- Prefer safe delete: move file to `.tickcats/.trash/` instead of permanent removal.
- Reload board after confirmed delete.
- Show status message after delete/cancel.
- If no ticket selected, show `No ticket selected`.

## Out of Scope

- No empty-trash command.
- No restore command.
- No multi-delete.

## Tests

- `x` enters delete confirm mode when ticket selected.
- `n` / `esc` cancels.
- `y` moves file to `.tickcats/.trash/` and reloads board.
- Empty column does not enter confirm mode.

## Acceptance Checks

- In TUI, select ticket, press `x`, then `y`; ticket disappears from board and exists in trash.
- `go test ./...` passes.
