# Implementation Plan: TC-AYQZGD custom column colors

Ticket: `.tickcats/doing/tc-ayqzgd-introduce-custom-colors-for-configurable-columns.md`

## Goal

Allow users to assign a custom predefined color to each configured board column. The color must be stored in `.tickcats/config.json`, selectable from a visual swatch picker in the TUI config page, and applied to the column header and ticket cells. Existing theme-based colors remain the fallback when no custom column color is set.

## Constraints

- Keep status folder-derived; do not add ticket frontmatter state.
- Keep the feature local-only; no sync, auth, collaboration, or AI additions.
- Preserve existing config files: old `columns` entries without `color` must still load.
- Use three-character hex colors only: `#rgb`.
- Valid hex digits for the predefined palette: `0`, `4`, `8`, `a`, `d`, `f`.
- The picker palette should be deterministic and generated in code, not hand-maintained.

## Proposed UX

In the TUI config page (`c`):

- `tab` to the **Columns** field.
- Select a column with `j/k`.
- Press `C` to open a color picker for that column.
- Navigate swatches with `h/l/j/k`.
- Press `enter` to save the selected color immediately.
- Press `esc` to cancel without changing color.
- Optional reset shortcut: `x` clears the custom color and returns the column to theme fallback.

The columns table should display a color swatch and code, for example:

```text
Columns #  Name                  Folder ID             Color    Actions
        > 1  Backlog             backlog               ██ #88a  a r K/J d C
```

## Data model and store changes

### 1. Extend `store.Column`

File: `internal/store/config.go`

Add a color field:

```go
type Column struct {
    ID          string `json:"id"`
    DisplayName string `json:"name"`
    Color       string `json:"color,omitempty"`
}
```

Notes:

- Empty `Color` means “use theme fallback”.
- Existing configs remain valid because `omitempty` and zero value are compatible.
- Update `RenameColumn` to preserve the existing color when replacing the column struct.

### 2. Add palette and validation helpers

File: `internal/store/config.go` or new `internal/store/color.go`

Add helpers:

- `ColumnColorPalette() []string`
  - Generate all `6^3 = 216` colors from `[]byte("048adf")`.
  - Return colors in deterministic RGB nested-loop order.
- `NormalizeColumnColor(raw string) (string, error)`
  - Trim whitespace.
  - Empty string is allowed and returns `""` for reset.
  - Lowercase input before validation.
  - Require exactly `#` + 3 valid chars.
- `IsColumnColor(value string) bool`
  - Convenience wrapper used by tests/TUI if needed.

### 3. Add color persistence helper

File: `internal/store/config.go`

Add:

```go
func SetColumnColor(boardRoot string, id string, color string) error
```

Behavior:

- Load config.
- Find configured column by ID.
- Normalize/validate `color`.
- Set `columns[idx].Color`.
- Save config.
- Return useful errors for unknown column or invalid color.

### 4. Store tests

File: `internal/store/column_crud_test.go` or new `internal/store/color_test.go`

Add tests for:

- Palette length is 216.
- Palette contains examples: `#000`, `#f00`, `#0f0`, `#00f`, `#fff`.
- Valid colors normalize correctly, including uppercase to lowercase if accepted.
- Invalid values are rejected: `red`, `#12f`, `#ffff`, `#ggg`, missing `#`.
- `SetColumnColor` persists `color` in `config.json`.
- `SetColumnColor(root, id, "")` clears the color.
- `RenameColumn` preserves a column’s custom color.

## TUI model changes

### 1. Add picker state

File: `internal/tui/model.go`

Extend `configAction`:

```go
configActionPickColor
```

Add model fields:

```go
configColorIdx int
```

Optionally add cached picker width/columns only if rendering needs it; otherwise derive from layout.

### 2. Column color helper

File: `internal/tui/util.go`

Add helpers:

- `func (m Model) columnColor(colIndex int) lipgloss.Color`
  - Look up `m.Config.GetColumns()[colIndex].Color`.
  - If non-empty and valid, return it.
  - Otherwise return `m.themeColor(colIndex)`.
- `func (m Model) columnStyle(colIndex int) lipgloss.Style`
  - Bold foreground using `m.columnColor(colIndex)`.

Keep `themeColor` unchanged because deadline urgency still depends on the selected theme gradient.

## TUI config flow

### 1. Open picker from columns field

File: `internal/tui/config_view.go`

In `updateConfig`, when `configField == 2`:

- Handle `C`.
- Read selected column.
- Build palette via `store.ColumnColorPalette()`.
- If selected column already has a color, initialize `configColorIdx` to that palette index.
- Otherwise initialize to the nearest/simple default, likely index `0` (`#000`).
- Set `configAction = configActionPickColor`.
- Clear `Status`.

### 2. Handle picker input

In `updateConfigAction`, add `configActionPickColor`:

- `esc`: cancel and return to columns table.
- `h/left`: move one swatch left.
- `l/right`: move one swatch right.
- `j/down`: move one picker row down.
- `k/up`: move one picker row up.
- `enter`: call `store.SetColumnColor(m.Root, selectedColumn.ID, selectedColor)`.
  - On success: cancel action, reload config/board via `syncConfigAndOrder`, notify “Column color updated”.
  - On error: put error in `Status`.
- Optional `x`: call `store.SetColumnColor(..., "")`, cancel action, notify “Column color cleared”.

### 3. Render picker

Add `renderConfigColorPicker(width int) string`.

Rendering guidance:

- Use visible square swatches, e.g. `"██"` or two spaces with background color.
- Use `lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("██")` or background color for filled blocks.
- Mark the focused swatch with brackets or inverse style, e.g. `[██]`.
- Show selected color code and target column name above the grid.
- Keep the grid compact; derive columns from available width:
  - swatch cell width around 5–7 chars including spaces/brackets.
  - clamp to at least 6 columns.
- With 216 colors, expect multiple rows; this is acceptable in the config dialog.

### 4. Update table and help text

File: `internal/tui/config_view.go`

- Change table header from `Actions` only to include `Color`.
- Render each column’s custom color if set; otherwise render theme fallback with a marker like `theme`.
- Include `C color` in the config help line.
- Include picker-specific help while picker is open: `h/l/j/k move  enter set  x clear  esc cancel`.

## Board rendering changes

File: `internal/tui/render_board.go`

### 1. Header color

Update `renderColumn`:

- Use `m.columnColor(index)` for column header border and header text.
- Keep selected-column emphasis by making the selected header bold or using the existing selected style behavior.
- Non-selected columns with custom colors should still show their configured color in the header.

### 2. Column cells

Update ticket row styling in `styledTicketColumnLines`:

- Use the configured column color for ticket lines in that column.
- Focused row should be bold in the column color.
- Multi-selected row should remain visually distinct; if needed, combine bold/underline with the column color instead of replacing it with global pink.
- Deadline date color should remain controlled by urgency/theme unless a separate design is chosen later.

### 3. Detail view metadata

File: `internal/tui/render_detail.go`

Where state/column labels use `m.colStyle(...)`, switch to the new column-aware style so detail metadata reflects custom colors too.

## Documentation updates

### README

File: `README.md`

- Update the custom columns JSON example:

```json
{ "id": "ready", "name": "Ready", "color": "#0af" }
```

Use only allowed digits in examples, e.g. `#0af`, `#f80`, `#8ad`.

- Update the Configuration table:
  - Columns: add/rename/reorder/delete and assign custom colors.
  - Theme: theme remains fallback for columns without custom colors.

### Flow docs

File: `docs/flows/configuration.md`

- Add `C` color picker transitions to the flowchart.
- Add `configActionPickColor` to the architecture section.
- Mention `store.SetColumnColor` and palette validation.

## Test plan

Run after implementation:

```bash
gofmt -w internal/store internal/tui
go test ./...
go vet ./...
```

Focused tests to add/update:

- `internal/store`: color validation, palette generation, persistence, rename preservation.
- `internal/tui/color_test.go`: custom color helper falls back to theme and overrides when configured.
- `internal/tui/model_test.go`: config columns table renders color column; `C` picker opens; navigation changes selected swatch; `enter` persists; `esc` cancels.
- Board rendering test if existing assertions cover ANSI/color output.

## Implementation order

1. Store model + validation + `SetColumnColor`.
2. Store tests and green `go test ./internal/store`.
3. TUI color helper and board rendering fallback/override logic.
4. Config picker state, update handling, and rendering.
5. TUI tests and green `go test ./internal/tui`.
6. README and flow docs.
7. Full verification: `gofmt`, `go test ./...`, `go vet ./...`.

## Risks and mitigations

- **ANSI-heavy render tests may be brittle**: prefer helper-level tests for color choice and minimal string containment tests for config labels.
- **216 swatches may overflow small terminals**: compute picker columns from available width and let the dialog height constrain output if needed.
- **Renaming could drop color**: explicitly preserve `Column.Color` in `RenameColumn` and add a regression test.
- **Theme and custom color interactions could confuse users**: label table values as custom color vs theme fallback.
