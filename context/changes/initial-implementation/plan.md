# Initial Implementation Plan

## Strategy

Build TickCats from the core engine outward. The first milestone should prove the product rule without TUI complexity: local markdown tickets can be created, loaded, moved through folder states, labeled through the title line, and ranked by pick-next.

## Pivot: Simplified Ticket Model

Use the ticket title as the primary lightweight metadata surface.

- No `type` frontmatter.
- Ticket kind is inferred from the title prefix:
  - `Feat:` → feature
  - `Bug:` → bug
  - `Task:` → task
  - missing kind prefix → task by default
- Status/filter labels are written before the title prefix:
  - `[blocked] Feat: add import validation`
  - `[to refine] Task: clean up parser errors`
  - `[blocked] [to refine] Bug: crash on empty backlog`
- Labels are free-form enough to allow examples like `[idea] [to refine] Feat: feature description`.
- v1 only gives special behavior to `[blocked]` and `[to refine]`.
- Future improvement: serialize/discover used labels to support filtering; not needed in v1.
- Future improvement: TUI-assisted label toggling, e.g. quickly add/remove `[blocked]`, `[to refine]`, or other discovered labels without manually editing the title text.
- `created` and `updated` remain frontmatter fields. `updated` is only for content/metadata updates; folder moves do not change it. Since in-app markdown editing is out of scope, users may maintain `updated` manually for now.

## Slice 1 — Project Skeleton and Core Types

**Goal:** Establish a conventional Go layout and the domain types every later slice uses.

Tasks:
- Create `cmd/tickcats/main.go` with a minimal CLI entrypoint.
- Create internal packages, likely:
  - `internal/ticket` for title parsing, priority, labels, readiness, and parsing-facing structs.
  - `internal/store` for `.tickcats/` filesystem operations.
- Define core enums/constants:
  - inferred kinds: `feature`, `task`, `bug`
  - priorities: `P0`, `P1`, `P2`, `P3`
  - states/folders: `backlog`, `ready`, `doing`, `done`
  - special title labels: `blocked`, `to refine`
- Add tests for priority ordering, folder-state parsing, title prefix parsing, and label parsing.

Acceptance checks:
- `go test ./...` passes.
- `go run ./cmd/tickcats` prints a minimal help/version message or placeholder without error.

## Slice 2 — `.tickcats/` Initialization

**Goal:** Create the local private board folders safely.

Tasks:
- Implement `tickcats init` or equivalent function that creates:
  - `.tickcats/backlog/`
  - `.tickcats/ready/`
  - `.tickcats/doing/`
  - `.tickcats/done/`
- Ensure init is idempotent.
- Ensure `.tickcats/` is added to `.gitignore`; create `.gitignore` if absent.
- Add filesystem tests using temp directories.

Acceptance checks:
- Running init twice does not fail or duplicate content.
- Init never overwrites existing ticket files.
- `.tickcats/` is ignored for v1 Private mode.

## Slice 3 — Ticket Creation and Markdown Parsing

**Goal:** Support the simplified v1 ticket schema.

Tasks:
- Implement ticket creation with title prefixes rather than type metadata:
  - `Feat:` for features
  - `Task:` for tasks
  - `Bug:` for bugs
- Frontmatter fields:
  - required: `title`, `priority`, `created`, `updated`
  - no `type`
  - no `blocked_by`
- Body sections can start with a simple generic shape:
  - `## Context`
  - `## Acceptance Criteria`
- Parse frontmatter and markdown body enough to inspect title, priority, labels, and Acceptance Criteria.
- Add tests for conventional title parsing, missing prefix defaulting to task, labels before title prefix, and malformed files.

Acceptance checks:
- Generated tickets parse back into valid ticket structs.
- Missing `Feat:`/`Task:`/`Bug:` prefix defaults kind to task.
- `[blocked]` marks a ticket blocked.
- `[to refine]` marks a ticket not ready.
- Empty or missing Acceptance Criteria makes a ticket not ready.

## Slice 4 — Load Board and Move Tickets

**Goal:** Treat the filesystem as the board.

Tasks:
- Load tickets from `.tickcats/{backlog,ready,doing,done}/`.
- Derive ticket state from parent folder, not frontmatter.
- Implement move operation between valid states using file moves.
- Preserve filename and markdown content during moves.
- Do not update `updated` when moving tickets between folders.

Acceptance checks:
- Board loading returns tickets grouped by folder state.
- Moving from `ready` to `doing` changes folder state and preserves content.
- Invalid state folders are ignored or surfaced as warnings, not treated as valid board columns.

## Slice 5 — Pick-Next Rule

**Goal:** Implement the core business rule.

Rule:
- Eligible tickets live in `.tickcats/ready/`.
- `title` is non-empty.
- title labels do not include `[blocked]`.
- title labels do not include `[to refine]`.
- `## Acceptance Criteria` is non-empty.
- Sort by priority: `P0` > `P1` > `P2` > `P3`.
- Tie-break by oldest `created` timestamp.
- If still tied, return candidates deterministically for manual choice later.

Tasks:
- Implement readiness checks.
- Implement ranking/tie detection.
- Add table-driven tests for priority, blocked labels, to-refine labels, missing acceptance criteria, missing kind prefix, and timestamp ties.

Acceptance checks:
- Pick-next returns the expected ticket across mixed folders and priorities.
- `[blocked]` ready tickets are excluded.
- `[to refine]` ready tickets are excluded.
- Backlog tickets are excluded even if their content is complete.

## Slice 6 — Minimal CLI Dogfood Loop

**Goal:** Expose the engine before building the full TUI.

Tasks:
- Add minimal commands for engine dogfooding:
  - `tickcats init`
  - `tickcats new feat|task|bug`
  - `tickcats list`
  - `tickcats move <ticket> <state>`
  - `tickcats pick-next`
- Keep UI plain text for now; full TUI comes after the engine is stable.
- Document temporary CLI commands in `README.md` once created.

Acceptance checks:
- From a temp repo, user can initialize, create a ticket, move it to ready, and run pick-next.
- Commands work offline.
- `go test ./...` passes.

## Deferred Until After Core Engine

- Bubble Tea board UI.
- Full-screen ticket detail view.
- Command palette.
- Basic search/filter.
- Label serialization/filtering from discovered title labels.
- TUI-assisted label toggling for `[blocked]`, `[to refine]`, and other discovered labels.
- Release packaging via GitHub Releases/Homebrew/npm.

## Open Decisions Before Coding

1. What filename format should generated tickets use? Proposed: `YYYYMMDD-HHMM-<slug>.md`, e.g. `20260530-1030-add-import-validation.md`. This is readable, sorts by creation time, and avoids needing a separate counter file.
2. How should malformed tickets be handled? Proposed: board loading should skip malformed tickets but report warnings with file paths; commands that target a malformed ticket should fail with a clear parse error. This keeps one bad file from breaking the whole board while preserving visibility.
3. Should `## Context` + `## Acceptance Criteria` be the only generated body sections for all ticket kinds in v1, or should `feat`/`bug` still get extra optional headings while sharing the same parser?
