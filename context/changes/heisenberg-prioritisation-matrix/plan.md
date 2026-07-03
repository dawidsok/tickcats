# Heisenberg Prioritisation Matrix Implementation Plan

## Overview

Add a default-on, config-toggleable urgency/importance matrix for TickCats ticket ordering. Tickets can be marked important from the board and detail views; urgency comes from deadlines within 7 days, with overdue tickets most urgent.

## Current State Analysis

TickCats already has ticket priority, optional deadlines, config persistence, sort persistence, and TUI hotkeys. The missing pieces are a stored `important` flag, a matrix-aware priority sort, a forced deadline sort mode, and direct keyboard toggles for importance.

## Desired End State

When matrix prioritisation is enabled, the existing `priority` sort orders tickets by matrix bucket: urgent+important, important, urgent, then neither. Within buckets, tickets sort by deadline, then P0-P3 priority, then created time, then filename. The config view can disable/re-enable this behavior, and the `s` sort cycle includes a deadline-only sort mode for users who want to force pure deadline ordering.

### Key Discoveries:

- Ticket parsing already accepts optional `deadline` frontmatter and maps it into `ticket.Ticket` at `internal/ticket/markdown.go:19` and `internal/ticket/markdown.go:119`.
- Config defaults are derived from an empty `Config{}` at `internal/store/config.go:21`; a default-on toggle should use an inverted persisted field so existing boards enable the matrix without migration.
- Sort modes are centralized in `internal/store/sort.go:10`, while actual TUI board sorting happens in `internal/tui/actions.go:204`.
- Board and detail hotkeys are routed in `internal/tui/update.go:84` and `internal/tui/update.go:254`; `i` is currently unused in both.
- Detail metadata already shows priority and deadline at `internal/tui/render_detail.go:67`; board rendering already shows priority and deadline at `internal/tui/render_board.go:155` and `internal/tui/render_board.go:172`.

## What We're NOT Doing

- We are not changing the pick-next rule. It must remain highest P-priority ready ticket with required acceptance criteria, excluding `[blocked]` and `[to refine]`.
- We are not adding a configurable urgency window; urgency is fixed at due within 7 days for v1 simplicity.
- We are not replacing P0-P3 priorities or changing ticket creation defaults.
- We are not adding collaboration, metrics, AI, GitHub sync, or cross-project behavior.
- We are not building a full metadata editor; this plan only adds direct importance toggles.

## Implementation Approach

Keep the change inside existing packages. Add optional `important: true` frontmatter support to `internal/ticket`, a small store-level rewrite function for toggling importance, a default-on config toggle, one new `deadline` sort mode, and TUI rendering/hotkeys. Use pure helper functions for matrix rank and deadline comparison so the sort logic is easy to test without running the TUI.

## Critical Implementation Details

### State sequencing

The config setting is default-on. Do not add `MatrixPrioritisation bool `json:"...,omitempty"`` because an omitted bool would default to false. Use an inverted persisted flag such as `disable_matrix_prioritisation,omitempty` while rendering the UI as “Matrix priority: on/off”.

### User experience spec

The board/detail hotkey should be `i` because it is unused in both modes. Toggling importance should rewrite the selected ticket, reload the board, preserve focus on the same ticket, and show a short notification.

## Phase 1: Metadata + Config Contracts

### Overview

Add the persistent data shape: optional ticket importance and a default-on config toggle.

### Changes Required:

#### 1. Ticket model and parser

**File**: `internal/ticket/markdown.go`

**Intent**: Add an `Important bool` field to parsed tickets and support optional `important: true` frontmatter.

**Contract**: `important` is optional; missing or false means not important. Parser accepts common bool forms that match Go's `strconv.ParseBool`; invalid values return a parse error. New ticket templates should omit `important` by default.

#### 2. Ticket markdown tests

**File**: `internal/ticket/markdown_test.go`

**Intent**: Lock parsing behavior for important tickets and malformed values.

**Contract**: Tests cover absent important, `important: true`, `important: false`, and invalid `important: maybe`.

#### 3. Store-level importance toggle

**File**: `internal/store/ticket_io.go` or a new `internal/store/important.go`

**Intent**: Provide one narrow function that toggles/persists importance for an existing ticket.

**Contract**: Add a store function such as `SetImportant(boardRoot string, name string, state State, important bool) error` or `ToggleImportant(...) (bool, error)`. It validates the filename, reads the ticket, updates only frontmatter, preserves the markdown body, updates `updated`, and omits `important` when false.

#### 4. Store importance tests

**File**: `internal/store/*_test.go`

**Intent**: Verify that toggling importance is safe and preserves existing ticket content.

**Contract**: Tests cover setting true, setting false/removing the field, preserving body and existing deadline, updating `updated`, and invalid ticket names.

#### 5. Config field and defaults

**File**: `internal/store/config.go`

**Intent**: Persist the matrix toggle while keeping matrix prioritisation enabled by default.

**Contract**: Add an inverted config field, e.g. `DisableMatrixPrioritisation bool `json:"disable_matrix_prioritisation,omitempty"``, plus a helper like `MatrixPrioritisationEnabled() bool` returning `!DisableMatrixPrioritisation`.

#### 6. Config tests

**File**: `internal/store/config_test.go` or existing config tests

**Intent**: Prove default-on behavior and persistence.

**Contract**: Tests cover empty config returning enabled, persisted disabled returning disabled, and re-enabled config omitting or falseing the disabled flag.

### Success Criteria:

#### Automated Verification:

- `go test ./internal/ticket ./internal/store` passes.
- `go test ./...` passes after the phase.

#### Manual Verification:

- A hand-edited ticket with `important: true` loads without warnings.
- A new ticket still omits `important` by default.

**Implementation Note**: After completing this phase and all automated verification passes, pause here for manual confirmation from the human that the manual testing was successful before proceeding to the next phase.

---

## Phase 2: Matrix and Deadline Sorting

### Overview

Make existing priority sorting matrix-aware when enabled and add forced deadline sorting.

### Changes Required:

#### 1. Sort mode contract

**File**: `internal/store/sort.go`

**Intent**: Add a deadline sort mode to the persisted sort cycle.

**Contract**: Add `SortDeadline SortMode = "deadline"` and include it in `SortModes`. Existing `sort.json` files with `priority`, `title`, `date`, or `manual` remain valid.

#### 2. Sort helpers

**File**: `internal/tui/actions.go` or a small new `internal/tui/sort.go`

**Intent**: Keep matrix and deadline ranking isolated from the TUI switch statement.

**Contract**: Add helper functions for: urgency (`deadline != nil && daysUntil(deadline, now) <= 7`), matrix bucket ranking, deadline comparison with nil deadlines last, and the combined tie-break order: deadline, priority rank, created, filename.

#### 3. Apply sort behavior

**File**: `internal/tui/actions.go`

**Intent**: Wire the new sort rules into board sorting.

**Contract**: In `SortPriority`, use matrix ordering when `m.Config.MatrixPrioritisationEnabled()` is true; otherwise keep current P0-P3 behavior. In `SortDeadline`, sort by deadline only, with overdue/earlier deadlines first, missing deadlines last, then priority, created, filename.

#### 4. Sorting tests

**File**: `internal/tui/model_test.go` or a focused `internal/tui/sort_test.go`

**Intent**: Lock the business ordering independent of rendering.

**Contract**: Tests cover bucket order urgent+important > important > urgent > neither, overdue before future deadlines, deadline ties by priority then created, disabled config falling back to old priority sort, and explicit deadline sort mode.

### Success Criteria:

#### Automated Verification:

- Matrix sort unit tests pass.
- Existing manual/title/date/priority sort tests still pass.
- `go test ./internal/tui` passes.
- `go test ./...` passes after the phase.

#### Manual Verification:

- With matrix enabled, `s` on priority sort shows urgent+important tickets above important-only tickets.
- With matrix disabled in config, priority sort returns to P0-P3 ordering.
- Cycling sort reaches `deadline`, and deadline sort places overdue tickets before future and no-deadline tickets.

**Implementation Note**: After completing this phase and all automated verification passes, pause here for manual confirmation from the human that the manual testing was successful before proceeding to the next phase.

---

## Phase 3: TUI Controls, Rendering, and Docs

### Overview

Expose the feature through keyboard-first interactions and document the behavior.

### Changes Required:

#### 1. Board and detail hotkeys

**File**: `internal/tui/update.go`

**Intent**: Let users toggle importance from the board and detail views.

**Contract**: Add `i` in `updateBoard` and `updateDetail`. The command toggles the selected/detail ticket, reloads the board, preserves focus/detail context, and notifies `Marked important` or `Marked not important`.

#### 2. TUI toggle action

**File**: `internal/tui/actions.go`

**Intent**: Centralize the toggle flow so board and detail use the same behavior.

**Contract**: Add a method such as `toggleImportant()` that resolves the selected/detail ticket, calls the store toggle function, reloads/sorts, and preserves cursor by ticket filename.

#### 3. Config view row

**File**: `internal/tui/config_view.go`

**Intent**: Add a config row for matrix prioritisation.

**Contract**: Increase config field count and render a `Matrix` row with `[on] [off]` or equivalent. `h/l` toggles it; Enter saves it. Existing editor/theme/columns flows keep working.

#### 4. Board/detail rendering

**File**: `internal/tui/render_board.go`

**Intent**: Make important tickets visible without crowding the board.

**Contract**: Add a minimal marker such as `!` or `★` beside the priority in ticket rows. Keep wrapping and selection prefixes intact.

**File**: `internal/tui/render_detail.go`

**Intent**: Show importance in metadata.

**Contract**: Add `Important: yes/no` near Priority/Deadline.

#### 5. Help/footer/docs

**Files**: `internal/tui/layout.go`, `internal/tui/help_dialog.go`, `README.md`, `docs/flows/configuration.md`, relevant user-flow docs

**Intent**: Update visible keyboard hints and documentation.

**Contract**: Mention `i` toggle, matrix config, and deadline sort. Keep docs factual; do not expand into non-v1 features.

#### 6. TUI and docs tests

**File**: `internal/tui/model_test.go`

**Intent**: Verify user-facing behavior.

**Contract**: Tests cover board `i` toggles frontmatter and re-sorts, detail `i` toggles the open ticket, config row persists disabled/enabled, rendering shows importance, help/footer mention the hotkey, and deadline sort appears in cycle/status.

### Success Criteria:

#### Automated Verification:

- TUI toggle/config/render tests pass.
- `go test ./...` passes.
- `go vet ./...` passes.

#### Manual Verification:

- In board view, pressing `i` marks/unmarks the focused ticket and updates ordering.
- In detail view, pressing `i` marks/unmarks the open ticket and metadata updates.
- Config view can turn matrix prioritisation off and on.
- Deadline sort can be selected with `s` and visibly sorts overdue/future/no-deadline tickets.

**Implementation Note**: After completing this phase and all automated verification passes, pause here for manual confirmation from the human that the manual testing was successful before proceeding to the next phase.

---

## Testing Strategy

### Unit Tests:

- Ticket parser handles optional `important` frontmatter.
- Store toggle rewrites frontmatter safely and preserves body/deadline.
- Config default-on helper returns enabled for empty config.
- Matrix bucket ranking and deadline comparison are deterministic with an injected `now`.

### Integration Tests:

- TUI board `i` writes `important: true`, reloads, and preserves focus.
- TUI detail `i` toggles the open ticket and keeps detail view usable.
- Config view persists matrix disabled/enabled and sorting honors it.
- Sort cycle includes deadline and persists it in `sort.json`.

### Manual Testing Steps:

1. Create four ready tickets: urgent+important, important only, urgent only, and neither; verify priority sort order.
2. Disable matrix in config; verify P0-P3 priority order returns.
3. Re-enable matrix; verify the same board reorders without editing tickets.
4. Add overdue, due-soon, future, and no-deadline tickets; verify deadline sort.
5. Toggle importance from board and detail views; inspect markdown frontmatter.

## Performance Considerations

Board data volume is local and small. Matrix sorting remains `O(n log n)` per column, same shape as current sorting. Do not add caching or background indexing.

## Migration Notes

Existing tickets need no migration: missing `important` means false. Existing configs need no migration because the matrix toggle is default-on via an inverted disabled flag. Existing `sort.json` files keep their mode; when mode is `priority`, the new matrix behavior applies unless config disables it.

## References

- Product guardrail: `context/foundation/prd.md`
- Ticket frontmatter/parser: `internal/ticket/markdown.go:19`
- Config persistence: `internal/store/config.go:21`
- Sort mode persistence: `internal/store/sort.go:10`
- Board sorting implementation: `internal/tui/actions.go:204`
- Board/detail key routing: `internal/tui/update.go:84`, `internal/tui/update.go:254`
- Detail metadata rendering: `internal/tui/render_detail.go:67`
- Board ticket rendering/deadline urgency helpers: `internal/tui/render_board.go:155`, `internal/tui/render_board.go:260`

## Progress

> Convention: `- [ ]` pending, `- [x]` done. Append ` — <commit sha>` when a step lands. Do not rename step titles.

### Phase 1: Metadata + Config Contracts

#### Automated

- [x] 1.1 `go test ./internal/ticket ./internal/store` passes.
- [x] 1.2 `go test ./...` passes after the phase.

#### Manual

- [x] 1.3 A hand-edited ticket with `important: true` loads without warnings.
- [x] 1.4 A new ticket still omits `important` by default.

### Phase 2: Matrix and Deadline Sorting

#### Automated

- [x] 2.1 Matrix sort unit tests pass.
- [x] 2.2 Existing manual/title/date/priority sort tests still pass.
- [x] 2.3 `go test ./internal/tui` passes.
- [x] 2.4 `go test ./...` passes after the phase.

#### Manual

- [ ] 2.5 With matrix enabled, `s` on priority sort shows urgent+important tickets above important-only tickets.
- [ ] 2.6 With matrix disabled in config, priority sort returns to P0-P3 ordering.
- [ ] 2.7 Cycling sort reaches `deadline`, and deadline sort places overdue tickets before future and no-deadline tickets.

### Phase 3: TUI Controls, Rendering, and Docs

#### Automated

- [ ] 3.1 TUI toggle/config/render tests pass.
- [ ] 3.2 `go test ./...` passes.
- [ ] 3.3 `go vet ./...` passes.

#### Manual

- [ ] 3.4 In board view, pressing `i` marks/unmarks the focused ticket and updates ordering.
- [ ] 3.5 In detail view, pressing `i` marks/unmarks the open ticket and metadata updates.
- [ ] 3.6 Config view can turn matrix prioritisation off and on.
- [ ] 3.7 Deadline sort can be selected with `s` and visibly sorts overdue/future/no-deadline tickets.
