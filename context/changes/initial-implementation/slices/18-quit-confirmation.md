# Slice 18 — Quit Confirmation

## Goal

Prevent accidental app exit with a simple global quit confirmation flow.

## Scope

- Pressing `q` from any mode opens quit confirmation.
- Quit confirmation keys:
  - `y` confirms quit
  - `q` confirms quit
  - `n` cancels
  - `esc` cancels
- This means both `q y` and `q q` quit.
- Preserve `ctrl+c` as immediate quit everywhere.
- Show clear footer/status text while confirming.
- After cancel, return to the exact previous view/mode.

## Out of Scope

- No unsaved-change detection.
- No session restore.
- No per-mode quit behavior.

## Tests

- Board mode `q` enters quit confirmation.
- Detail mode `q` enters quit confirmation.
- Move mode `q` enters quit confirmation.
- Quit confirmation `n`/`esc` returns to previous mode.
- Quit confirmation `y` quits.
- Quit confirmation `q` quits.
- `ctrl+c` quits immediately from all modes.

## Acceptance Checks

- `q y` quits.
- `q q` quits.
- `q esc` does not quit.
- `q n` does not quit.
- `go test ./...` passes.
