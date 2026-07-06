---
name: tc-plan-review
description: >
  Tickcats-pipeline variant of 10x-plan-review. Review a refined ticket's
  implementation plan for substance, feasibility, and architectural fitness.
  Use when the user says "tc plan review", "review this ticket's plan",
  "is this ticket ready to implement", or references a .tickcats/ ticket and
  asks for plan feedback. Use AFTER /tc-plan, BEFORE /tc-implement. For the
  classic context/changes workflow use /10x-plan-review instead.
argument-hint: "[ticket-id-or-path]"
---

# Ticket Plan Review

Catch substance problems in a ticket's implementation plan before a line of code is written. A flawed plan costs hours — a flawed review costs minutes.

Where `/tc-impl-review` asks "did we build what the ticket planned?", this asks "will this ticket's plan actually work?"

The plan under review is the **ticket body** — a refined tickcats ticket with sections: Context, Requirements, Acceptance Criteria, Architecture, Current State, Implementation Plan, What We're NOT Doing, Progress. There is no separate review file: findings are appended to the ticket itself under `## Plan Review`.

Two modes:
- **Fresh review**: analyze → findings → interactive triage
- **Resume triage**: the ticket already has a `## Plan Review` with `Decision: PENDING` findings → jump to per-finding triage

## Tickcats facts

- Columns are folders: `.tickcats/{backlog,ready,doing,done,wont-do}/`. A ticket's state IS its folder — never a frontmatter field.
- Frontmatter keys: `title`, `id` (TC-XXXXXX), `priority` (P0–P3), `created`, `updated`.
- Labels live in title brackets: `[blocked]`, `[to refine]`.
- Dependencies are a convention: `Prerequisites: TC-XXX, TC-YYY` line in the ticket's Context section.
- Moves go through the CLI: `tickcats move <bare-filename> <from> <to>`.

## Input resolution

1. Argument is a path to a ticket file → use it.
2. Argument is a `TC-XXXXXX` id → `grep -rl "id: TC-XXXXXX" .tickcats/` to find the file.
3. No argument → list tickets in `.tickcats/ready/` (highest priority first) via AskUserQuestion; if `ready/` is empty, say so and STOP — nothing is planned for review.
4. `--quick` flag → document-only mode (skip Step 3).

The ticket is expected in `ready/`. If it's in `backlog/`, warn that it may not be fully planned yet but proceed if the user confirms. If it's in `done/` or `wont-do/`, refuse: "This ticket is closed. Plan reviews apply before implementation." If it's in `doing/`, warn that implementation has started and reviews may arrive too late — proceed only on confirmation.

If the ticket already has a `## Plan Review` section containing `Decision: PENDING` findings, offer to **resume triage** (skip to Step 7) instead of re-reviewing.

## Step 1: Load and internal consistency scan

Read the ticket file fully. Read `context/foundation/prd.md` (if present) for the FR/US references the Context section cites, and `context/foundation/lessons.md` if present — accepted rules are priors: a finding that restates a known recurring rule should weigh more, not less. Extract:

- **Context** — why, PRD refs, Prerequisites, outcome sentence
- **Requirements** and **Acceptance Criteria** — the promised end state
- **Architecture** — mermaid diagram(s) + placement prose
- **Current State** — research digest with file:line refs, constraints, gotchas
- **Implementation Plan** — phases: File/Intent/Contract per change, Success Criteria (Automated/Manual)
- **What We're NOT Doing** — scope boundaries
- **Progress** — the checkbox section `/tc-implement` will execute against

Before any code verification, check the plan against itself. These scans often catch the highest-value issues — problems the plan author discovered but didn't follow through on:

- **Contradiction**: does Current State document a limitation a phase ignores? Do "What We're NOT Doing" items reappear in phases? Does a phase assume behavior elsewhere acknowledged as broken?
- **Promise gap**: every capability promised in Requirements / Acceptance Criteria should have a backing phase. If an AC says "rate limiting works" but no phase builds it, the implementer hits a gap mid-build.
- **Contract breaks** (when the plan defines or uses API endpoints): trace data flow across endpoints — if step B needs a token/ID from step A, does A's response include it? Flag unresolved design decisions the implementer would have to guess at.
- **Contract surfaces touched**: if `docs/reference/contract-surfaces.md` exists, extract its H2 headings as surface names and `grep -F` the ticket body with one `-e <name>` per heading. For each hit, read that H2 section and verify the plan reports the surface's current shape accurately and flags any rename/schema change as breaking with a migration story. If the file doesn't exist, skip silently.

### Ticket-specific checks (mechanical — CRITICAL under Plan Completeness when violated)

- **Title label**: title must NOT contain `[to refine]` — a ready ticket claiming to need refinement is contradictory. If `[blocked]`, verify every Prerequisites ID is in `done/`; a blocked ticket doesn't belong in `ready/`.
- **Prerequisites exist**: for each `TC-XXXXXX` in the Context `Prerequisites:` line, `grep -rl "id: TC-XXXXXX" .tickcats/` must hit. Dangling IDs are CRITICAL — `/tc-implement`'s unblocking sweep depends on them.
- **AC ↔ plan**: every Acceptance Criteria checkbox maps to at least one phase; every phase serves at least one AC or a prerequisite of one.
- **Progress ↔ phases**: exactly one `## Progress` section; each `### Phase N: <name>` in the Implementation Plan has matching `- [ ] N.M <step>` rows in Progress; checkboxes appear ONLY in Acceptance Criteria and Progress — `- [ ]` anywhere else will confuse `/tc-implement`'s parser.
- **Architecture present and honest**: the Architecture section must contain at least one ` ```mermaid ` block, and the components/modules it names must correspond to the files the Implementation Plan touches. A diagram describing modules no phase touches (or phases touching areas absent from the diagram) is a Plan Completeness finding — the diagram is the map a repo-naive dev navigates by.

## Step 2: Grounding

Quick, no sub-agents:
- **Paths**: `ls -l` on ≥5 file paths from the plan's **File:** lines. Non-existent paths (that aren't explicitly marked as new files) are critical.
- **Symbols**: grep for specific functions/config keys named in Contract lines and Current State file:line refs.
- **Context ↔ plan consistency**: does the outcome sentence in Context match what the phases build?

Report inline: `Grounding: 5/5 paths ✓, 3/3 symbols ✓, prereqs 2/2 on board ✓`. Only escalate to a finding on failure.

## Step 3: Codebase verification (deep mode only)

Skip if `--quick`.

From Steps 1–2, identify the **3–5 riskiest claims** in the ticket — things that, if wrong, force significant rework. Launch **one** sub-agent (`subagent_type: "general-purpose"`) with three combined tasks:

1. **Verify the riskiest claims** against the actual code. For each: what does the code show, does it confirm or contradict the plan, with file:line evidence.
2. **Blast-radius sweep**: for functions, constants, or endpoints the plan modifies, grep for other callers/importers not mentioned in the ticket. These are files the plan doesn't know it's affecting.
3. **Pattern check** (only if the plan introduces new patterns): do existing files in the touched areas already solve this? Pattern proliferation is a common finding.

Give the sub-agent targeted questions with relevant file paths — don't dump the full ticket. A focused prompt finds more than a broad sweep because the agent knows what to look for.

## Step 4: Substance analysis

Analyze the plan against five dimensions. Only produce findings for real issues — don't pad with "no issues found".

### End-State Alignment
Walking phases sequentially, does the system reach the state Requirements and Acceptance Criteria describe? Could all success criteria pass while the goal remains unmet? Any "last mile" gap where the plan does 90% and stops short?

### Lean Execution
For each phase: "if I removed this, would the end state still be achievable?" Watch for premature abstraction, "while we're here" additions, framework-where-a-function-would-do, scope contradictions ("NOT Doing" items appearing in phases).

### Architectural Fitness
Does this fit the existing system? New patterns where existing ones would work (pattern proliferation). Clean module boundaries and correct dependency direction. High-blast-radius changes — phases touching many files across modules, changes to shared utilities. Vague "refactor as needed" that will spiral. Does the Architecture diagram's placement story hold up against Current State evidence?

### Blind Spots
What didn't the ticket consider? Error paths (only happy path described?), rollback story (phase 3 fails — can we revert?), resource/cost impact, default value changes, testing gaps, security boundaries. Remember the ticket must stand alone for a dev who has never seen the repo — unstated tribal knowledge is a blind spot here, not a nitpick.

### Plan Completeness
Is the ticket self-contained and actionable? File paths specific (not "somewhere in src/")? Changes at function/method level with Contracts? Success criteria with runnable commands? TBDs, TODOs, placeholder sections? All mechanical ticket-specific checks from Step 1 land here.

## Step 5: Compile findings

Each finding has:

- **ID**: F1, F2, F3… (if the ticket has prior reviews, continue numbering from the highest existing F-number so IDs stay unique across the ticket's history)
- **Severity**: CRITICAL / WARNING / OBSERVATION (how bad if ignored)
- **Impact**: LOW / MEDIUM / HIGH (how much focus the decision needs)
- **Dimension**: one of the five above
- **Title**: one line
- **Location**: ticket section or phase
- **Detail**: what's wrong with evidence — the ticket's claim vs. what's actually true, or what's missing
- **Fix options**: 1 or 2 (see below)

### Impact

Orthogonal to severity. A CRITICAL with LOW impact (obvious fix) is cheap to resolve; a WARNING with HIGH impact (unclear tradeoffs, wide blast) deserves careful thought.

| Impact | Meaning |
|---|---|
| 🏃 **LOW** | Quick decision. Fix is obvious and narrowly scoped. Safe to batch. |
| 🔎 **MEDIUM** | Worth pausing. Real tradeoff or non-trivial edit — think before deciding. |
| 🔬 **HIGH** | Architectural stakes. Wide blast radius, strategic implications, or unclear best path. |

### Fix options

Default to **one** fix. Only present two when there's a genuine tradeoff a smart reviewer would want to weigh — e.g. "minimal edit that patches the symptom" vs. "refactor that removes the class of problem". If you find yourself inventing a weak second option to satisfy a template, don't.

**LOW-impact findings**: just `Fix: [one line]`.

**MEDIUM/HIGH-impact findings**: each option gets:
```
[1-sentence approach] · Strength: [advantage, grounded in ticket/codebase evidence] · Tradeoff: [cost or risk] · Confidence: HIGH|MED|LOW — [1-line why] · Blind spot: [what we haven't verified, or "None significant"]
```

When offering two options, mark exactly one `⭐ Recommended`.

### Dimension verdicts and overall verdict

Each dimension: **PASS** / **WARNING** / **FAIL**.

- **SOUND** — safe to implement. All PASS, or PASS with minor warnings.
- **REVISE** — needs targeted fixes. Multiple warnings or 1 non-critical FAIL.
- **RETHINK** — fundamental problems. Multiple FAILs or wrong approach.

Sort findings by severity: CRITICAL → WARNING → OBSERVATION. Cap at 10 — consolidate related findings if you have more.

## Step 6: Present report and append to ticket

Print the report as plain text with box-drawing, same register as this example. Findings grouped by severity; omit empty groups. PASS dimensions appear only in the verdicts table, never as findings.

```
═══════════════════════════════════════════════════════════
  PLAN REVIEW: [Ticket title] (TC-XXXXXX)
  Mode: Deep / Quick  |  Date: YYYY-MM-DD
  Findings: [N critical] [N warnings] [N observations]
═══════════════════════════════════════════════════════════

  End-State Alignment    PASS    ✅
  Lean Execution         WARNING ⚠️   (1 finding)
  Architectural Fitness  PASS    ✅
  Blind Spots            FAIL    ❌   (1 finding)
  Plan Completeness      WARNING ⚠️   (1 finding)

  Grounding: 5/5 paths ✓, 3/3 symbols ✓, prereqs 2/2 on board ✓
  ► Overall: REVISE

═══════════════════════════════════════════════════════════
  CRITICAL FINDINGS ❌
═══════════════════════════════════════════════════════════

  F1 — No rollback for 50M-row backfill
  ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
    Severity:  ❌ CRITICAL
    Impact:    🔬 HIGH — architectural stakes; think carefully before deciding
    Dimension: Blind Spots
    Location:  Phase 3 — Database Changes

    Detail:
    Plan adds a NOT NULL column to users (50M rows) but no phase
    covers rollback if the backfill fails mid-way.

    Fix A ⭐ Recommended: Make column nullable + separate restartable backfill
      Strength:   Restartable; partial progress isn't destructive.
      Tradeoff:   Two deploys (add nullable → backfill → enforce NOT NULL).
      Confidence: HIGH — this exact approach shipped cleanly last quarter.
      Blind spot: Enforce step still needs its own rollback note.

    Fix B: Add explicit rollback phase with full table snapshot
      Strength:   Single deploy; rollback is atomic.
      Tradeoff:   50M-row snapshot is expensive in disk and lock time.
      Confidence: MEDIUM — snapshot cost unverified at this size.

═══════════════════════════════════════════════════════════
```

### Formatting rules for the report

- The **finding title line** holds only the ID and short title. Everything else goes below as labeled fields.
- **Always pair icons with a word** — `❌ CRITICAL`, never a bare icon.
- **Impact always carries its one-line meaning** (copy from the Impact table) so LOW/MEDIUM/HIGH is self-explanatory at the point of use.
- Severity, Impact, Dimension, Location each on their own line with aligned labels; Detail on its own line under a `Detail:` label.

### Appending the review to the ticket

There is NO separate review file. Append to the ticket body:

- If the ticket has no `## Plan Review` heading, add one at the bottom (above `## Impl Review` if that exists).
- Under it, append a dated subsection — always dated, so reruns stack chronologically:

```markdown
## Plan Review

### Review 2026-07-06

- **Mode**: Deep / Quick
- **Verdict**: SOUND / REVISE / RETHINK
- **Findings**: [N critical] [N warnings] [N observations]

| Dimension | Verdict |
|-----------|---------|
| End-State Alignment | PASS/WARNING/FAIL |
| Lean Execution | PASS/WARNING/FAIL |
| Architectural Fitness | PASS/WARNING/FAIL |
| Blind Spots | PASS/WARNING/FAIL |
| Plan Completeness | PASS/WARNING/FAIL |

Grounding: [grounding line]

#### F1 — No rollback for 50M-row backfill

- **Severity**: ❌ CRITICAL
- **Impact**: 🔬 HIGH — architectural stakes; think carefully before deciding
- **Dimension**: Blind Spots
- **Location**: Phase 3 — Database Changes
- **Detail**: Plan adds a NOT NULL column to users (50M rows) but no phase covers rollback if the backfill fails mid-way.
- **Fix A ⭐ Recommended**: Make column nullable + separate restartable backfill
- **Fix B**: Add explicit rollback phase with full table snapshot
- **Decision**: PENDING
```

If a `## Plan Review` section already exists, keep it — append only a new `### Review <YYYY-MM-DD>` subsection under it. Update the frontmatter `updated:` date. The `Decision: PENDING` lines enable resume mode.

Then ask:

```
question: "Plan review complete. How would you like to proceed?"
header: "Plan Review — [N] findings"
options:
  - label: "Triage findings"
    description: "Walk through each finding and decide."
  - label: "Triage later"
    description: "Review is saved in the ticket. Resume with /tc-plan-review <ticket>."
multiSelect: false
```

## Step 7: Interactive triage

### Resume mode

If entered via a ticket with an existing review: parse `#### F` headers in the latest `### Review` subsection, filter to `Decision: PENDING`. If none, say "All findings triaged" and stop.

### Triage loop

Walk findings in severity order. For each, AskUserQuestion:

```
question: "F[N] — [title]\n\nSeverity: [sev icon] [SEV]\nImpact: [impact icon] [LEVEL] — [meaning]\nDimension: [dim]\nLocation: [loc]\n\nDetail: [detail]\n\n[Fix A block]\n\n[Fix B block]"
header: "Finding [current] of [total remaining]"
options:
  - label: "Apply Fix A ⭐"
    description: "[Fix A one-liner]"
  - label: "Apply Fix B"
    description: "[Fix B one-liner]"
  - label: "Fix differently"
    description: "Different approach — let's discuss."
  - label: "Skip"
    description: "Not worth addressing now."
  - label: "Accept risk"
    description: "Understood — I'll handle during implementation."
  - label: "Disagree"
    description: "Not actually an issue — dismiss."
multiSelect: false
```

**With 1 fix option:** same, but replace "Apply Fix A/B" with a single "Fix in ticket".

**Handling responses:**
- **Apply Fix A/B / Fix in ticket**: show the exact edit (before/after), confirm briefly, then Edit the ticket's plan sections **in place** — Requirements, Architecture, Implementation Plan, Progress, whatever the fix touches. Minimal targeted edits; don't restructure the ticket for one finding. Keep AC/Progress/phases consistent after the edit (the Step 1 mechanical checks must still hold). Mark FIXED (record which fix).
- **Fix differently**: ask the preferred approach, apply the same way, mark FIXED.
- **Skip** → SKIPPED. **Accept risk** → ACCEPTED. **Disagree** → DISMISSED. Move on, don't argue.

After each decision, update the finding's `Decision:` line in the ticket's Plan Review section (e.g. `Decision: FIXED via Fix A (2026-07-06)`).

### Summary

```
═══════════════════════════════════════════════════════════
  TRIAGE COMPLETE
═══════════════════════════════════════════════════════════

  Fixed:     F1 (Fix A), F3   (2)
  Skipped:   F4               (1)
  Accepted:  F2               (1)
  Dismissed: F5               (1)

  ► Verdict after fixes: [updated if fixes changed it, e.g. REVISE → SOUND]
═══════════════════════════════════════════════════════════
```

Update the `**Verdict**:` line in the review subsection if fixes changed it.

### On RETHINK

If the final verdict (after any triage) is RETHINK, the ticket isn't ready for `/tc-implement` — offer to send it back to planning via AskUserQuestion:

- **"Move back to backlog"** → run `tickcats move <bare-filename> ready backlog`, re-add `[to refine]` to the frontmatter `title:` (labels live in ONE comma-separated bracket group at the very start of the title, e.g. `[blocked, to refine] Feat: …` — never two groups), and point the user at `/tc-plan` to replan with the review findings as input.
- **"Leave in ready"** → note in the review subsection that the RETHINK verdict was acknowledged and the ticket stays; the user owns the risk.

## Notes

- This is a **review** skill. Analyze and report — don't rewrite the plan unless asked during triage.
- Be specific. "Phase 3 introduces a second event system alongside the existing EventBus in `src/core/events.ts`" — not "architecture might have issues".
- Distinguish "won't work" (FAIL) from "could be better" (WARNING).
- If the plan is genuinely good, say so briefly, append a clean SOUND review, and stop. Don't manufacture findings.
- Impact is about *decision effort*, not *severity*.
- The ticket is the single artifact — every edit (fixes, decisions, verdicts) lands in the ticket file, never in a side file. Column moves only via `tickcats move`, and only in the RETHINK branch with user consent.
- During triage, keep momentum. User already read the report — present the finding, take the decision, move on.
