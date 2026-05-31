# Slice 12 — Board Sorting

## Goal

Make board columns deterministic and useful by sorting tickets by work priority instead of filename only.

## Scope

- Sort tickets in each board column by default:
  1. priority: `P0` → `P3`
  2. oldest `created`
  3. filename ascending
- Keep pick-next sorting behavior aligned with board sorting where possible.
- Preserve selection after reload/move/edit/delete when the selected ticket still exists.
- Document current sort policy in help/docs if needed.

## Out of Scope

- No persisted per-project sort preference.
- No manual reorder storage.
- No hotkey sort switching yet.
- No drag/drop or multi-select.

## Tests

- `store.LoadBoard` returns each column sorted by priority, then created, then filename.
- Equal priority + equal created sorts by filename.
- Board view follows sorted store order.
- Moving/editing/reloading keeps selection near the affected ticket.

## Acceptance Checks

- In TUI, higher-priority tickets appear above lower-priority tickets in each column.
- Older tickets appear first when priority is equal.
- `go test ./...` passes.
