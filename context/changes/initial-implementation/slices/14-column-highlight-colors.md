# Slice 14 — Column Highlight Colors

## Goal

Make the selected board column easier to identify by giving each workflow column its own highlight color.

## Scope

- Assign a distinct accent color per state:
  - backlog
  - ready
  - doing
  - done
- When a column is selected, render its border/header with that state color.
- Keep non-selected columns muted.
- Keep selected ticket row visibly selected within the active column.
- Preserve readable contrast on dark terminal backgrounds.

## Out of Scope

- No user-configurable themes.
- No light/dark theme detection.
- No color changes in CLI output.
- No semantic ticket priority colors yet.

## Tests

- Selected backlog/ready/doing/done columns use different border/header colors.
- Non-selected columns remain muted.
- Selected row remains visibly selected.

## Acceptance Checks

- Moving between columns with `h/l` changes selected column color.
- Each selected state has a recognizable color.
- `go test ./...` passes.
