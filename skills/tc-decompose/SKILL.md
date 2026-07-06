---
name: tc-decompose
description: >
  Tickcats-pipeline variant of 10x-roadmap. Decompose context/foundation/prd.md
  directly into tickcats ticket stubs on the .tickcats/ board — this replaces
  the roadmap.md file entirely: no roadmap document is ever written. Foundations
  become task tickets, vertical slices become feat tickets, all labeled
  [to refine] in backlog, with priorities and [blocked]/Prerequisites wiring
  derived from the dependency graph. Trigger phrases: "decompose the PRD into
  tickets", "tc decompose", "PRD to tickcats", "create ticket stubs from PRD".
  Use AFTER /tc-prd, BEFORE /tc-refine or /tc-plan. For the classic
  roadmap-file workflow use /10x-roadmap instead.
argument-hint: "[path-to-prd]"
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash
  - Glob
  - Agent
  - AskUserQuestion
  - TaskCreate
  - TaskUpdate
---

# tc-decompose: PRD → tickcats ticket stubs

This skill is the bridge between **product** (PRD) and **per-ticket planning** (`/tc-plan`). Its single job: read a PRD, ask the 2-3 load-bearing framing questions, decompose the product into Foundations (cross-cutting enablers) and vertical Slices, and emit them as **ticket stubs** on the tickcats board — every one labeled `[to refine]`, priority set, dependencies wired via `Prerequisites:` lines and `[blocked]` labels.

**No roadmap.md is ever written.** The board IS the decomposition. Column = state, ticket = unit of work, `tickcats list` = the at-a-glance view. `context/foundation/` still holds prd.md, shape-notes.md, lessons.md — those survive; the roadmap file does not.

It is a **decomposition + sequencing** skill, not a planner. It NEVER picks frameworks, file paths, schemas, or implementation details — those belong to `/tc-plan`. It NEVER assigns time estimates, t-shirt sizes, or calendar dates. Stubs are intentionally thin: Context + draft Acceptance Criteria, nothing more. Full planning is `/tc-plan`'s job; sharpening stubs is `/tc-refine`'s.

## When to use, when to skip

**Use when**: `context/foundation/prd.md` exists with non-trivial content (FRs and user stories populated) AND the user wants the PRD broken into board tickets. Typical trigger: just finished `/tc-prd`.

**Skip when**: the PRD is hollow (large `## Open Questions`, `# TODO` markers) — point at `/tc-prd` (or upstream `/tc-shape`) first. Skip when the user wants to plan a *single* ticket in detail — that's `/tc-plan`. Skip when the user wants the classic roadmap-file workflow — that's `/10x-roadmap`.

## Board conventions (shared across all tc-* skills)

| Column | Meaning |
|---|---|
| backlog | stubs from tc-decompose, titled `[to refine] …` |
| ready | fully planned by tc-plan, prerequisites met |
| doing | tc-implement in progress |
| done | implemented (tc-impl-review appends findings here) |
| wont-do | split-away originals, rejected scope |

- **Ticket title format**: `[labels] Feat|Task: <change-id>: <human title>` — change-id is kebab-case, stable, assigned at decomposition (survives splits as `<change-id>a`, `<change-id>b`).
- **Blocked**: `[blocked]` in title until all `Prerequisites:` ticket IDs are in `done/`.
- **Labels**: `[to refine]` and `[blocked]` live in the title brackets; `tickcats pick-next` natively excludes both.
- Tickcats has **no native dependency fields** — dependencies are a convention: a `Prerequisites: TC-XXXXXX, TC-YYYYYY` line in the ticket's `## Context` section, mirrored by a `[blocked]` label on the title while any prerequisite is unmet.

## Stub anatomy (this skill's output contract)

Every stub this skill creates has exactly: frontmatter (title, id, priority) + `## Context` + draft `## Acceptance Criteria`. Nothing more — no Requirements, no Architecture, no Implementation Plan. `/tc-plan` writes those later.

```markdown
---
title: "[to refine] Feat: srs-review-session: User can review due cards"
id: TC-ABCDEF
priority: P1
---
## Context
<Why this exists — 1-2 sentences tied to the PRD's vision.>
PRD refs: FR-003, US-02
Prerequisites: TC-XXXXXX (minimal-auth), TC-YYYYYY (card-storage)
Outcome: user can <verb-led, user-visible capability>.

## Acceptance Criteria
- [ ] <primary criterion derived from the Outcome sentence>
- [ ] <secondary criteria from the referenced FRs' testable clauses>
```

Foundations use the same shape; their Outcome starts with `(foundation)` and their Context names the downstream slices they unlock. Root tickets (no prerequisites) write `Prerequisites: —`.

## Interactive prompts

Whenever the procedure says *"ask the user"*, use the host's interactive-question tool (Claude Code → `AskUserQuestion`; other harnesses → any tool that asks a structured question with options; none available → plain conversational message with labelled options). State which tool you selected the first time you ask.

## Process

### Step 0: Preflight — CLI, board, gitignore

Run these checks in order; each gates the next.

**0a. CLI on PATH:**

```bash
command -v tickcats
```

If missing, STOP with:

> `tickcats` CLI not found on PATH. Install it first (e.g. `bun install -g tickcats` or your package manager of choice), then re-invoke `/tc-decompose`.

**0b. Board exists:**

```bash
test -d .tickcats || tickcats init
```

If `.tickcats/` is absent, run `tickcats init` and report that the board was initialized.

**0c. Board is committed — reverse the tickcats default.** This workflow treats tickets as shareable, reviewable artifacts, so `.tickcats/` must NOT be gitignored. If `.gitignore` exists and contains a line matching `.tickcats` (with or without trailing slash), remove that line and tell the user:

> Removed `.tickcats/` from .gitignore — in the tc-* workflow the board is committed so tickets can be shared and reviewed.

If there is no `.gitignore` or no such line, continue silently.

### Step 1: Read PRD and supplementary inputs

Resolve the PRD path: argument if provided (strip leading `@`), else `context/foundation/prd.md`. If the file is missing, ask:

Interactive question:
- question: "No PRD found at `<resolved-path>`. How would you like to proceed?"
  header: "Input?"
  options:
  - label: "Run /tc-prd first (Recommended)"
    description: "Stop here. Run /tc-prd to produce prd.md, then re-invoke /tc-decompose."
  - label: "Provide a different path"
  - label: "Cancel"
  multiSelect: false

On "Run /tc-prd first" or "Cancel": STOP.

**Read the PRD FULLY** (no limit/offset). Then read, best effort:

- `context/foundation/shape-notes.md` — lift any `## Forward:` sections (especially forward-parked roadmap/decomposition content) as candidate inputs; the user already parked them there during shaping.
- `context/foundation/tech-stack.md` — informs Foundations (auth scaffold, deploy skeleton — anything the stack choice implied).
- `context/foundation/lessons.md` — scan for rules touching ordering or readiness. Priors, not gospel.

**PRD readiness check** — score 0-4, one point per signal:

1. **Vision & Problem Statement non-trivial** — exists, ≥ 2 sentences, no `# TODO`.
2. **≥ 1 populated user story** — `### US-NN:` heading with a Given/When/Then block beneath it.
3. **≥ 1 must-have FR** — line matching `^- FR-\d{3}: .* (P|p)riority: must-have$`.
4. **Business Logic populated** — first non-blank line of `## Business Logic` is declarative, not `# TODO: domain rule`.

Document the score explicitly in the conversation:

```
PRD readiness check (heuristic, 4 signals, 1 point each):
  [✓|✗] Vision & Problem Statement non-trivial
  [✓|✗] ≥ 1 populated user story
  [✓|✗] ≥ 1 must-have FR
  [✓|✗] Business Logic populated

  Score: <N>/4
  Open Questions in PRD: <count>
```

**Score ≥ 3**: proceed. **Score < 3**: warn — name the missing signals and their consequence (tickets from a hollow PRD will carry `[blocked]` labels whose first blocker is a PRD gap), then ask:

Interactive question:
- question: "How would you like to proceed?"
  header: "Thin PRD"
  options:
  - label: "Firm up PRD first (Recommended)"
    description: "Stop here. Resolve the PRD's Open Questions / TODOs, then re-invoke /tc-decompose."
  - label: "Proceed anyway"
    description: "Decompose from what's there. Hollow areas surface as [blocked] stubs whose Context names the PRD gap."
  - label: "Cancel"
  multiSelect: false

On "Firm up PRD first" or "Cancel": STOP.

### Step 2: Lean interview — 2-3 anchor questions, each with a strong Recommend

Capped interview: at most **three anchor questions** — `main_goal`, `north_star`, `top_blocker` — each carrying one strong **Recommend** grounded in a quoted artifact line, plus 1-2 alternatives each with its own one-line "why this is also reasonable" rationale. The user picks Recommend, picks an alternative, or overrides freely. Never more than 3 questions; skip an anchor only when the PRD literally states its value (announce the skip with the quote that locks it). If `shape-notes.md` carried forward-parked framing, feed it into the Recommend — don't re-elicit.

- **`main_goal`** — `market-feedback | quality | low-complexity | speed | learn | other`. Signals: timeline/budget frontmatter, target scale, Success Criteria phrasing ("learn from real users" → market-feedback), Vision tone.
- **`north_star`** — the smallest end-to-end user-visible flow that, shipped first, proves the PRD's core hypothesis. Options name concrete US-NN candidates, not abstract values. This anchor decides which slice gets sequenced (and prioritized) earliest.
- **`top_blocker`** — `skills | capacity | time | decisions | external | motivation | none`. Signals: ≥ 3 unresolved Open Questions → decisions; scope-vs-deadline mismatch → time/capacity; uncontracted vendor → external.

Question format (one structured question per anchor, sequential):

Interactive question:
- question: "<plain-language anchor question, in the user's language>"
  header: "<Goal | North star | Blocker>"
  options:
  - label: "<Recommend value> (Recommended)"
    description: "<One-line why, with the artifact quote that grounds it.>"
  - label: "<Alternative>"
    description: "Reasonable when <condition the artifacts partially support>; you'd pick this when <consequence>."
  - label: "Something else — I'll explain"
    description: "Free-form. Name the value and the reason."
  multiSelect: false

Rules: Recommend is always option 1; at most 2 alternatives; no strawmen (if only one value is plausible, present Recommend + free-form only and say so); mirror the user's language end-to-end. After the answers land, emit a short plain-markdown recap locking the framing — no new questions. The answers bias sequencing (Step 3) and priorities (Step 5).

### Step 3: Decompose — Foundations and Slices (in memory first)

Build the full ticket set in memory before touching the CLI.

**3a. Foundations** (→ `task` tickets, F-style enablers). A foundation is a cross-cutting prerequisite with no user-visible outcome of its own that unblocks named vertical slices. Sources: tech-stack.md implications (auth provider → auth scaffold), PRD NFRs needing infrastructure, PRD Access Control beyond "single user, no auth". Guardrails:

- **Minimal-unlock cap**: a foundation is the smallest enabler that lets a named slice proceed — never "complete the data/API/UI/auth layer". If its outcome sounds layer-complete, split it or fold the minimum into the first consuming slice.
- **Progressive disclosure**: introduce technical elements when the first slice needs them. "We'll need this eventually" is not a foundation.
- Every foundation's Context must name the downstream slices it unlocks. No unlock → no foundation.

**3b. Slices** (→ `feat` tickets, vertical user-facing). Walk `## User Stories` and `## Functional Requirements`; group into end-to-end slices where each:

- Delivers a single user-visible capability stated as "user can …".
- Touches every layer needed (data + logic + interface), top to bottom.
- Is generally one US-NN, occasionally 1-2 tightly-coupled FRs (e.g. create + list of the same entity).

Do NOT slice horizontally ("the database ticket", "the API ticket") — that's the anti-pattern this skill exists to prevent. Keep slices roughly comparable in weight; split an oversized candidate along user-visible outcomes, never by layer.

**Hard rule — never invent slices.** Every slice traces to a PRD US-NN or FR-NNN. Interview-surfaced extras become a note to the user, not a ticket.

**3c. Change IDs and titles.** Each ticket gets a stable kebab-case change-id (outcome-oriented: `first-gated-generation`, `minimal-auth`). Title format: `[to refine] Task: <change-id>: <human title>` for foundations, `[to refine] Feat: <change-id>: <human title>` for slices.

**3d. Dependency graph.** For each ticket, list prerequisite tickets (foundation a slice needs; slice whose data/capability another slice consumes). Verify: no cycles, every prerequisite exists in the set, topological order exists. Place the north star as early as its prerequisites allow; break remaining ties by `main_goal` (speed → strict must-have path first; market-feedback → riskiest assumption first; learn → unfamiliar tech first; quality → foundations eagerly).

**3e. Preview gate.** Present the full mapping before any writes:

```
FOUNDATIONS → task tickets
  <change-id>   P0   (unlocks: <slice change-ids>)
SLICES → feat tickets
  <change-id>   P1   ← north star
  <change-id>   P2   [blocked] (needs: <change-ids>)

N tickets total. Proceed, or adjust?
```

Wait for confirmation. Accept priority/scope overrides, show the revised table, confirm again. Do not create tickets until the user says go.

### Step 4: Derive priorities

| Condition (first match wins) | Priority |
|---|---|
| Foundation whose unlock fan-out ≥ 2 (blocks ≥ 2 downstream tickets) | P0 |
| North-star slice, or any ticket immediately actionable (no prerequisites) | P1 |
| Ticket with downstream dependents but not immediately actionable | P2 |
| Leaf ticket (nothing depends on it) | P2 |
| Ticket blocked by unmet prerequisites AND not on the north-star path | P3 |

`[blocked]` and P3 are not synonyms: a blocked ticket on the north-star critical path keeps its derived priority so `tickcats list` shows what matters next; P3 is for blocked leaves.

### Step 5: Create tickets in dependency order, back-fill real IDs

Ticket IDs (`TC-XXXXXX`) are only known after creation, so **create in topological order** — every ticket's prerequisites are created (and their IDs known) before the ticket itself.

For each ticket, in order:

```bash
FILE=$(tickcats new task "<change-id>: <human title>")   # foundations
FILE=$(tickcats new feat "<change-id>: <human title>")   # slices
```

Pass the title WITHOUT labels and WITHOUT a kind prefix — `tickcats new` adds the `Task:`/`Feat:` prefix itself and does NOT parse labels from its argument (labels passed here would land after the kind prefix as plain text and never be recognized).

`tickcats new` prints the created file path — **capture it** and record the ticket's `id` from its frontmatter, mapped to its change-id. Then edit the file directly:

1. **Add labels to the frontmatter `title:`** — prepend ONE bracket group at the very start, comma-separated: `title: "[to refine] Feat: <change-id>: <title>"`, or for tickets with unmet prerequisites `title: "[blocked, to refine] Feat: <change-id>: <title>"`. Never write two bracket groups (`[blocked] [to refine]`) — only the first is parsed. (At decomposition time nothing is in `done/`, so every ticket with any prerequisite starts blocked.)
2. **Set `priority:`** in the YAML frontmatter per Step 4.
3. **Write the `## Context` section** per the stub anatomy: why-sentence, `PRD refs: FR-NNN, US-NN` (literal PRD IDs, no paraphrase), `Prerequisites: TC-XXXXXX (<change-id>), …` using the real IDs captured from earlier creations (or `—`), `Outcome: <verb-led sentence>`.
4. **Write the draft `## Acceptance Criteria`** checklist: the Outcome sentence as the primary criterion, plus criteria derived from the referenced FRs' testable clauses. 2-5 items — a draft for `/tc-refine` and `/tc-plan` to sharpen, not a full spec.

Nothing more goes in the body. If `tickcats new` fails, print the error verbatim and STOP — a half-populated board is worse than none. If a ticket with the same change-id prefix already exists in any column (check `tickcats list` before creating), skip it and note "already exists" in the report.

All stubs stay in `backlog/`. This skill never moves tickets to `ready/` — that's `/tc-plan`'s promotion after full planning.

### Step 6: Self-review

Before reporting, verify the created board:

1. **PRD coverage** — every must-have FR (grep `^- FR-\d{3}: .* must-have$`) and every `### US-NN:` appears in at least one ticket's `PRD refs`. Uncovered must-have → FAIL.
2. **No dangling Prerequisites** — every `TC-XXXXXX` in any `Prerequisites:` line exists on the board. Grep the ticket files, don't trust memory.
3. **No cycles** — the Prerequisites graph is a DAG.
4. **Blocked-label consistency** — every ticket with a non-empty `Prerequisites:` line (pointing at not-done tickets) carries `[blocked]`; no ticket without unmet prerequisites carries it.
5. **Stub anatomy** — every ticket has frontmatter priority, `## Context` with PRD refs + Prerequisites + Outcome, and a non-empty `## Acceptance Criteria` checklist; every title carries `[to refine]` and the `Feat|Task: <change-id>:` shape.
6. **No invented tickets** — every ticket's PRD refs contain at least one real `FR-\d{3}` or `US-\d{2}`.

If any check fails, **fix the ticket files** (they're on disk — edit them), re-verify, and only then report. If a failure requires re-decomposition (uncovered FR, cycle), report the specific failure and STOP rather than papering over it.

Then show the board:

```bash
tickcats list
```

### Step 7: Hand off

Summarize:

```
═══════════════════════════════════════════════
  BOARD POPULATED — <N> ticket stubs in backlog
═══════════════════════════════════════════════
  Foundations (task):  <count>
  Slices (feat):       <count>
  Priorities:          P0: n  P1: n  P2: n  P3: n
  Blocked:             <count> ([blocked], waiting on prerequisites)
  PRD coverage:        <covered must-have FRs>/<total>
  North star:          <change-id> — <outcome>
═══════════════════════════════════════════════
```

Then **recommend a single next move**, not a menu:

- Default: `/tc-refine` — grill and sharpen the `[to refine]` stubs (scope, AC, priorities) before planning.
- If the stubs are already sharp (small PRD, high readiness score): `/tc-plan <ticket>` on the highest-priority unblocked stub — name it.

STOP. Do not chain into another skill automatically — the user reviews the board first.

## Critical guardrails

1. **No roadmap.md, ever.** The board is the decomposition artifact. If you find yourself drafting a roadmap document, stop.
2. **PRD is the source.** Every ticket traces to PRD IDs; the interview frames sequencing, it never grows the PRD.
3. **Vertical slices first.** Foundations are the only cross-cutting exception, and each must name what it unlocks.
4. **Stubs stay thin.** Context + draft AC only. Requirements, architecture, phases belong to `/tc-plan`.
5. **No estimates, no time units, no complexity scores.** Order lives in Prerequisites; priority in frontmatter.
6. **No low-level technical details.** No file paths, schemas, libraries in stubs — `/tc-plan`'s territory.
7. **Create in dependency order.** Real TC-XXXXXX ids in Prerequisites lines, never change-id placeholders left behind.
8. **Board is committed.** Step 0c removes `.tickcats/` from .gitignore; never re-add it.
9. **Never chain automatically.** Step 7 announces the next move; the user invokes it.
