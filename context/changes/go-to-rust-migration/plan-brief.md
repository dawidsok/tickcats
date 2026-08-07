# Go to Rust Migration — Plan Brief

> Full plan: `context/changes/go-to-rust-migration/plan.md`

## What & Why

Replace TickCats' Go implementation with Rust as the long-term maintenance stack. The migration starts with a feature-retention gate: current behavior is listed individually so the user can retain, preserve-data-only, drop, or defer it before any Rust porting work begins.

## Starting Point

TickCats has 6,191 lines of non-test Go and 5,878 lines of Go tests across CLI, ticket, store, and TUI modules. The implementation is broader than the PRD and also contains known inconsistencies, so neither the PRD nor current executable behavior can be treated as the sole migration scope without review.

Current matrix state:

- **69 feature/integration rows:** 34 retained, 17 replaced, 1 preserve-data-only, 9 dropped, 8 awaiting review.
- **16 persisted-data rows:** all preservation is mandatory.
- **7 known defect/ambiguity rows:** 6 resolved, 1 awaiting review.

## Desired End State

The repo ships one Rust `tickcats` binary. Existing `.tickcats/` boards load without migration, approved CLI contracts remain script-compatible, retained TUI workflows preserve keyboard behavior, and current GitHub/Homebrew artifacts remain installable. Go is removed only after release proof.

## Key Decisions Made

| Decision | Choice | Why |
|---|---|---|
| Long-term stack | Rust | This is a permanent migration, not only a learning prototype. |
| Feature scope | Matrix gate before implementation | Prevents low-value current behavior from being ported automatically. |
| Baseline | Main plus separately labeled in-flight work | Distinguishes shipped, PRD-only, undocumented, and unfinished behavior. |
| Migration style | Parallel Rust port | Keeps Go as reference and rollback until Rust ships successfully. |
| Data policy | Preserve every known persisted field | Dropping a UI feature must not erase existing board data. |
| Defects | Define intended behavior before parity freeze | Avoids canonizing accidental Go behavior. |
| CLI parity | Exact for scripts | Protects exit codes, streams, paths, completions, and Agent Skills. |
| TUI parity | Workflow, not pixel parity | Preserves muscle memory without binding Rust to Bubble Tea glyphs. |
| Testing | Feature-to-contract matrix | Verifies behavior without translating every Go test. |
| Distribution | Current surface | Retains five targets, GitHub archives/checksums, completions, and Homebrew. |
| Cutover | Release proof first | Go stays until product, integration, and artifact checks all pass. |
| Workflow columns | Fixed Backlog/Ready/WIP/Done, no WIP limit | Drops Won't Do from the new default; WIP keeps `doing/` and remains a label rather than an enforced capacity. |
| Legacy columns | Leave untouched and warn | Custom and `wont-do/` folders stay on disk but are excluded from the fixed board and reported with ticket counts. |
| Workflow transitions | Adjacent only, both directions | CLI and TUI allow Backlog↔Ready↔WIP↔Done; direct first/last shortcuts are removed. |
| Ticket identity | Stable `TC-XXXXXX` IDs | IDs survive filename/title changes and remain usable by prerequisites, skills, and commit references. |
| Legacy ID repair | Explicit `ids migrate` command | Migration touches only four fixed columns, warns about skipped legacy folders, and never runs during load. |
| Deadlines | Read-only matrix input | Dates appear on cards/detail and drive matrix urgency; editing and deadline sorting are removed. |
| Importance matrix | Retain with direct `M` toggle | Important toggle and urgency buckets remain; `M` replaces the config screen and persists the mode immediately. |
| Board sorting | Fixed priority/matrix order | Removes title/date/manual modes, sort cycling, reorder prompts, and writes to `sort.json`. |
| Deletion | Retain soft delete | `x` confirms and moves one visible ticket to `.trash/`; permanent deletion is not exposed. |
| Malformed tickets | Load valid files and warn | Invalid files remain untouched and cannot block the rest of the board. |
| Column colors | Preserve data only | Rust ignores legacy color values and omits the palette, setter, rendering override, and planned picker. |
| Themes | Neutral with semantic accents | Legacy theme values are ignored; color is reserved for focus, urgency, warnings, and success/error, never as the only signal. |
| External editor | `$EDITOR`, then `vi` | Shell-word parsing supports quoted arguments without invoking a shell; board editor settings are removed. |
| TUI creation | Four-field form, then board | Retains kind/title/priority/refine fields; creation focuses the new ticket and `e` replaces the post-create editor prompt. |
| TUI launch | No args plus `tui` | Keeps the shortest interactive launch and an explicit scriptable alias. |
| Board path | Retain global `--path` | Every command and TUI launch can target an explicit board directory. |
| Pick path output | `pick-next --print-path` | Replaces the unreachable conflicting `pick-next --path` while keeping script-friendly output. |
| Board init | Four folders plus `.gitignore` | Explicit, idempotent setup preserves local/private-by-default behavior. |
| Intro ticket | Guided ordinary ticket | It teaches help, editing/refinement, pick-next, and progression, but has no protection, regeneration, or hidden onboarding state. |
| CLI ticket creation | `feat|task|bug`, P2, `[to refine]`, optional `--ac` | Removes undocumented kind/AC aliases and aligns CLI/TUI readiness defaults. |
| CLI list | Retain four-column text output | Keeps noninteractive human/agent inspection with stable filenames, IDs, priorities, titles, and warnings. |
| CLI help | `help`, `--help`, and `-h` | Keeps conventional global/command discovery and explicit unknown-command errors. |
| Pick display | Mark recommended Ready card(s) | Removes the top banner; exact ties mark every tied card and show “choose one” without forcing a dialog. |
| Navigation | Vim keys plus arrows, no counts | Retains `h/j/k/l`, arrows, and `d/u`, but removes numeric motion-prefix state. |
| Narrow layout | Sliding full-width columns | Shows as many readable columns as fit and labels hidden sides instead of compressing all four. |
| Ticket detail | Side panel with narrow fallback | Wide terminals keep the board visible; narrow terminals use full-screen detail for readability. |
| Multi-select | Drop | Every action targets one focused ticket; selection state, markers, and bulk-move failure handling disappear. |
| Search | Retain current fuzzy model | Keeps priority/title/body subsequence matching, typing/navigation phases, counts, and cross-column results. |
| TUI help | Retain `?` overlay | Keeps the reduced mode-specific keymap discoverable after the first-run ticket is gone. |
| Quit | Immediate `q`/Ctrl-C | Removes confirmation state because the TUI has no unsaved edits; terminal restoration remains mandatory. |

## Scope

**In scope:**

- Review of every current, PRD-only, and in-flight feature.
- Persisted ticket/config/sort/folder compatibility.
- Approved CLI, store, and TUI behavior.
- Shell completions, retained Agent Skills, GitHub releases, and Homebrew.
- Final removal of Go after proof.

**Out of scope:**

- New product features or data format.
- Sync, collaboration, metrics, auth, AI, dashboards, or hosted services.
- Pixel-perfect TUI output.
- crates.io/cargo-install, Scoop, plugins, or public library APIs.
- Mechanical test-for-test translation.

## Architecture / Approach

```text
feature + defect decisions
          ↓
approved contracts and copied fixtures
          ↓
Rust ticket/data → store/CLI → retained TUI
          ↓
completions/skills/release proof
          ↓
Go removal
```

## Phases at a Glance

| Phase | What it delivers | Key risk |
|---|---|---|
| 0. Approve scope | Final feature and defect decisions | Porting starts before all Review rows are resolved. |
| 1. Freeze + scaffold | Rust package, fixtures, process harness, dual CI | Golden tests preserve an accidental bug. |
| 2. Ticket + data | Parser/domain/config/sort compatibility | A rewrite silently erases optional data. |
| 3. Store + CLI | Useful script-compatible Rust binary | Filesystem and output edge cases drift. |
| 4. TUI | Only retained keyboard workflows | Terminal event/render differences. |
| 5. Integrations + release | Completions, skills, five targets, Homebrew | Packaging succeeds locally but not for users. |
| 6. Cutover | Rust-only repository | Go is removed before rollback evidence exists. |

**Prerequisites:** Rust toolchain; Go retained through Phase 5; user approval of all Phase 0 rows.

**Estimated effort:** Phase 0 is one focused review session. A near-full retention decision is roughly 10–16 focused implementation/QA sessions, dominated by TUI and release parity; dropping features reduces later phases directly.

## Open Risks & Assumptions

- TUI and release parity will cost more than ticket/store porting.
- Partial multi-file failure handling remains the only unresolved defect decision.
- Agent Skills consume CLI paths/output and may expose contracts absent from unit tests.
- Existing unknown config/frontmatter fields are not preserved more strongly than current Go behavior; all **known** fields are.

## Success Criteria Summary

- All 8 feature and 1 defect Review rows are resolved before Rust implementation.
- Existing boards and every known persisted field survive Rust read/write operations.
- Retained CLI/TUI/integration checks and five-platform release proof pass before Go deletion.
