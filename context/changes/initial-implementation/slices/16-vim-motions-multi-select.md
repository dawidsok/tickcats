# Slice 16 — Vim Motions and Multi-Select

## Goal

Add faster vim-style navigation and allow selecting multiple tickets for batch actions.

## Scope

- Add vim motions in board mode:
  - `g g` jumps to first ticket in current column
  - `G G` jumps to last ticket in current column
  - `0` jumps to backlog column
  - `$` jumps to done column
- Add visual selection mode:
  - `v` enters/leaves selection mode
  - `j/k` expands selection within current column
  - `esc` clears selection and returns to board mode
- Track selected ticket set by stable ticket path.
- Render selected tickets distinctly from cursor row.
- Batch move selected tickets in move mode with `h/l`.
- Batch safe-delete selected tickets with `x` + confirmation.

## Out of Scope

- No cross-column range selection in v1 of this slice unless cheap.
- No mouse selection.
- No clipboard/register behavior.
- No arbitrary count prefixes like `5j` yet.
- No batch edit.

## Tests

- `gg` selects first ticket.
- `GG` selects last ticket.
- `0` / `$` jump to first/last columns.
- `v` toggles visual selection mode.
- Selection expands with `j/k` and stores paths.
- Batch move moves all selected tickets to target state.
- Batch delete moves all selected tickets to trash after confirmation.

## Acceptance Checks

- User can select several tickets in one column and move/delete them together.
- Cursor row and selected rows are visually distinguishable.
- `go test ./...` passes.
