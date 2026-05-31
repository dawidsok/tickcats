# Slice 13 — Page Scroll Keys

## Goal

Add vim-like page scroll shortcuts for long board columns and ticket details.

## Scope

- Support `d` as page down.
- Support `u` as page up.
- In board mode, `d/u` move selection down/up by one visible column page.
- In detail mode, `d/u` scroll content down/up by one visible detail page.
- In move mode, keep `d/u` ignored or no-op unless later assigned.
- Preserve existing `j/k` line movement.

## Out of Scope

- No half-page scroll distinction yet.
- No `ctrl+d` / `ctrl+u` unless cheap.
- No mouse wheel support.

## Tests

- Board mode `d` jumps selection down by visible page and updates column scroll.
- Board mode `u` jumps selection up by visible page.
- Detail mode `d` jumps detail scroll down by visible page.
- Detail mode `u` jumps detail scroll up by visible page.
- Scroll clamps at top/bottom.

## Acceptance Checks

- Large backlog can be navigated quickly with `d/u`.
- Long ticket detail can be navigated quickly with `d/u`.
- `go test ./...` passes.
