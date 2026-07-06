---
name: tickcats-from-roadmap
description: >
  Convert a 10x-roadmap output (context/foundation/roadmap.md) into tickcats
  tickets. Foundations become task tickets; vertical slices become feat tickets.
  Priorities, acceptance criteria, and blocked status are derived from the
  roadmap automatically. Use when the user says "create tickcats tickets from
  roadmap", "populate tickcats from roadmap", "roadmap to tickets", "add roadmap
  to tickcats", "tickcats from roadmap", or "turn the roadmap into tickets".
  Requires tickcats CLI and an initialized .tickcats board in the project.
---

# tickcats-from-roadmap

Convert `context/foundation/roadmap.md` into tickcats tickets. Foundations → `task`, slices → `feat`. Shows a full mapping preview before writing anything.

## Phase 0: Preflight

Check the board exists:

```bash
ls .tickcats/
```

If `.tickcats/` is missing, stop immediately:

> No tickcats board found. Run `tickcats init` first, then re-invoke this skill.

## Phase 1: Locate the roadmap

Default path: `context/foundation/roadmap.md`.

If an argument was passed (e.g. `/tickcats-from-roadmap path/to/roadmap.md`), use that path instead.

If the file doesn't exist at the resolved path, ask the user:

- Provide a different path
- Run `/10x-roadmap` first to generate one
- Cancel

If the file exists, read it fully.

## Phase 2: Parse and preview

Extract from the roadmap:

**From `## Backlog Handoff` table:** ID (F-NN / S-NN), Change ID, suggested issue title, ready status.

**From `## Foundations` and `## Slices` sections:** for each ID, collect:
- `Outcome` field → acceptance criteria source
- `Prerequisites` field → dependency chain
- `Unlocks` field (foundations only) → how many slices this unblocks (fan-out count)
- `Status` field → whether to add `[blocked]` prefix
- `PRD refs` field → context section content

**Derive priorities:**

| Condition | Priority |
|---|---|
| F-NN with Unlocks fan-out ≥ 2 | P0 |
| F-NN with Unlocks fan-out = 1 | P1 |
| S-NN that is the roadmap's North star | P1 |
| S-NN with Status: ready | P1 |
| S-NN with Status: proposed, has downstream slices | P2 |
| S-NN with no downstream slices (leaf) | P2 |
| Any item with Status: blocked | P3 |

**Identify `ready` targets:** tickets to move from backlog → ready after creation:
- Foundations with Status: ready
- Slices with Status: ready
- Slices whose Prerequisites are all foundations already reported as `done` in the roadmap

**Present mapping table before any writes:**

```
Roadmap: context/foundation/roadmap.md
Project: <project from frontmatter>

FOUNDATIONS → task tickets
  F-01  <change-id>  P0  → ready
  F-02  <change-id>  P1  → ready

SLICES → feat tickets
  S-01  <change-id>  P1  → ready      (after F-01)
  S-02  <change-id>  P1  → backlog    (after F-01, S-01)
  S-03  <change-id>  P2  → backlog    (after F-01, S-01)
  S-05  [blocked] <change-id>  P3  → backlog

N tickets total. Proceed, or adjust any priorities?
```

Wait for confirmation. Accept:
- "yes" / "go" / "proceed" → continue
- Priority overrides (e.g. "S-03 should be P1") → update and show revised table, confirm again

## Phase 3: Create tickets in dependency order

**Create foundations first (F-01, F-02, ... in order):**

```bash
FILE=$(tickcats new task "<change-id>: <suggested title>")
```

`tickcats new` prints the relative file path. Capture it. Then edit the file:

1. **Set priority** in YAML frontmatter: `priority: P0` (or P1)
2. **Rewrite `## Acceptance Criteria`** using the Outcome field as the primary criterion, plus any Unlocks items:
   ```
   ## Acceptance Criteria
   - [ ] <Outcome sentence verbatim from roadmap>
   - [ ] Unlocks: <downstream S-NN slice names>
   ```
3. **Add `## Context`** section body:
   ```
   ## Context
   Roadmap: <F-NN> — <Change ID>
   PRD refs: <PRD refs field verbatim>
   <Outcome sentence>.
   Unlocks: <Unlocks field verbatim>.
   ```

**Create slices in topological order (Prerequisites satisfied before dependents):**

```bash
FILE=$(tickcats new feat "<change-id>: <suggested title>")
```

Pass the title WITHOUT labels — `tickcats new` does not parse labels from its argument (a `[blocked]` passed here would land after the `Feat:` prefix as plain text and never be recognized by `pick-next` or the TUI).

Edit the file:

1. **For blocked slices, add the label to the frontmatter `title:`** — prepend ONE bracket group at the very start: `title: "[blocked] Feat: <change-id>: <suggested title>"`. Multiple labels go in the same group, comma-separated (`[blocked, to refine]`) — never two bracket groups; only the first is parsed.
2. **Set priority** in YAML frontmatter
3. **Rewrite `## Acceptance Criteria`** from the Outcome field:
   ```
   ## Acceptance Criteria
   - [ ] <Outcome sentence verbatim from roadmap>
   ```
   If the roadmap Outcome references specific PRD refs (FR-NNN, US-NN), add them as context lines after the criterion.
4. **Add `## Context`** section:
   ```
   ## Context
   Roadmap: <S-NN> — <Change ID>
   PRD refs: <PRD refs field verbatim>
   Prerequisites: <prerequisite Change IDs, comma-separated>
   <Outcome sentence>.
   ```

Track the file path printed by each `tickcats new` call alongside its roadmap ID — needed for Phase 4.

## Phase 4: Move ready tickets to ready column

For each ticket identified as ready in Phase 2:

```bash
tickcats move <bare-filename> backlog ready
```

`tickcats move` takes the BARE filename (e.g. `tc-abcdef-my-ticket.md`), not a path — strip the `.tickcats/backlog/` prefix from the path `tickcats new` printed, or it errors with "ticket name must be a file name". It prints the ticket's new path on success.

Leave all other tickets in backlog. Blocked tickets (with `[blocked]` prefix) stay in backlog unconditionally.

## Phase 5: Report

Print a final summary:

```
Created N tickets from context/foundation/roadmap.md

FOUNDATIONS
  TC-XXXXXX  P0  task  <change-id>: <title>   → ready
  TC-YYYYYY  P1  task  <change-id>: <title>   → ready

SLICES
  TC-AAAAAA  P1  feat  <change-id>: <title>   → ready
  TC-BBBBBB  P1  feat  <change-id>: <title>   → backlog  (needs <change-id>)
  TC-CCCCCC  P2  feat  <change-id>: <title>   → backlog  (needs <change-id>)
  TC-DDDDDD  P3  feat  [blocked] <change-id>  → backlog

Run `tickcats list` to see the board.
Run `tickcats pick-next` to start with the highest-priority ready ticket.
```

Do NOT run `tickcats tui` or `tickcats pick-next` automatically.

## Edge cases

**Roadmap has no `## Backlog Handoff` table:** fall back to parsing `## At a glance` table directly (same columns, same data). If neither table exists, the roadmap is malformed — tell the user and stop.

**Roadmap has Status: done items:** skip them. Do not create tickets for work already archived.

**`tickcats new` fails:** print the error verbatim and stop. Do not attempt partial recovery or continue creating remaining tickets — a half-populated board is worse than none.

**Duplicate Change IDs already exist in the board:** check with `tickcats list` output before creating. If a ticket with the same Change ID prefix already exists in any column, skip that item and note it in the Phase 5 report as "already exists".
