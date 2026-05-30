# Slice 08 — TUI Ticket Detail

## Goal

Open selected ticket in full-screen read-only detail view.

## Scope

- Press `enter` on selected ticket to open detail view.
- Detail view replaces board.
- Show ticket title, state, priority, filename, labels, and markdown body.
- Long content scrolls with `j` / `k`.
- `esc` returns to board.
- `q` quits from either view.
- Direct hotkey footer only; no focusable action buttons.

## Out of Scope

- No inline markdown editing.
- No external editor integration yet.
- No move/block/toggle actions yet.
- No command palette.

## Tests

- Enter on selected ticket switches to detail mode.
- Enter on empty column stays on board.
- Esc returns to board.
- Detail scroll clamps at top and bottom.
- Detail view contains selected ticket metadata and body.

## Acceptance Checks

- `tickcats` opens TUI by default.
- Selected ticket opens with `enter`.
- Long detail content scrolls with `j/k`.
- `esc` returns to board.
- `go test ./...` passes.
