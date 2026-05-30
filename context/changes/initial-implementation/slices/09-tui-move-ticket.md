# Slice 09 — TUI Move Ticket

## Goal

Move selected ticket between workflow columns from TUI.

## Scope

- Board mode uses lowercase `h/j/k/l` for selection navigation.
- Press `m` to enter move mode.
- In move mode, press `l` to move selected ticket one state to the right:
  - backlog → ready
  - ready → doing
  - doing → done
  - done → no-op message
- In move mode, press `h` to move selected ticket one state to the left:
  - done → doing
  - doing → ready
  - ready → backlog
  - backlog → no-op message
- In move mode, press `esc` to return to board mode.
- In board mode, `e` shows an edit-not-implemented message; later it should open `$EDITOR`.
- In board mode, `d` or `enter` opens ticket detail.
- Use existing `store.Move` so filesystem remains source of truth.
- Reload board after successful move.
- Keep selection near moved ticket's new column.
- Show status/error message in footer.

## Out of Scope

- No move-to-specific-state menu.
- No manual reordering with move-mode `j` / `k`; defer until sort modes/manual order storage are designed.
- No detail-view move action.
- No command palette.

## Tests

- Move mode `l` moves selected backlog ticket to ready in model + filesystem.
- Move mode `h` moves selected ready ticket to backlog in model + filesystem.
- Move key on empty column does not panic.
- Move mode `l` on done ticket shows no-op message.
- Move mode `h` on backlog ticket shows no-op message.
- Move mode `j` / `k` shows manual-reorder-not-implemented message.
- Move failure shows error message.

## Acceptance Checks

- In TUI, select backlog ticket, press `m`, then `l`; ticket appears in Ready column.
- In TUI, select ready ticket, press `m`, then `h`; ticket appears in Backlog column.
- `go test ./...` passes.
