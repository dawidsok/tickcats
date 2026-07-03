# Heisenberg Prioritisation Matrix — Plan Brief

> Full plan: `context/changes/heisenberg-prioritisation-matrix/plan.md`

## What & Why

Add a default-on urgency/importance matrix for ticket ordering. Important is user-marked; urgent means overdue or due within 7 days; urgent+important tickets rise to the top so the board better reflects what should be worked on next.

## Starting Point

TickCats already supports P0-P3 priority, optional deadlines, persisted config, persisted sort mode, board/detail views, and keyboard-first actions. It does not yet store importance or combine importance with deadline urgency in sorting.

## Desired End State

Users can press `i` on the board or detail view to mark/unmark a ticket as important. Priority sort becomes matrix-aware by default, config can disable it, and the sort cycle gains a deadline sort for forcing pure deadline ordering.

## Key Decisions Made

| Decision | Choice | Why |
| --- | --- | --- |
| Complexity | Medium | The feature crosses storage, sort rules, config UI, and TUI hotkeys but stays local. |
| Default | On by default | User explicitly chose matrix behavior as the default. |
| Storage | Optional `important: true` frontmatter | Explicit, parseable, and easy to toggle without changing titles. |
| Edit view | Detail view hotkey | Fast keyboard flow without requiring the external editor. |
| Urgency | Due within 7 days | Captures “closer to deadline” while avoiding another config knob. |
| Sort integration | Priority sort becomes matrix-aware | Minimal UI change and makes the config toggle meaningful. |
| Tie-breaks | Deadline, priority, created | Keeps deadline intent while preserving existing P0-P3 signal. |

## Scope

**In scope:**

- Parse and persist optional ticket importance.
- Add default-on config toggle for matrix prioritisation.
- Make priority sort matrix-aware when enabled.
- Add deadline sort mode.
- Add board/detail `i` hotkey and visible importance metadata.
- Update tests and docs.

**Out of scope:**

- Changing pick-next behavior.
- Configurable urgency thresholds.
- Replacing P0-P3 priority.
- Any sync, collaboration, metrics, AI, or cross-project feature.

## Architecture / Approach

Keep the feature in existing packages. `internal/ticket` parses `important`, `internal/store` persists config and rewrites frontmatter for toggles, and `internal/tui` owns sorting, hotkeys, rendering, and config view wiring. Existing `priority` sort mode is reused; only a new `deadline` mode is added.

## Phases at a Glance

| Phase | What it delivers | Key risk |
| --- | --- | --- |
| 1. Metadata + Config Contracts | Ticket `important` parsing/toggling and default-on config | Preserving frontmatter/body safely when rewriting tickets |
| 2. Matrix and Deadline Sorting | Matrix-aware priority sort plus deadline sort | Getting bucket/tie-break ordering exactly right |
| 3. TUI Controls, Rendering, and Docs | `i` hotkeys, config row, visible metadata, docs | Preserving focus and avoiding config-view regressions |

**Prerequisites:** Existing TUI tests should be passing before phase 1 starts.  
**Estimated effort:** ~2-3 focused sessions across 3 phases.

## Open Risks & Assumptions

- “Heisenberg prioritisation matrix” is treated as an Eisenhower-style urgent/important matrix.
- Default-on config requires an inverted persisted flag; using a plain bool would accidentally default to off.
- Urgency uses calendar dates, not exact times, matching existing deadline handling.

## Success Criteria (Summary)

- Users can toggle importance from board and detail views and see it persisted in markdown.
- Matrix-enabled priority sort orders urgent+important, important, urgent, then neither.
- Config can disable matrix behavior, and deadline sort can be forced via the sort cycle.
