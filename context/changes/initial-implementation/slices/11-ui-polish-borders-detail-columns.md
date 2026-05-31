# Slice 11 — UI Polish: Borders and Detail Columns

## Goal

Improve TUI readability with bordered panels and a two-column detail view.
inspiration is lazygit styling

## Scope

- Add borders around board columns and next picked ticket
- Preserve terminal-size-aware layout.
- Detail view becomes two columns:
  - left column: ticket markdown content, about 2/3 width
  - right column: metadata, about 1/3 width
- Metadata column includes: title, state, priority, filename, labels, created, updated.
- Detail content column keeps `j/k` scrolling.
- Footer remains mode-specific.
- Footer should also be separated by line

## Out of Scope

- No inline editing.
- No theme customization
- No responsive breakpoint beyond sane minimum widths.

## Tests

- Board view renders border characters.
- Detail view includes content and metadata.
- Detail width split uses roughly 2/3 + 1/3 of terminal width.
- Detail scroll still clamps.

## Acceptance Checks

- TUI board is visibly columned with borders.
- Detail view has content left and metadata right.
- `go test ./...` passes.
