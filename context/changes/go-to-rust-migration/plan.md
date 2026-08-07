# Go to Rust Migration Implementation Plan

## Overview

Replace the Go implementation of TickCats with a long-term Rust implementation without changing the repo-local product model or losing existing board data. This is a feature-selection migration, not an automatic full rewrite: Phase 0 reviews the current feature inventory below and decides what is worth carrying forward before Rust implementation begins.

The Go binary remains the executable reference until every retained contract, TUI workflow, integration, and release artifact has passed its acceptance check. Go is deleted only after release proof.

## Current State Analysis

TickCats currently has 6,191 lines of non-test Go and 5,878 lines of Go tests. The code is split into `cmd/tickcats`, `internal/ticket`, `internal/store`, and `internal/tui`; `go test ./...` passes on the planning branch.

The implemented product is broader than `context/foundation/prd.md`:

- The PRD names four workflow columns, while the app initializes five (including Won't Do) and supports arbitrary configured columns (`internal/store/config.go:29-41`).
- The app includes IDs, deadlines, importance/matrix prioritisation, five sort modes, fuzzy search, multi-select, themes, file watching, and column administration.
- The PRD requires a command palette and in-TUI metadata editing, but neither is implemented (`context/foundation/prd.md:91-106,139-149`; `internal/tui/model.go:49-74`).
- Column-color storage exists, but the color-picker UI is only an untracked/in-flight plan (`internal/store/color.go`; root `plan.md`).
- The Agent Skills installer exists only in the current working tree (`scripts/install-skills.sh` and the unstaged README change); the shipped skills themselves are committed.

### Key discoveries

- Folder location is status; frontmatter must not gain workflow state (`internal/store/board.go:16-115`).
- The parser is a small line-oriented dialect, not full YAML (`internal/ticket/markdown.go:145-230`).
- Pick-next is the load-bearing contract: Ready only, meaningful Acceptance Criteria, no `[blocked]` or `[to refine]`, then priority, created time, and filename (`internal/store/pick.go:35-89`).
- Nominal reads can rewrite `config.json` during folder reconciliation (`internal/store/config.go:55-79,268-327`).
- Script consumers include shell completions and bundled Agent Skills, so exit codes, stdout/stderr, and printed paths matter (`completions/`; `skills/`).
- Current release behavior is five target archives, checksums, GitHub Releases, and Homebrew (`.goreleaser.yml`; `context/changes/release-verification/verification.md`).

## Desired End State

The repository ships one Rust `tickcats` binary as the long-term implementation. Existing `.tickcats/` boards remain readable and writable without a data migration. Every retained feature has an approved matrix decision and an observable acceptance check; dropped features are removed deliberately rather than omitted accidentally.

The cutover is complete when:

- No feature or defect row remains `Review`.
- Every known persisted field is preserved, including fields whose UI was dropped.
- Retained CLI contracts match approved stdout, stderr, exit-code, argument, and path behavior.
- Retained TUI workflows preserve keys, state transitions, data effects, and width safety; exact glyph parity is not required.
- GitHub archives, checksums, shell completions, Homebrew, and the current five target platforms are proven with Rust artifacts.
- Go source and release tooling are removed only after all previous checks pass.

## Scope Rules

### In scope

- Feature inventory and retain/drop/defer decisions.
- Known-defect decisions before freezing compatibility behavior.
- Parallel Rust implementation of approved ticket, store, CLI, and TUI behavior.
- Semantic compatibility for all known ticket, config, sort, and folder data.
- Contract fixtures comparing Go and Rust where current Go behavior is approved.
- Existing shell completions, Agent Skills integration, GitHub release archives, checksums, Homebrew, and supported platform targets.
- Final Go removal after release proof.

### Out of scope

- New sync, collaboration, metrics, auth, AI, cross-project dashboards, or hosted services.
- A new `.tickcats/` format or workflow state in frontmatter.
- Pixel-perfect Bubble Tea rendering in Ratatui.
- New package channels such as crates.io/cargo-install or Scoop during the migration.
- Porting every Go test one-for-one or mirroring Go package boundaries when a smaller Rust module is sufficient.
- Public library/plugin architecture.

## Phase 0 Decision Vocabulary

### Baseline status

- **Shipped** — committed on `main` and reachable by users.
- **Undocumented** — committed behavior not accurately described in public docs.
- **PRD-only** — required by the PRD but not implemented.
- **In flight** — present only in the planning working tree or an uncommitted plan.
- **Repo tooling** — integration around the binary rather than compiled product behavior.

### Allowed feature decisions

- **Retain** — port and verify in Rust.
- **Replace** — intentionally substitute a simpler approved behavior and verify the new contract.
- **Preserve-data-only** — keep persisted values round-trippable but omit the user interaction.
- **Drop** — intentionally remove; update docs/integrations.
- **Defer** — exclude from initial Rust cutover and record a follow-up.
- **Review** — human decision still required; Phase 1 cannot start.

`Retain (contract)` and `Preserve (decided)` rows were settled by the PRD/repository rules or this planning interview. Changing them requires an explicit product-contract change, not a silent matrix edit.

## Current Feature Inventory and Decision Matrix

### Core product and board behavior

| ID | Current feature and evidence | Baseline | Data impact | Decision / owner | Rust acceptance check |
|---|---|---|---|---|---|
| CORE-01 | Repo-local, offline, single-user board; no auth/server (`context/foundation/prd.md`) | Shipped | Entire `.tickcats/` tree | Retain (contract) | Core workflows run with network disabled and no account/config outside the repo. |
| CORE-02 | Status comes from configured column folders, never ticket frontmatter (`internal/store/board.go`) | Shipped | Folder paths | Retain (contract) | Moving a file changes status; a frontmatter `status` key has no effect. |
| CORE-03 | Exact pick-next eligibility and ranking (`internal/store/pick.go`) | Shipped | Reads Ready tickets | Retain (contract) | Shared fixtures cover eligibility, priority, oldest-created ordering, filename order, and exact ties. |
| CORE-04 | Plain Markdown tickets editable outside TickCats (`internal/ticket/markdown.go`) | Shipped | Ticket bytes | Retain (contract) | Existing real tickets load before and after external edits. |
| CORE-05 | Five default columns: Backlog, Ready, Doing, Done, Won't Do (`internal/store/config.go:29-41`) | Shipped; exceeds PRD | Folders/config | Replace — fixed Backlog/Ready/WIP/Done; WIP displays from `doing/` and has no enforced capacity (user, 2026-08-06) | Fresh init creates only `backlog/`, `ready/`, `doing/`, and `done/`; UI labels `doing/` as WIP and accepts any ticket count. |
| CORE-06 | Arbitrary configured columns and display-name/slug resolution (`internal/store/config.go`; `internal/store/state.go`) | Shipped | Folders/config | Drop — Rust board supports only Backlog/Ready/WIP/Done; legacy folders remain untouched and emit ticket-count warnings (user, 2026-08-06) | `wont-do/` and custom folders are excluded without file/config mutation and reported with ticket counts. |
| CORE-07 | Config reconciles with folders: remove missing, append discovered, ignore dot-folders (`internal/store/config.go:268-327`) | Shipped; partly undocumented | May rewrite config | Replace — fixed-column loading never reconciles folder state into config (user, 2026-08-06) | Loading a board does not rewrite `config.json`; only four fixed folders are actionable. |
| CORE-08 | Stable `TC-XXXXXX` IDs and missing/invalid/duplicate warnings (`internal/ticket/id.go`; `internal/store/board.go`) | Shipped | `id` | Retain — every new ticket gets a stable ID independent of filename/title (user, 2026-08-06) | New IDs match the approved alphabet/length, remain unchanged across edits/moves, and duplicate/invalid IDs warn. |
| CORE-09 | Optional deadline/SLA metadata and mutation (`internal/store/deadline.go`) | Shipped; post-PRD | `deadline`, `updated` | Replace — preserve deadline as matrix urgency input, but remove direct editing and deadline sort (user, 2026-08-06) | Existing `deadline` values survive rewrites and affect the retained matrix only; no deadline mutation command/dialog exists. |
| CORE-10 | Optional important flag and urgency/importance matrix (`internal/store/important.go`; `internal/tui/sort.go`) | Shipped; post-PRD | `important`, config | Retain — full urgency/importance matrix, including important toggle and config switch (user, 2026-08-06) | Matrix buckets use preserved deadlines plus `important`; enabled/disabled ordering and persistence match approved fixtures. |
| CORE-11 | Priority/title/date/deadline/manual sort and per-column manual order (`internal/store/sort.go`; `internal/tui/actions.go`) | Shipped; post-PRD | `sort.json` | Review — user | Approved modes order fixed fixtures and persist/reconcile manual order. |
| CORE-12 | Soft delete into `.trash` with collision refusal (`internal/store/delete.go`) | Shipped; undocumented | `.trash/` | Review — user | Approved delete/restoration boundary is verified without permanent loss. |
| CORE-13 | Invalid tickets and IDs produce non-fatal board warnings (`internal/store/board.go:62-110`) | Shipped | No rewrite | Review — user | Good tickets load while malformed/duplicate-ID fixtures emit approved warnings. |
| CORE-14 | Column color field, 216-color palette, validation, and setter; no current picker/render use (`internal/store/color.go`; `internal/store/config.go:242-264`) | Undocumented | `columns[].color` | Review — user | Field always survives; palette/setter are ported only if retained beyond data preservation. |

### Scriptable CLI

| ID | Current feature and evidence | Baseline | Data impact | Decision / owner | Rust acceptance check |
|---|---|---|---|---|---|
| CLI-01 | No command opens TUI; explicit `tui` does the same (`cmd/tickcats/main.go:42-62`) | Shipped | TUI may mutate | Review — user | Both approved entry forms launch the same model against `--path`. |
| CLI-02 | Global `--path <dir>` may appear before dispatch and selects board root (`cmd/tickcats/main.go:28-45`) | Shipped | Selects all files | Review — user | Position, missing-value, custom basename, and default cases have process tests. |
| CLI-03 | `init` creates defaults and idempotently adds `<board-basename>/` to adjacent `.gitignore` (`internal/store/init.go`) | Shipped | Folders, `.gitignore` | Review — user | Temp-repo fixture covers first run, repeat run, existing board, and custom path. |
| CLI-04 | `new feat\|feature\|task\|bug\|fix`, P2 default, generated ID, backlog destination (`cmd/tickcats/main.go:79-100`; `internal/store/create.go`) | Shipped; aliases undocumented | New ticket | Review — user | Golden cases cover every approved kind/alias and exact printed path. |
| CLI-05 | `new --ac\|--acceptance` joins remaining text into one Acceptance Criteria bullet (`cmd/tickcats/main.go:234-244`) | Shipped; alias undocumented | Ticket body | Review — user | Golden cases cover missing, blank, and multiword acceptance text. |
| CLI-06 | `list` prints configured columns, filenames, ID/emdash, priority/title, and warnings (`cmd/tickcats/main.go:102-122`) | Shipped | Config may reconcile | Review — user | Exact stdout/stderr/exit and config side effect match approved fixtures. |
| CLI-07 | `move <ticket> <from> <to>` accepts IDs/display names, validates paths, and refuses collisions (`cmd/tickcats/main.go:124-148`) | Shipped | Renames file | Replace — allow only adjacent Backlog↔Ready↔WIP↔Done moves (user, 2026-08-06) | Golden cases accept both directions between adjacent folders and reject skipped-column moves without mutation. |
| CLI-08 | Human `pick-next` output, no-ticket success, and tie candidate list (`cmd/tickcats/main.go:150-182`) | Shipped | Config may reconcile | Retain (contract) | Exact approved stdout/stderr/exit for none, one, warning, and tie. |
| CLI-09 | Intended `pick-next --path` path-only mode (`cmd/tickcats/main.go:206-231`) | Shipped but unreachable via normal dispatch | None | Review via DEF-01 — user | Process-level test freezes the approved non-conflicting contract. |
| CLI-10 | `ids migrate` rewrites IDs and filenames and prints mappings (`cmd/tickcats/main.go:184-197`) | Shipped | Many tickets | Retain — explicit only; scan four fixed columns and warn/count skipped legacy-column tickets (user, 2026-08-06) | Command never runs on load, handles fixed-column collisions deterministically, and leaves unsupported folders untouched. |
| CLI-11 | Hidden `__complete tickets\|columns` emits live completion candidates (`cmd/tickcats/main.go:256-297`) | Shipped | Config may reconcile | Retain (distribution) | Exact candidate output works on copied dynamic-column boards. |
| CLI-12 | Static `help`, aliases `help\|--help\|-h`, unknown-command errors (`cmd/tickcats/main.go:63-68,305-321`) | Shipped | None | Review — user | Approved command vocabulary and exit statuses have process tests. |
| CLI-13 | Returned errors render as `Error: …` on stderr with exit 1; warnings use `Warning: …` (`cmd/tickcats/main.go:21-26,299-303`) | Shipped | None | Retain (script contract) | Process harness captures exact prefixes, streams, and status. |

### TUI workflows

| ID | Current feature and evidence | Baseline | Data impact | Decision / owner | Rust acceptance check |
|---|---|---|---|---|---|
| TUI-01 | Board shows pick-next banner, configured columns, focused ticket, status/warnings, and footer (`internal/tui/render_board.go`) | Shipped | None | Review — user | Retained elements render within approved widths. |
| TUI-02 | Vim/arrow column-row navigation, clamped cursors, `d/u` half pages (`internal/tui/update.go:89-116`; `navigation.go`) | Shipped | None | Review — user | State-transition tests cover boundaries and scroll offsets. |
| TUI-03 | Up to six-digit count prefixes for `h/j/k/l`; leading zero and non-motion consumption rules (`internal/tui/update.go:89-197`) | Shipped; undocumented | None | Review — user | Focused tests freeze approved count semantics. |
| TUI-04 | Horizontal column window, hidden-column indicators, wrapping, narrow-width safety (`internal/tui/layout.go`; `render_board.go`) | Shipped | None | Review — user | Snapshot/property checks assert no overflow at agreed terminal sizes. |
| TUI-05 | Detail view with markdown highlighting, metadata, scrolling, and identity preservation (`internal/tui/render_detail.go`; `movement.go`) | Shipped | None | Review — user | Open/scroll/reload/move/delete scenarios preserve or report identity. |
| TUI-06 | New-ticket form: kind, title, P0-P3, default P2, default `[to refine]` (`internal/tui/create.go`) | Shipped; differs from CLI | Creates ticket | Review — user | Approved defaults, validation, field navigation, and output are tested. |
| TUI-07 | Post-create editor prompt with yes/no/don't-ask and persisted skip preference (`internal/tui/create.go:128-179`) | Shipped | Config | Review — user | Dialog branches and `skip_editor_prompt` persistence pass. |
| TUI-08 | External editor command: config, `$EDITOR`, then `vi`; whitespace-split args; board reload (`internal/tui/editor.go`; `actions.go`) | Shipped | Ticket/config | Review — user | Approved command resolution/arguments and reload behavior pass. |
| TUI-09 | `p` progress and `b` move back one column (`internal/tui/update.go:124-136`) | Shipped; `b` undocumented in README | Renames ticket | Retain — these are the primary adjacent workflow actions (user, 2026-08-06) | `p`/`b` move exactly one column, stop at board edges, and preserve focus. |
| TUI-10 | Move mode `m`, `h/l`, `H/L`, and cancel (`internal/tui/update.go:199-223`) | Shipped | Renames tickets | Replace — retain adjacent `h/l` and cancel; drop direct `H/L` first/last shortcuts (user, 2026-08-06) | Move mode permits one adjacent step per action and rejects/skips no workflow columns. |
| TUI-11 | Cross-column multi-select `v` and ordered bulk moves; delete remains single-ticket (`internal/tui/actions.go:312-465`) | Shipped; partly undocumented | Renames tickets | Review — user | Marker, move ordering, retained selection, and reload cleanup pass. |
| TUI-12 | Manual `j/k` reorder; non-manual modes prompt to switch (`internal/tui/update.go:224-247`; `actions.go`) | Shipped | `sort.json` | Review — user | Prompt branches and persisted order pass. |
| TUI-13 | `x` soft-delete confirmation for focused ticket (`internal/tui/actions.go:22-58`) | Shipped | `.trash/` | Review — user | Confirm/cancel/collision behavior passes. |
| TUI-14 | Deadline dialog: today, tomorrow, +7 days, custom date, clear (`internal/tui/deadline_dialog.go`) | Shipped | Ticket metadata | Drop — deadline is preserved data only (user, 2026-08-06) | Rust TUI has no deadline key, dialog, urgency color, or date-based ordering. |
| TUI-15 | `i` important toggle and matrix-aware priority display/order (`internal/tui/actions.go:77-97`; `sort.go`) | Shipped | Ticket/config | Retain — full matrix behavior remains user-facing (user, 2026-08-06) | Toggle, urgent threshold, buckets, and P-label suppression pass using preserved deadline values. |
| TUI-16 | `s` cycles priority → title → date → deadline → manual (`internal/tui/actions.go:172-191`) | Shipped | `sort.json` | Review — user | Any retained sort set excludes deadline mode; exact approved cycle and tie-break ordering pass. |
| TUI-17 | `/` fuzzy subsequence search over priority/title/body with typing/navigation phases and two-step Esc (`internal/tui/search.go`) | Shipped; partly undocumented | None | Review — user | Query, count, cross-column nav, detail open, clear, and exit pass. |
| TUI-18 | Config editor presets/custom command (`internal/tui/config_view.go`) | Shipped | `config.editor` | Review — user | Approved presets/custom value persist and reload. |
| TUI-19 | Six themes with deterministic dynamic-column gradient (`internal/tui/model.go:79-99`; `color_test.go`) | Shipped | `config.theme` | Review — user | Approved theme identities and deterministic colors pass. |
| TUI-20 | Config matrix on/off (`internal/tui/config_view.go`) | Shipped | `disable_matrix_prioritisation` | Retain — user can enable/disable matrix prioritisation (user, 2026-08-06) | Toggle persists without erasing other config fields and immediately reapplies ordering/display. |
| TUI-21 | Config column add/rename/reorder/delete with Backlog/Done/first-column restrictions (`internal/tui/config_view.go`) | Shipped | Folders/config | Drop — fixed workflow has no column administration (user, 2026-08-06) | Rust TUI has no column CRUD/reorder controls; legacy column config remains preserved on disk. |
| TUI-22 | Column color picker/rendering described in root `plan.md`; storage only is committed | In flight | `columns[].color` | Review — user | If retained, picker/render/reset workflow and data persistence pass; otherwise preserve data only. |
| TUI-23 | Scrollable help overlay and mode-specific key reference (`internal/tui/help_dialog.go`) | Shipped | None | Review — user | Open/scroll/close and approved help contents pass. |
| TUI-24 | Quit confirmation from board/detail/move; Ctrl-C immediate (`internal/tui/update.go:18-70`) | Shipped | None | Review — user | Confirm/cancel restores the prior state correctly. |
| TUI-25 | Manual reload plus debounced external file watcher; focus/detail/selection reconciliation (`internal/tui/watcher.go`; `actions.go`) | Shipped | Reads board | Review — user | Manual reload and approved watcher lifecycle/coalescing scenarios pass. |
| TUI-26 | Three-second generation-safe success/error/info notifications (`internal/tui/model.go:199-224`) | Shipped; undocumented | None | Review — user | Newer notification is not cleared by an older timer. |
| TUI-27 | Command palette with create/move/edit/pick actions (`context/foundation/prd.md:139-149`) | PRD-only | Depends on selected actions | Review — user | If retained, specify keys/actions first; do not invent it under migration parity. |
| TUI-28 | Direct in-TUI ticket metadata editing (`context/foundation/prd.md:99-105,136`) | PRD-only | Ticket metadata | Review — user | If retained, define fields/validation first; external-editor support does not satisfy it. |
| TUI-29 | Direct pick-next hotkey, separate from the always-visible recommendation (`context/foundation/prd.md:164-170`) | PRD-only | None | Review — user | If retained, define whether it focuses, opens, or only announces the recommendation before porting. |

### Distribution, documentation, and repository integrations

| ID | Current feature and evidence | Baseline | Data impact | Decision / owner | Rust acceptance check |
|---|---|---|---|---|---|
| OPS-01 | GitHub release on semantic-version-shaped tags (`.github/workflows/release.yml`) | Shipped | Release metadata | Retain (decided) | Rust release workflow triggers on the same tag class with required permissions/secrets. |
| OPS-02 | Five targets: macOS amd64/arm64, Linux amd64/arm64, Windows amd64 (`.goreleaser.yml`) | Shipped | Artifact names | Retain (decided) | Matrix produces exactly the five approved installable binaries. |
| OPS-03 | tar.gz except Windows zip; `tickcats_<version>_<os>_<arch>`; README/LICENSE/completions; `checksums.txt` (`.goreleaser.yml`) | Shipped | Artifact contract | Retain (decided) | Archive-content and checksum assertions pass. |
| OPS-04 | Homebrew tap formula installs binary/completions and runs init smoke test (`.goreleaser.yml:60-80`) | Shipped | External tap | Retain (decided) | Generated formula/cask equivalent passes the same install test. |
| OPS-05 | Direct GitHub archive download (`README.md`) | Shipped | None | Retain (decided) | Install docs and artifact smoke test cover direct download. |
| OPS-06 | `go install` installation (`README.md`) | Shipped | None | Drop at cutover (decided) | No post-cutover docs claim a Go install path. |
| OPS-07 | Bash, Zsh, Fish static/dynamic completions (`completions/`) | Shipped | Calls CLI helpers | Retain (decided) | Syntax/package checks and live ticket/column candidates pass for all three scripts. |
| OPS-08 | Bundled `tc-*` and roadmap Agent Skills invoke exact CLI/path/folder contracts (`skills/`) | Repo tooling | May mutate boards/gitignore | Review — user | Retained skills pass representative create/list/move/pick workflow against Rust. |
| OPS-09 | Interactive multi-harness skill installer (`scripts/install-skills.sh`; unstaged README change) | In flight | User home dirs | Review — user | If retained, shell check and temp-HOME install test pass; otherwise exclude from migration commit. |
| OPS-10 | Generated changelog groups/filters in GoReleaser (`.goreleaser.yml:40-58`) | Shipped; operational | Release notes | Review — user | If retained, a dry run proves equivalent grouping/filtering. |
| OPS-11 | Public README, installation, architecture, flow docs, and completion instructions (`README.md`; `docs/`) | Shipped; some stale | None | Retain (decided) | Docs describe only approved Rust behavior/install paths and distinguish deferred features. |
| OPS-12 | Single standalone binary with no runtime and offline core workflows (`README.md`) | Shipped | Distribution | Retain (contract) | Fresh-machine artifact smoke test needs no Go/Rust/runtime/network. |

### Feature decision gate

Before Phase 1:

1. Review each `Review` row with the user.
2. Replace it with Retain, Replace, Preserve-data-only, Drop, or Defer.
3. Add a one-sentence rationale and decision date in the `Decision / owner` cell.
4. Ensure each Retain row has a concrete acceptance check; each Drop/Defer row names affected docs/integrations.
5. Summarize counts in `plan-brief.md`.
6. Do not choose Rust modules/crates or translate tests for dropped rows.

## Persisted Data Compatibility Matrix

These rows are already **Preserve (decided)**. A feature may be dropped while its data remains readable and writable. Semantic preservation means existing values survive operations that rewrite the containing file; byte identity is required only for operations that currently move without rewriting.

| ID | Persisted contract | Current behavior/evidence | Required Rust check |
|---|---|---|---|
| DATA-01 | Column folder determines status | Configured folders scanned; hidden folders ignored (`internal/store/board.go`) | Fixed folders load as Backlog/Ready/WIP/Done; unsupported legacy folders remain byte-untouched and emit ticket-count warnings. |
| DATA-02 | Ticket frontmatter dialect | Simple first-colon parser, quote trimming, duplicate-last-wins, CRLF accepted (`internal/ticket/markdown.go`) | Fixture corpus freezes approved parser semantics; do not substitute full YAML semantics. |
| DATA-03 | Required ticket fields | `title`, `priority`, `created`, `updated` | Missing/blank/invalid fixtures fail with approved errors. |
| DATA-04 | Optional ticket fields | `id`, `deadline`, `important` | Every known optional value survives read/write even if its UI is dropped. |
| DATA-05 | Markdown body and Acceptance Criteria | Exact `## Acceptance Criteria`; bare `-` is empty | Existing bodies survive; readiness matches Go fixtures. |
| DATA-06 | Labels and kind prefixes | One leading label group; Feat/Feature, Task, Bug/Fix; fallback Task (`internal/ticket/title.go`) | Existing titles parse and normalize only when current operations normalize them. |
| DATA-07 | Priorities | Case-insensitive P0-P3, strict invalid rejection (`internal/ticket/priority.go`) | All ranks and invalid cases match approved contract. |
| DATA-08 | Ticket IDs | `TC-` plus six restricted characters; missing/invalid/duplicate warning semantics | Existing IDs never change unless approved ID migration is invoked. |
| DATA-09 | Ticket filenames | ID/title slug names; filename is stable reference for CLI/completions/manual order | Move-only operations preserve names; approved migration collision suffixes are deterministic. |
| DATA-10 | Config preferences | `editor`, `theme`, `skip_editor_prompt`, `disable_matrix_prioritisation` | Deserialize/save cycle retains every known field. |
| DATA-11 | Config columns | Ordered `{id,name,color}` entries | Existing custom columns and colors survive every config write. |
| DATA-12 | Sort config | `mode` and per-column `manual_order` filename arrays | Existing mode/order survive load, reconciliation, and save. |
| DATA-13 | Trash | `.trash/<ticket>.md` soft-delete storage | Existing trash is not treated as a column or deleted at cutover. |
| DATA-14 | Git ignore entry | Init appends `<board-basename>/` idempotently | Default/custom board path fixtures preserve newline and duplicate behavior. |
| DATA-15 | Warnings and malformed files | Bad files remain on disk and are skipped with warnings | Rust never deletes or rewrites a malformed ticket merely by loading a board. |
| DATA-16 | Move byte preservation | Move/trash use rename; metadata mutations rewrite LF text | Fixture checks byte identity for moves and approved normalization for mutations. |

## Known Defect and Ambiguity Register

Each row must be explicitly Preserve, Fix-before-freeze, Fix-in-Rust-with-contract, or Drop-with-feature before parity fixtures are finalized.

| ID | Current inconsistency | Evidence | Decision | Required resolution |
|---|---|---|---|---|
| DEF-01 | Global `--path <dir>` consumes `pick-next --path`, making path-only output unreachable through the real binary | `cmd/tickcats/main.go:28-39,150-165` | Review | Choose a non-conflicting intended CLI, then test the real process rather than the helper. |
| DEF-02 | `ids migrate` scans only legacy states and silently misses custom columns | `internal/store/ids.go:41-42` | Resolve with fixed-column scope (user, 2026-08-06) | Scan Backlog/Ready/WIP/Done only and emit skipped legacy-folder ticket counts instead of silently ignoring them. |
| DEF-03 | Soft delete accepts only legacy states while move accepts configured columns | `internal/store/delete.go:17-19` | Drop with custom-column feature (user, 2026-08-06) | Rust delete applies only to visible fixed-column tickets; unsupported legacy folders are never mutated. |
| DEF-04 | Watcher subscribes only to columns present at startup | `internal/tui/watcher.go:31-42` | Resolve via fixed columns (user, 2026-08-06) | If file watching is retained, subscribe only to the four fixed folders; no dynamic subscription exists. |
| DEF-05 | Column deletion and ID migration can partially mutate files before a later error | `internal/store/config.go:219-239`; `internal/store/ids.go` | Review | Preserve partial behavior or specify preflight/rollback semantics. |
| DEF-06 | Read-like commands can rewrite `config.json` during reconciliation | `internal/store/config.go:55-79` | Review | Preserve, separate sync from reads, or constrain side effects explicitly. |
| DEF-07 | External editor command uses whitespace splitting, not shell quoting | `internal/tui/editor.go` | Review | Freeze simple token behavior or adopt and document a different command contract. |

## Implementation Approach

Use a parallel, compatibility-first port:

```text
approved feature/data/defect matrices
                 ↓
contract fixtures + Go reference binary
                 ↓
Rust ticket/data → Rust store/CLI → Rust TUI
                 ↓
completions/skills/release proof
                 ↓
Rust cutover and Go deletion
```

Prefer one Rust package and the fewest modules needed by retained features. Select dependencies only after Phase 0; likely categories are JSON serialization, date/time, terminal UI/events, filesystem watching, temp directories, and ID randomness. Do not add a full YAML parser, plugin framework, service layer, or one-implementation traits.

Golden comparisons apply only where current Go behavior is approved. When a defect is fixed, both binaries need not match; the fixture records the approved intended contract and the plan documents whether Go is fixed first or Rust intentionally differs.

## Critical Implementation Details

The data-preservation gate is independent from feature retention. Dropping deadlines, matrix sorting, custom colors, or manual ordering from the Rust UI does not permit erasing their existing ticket/config/sort values.

Fixture boards must be copied before each command because config reconciliation and mutations can write during tests. Compare filesystem side effects as well as stdout, stderr, and exit status.

## Phase 0: Approve Migration Scope

### Overview

Turn the inventory above into an approved migration contract before writing Rust code.

### Changes Required

#### 1. Feature decisions

**File**: `context/changes/go-to-rust-migration/plan.md`

**Intent**: Decide the value of each current, PRD-only, and in-flight behavior.

**Contract**: No feature row remains `Review`; every decision has a rationale/date and an acceptance or removal consequence.

#### 2. Defect decisions

**File**: `context/changes/go-to-rust-migration/plan.md`

**Intent**: Prevent golden tests from canonizing accidental Go behavior.

**Contract**: Every DEF row records Preserve, Fix-before-freeze, Fix-in-Rust-with-contract, or Drop-with-feature.

#### 3. Brief summary

**File**: `context/changes/go-to-rust-migration/plan-brief.md`

**Intent**: Make the approved scope reviewable without rereading the full plan.

**Contract**: Brief reports retained/preserve-only/dropped/deferred counts and the final defect decisions.

### Success Criteria

#### Automated Verification

- A repository check reports zero `| Review` decisions in the feature and defect matrices.
- Every retained feature ID appears in at least one later phase acceptance mapping.
- Every DATA row remains Preserve.

#### Manual Verification

- User approves the feature counts, rationales, and defect dispositions.
- User confirms no in-flight or PRD-only behavior was mistaken for shipped behavior.

---

## Phase 1: Freeze Approved Contracts and Scaffold Rust

### Overview

Create the smallest Rust project and test oracle that represent only approved behavior and all preserved data.

### Changes Required

#### 1. Rust package skeleton

**File**: `Cargo.toml`, `Cargo.lock`, `src/main.rs`, `src/lib.rs`, `.gitignore`

**Intent**: Add a `tickcats` Rust binary beside Go with no product implementation yet.

**Contract**: Stable Rust builds/tests the package; `/target/` is ignored; package and binary are named `tickcats`.

#### 2. Contract fixture corpus

**File**: `tests/fixtures/`, `tests/contracts/`

**Intent**: Capture approved ticket, board, config, sort, warning, error, and CLI behavior.

**Contract**: Each retained CORE/CLI feature and every DATA row maps to at least one fixture/check. Include malformed frontmatter, CRLF, labels, priorities, IDs, deadlines, importance, custom columns/colors, manual sort, collisions, and pick ties.

#### 3. Go/Rust process harness

**File**: `scripts/compare-go-rust.sh`, `tests/cli_contract.rs`

**Intent**: Compare real process behavior and filesystem side effects without mutating source fixtures.

**Contract**: Harness copies fixtures, captures stdout/stderr/exit code, snapshots resulting files, and supports intended-contract expectations for approved Go defects.

#### 4. Temporary dual-language CI

**File**: `.github/workflows/ci.yml`

**Intent**: Make the parallel port continuously verifiable.

**Contract**: CI runs Go tests, Rust format/lint/tests, and available contract cases. Post-cutover cleanup is specified in Phase 6.

### Success Criteria

#### Automated Verification

- `go test ./...` passes.
- `cargo fmt --check`, `cargo clippy -- -D warnings`, and `cargo test` pass for the skeleton.
- Fixture manifest covers all approved feature IDs and DATA rows.
- Contract harness proves it uses copied fixture trees and records streams/status/filesystem effects.

#### Manual Verification

- Confirm the Rust module/dependency set contains nothing for dropped/deferred features.
- Confirm fixture expectations reflect approved intent rather than undocumented accidents.

---

## Phase 2: Port Ticket and Persisted Data Contracts

### Overview

Port parsing, domain values, and semantic round-trip behavior before filesystem actions or TUI work.

### Changes Required

#### 1. Ticket parser and model

**File**: `src/ticket.rs` or `src/ticket/`

**Intent**: Implement DATA-02 through DATA-08 without broadening the frontmatter dialect.

**Contract**: Required/optional fields, RFC3339 timestamps, date/bool parsing, body retention, Acceptance Criteria detection, title labels/kinds, priorities, and ID validation match approved fixtures.

#### 2. Config and sort models

**File**: `src/store/config.rs`, `src/store/sort.rs`

**Intent**: Preserve DATA-10 through DATA-12 independently of retained UI features.

**Contract**: Every known config/sort field survives load/save; custom column colors and manual order are never erased.

#### 3. Compatibility tests

**File**: `tests/ticket_contract.rs`, `tests/data_roundtrip.rs`

**Intent**: Replace test-for-test translation with focused behavior coverage.

**Contract**: Every DATA row has positive, malformed, and round-trip cases where relevant.

### Success Criteria

#### Automated Verification

- Rust ticket/data tests pass for every DATA row.
- Existing `.tickcats-test/**/*.md`, `config.json`, and `sort.json` fixtures load successfully.
- Semantic round-trip tests prove no known persisted field is erased.
- `go test ./internal/ticket ./internal/store` still passes.

#### Manual Verification

- Spot-check representative real tickets and config/sort files in both implementations.

---

## Phase 3: Port Retained Store and CLI Features

### Overview

Build a useful non-interactive Rust binary by implementing approved board operations and script-exact CLI contracts.

### Changes Required

#### 1. Board and filesystem operations

**File**: `src/store/`

**Intent**: Implement retained CORE features and DATA-01, DATA-09, DATA-13 through DATA-16.

**Contract**: Folder scanning, warnings, init/gitignore, create, move, configured columns, reconciliation, pick-next, trash, metadata updates, sort persistence, and ID migration exist only where retained; defect decisions define edge behavior.

#### 2. CLI dispatch

**File**: `src/cli.rs`, `src/main.rs`

**Intent**: Implement retained CLI rows with script-exact contracts.

**Contract**: Approved arguments, aliases, stdout/stderr, exit status, warnings, and printed paths are stable. Help prose may differ while command vocabulary remains approved.

#### 3. Completion helpers and integration contracts

**File**: `src/cli.rs`, `tests/cli_contract.rs`

**Intent**: Keep hidden helpers and script consumers working.

**Contract**: `__complete tickets|columns` emits approved live candidates without extra output.

### Success Criteria

#### Automated Verification

- `cargo test` passes all retained CORE/CLI feature mappings.
- Process contract harness passes for approved `init`, `new`, `list`, `move`, `pick-next`, ID, completion, help, warning, and error cases.
- Filesystem snapshots pass for init, create, move, metadata mutation, reconciliation, collisions, and malformed files.
- `go test ./...` still passes.

#### Manual Verification

- Run retained Rust commands against a copied `.tickcats-test/` board.
- Confirm dropped/deferred commands are absent or clearly reported as unsupported per their matrix decisions.

---

## Phase 4: Port Retained TUI Workflows

### Overview

Implement only retained TUI rows with workflow parity: preserve approved keys, state transitions, data effects, offline operation, and width safety without requiring Bubble Tea glyph-for-glyph output.

### Changes Required

#### 1. TUI model and event loop

**File**: `src/tui/model.rs`, `src/tui/update.rs`

**Intent**: Represent approved views, overlays, cursor/scroll state, selection, notifications, and watcher messages.

**Contract**: Each retained TUI row maps to a model transition test; dropped modes have no speculative scaffolding.

#### 2. Board/detail/forms/dialog rendering

**File**: `src/tui/`

**Intent**: Render approved workflows with Ratatui/Crossterm or the smallest selected equivalent.

**Contract**: Retained board/detail/create/config/search/help/dialog elements fit agreed narrow and medium widths; terminal library differences may change borders/glyphs.

#### 3. Store/editor/watcher integration

**File**: `src/tui/actions.rs`, `src/tui/editor.rs`, `src/tui/watcher.rs`

**Intent**: Route every retained mutation through tested store operations and preserve approved reload/editor behavior.

**Contract**: Defect decisions determine dynamic-column delete/watching and editor argument semantics.

### Success Criteria

#### Automated Verification

- Every retained TUI ID has a state-transition, render-bound, or action integration check.
- `cargo test` passes TUI and CLI contracts.
- Narrow/medium terminal render checks do not exceed agreed dimensions.
- Dropped/deferred TUI IDs have no unreachable implementation scaffolding.

#### Manual Verification

- Run the Rust TUI against a copied `.tickcats-test/` board and execute every retained TUI matrix scenario.
- Confirm muscle-memory keys and external-editor handoff work in a real terminal.
- Confirm file reload and terminal restoration behave correctly on exit/error.

---

## Phase 5: Prove Integrations and Distribution

### Overview

Prove that the Rust binary works with retained repository tooling and the current distribution surface before deleting Go.

### Changes Required

#### 1. Shell completions

**File**: `completions/tickcats.bash`, `completions/_tickcats.zsh`, `completions/tickcats.fish`

**Intent**: Keep approved command/static options and live ticket/column candidates aligned with Rust.

**Contract**: Checked-in filenames and Homebrew destinations remain stable; syntax and candidate tokenization pass for all three shells.

#### 2. Agent Skills integration

**File**: `skills/`, `scripts/install-skills.sh` if retained

**Intent**: Verify retained skills and installer behavior against Rust rather than assuming CLI unit parity is enough.

**Contract**: Approved skills consume Rust printed paths/output and folder behavior successfully; Go-specific guidance is removed.

#### 3. Rust release workflow

**File**: `.github/workflows/release.yml`, Rust release configuration/scripts

**Intent**: Replace GoReleaser with the smallest Rust release path that preserves OPS-01 through OPS-05 and OPS-07.

**Contract**: Semantic-version tag trigger produces the same five target archives, archive names/formats/contents, executable bits, and `checksums.txt`; Homebrew tap publishing keeps current secret and smoke-test behavior.

#### 4. Installation and architecture docs

**File**: `README.md`, `docs/installation.md`, `docs/architecture.md`, `docs/flows/`, `context/foundation/tech-stack.md`

**Intent**: Describe the approved Rust product and no longer advertise dropped/deferred/Go-only behavior.

**Contract**: Homebrew/direct download remain; `go install` is removed; crates.io/Scoop are not claimed.

### Success Criteria

#### Automated Verification

- Bash/Zsh/Fish completion checks pass against the Rust binary.
- Retained Agent Skills smoke tests pass in temporary repositories.
- Release dry run produces exactly macOS amd64/arm64, Linux amd64/arm64, and Windows amd64 artifacts plus checksums.
- Archive-content checks find the binary, LICENSE, README, and all completion scripts.
- Homebrew formula/cask equivalent passes `tickcats --path <temp>/.tickcats init`.
- Rust format/lint/tests/contracts and Go tests still pass.

#### Manual Verification

- Install one macOS/Linux artifact and one Windows artifact or test equivalent.
- Install via the generated Homebrew path and verify completions plus core CLI/TUI smoke flows.
- Review public docs for stale Go commands and unapproved feature claims.

---

## Phase 6: Cut Over and Remove Go

### Overview

Make Rust the sole implementation only after the release-proof gate is green.

### Changes Required

#### 1. Release-proof gate

**File**: `context/changes/go-to-rust-migration/plan.md`

**Intent**: Record evidence that product, data, CLI, TUI, integration, and packaging contracts passed.

**Contract**: All prior automated/manual progress items are complete; no Review rows or accepted parity gaps remain.

#### 2. Remove Go implementation and tooling

**File**: `cmd/`, `internal/`, `go.mod`, `go.sum`, `.goreleaser.yml`, ignored root Go binary if present

**Intent**: End dual maintenance after the Rust release path is proven.

**Contract**: No Go source/build/release command remains; historical verification docs may remain clearly historical.

#### 3. Simplify CI and final docs

**File**: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, README/docs

**Intent**: Remove temporary Go/reference steps and leave one Rust build path.

**Contract**: CI runs only approved Rust checks; install and contributor commands are Rust-only.

### Success Criteria

#### Automated Verification

- `cargo fmt --check`, `cargo clippy -- -D warnings`, and `cargo test` pass after Go deletion.
- Repository search finds no active `go test`, `go build`, `go install`, GoReleaser, or `cmd/tickcats` instructions.
- A release dry run still produces the approved artifacts from the Rust-only tree.
- Fresh-board and existing-board contract fixtures pass with only Rust installed.

#### Manual Verification

- Install the final Rust artifact and run all retained smoke workflows.
- Confirm rollback is possible by reverting the cutover commit or republishing the previous release.
- Approve removal of the Go reference implementation.

## Testing Strategy

### Contract tests

- One manifest maps retained feature IDs and all DATA IDs to executable checks.
- Process tests capture stdout, stderr, exit status, and resulting filesystem tree.
- Fixture input is copied for every invocation.
- Intended-contract fixtures replace Go comparison where an approved defect is fixed.

### Focused unit tests

- Parser dialect, title labels/kinds, priorities, IDs, Acceptance Criteria.
- Config/sort semantic preservation.
- Pick-next eligibility/ranking/ties.
- Approved path validation, collision, and multi-file preflight behavior.
- Retained TUI state transitions and width bounds.

### Manual tests

- Real terminal interaction for every retained TUI row.
- External editor and file-watcher behavior.
- Shell completion in Bash/Zsh/Fish.
- Release artifact and Homebrew install smoke tests.

Do not translate Go tests mechanically. Keep Go tests as a reference until cutover, then retain Rust tests that protect approved behavior and data.

## Performance Considerations

The PRD defines small local boards. Use straightforward directory scans and sorting; do not add caches, databases, async runtimes, or indexing unless a measured retained workflow requires them. Preserve responsive TUI behavior at current fixture sizes rather than inventing a scale target.

## Migration and Rollback Notes

There is no user-data migration. Existing folders, tickets, config, sort order, trash, IDs, deadlines, importance, and column colors must survive unchanged semantically.

Before Phase 6, rollback is selecting the Go binary/release. Phase 6 should be one isolated Conventional Commit so reverting it restores the Go reference and release tooling without reverting the Rust implementation.

## References

- Product contract: `context/foundation/prd.md`
- Current stack rationale: `context/foundation/tech-stack.md`
- CLI: `cmd/tickcats/main.go`
- Ticket parser/title/priority/ID: `internal/ticket/`
- Store: `internal/store/`
- TUI: `internal/tui/`
- Release: `.github/workflows/release.yml`, `.goreleaser.yml`
- Release verification: `context/changes/release-verification/verification.md`
- Public behavior: `README.md`, `docs/`, `completions/`, `skills/`
- In-flight column-color picker: root `plan.md`
- In-flight skills installer: `scripts/install-skills.sh` and current README diff

## Progress

> Convention: `- [ ]` pending, `- [x]` done. Append ` — <commit sha>` when a step lands. Do not rename step titles.

### Phase 0: Approve Migration Scope

#### Automated

- [ ] 0.1 Zero feature or defect matrix rows remain Review.
- [ ] 0.2 Every retained feature ID maps to a later acceptance check.
- [ ] 0.3 Every DATA row remains Preserve.

#### Manual

- [ ] 0.4 User approves feature counts, rationales, and defect dispositions.
- [ ] 0.5 User confirms shipped, PRD-only, and in-flight classifications.

### Phase 1: Freeze Approved Contracts and Scaffold Rust

#### Automated

- [ ] 1.1 Go baseline tests pass.
- [ ] 1.2 Rust format, lint, and skeleton tests pass.
- [ ] 1.3 Fixture manifest covers all approved feature IDs and DATA rows.
- [ ] 1.4 Process harness captures streams, status, and copied-tree side effects.

#### Manual

- [ ] 1.5 Rust modules/dependencies contain nothing for dropped/deferred features.
- [ ] 1.6 Fixture expectations match approved intent.

### Phase 2: Port Ticket and Persisted Data Contracts

#### Automated

- [ ] 2.1 Rust ticket/data tests pass for every DATA row.
- [ ] 2.2 Existing ticket/config/sort fixtures load successfully.
- [ ] 2.3 Round-trip checks preserve every known field.
- [ ] 2.4 Go ticket/store tests still pass.

#### Manual

- [ ] 2.5 Representative real tickets and config/sort files match in Go and Rust.

### Phase 3: Port Retained Store and CLI Features

#### Automated

- [ ] 3.1 Retained CORE/CLI mappings pass Rust tests.
- [ ] 3.2 Process contracts pass for all retained commands and errors.
- [ ] 3.3 Filesystem side-effect snapshots pass.
- [ ] 3.4 Full Go suite still passes.

#### Manual

- [ ] 3.5 Retained Rust commands work against a copied real board.
- [ ] 3.6 Dropped/deferred commands follow approved absence/error behavior.

### Phase 4: Port Retained TUI Workflows

#### Automated

- [ ] 4.1 Every retained TUI ID has an automated state/render/action check.
- [ ] 4.2 Rust TUI and CLI contracts pass.
- [ ] 4.3 Agreed narrow/medium terminal bounds pass.
- [ ] 4.4 No scaffolding remains for dropped/deferred TUI rows.

#### Manual

- [ ] 4.5 Every retained TUI matrix scenario passes in a real terminal.
- [ ] 4.6 External editor, watcher, and terminal restoration pass.

### Phase 5: Prove Integrations and Distribution

#### Automated

- [ ] 5.1 Bash, Zsh, and Fish completion checks pass.
- [ ] 5.2 Retained Agent Skills smoke tests pass.
- [ ] 5.3 Release dry run produces exactly five approved target artifacts and checksums.
- [ ] 5.4 Archive names, formats, contents, and executable bits pass.
- [ ] 5.5 Homebrew install smoke test passes.
- [ ] 5.6 Rust checks/contracts and Go tests pass together.

#### Manual

- [ ] 5.7 Representative macOS/Linux and Windows artifacts install and run.
- [ ] 5.8 Homebrew installation, completions, CLI, and TUI smoke flows pass.
- [ ] 5.9 Public docs contain no stale Go or unapproved-feature claims.

### Phase 6: Cut Over and Remove Go

#### Automated

- [ ] 6.1 Rust format, lint, and tests pass after Go deletion.
- [ ] 6.2 Active repository instructions contain no Go build/install/release commands.
- [ ] 6.3 Rust-only release dry run produces approved artifacts.
- [ ] 6.4 Fresh and existing board contracts pass with only Rust installed.

#### Manual

- [ ] 6.5 Final Rust artifact passes all retained smoke workflows.
- [ ] 6.6 Cutover commit has a tested revert path.
- [ ] 6.7 User approves removal of the Go reference implementation.
