# Go to Rust Migration — Plan Brief

> Full plan: `context/changes/go-to-rust-migration/plan.md`

## What & Why

Replace TickCats' Go implementation with Rust as the long-term maintenance stack. The migration starts with a feature-retention gate: current behavior is listed individually so the user can retain, preserve-data-only, drop, or defer it before any Rust porting work begins.

## Starting Point

TickCats has 6,191 lines of non-test Go and 5,878 lines of Go tests across CLI, ticket, store, and TUI modules. The implementation is broader than the PRD and also contains known inconsistencies, so neither the PRD nor current executable behavior can be treated as the sole migration scope without review.

Current matrix state:

- **68 feature/integration rows:** 15 retained by contract/interview, 2 replaced, 3 dropped, 48 awaiting review.
- **16 persisted-data rows:** all preservation is mandatory.
- **7 known defect/ambiguity rows:** 2 resolved, 5 awaiting review.

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
- `pick-next --path`, custom-column ID migration, and partial multi-file failures still need intended-behavior decisions.
- Agent Skills consume CLI paths/output and may expose contracts absent from unit tests.
- Existing unknown config/frontmatter fields are not preserved more strongly than current Go behavior; all **known** fields are.

## Success Criteria Summary

- All 48 feature and 5 defect Review rows are resolved before Rust implementation.
- Existing boards and every known persisted field survive Rust read/write operations.
- Retained CLI/TUI/integration checks and five-platform release proof pass before Go deletion.
